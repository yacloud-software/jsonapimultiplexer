package main

import (
	"flag"
	"fmt"
	"golang.conradwood.net/apis/common"
	lb "golang.conradwood.net/apis/h2gproxy"
	pb "golang.conradwood.net/apis/jsonapimultiplexer"
	"golang.conradwood.net/go-easyops/authremote"
	"golang.conradwood.net/go-easyops/client"
	"golang.conradwood.net/go-easyops/utils"
	//	rf "grpc/reflection/v1alpha"
	"github.com/jhump/protoreflect/grpcreflect"
	rf "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"os"
)

var (
	rescan  = flag.Bool("rescan", false, "initiate rescan of targets, e.g. after redeployment of a service")
	inspect = flag.String("inspect", "", "inspect a service (path/name) by reflection")
)

func main() {
	flag.Parse()
	if *inspect != "" {
		doinspect()
		os.Exit(0)
	}
	if *rescan {
		ctx := authremote.Context()
		_, err := pb.GetJSONApiMultiplexerClient().Rescan(ctx, &common.Void{})
		utils.Bail("failed to rescan", err)
		os.Exit(0)

	}
	cl := pb.NewJSONApiMultiplexerClient(client.Connect("cnw/testing/cnw/latest"))
	sr := &lb.ServeRequest{}
	resp, err := cl.Serve(authremote.Context(), sr)
	if err != nil {
		fmt.Printf("Failed: %s\n", err)
		os.Exit(10)
	}
	fmt.Printf("done: %v", resp)
}

func doinspect() {
	con := client.Connect(*inspect)
	fmt.Printf("Connected to %s\n", *inspect)
	c := rf.NewServerReflectionClient(con)
	ctx := authremote.Context()
	//	stream, err := c.ServerReflectionInfo(ctx)
	//	utils.Bail("Failed to query serverreflectioninfo", err)
	rc := grpcreflect.NewClient(ctx, c)
	ls, err := rc.ListServices()
	utils.Bail("failed to list", err)
	for _, s := range ls {
		fmt.Printf(" Service: %s\n", s)
	}

}





