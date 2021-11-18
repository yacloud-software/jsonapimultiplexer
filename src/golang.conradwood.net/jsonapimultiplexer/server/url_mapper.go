package main

import (
	"context"
	"fmt"
	lb "golang.conradwood.net/apis/h2gproxy"
	"golang.conradwood.net/go-easyops/utils"
	um "golang.yacloud.eu/apis/urlmapper"
	"strings"
)

var (
	mapper um.URLMapperClient
)

/*
 attempt to get an autorouterdef from local config file
 not having a match is not an error.
 invalid urls etc are not an error either
*/
func (a *AutoConfigFile) FindMatchByPathFromMapper(ctx context.Context, req *lb.ServeRequest) (AutoRouterDef, error) {
	pathParts := strings.Split(req.Path, "/")
	if len(pathParts) < 2 {
		// too few parts in path, don't bother a lookup
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
		return nil, nil
	}
	apipath := req.Path[:idx]
	fmt.Printf("attempt to find match in mapper for \"%s%s (%s)\"\n", req.Host, req.Path, apipath)
	jmr, err := mapper.GetJsonMapping(ctx, &um.GetJsonMappingRequest{Domain: req.Host, Path: apipath})
	if err != nil {
		if *debug {
			fmt.Printf("jsonmapping error: %s\n", utils.ErrorString(err))
		}
		return nil, nil
	}
	mname := pathParts[len(pathParts)-1]
	fmt.Printf("urlmapper match: %v\n", jmr)
	are := &AutoRoutingEntry{ServiceName: jmr.GRPCService}
	res := &AutoRouter{Mapping: jmr, Info: isinfo, cfg: are, MethodName: mname, Prefix: apipath, html: false}
	return res, nil
}
