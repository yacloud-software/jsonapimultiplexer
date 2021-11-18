package main

import (
	"flag"
	"fmt"
	"golang.conradwood.net/apis/common"
	lb "golang.conradwood.net/apis/h2gproxy"
	pb "golang.conradwood.net/apis/jsonapimultiplexer"
	"golang.conradwood.net/go-easyops/client"
	"golang.conradwood.net/go-easyops/tokens"
	"golang.conradwood.net/go-easyops/utils"
	//	rf "grpc/reflection/v1alpha"
	rf "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"

	"io"
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
		ctx := tokens.ContextWithToken()
		_, err := pb.GetJSONApiMultiplexerClient().Rescan(ctx, &common.Void{})
		utils.Bail("failed to rescan", err)
		os.Exit(0)

	}
	cl := pb.NewJSONApiMultiplexerClient(client.Connect("cnw/testing/cnw/latest"))
	sr := &lb.ServeRequest{}
	resp, err := cl.Serve(tokens.ContextWithToken(), sr)
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
	ctx := tokens.ContextWithToken()
	stream, err := c.ServerReflectionInfo(ctx)
	utils.Bail("Failed to query serverreflectioninfo", err)
	fmt.Printf("Receiving from %v\n", stream)
	for {
		m, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("stream.Recv failed with %v  %v\n", stream, err)
			break
		}

		fmt.Printf("%v\n", m)
	}

	fmt.Printf("%v\n", stream)
}
