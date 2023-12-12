package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/golang/protobuf/jsonpb"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	lb "golang.conradwood.net/apis/h2gproxy"
	"golang.conradwood.net/go-easyops/utils"
	"html/template"
	"path/filepath"
	"strings"
)

func isBrowser(req *lb.ServeRequest) bool {
	ua := strings.ToLower(req.UserAgent)
	if strings.Contains(ua, "mozilla") {
		return true
	}
	return false
}

func serveServiceInfo(ctx context.Context, req *lb.ServeRequest, a *AutoRouter, t *RPCTarget) (*lb.ServeResponse, error) {
	if isBrowser(req) {
		return serveServiceInfoBrowser(ctx, req, a, t)
	}
	return serveServiceInfoApi(ctx, req, a, t)
}

// service 'ServiceInfo' as text
func serveServiceInfoApi(ctx context.Context, req *lb.ServeRequest, a *AutoRouter, t *RPCTarget) (*lb.ServeResponse, error) {
	res := &lb.ServeResponse{MimeType: "application/json", HTTPResponseCode: 200, GRPCCode: 0}
	// overview over all methods
	s := "ServiceInfo:\n"
	for _, m := range t.GetNonInternalMethods() {
		s = s + fmt.Sprintf("   %s\n", m.GetName())
	}
	res.Body = []byte(s)
	return res, nil
}

type ServiceInfoData struct {
	RPCTarget  *RPCTarget
	Request    *lb.ServeRequest
	AutoRouter *AutoRouter
	BaseURL    string
}

// service 'ServiceInfo' as html
func serveServiceInfoBrowser(ctx context.Context, req *lb.ServeRequest, a *AutoRouter, t *RPCTarget) (*lb.ServeResponse, error) {
	sid := &ServiceInfoData{
		Request:    req,
		RPCTarget:  t,
		AutoRouter: a,
		BaseURL:    "https://" + req.Host + "/" + filepath.Dir(req.Path),
	}
	res := &lb.ServeResponse{MimeType: "text/html", HTTPResponseCode: 200, GRPCCode: 0}
	tf, err := utils.ReadFile("templates/serviceinfo.html")
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New("serviceinfo").Parse(string(tf))
	if err != nil {
		return nil, err
	}
	var s bytes.Buffer
	err = tmpl.Execute(&s, sid)
	if err != nil {
		return nil, err
	}
	res.Body = s.Bytes()
	return res, nil
}

func serveInfoURL(m *desc.MethodDescriptor) (*lb.ServeResponse, error) {
	// info about an RPC
	res := &lb.ServeResponse{HTTPResponseCode: 200, GRPCCode: 0}
	fmt.Printf("[autorouter] M: %v\n", m)
	msg := dynamic.NewMessage(m.GetInputType())
	fill(msg)
	ms := &jsonpb.Marshaler{EmitDefaults: true, Indent: "  "}
	//input, err := ms.MarshalToString(msg)
	input, err := msg.MarshalJSONPB(ms)
	if err != nil {
		fmt.Printf("[autorouter] Marshal error: %s\n", err)
		return res, err
	}

	msg = dynamic.NewMessage(m.GetOutputType())
	output, err := ms.MarshalToString(msg)
	if err != nil {
		fmt.Printf("[autorouter] Marshal error: %s\n", err)
		return res, err
	}
	res.Body = []byte(fmt.Sprintf("Input:\n%s\nOutput:\n%s\n", input, output))
	fmt.Printf("[autorouter] I: #%v\n", string(input))
	fmt.Printf("[autorouter] O: #%v\n", output)
	return res, nil
}

func fill(m *dynamic.Message) {
	for _, f := range m.GetKnownFields() {
		ms := f.GetMessageType()
		if ms == nil {
			continue
		}
		fmt.Printf("Fields: %s\n", f.GetName())
		msg := dynamic.NewMessage(ms)
		m.SetField(f, msg)
		fill(msg)
	}
}



