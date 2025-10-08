package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	lb "golang.conradwood.net/apis/h2gproxy"
	"golang.conradwood.net/go-easyops/cache"
	"golang.yacloud.eu/apis/urlmapper"
	um "golang.yacloud.eu/apis/urlmapper"
)

var (
	um_cache = cache.New("jsonapi_urlmapper_cache", time.Duration(30)*time.Minute, 100)
	um_new   = flag.Bool("urlmapper_new_style", true, "use new style urlmapper lookup")
	mapper   um.URLMapperClient
)

type cache_entry struct {
	res AutoRouterDef
	err error
}

/*
attempt to get an autorouterdef from local config file
not having a match is not an error.
invalid urls etc are not an error either
*/
func (a *AutoConfigFile) FindMatchByPathFromMapper(ctx context.Context, req *lb.ServeRequest) (AutoRouterDef, error) {
	return urlmapper_newstyle(ctx, req)
}
func urlmapper_newstyle(ctx context.Context, req *lb.ServeRequest) (AutoRouterDef, error) {
	mr := &urlmapper.MappingRequest{Path: "https://" + req.Host + req.Path}
	m, err := urlmapper.GetURLMapperClient().GetMapping(ctx, mr)
	if err != nil {
		return nil, err
	}
	if !m.MappingFound {
		fmt.Printf("No urlmapper endpoint for path \"%s\"\n", mr.Path)
		return nil, nil
	}
	isinfo := false
	fmt.Printf("urlmapper match: %v\n", m)
	jmr := &urlmapper.JsonMappingResponse{}
	are := &AutoRoutingEntry{ServiceName: m.RegistryName}
	res := &AutoRouter{Mapping: jmr, Info: isinfo, cfg: are, MethodName: m.RPCName, Prefix: req.Path, html: false}
	return res, nil
}
