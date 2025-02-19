package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	lb "golang.conradwood.net/apis/h2gproxy"
	"golang.conradwood.net/go-easyops/cache"
	"golang.conradwood.net/go-easyops/errors"
	"golang.conradwood.net/go-easyops/utils"
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
	if *um_new {
		return urlmapper_newstyle(ctx, req)
	}
	return urlmapper_oldstyle(ctx, req)
}
func urlmapper_newstyle(ctx context.Context, req *lb.ServeRequest) (AutoRouterDef, error) {
	m, err := urlmapper.GetURLMapperClient().GetMapping(ctx, &urlmapper.MappingRequest{Path: "https://" + req.Host + req.Path})
	if err != nil {
		return nil, err
	}
	if !m.MappingFound {
		fmt.Printf("No endpoint for path \"%s\"\n", req.Path)
		return nil, errors.NotFound(ctx, "endpoint not found")
	}
	isinfo := false
	fmt.Printf("urlmapper match: %v\n", m)
	jmr := &urlmapper.JsonMappingResponse{}
	are := &AutoRoutingEntry{ServiceName: m.RegistryName}
	res := &AutoRouter{Mapping: jmr, Info: isinfo, cfg: are, MethodName: m.RPCName, Prefix: req.Path, html: false}
	return res, nil
}

/*
**************************************************************************************************************

	old style urlmapper call

**************************************************************************************************************
*/
func urlmapper_oldstyle(ctx context.Context, req *lb.ServeRequest) (AutoRouterDef, error) {
	key := fmt.Sprintf("%s_%s", req.Host, req.Path)
	if *debug {
		fmt.Printf("cache key: \"%s\"\n", key)
	}
	ao := um_cache.Get(key)
	if ao != nil {
		ax := ao.(*cache_entry)
		return ax.res, ax.err
	}
	pathParts := strings.Split(req.Path, "/")
	if len(pathParts) < 2 {
		// too few parts in path, don't bother a lookup
		um_cache.Put(key, &cache_entry{})
		return nil, nil
	}
	if mapper == nil {
		mapper = um.GetURLMapperClient()
	}
	isinfo := false
	if strings.HasSuffix(req.Path, "/Info") {
		isinfo = true
		np := strings.TrimSuffix(req.Path, "/Info")
		if *debug {
			fmt.Printf("path %s contains info. new path: \"%s\"\n", req.Path, np)
		}
		req.Path = np
		pathParts = pathParts[:len(pathParts)-1]
	}

	idx := strings.LastIndex(req.Path, "/")
	if idx == -1 {
		// url has no '/'
		um_cache.Put(key, &cache_entry{})
		return nil, nil
	}
	apipath := req.Path[:idx]
	fmt.Printf("attempt to find match in mapper for \"%s%s (%s)\"\n", req.Host, req.Path, apipath)
	jmr, err := mapper.GetJsonMapping(ctx, &um.GetJsonMappingRequest{Domain: req.Host, Path: apipath})
	if err != nil {
		if *debug {
			fmt.Printf("jsonmapping error: %s\n", utils.ErrorString(err))
		}
		um_cache.Put(key, &cache_entry{})
		return nil, nil
	}
	mname := pathParts[len(pathParts)-1]
	fmt.Printf("urlmapper match: %v\n", jmr)
	are := &AutoRoutingEntry{ServiceName: jmr.GRPCService}
	res := &AutoRouter{Mapping: jmr, Info: isinfo, cfg: are, MethodName: mname, Prefix: apipath, html: false}
	um_cache.Put(key, &cache_entry{res: res})
	return res, nil
}
