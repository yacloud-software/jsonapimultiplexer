package main

// this is the "autoconfig" for services that may be automatically exposed
// the new and preferred style for services with RPCInterceptor

import (
	"flag"
	"fmt"
	//	pb "golang.conradwood.net/apis/jsonapimultiplexer"
	lb "golang.conradwood.net/apis/h2gproxy"
	"golang.org/x/net/context"
	"gopkg.in/yaml.v2"
	//	"io/ioutil"
	"golang.conradwood.net/go-easyops/utils"
	"strings"
)

var (
	autodebug = flag.Bool("debug_auto", false, "debug autoconfig stuff")
)

type AutoRouterDef interface {
	Router() route
	Process(context.Context, *lb.ServeRequest) (*lb.ServeResponse, error)
}
type AutoConfigFile struct {
	Routes []*AutoRoutingEntry
}

type AutoRoutingEntry struct {
	ServicePrefix string
	ServicePath   string
	ServiceName   string
}

func (r *AutoRoutingEntry) String() string {
	return fmt.Sprintf("%s", r.ServicePath)
}

func ReadAutoRouting(fname string) (*AutoConfigFile, error) {
	fb, err := utils.ReadFile(fname)
	if err != nil {
		fmt.Printf("Failed to read file %s: %s\n", fname, err)
		return nil, err
	}
	gd := AutoConfigFile{}
	err = yaml.UnmarshalStrict(fb, &gd)
	if err != nil {
		fmt.Printf("Failed to parse file %s: %s\n", fname, err)
		return nil, err
	}
	return &gd, nil
}

func (a *AutoConfigFile) ByServicePrefix(prefix string) *AutoRoutingEntry {
	for _, c := range a.Routes {
		if c.ServicePrefix == prefix {
			return c
		}
	}
	if *debug {
		fmt.Printf("[autoconfig] No route for \"%s\"\n", prefix)
	}
	return nil
}

/*
 attempt to get an autorouterdef from local config file
 not having a match is not an error.
 invalid urls etc are not an error either
*/
func (a *AutoConfigFile) FindMatchByPathFromConfig(path string) (AutoRouterDef, error) {
	deb := "[autoconfig] "
	if *autodebug {
		fmt.Printf("%sFinding auto-config match for path \"%s\"...\n", deb, path)
	}

	if len(path) < 2 {
		if *autodebug {
			fmt.Printf("%sURL \"%s\" too short\n", deb, path)
		}
		return nil, nil
	}
	// find a servicematch:
	parts := strings.Split(path, "/")
	f := -1
	var are *AutoRoutingEntry
	prefix := ""
	for i, p := range parts {
		if p == "" {
			continue
		}
		re := a.ByServicePrefix(p)
		if re != nil {
			are = re
			prefix = p
			f = i
			break
		}
	}
	mn := "NONE"
	//	fmt.Printf("f=%d, len=%d\n", f, len(parts))
	if f >= (len(parts) - 1) {
		if *debug {
			fmt.Printf("Responding with htmlrouter()\n")
		}
		return GetHTMLRouter(), nil
	}
	if are == nil {
		if *debug {
			fmt.Printf("%sNo automatch found for \"%s\"\n", deb, path)
		}
		return nil, nil
	}
	if len(parts) > f+1 {
		mn = parts[f+1]
	}
	isInfo := strings.HasSuffix(path, "/Info")
	res := &AutoRouter{Info: isInfo, cfg: are, MethodName: mn, Prefix: prefix, html: false}
	for i := f + 2; i < len(parts); i++ {
		res.Options = append(res.Options, parts[i])
	}
	return res, nil

}

func (r *AutoRoutingEntry) Pref() string {
	return r.ServicePrefix
}

func (r *AutoRoutingEntry) Path() string {
	return r.ServicePath
}

func (r *AutoRoutingEntry) Name() string {
	return r.ServiceName
}


