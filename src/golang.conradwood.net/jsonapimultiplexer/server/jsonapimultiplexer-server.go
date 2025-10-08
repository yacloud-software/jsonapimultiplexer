package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"golang.conradwood.net/apis/common"
	lb "golang.conradwood.net/apis/h2gproxy"
	pb "golang.conradwood.net/apis/jsonapimultiplexer"
	"golang.conradwood.net/apis/quota"
	"golang.conradwood.net/go-easyops/auth"
	"golang.conradwood.net/go-easyops/client"
	"golang.conradwood.net/go-easyops/errors"
	"golang.conradwood.net/go-easyops/prometheus"
	"golang.conradwood.net/go-easyops/server"
	"golang.conradwood.net/go-easyops/utils"
	"google.golang.org/grpc"
)

var (
	port     = flag.Int("port", 10001, "The grpc server port")
	httpPort = flag.Int("http_port", 0, "set to non-0 to enable the debug http server port (jsonapi serving)")

	debug            = flag.Bool("debug", false, "debug mode enable")
	enablequota      = flag.Bool("enable_quota", false, "enable quotas for services")
	autoconfigfile   = flag.String("autofile", "configs/autorouting.yaml", "filename of a yaml file defining services with automatic routing")
	srv              *echoServer
	qc               quota.QuotaServiceClient
	autocfg          *AutoConfigFile
	forwardedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jsonapi_forwarded_rpc_counter",
			Help: "V=1 U=ops DESC=total counter of number of times we invoked an RPC call",
		},
		[]string{"type"},
	)
	failedForwardedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jsonapi_failed_rpc_counter",
			Help: "V=1 U=ops DESC=total counter of number of times we invoked an RPC call and it failed",
		},
		[]string{"type"},
	)
	prefixCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jsonapi_prefix_counter",
			Help: "V=1 U=ops DESC=total counter of number of times a given prefix was requested",
		},
		[]string{"prefix", "userid", "useremail"},
	)
)

type echoServer struct {
}

type route interface {
	Pref() string
	Path() string
	Name() string
}

func main() {
	var err error
	flag.Parse()
	server.SetHealth(common.Health_STARTING)

	prometheus.MustRegister(forwardedCounter, failedForwardedCounter, prefixCounter)
	utils.Bail("failed to read configfile", err)
	autocfg, err = ReadAutoRouting(*autoconfigfile)
	utils.Bail("failed to read autoconfigfile", err)
	go reprobe() // start the loop to refresh methods in exported services
	srv = new(echoServer)
	sd := server.NewServerDef()
	sd.SetPort(*port)
	sd.SetOnStartupCallback(startup)
	sd.SetRegister(server.Register(
		func(server *grpc.Server) error {
			pb.RegisterJSONApiMultiplexerServer(server, srv)
			return nil
		},
	))
	go startHTTP()
	err = server.ServerStartup(sd)
	utils.Bail("Unable to start server", err)
	os.Exit(0)
}
func startup() {
	server.SetHealth(common.Health_READY)
}

/************************************
* grpc functions
************************************/
func (e *echoServer) Rescan(ctx context.Context, req *common.Void) (*common.Void, error) {
	clearCache()
	return &common.Void{}, nil
}

// this is called by H2GProxy
func (e *echoServer) Serve(ctx context.Context, req *lb.ServeRequest) (*lb.ServeResponse, error) {
	return e.ServeHTML(ctx, req)
}
func (e *echoServer) ServeHTML(ctx context.Context, req *lb.ServeRequest) (*lb.ServeResponse, error) {
	if *debug {
		fmt.Printf("Request to serve %s/%s\n", req.Host, req.Path)
	}
	if req.Method == "OPTIONS" {
		resp := &lb.ServeResponse{Body: []byte("accepting all options")}
		return resp, nil
	}
	resp := &lb.ServeResponse{Body: []byte("hi")}
	are, err := autocfg.FindMatchByPathFromMapper(ctx, req)
	if err != nil {
		fmt.Printf("Failed to find best match by path in urlmapper.\n")
		return nil, err
	}
	/*
		if are == nil {
			//deprecated code path
			are, err = autocfg.FindMatchByPathFromConfig(req.Path)
			if err != nil {
				fmt.Printf("Failed to find best match by path in autocfg.\n")
				return nil, err
			}
		}
	*/
	if are == nil {
		return nil, errors.InvalidArgs(ctx, "no such endpoint", "endpoint not found: (path=%s)", req.Path)
	}

	if *enablequota {
		if !withinQuota(ctx, are.Router()) {
			resp.HTTPResponseCode = 429
			resp.GRPCCode = 429 // cannot use r.GrpcCode because r == nil
			resp.Body = []byte("your request exceeds your allocated quota. please try later")
			return resp, nil
		}
	}
	sp, err := are.Process(ctx, req)
	if *debug && err != nil {
		fmt.Printf("Failed to process: %s\n", utils.ErrorString(err))
	}
	return sp, err

}

func singleHeader(req *lb.ServeRequest, name string) string {
	for _, h := range req.Headers {
		if (h.Name == name) && (len(h.Values) > 0) {
			return h.Values[0]
		}
	}
	return ""
}

// return true if we're allowed to continue
func withinQuota(ctx context.Context, re route) bool {
	if !*enablequota {
		return true
	}
	if qc == nil {
		qc = quota.NewQuotaServiceClient(client.Connect("quota.QuotaService"))
	}
	uid := ""
	u := auth.GetUser(ctx)
	if u != nil {
		uid = u.ID
	}
	req := &quota.QuotaCall{
		SourceIP:     "0.0.0.0",
		User:         uid,
		TargetSystem: "jsonapi",
		TargetCall:   re.Name(),
	}
	resp, err := qc.AskForAccess(ctx, req)
	if err != nil {
		if *debug {
			fmt.Printf("Failed to check quota: %s\n", err)
		}
		return true
	}
	if resp.OverLimit {
		return false
	}
	return true
}
func incPrefix(ctx context.Context, p string) {
	user := auth.GetUser(ctx)
	email := ""
	id := ""
	if user != nil {
		email = user.Email
		id = user.ID
	}
	prefixCounter.With(prometheus.Labels{"prefix": p, "userid": id, "useremail": email}).Inc()
}

func ContextUserInfo(ctx context.Context) string {
	userid := ""
	u := auth.GetUser(ctx)
	if u != nil {
		userid = fmt.Sprintf("user: %s", u.String())
	}
	return fmt.Sprintf("CONTEXT - user=%s", userid)
}
