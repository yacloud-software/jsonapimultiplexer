package main

import (
	"fmt"
	sh "golang.conradwood.net/apis/htmlserver"
	lb "golang.conradwood.net/apis/h2gproxy"
	"golang.conradwood.net/go-easyops/client"
	"golang.org/x/net/context"
)

var (
	htmlr   *HTMLRouter
	hserver sh.HTMLServerServiceClient
)

type HTMLRouter struct {
	route route
}
type HTMLRoute struct {
}

func (h *HTMLRoute) Pref() string {
	return "html_pref"
}
func (h *HTMLRoute) Path() string {
	return "html_path"
}
func (h *HTMLRoute) Name() string {
	return "html_name"
}

func GetHTMLRouter() *HTMLRouter {
	if htmlr != nil {
		return htmlr
	}
	res := &HTMLRouter{
		route: &HTMLRoute{},
	}
	if hserver == nil {
		hserver = sh.NewHTMLServerServiceClient(client.Connect("htmlserver.HTMLServerService"))
	}
	htmlr = res
	return res
}
func (a *HTMLRouter) Router() route {
	return a.route
}
func (a *HTMLRouter) Process(ctx context.Context, req *lb.ServeRequest) (*lb.ServeResponse, error) {
	code := uint32(200)
	sp := &lb.ServeResponse{
		HTTPResponseCode: code,
		GRPCCode:         0,
		Body:             []byte(fmt.Sprintf("fooooo (%d)", code)),
		MimeType:         "text/plain",
	}
	sreq := &lb.ServeRequest{
		Headers:    req.Headers,
		Body:       req.Body,
		Path:       req.Path,
		Method:     req.Method,
		Parameters: req.Parameters,
	}
	r, err := hserver.ServeHTML(ctx, sreq)
	if err != nil {
		sp.HTTPResponseCode = 500
		sp.Body = []byte("Processing failure")
		return nil, err
	}
	sp.HTTPResponseCode = r.HTTPResponseCode
	sp.GRPCCode = r.GRPCCode
	sp.Body = r.Body
	sp.MimeType = r.MimeType
	return sp, nil
}
