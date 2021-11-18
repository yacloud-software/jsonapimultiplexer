package main

// this is the "autoconfig" for services that may be automatically exposed
// the new and preferred style for services with RPCInterceptor

// we have a basic enquiry mechanism:
// /foo/[service]/ServiceInfo -> return list of methods
// /foo/[service]/Method/Info -> return input/output json for this method

import (
	"flag"
	"fmt"
	"github.com/golang/protobuf/jsonpb"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"golang.conradwood.net/go-easyops/prometheus"
	um "golang.yacloud.eu/apis/urlmapper"
	//	pb "golang.conradwood.net/apis/jsonapimultiplexer"
	lb "golang.conradwood.net/apis/h2gproxy"
	"golang.conradwood.net/go-easyops/tokens"
	"golang.conradwood.net/go-easyops/utils"
	"golang.org/x/net/context"
	"strings"
	"sync"
	"time"
)

var (
	tlock         sync.Mutex
	reprobe_delay = flag.Int("reprobe_delay", 300, "delay in `seconds` between reprobes for API changes")
	idle_timeout  = flag.Int("idle_timeout", 600, "idle timeout in `seconds` after which to stop probing a service and removing it from cache")
)

type AutoRouter struct {
	Mapping    *um.JsonMappingResponse // only set if we come from urlmapper
	Info       bool                    // true if we want "info" about this entry
	Prefix     string                  // prefix we matched on
	cfg        *AutoRoutingEntry
	MethodName string
	Options    []string // options are set when we match a URL (see autoconfig.go)
	html       bool     // true if it's not a gRPC json thing but an html thing
}

func (a *AutoRouter) Router() route {
	return a.cfg
}

// main processing function -> h2gproxy gives us a serverequest and we return a serveresponse to lbproxy
func (a *AutoRouter) Process(ctx context.Context, req *lb.ServeRequest) (*lb.ServeResponse, error) {
	if *debug {
		fmt.Printf("[autorouter] Processing service: %s, method \"%s\"\n", a.cfg.ServiceName, a.MethodName)
	}

	t := getRPCTarget(ctx, a.cfg.ServiceName)
	if t == nil {
		fmt.Printf("[autorouter] No target for %s\n", a.cfg.ServiceName)
		return nil, fmt.Errorf("failed to call %s/%s:, no target\n", a.cfg.ServiceName, a.MethodName)
	}
	res := &lb.ServeResponse{MimeType: "application/json", HTTPResponseCode: 200, GRPCCode: 0}
	if a.html {
		res.MimeType = "text/html"
	}

	if a.MethodName == "ServiceInfo" {
		return serveServiceInfo(ctx, req, a, t)
	}
	m := t.GetMethod(a.MethodName)
	if m == nil {
		if *debug {
			fmt.Printf("[autorouter] no such method \"%s\" in %s\n", a.MethodName, a.Prefix)
		}
		hm := GetHTMLRouter()
		return hm.Process(ctx, req)
		//		return nil, fmt.Errorf("No method \"%s\" in service \"%s\"\n", a.MethodName, a.cfg.ServiceName)
	}
	if isInternal(m) {
		return nil, fmt.Errorf("No method \"%s\" in service \"%s\"\n", a.MethodName, a.cfg.ServiceName)
	}

	// if URL is '/info' -> serve the response/request syntax
	if a.Info {
		return serveInfoURL(m)
	}

	incPrefix(ctx, a.Prefix)
	// convert json into protobufs, make the RPC call and return the proto (as json)
	ax, err := t.call(ctx, m, req)
	if err != nil {
		fmt.Printf("[autorouter] Failure calling %s.%s: %s\n", a.cfg.ServiceName, a.MethodName, utils.ErrorString(err))
		return nil, err
	}
	res.Body = []byte(ax)

	return res, nil
}

// given a querystring of ?foo=bar we transfer the value bar into field foo
func parseQueryParametersToProto(req *lb.ServeRequest, m *dynamic.Message) error {
	// do clever stuff with req.Paramater and m.SetField(foo)
	for _, p := range req.Parameters {
		f := lookupFieldByName(m, p.Name)
		if f == nil {
			// no such field. ignore or error?
			if *debug {
				fmt.Printf("[autorouter] No such field: %s (value=%s) in %s\n", p.Name, p.Value, m.XXX_MessageName())
			}
			continue
		}
		err := setField(m, f, p.Value)
		if err != nil {
			// no field 'foo'
			// continue == ignore error, otherwise return error here
			if *debug {
				fmt.Printf("[autorouter] Unable to set field %s to %s: %s\n", p.Name, p.Value, err)
			}
			continue
		}
	}
	return nil
}

// find a fieldname by name. by doing a case-insensitive match
func lookupFieldByName(m *dynamic.Message, name string) *desc.FieldDescriptor {
	s := strings.ToLower(name)
	for _, f := range m.GetKnownFields() {
		mn := f.GetName()
		if strings.ToLower(mn) == s {
			return f
		}
	}
	return nil
}

// call a grpc with json input and return the protobuf as json
func (r *RPCTarget) call(ctx context.Context, method *desc.MethodDescriptor, req *lb.ServeRequest) (string, error) {
	input := req.Body
	if *debug {
		fmt.Printf("Body: \"%s\"\n", string(input))
	}
	var err error
	// create the proto
	svcname := method.GetService().GetFullyQualifiedName()
	msgdesc := method.GetInputType()
	msg := dynamic.NewMessage(msgdesc)
	args := msg
	// fill the proto with json input, if any
	if input != "" {
		um := jsonpb.Unmarshaler{AllowUnknownFields: false}
		err = um.Unmarshal(strings.NewReader(input), msg)
		if err != nil {
			fmt.Printf("Input: %s\n", input)
			return "", fmt.Errorf("Failed to parse json: %s", err)
		}
	}
	err = parseQueryParametersToProto(req, msg)
	if err != nil {
		return "", fmt.Errorf("Failed to parse parameters: %s", err)
	}

	if *debug {
		fmt.Printf("[autorouter] RPCCall to %s.%s (input:%s)\n", svcname, method.GetName(), string(input))
	}
	// create an empty response object
	res := dynamic.NewMessage(method.GetOutputType())
	fqmn := fmt.Sprintf("/%s/%s", svcname, method.GetName())
	// make the call (which fills our response object)
	forwardedCounter.With(prometheus.Labels{"type": "static"}).Inc()
	err = r.con.Invoke(ctx, fqmn, args, res)
	if err != nil {
		failedForwardedCounter.With(prometheus.Labels{"type": "static"}).Inc()
		if *debug {
			fmt.Printf("[autorouter] error from %s: %s", fqmn, err)
		}
		return "", err
	}
	// turn the response into json
	ms := jsonpb.Marshaler{EmitDefaults: true}
	b, err := ms.MarshalToString(res)
	if err != nil {
		if *debug {
			fmt.Printf("[autorouter] invalid json from %s: %s", fqmn, err)
		}
		return "", err
	}
	if *debug {
		fmt.Printf("[autorouter] result from %s\n", fqmn)
		fmt.Printf("[autorouter] result is: %s\n", b)
	}
	return b, nil
}

func isInternal(m *desc.MethodDescriptor) bool {
	if m.GetName() == "ServerReflectionInfo" {
		return true
	}
	return false
}

// periodically refresh list of known methods via grpc reflection
// this would be way better to handle with connections, e.g. a channel to the frameworks' client.Resolver
// another day perhaps...
func reprobe() {
	wait := true
	for {
		if wait {
			n := *reprobe_delay
			if *idle_timeout < *reprobe_delay {
				n = *idle_timeout
			}
			if n < 60 {
				time.Sleep(time.Duration(n) * time.Second)
			} else {
				utils.RandomStall(n)
			}
		}
		if *debug {
			fmt.Printf("[autorouter] Reprobing...\n")
		}
		wait = true
		for _, t := range rpctargets {
			if time.Since(t.lastUsed).Seconds() > float64(*idle_timeout) {
				if *debug {
					fmt.Printf("[autorouter] Removing %v, it's idle\n", t)
				}
				t.Remove()
				wait = false
				break
			}
			if time.Since(t.lastProbed).Seconds() > float64(*reprobe_delay) {
				// creating token because this is called on server start up
				t.ProbeRPC(tokens.ContextWithToken())
			}
		}
	}
}
