package main

import (
	"fmt"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"golang.conradwood.net/go-easyops/client"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	rf "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	//	_ "google.protobuf.descriptor"
	"strings"
	"time"
)

var (
	rpctargets []*RPCTarget
)

type RPCTarget struct {
	serviceName string
	con         *grpc.ClientConn
	methods     []*desc.MethodDescriptor
	lastProbed  time.Time
	closed      bool
	lastUsed    time.Time
}

func (t *RPCTarget) ServiceName() string {
	return t.serviceName
}
func (t *RPCTarget) GetNonInternalMethods() []*desc.MethodDescriptor {
	var res []*desc.MethodDescriptor
	for _, m := range t.methods {
		if isInternal(m) {
			continue
		}
		res = append(res, m)
	}
	return res
}

// get method by name case INSENSITIVE
func (t *RPCTarget) GetMethod(name string) *desc.MethodDescriptor {
	if t.methods == nil {
		return nil
	}
	s := strings.ToLower(name)
	for _, m := range t.methods {
		if m == nil {
			continue
		}
		if strings.ToLower(m.GetName()) == s {
			return m
		}
	}
	return nil
}

func clearCache() {
	tlock.Lock()
	orpc := rpctargets
	rpctargets = make([]*RPCTarget, 0)
	tlock.Unlock()
	for _, t := range orpc {
		if t == nil {
			continue
		}
		if t.con == nil {
			continue
		}
		t.con.Close()
		t.con = nil
		t.closed = true
	}
	um_cache.Clear() // clear urlmapper cache
}

// target, preferably cached, for a servicename
func getRPCTarget(ctx context.Context, serviceName string) *RPCTarget {
	tlock.Lock()
	for _, t := range rpctargets {
		if t == nil {
			continue
		}
		if t.serviceName == serviceName {
			tlock.Unlock()
			t.lastUsed = time.Now()
			return t
		}
	}
	tlock.Unlock()
	res := &RPCTarget{closed: true, serviceName: serviceName, lastProbed: time.Now(), lastUsed: time.Now()}

	// connect & analyse target w/o holding lock
	fmt.Printf("Dialing %s...\n", serviceName)
	res.con = client.Connect(serviceName)
	fmt.Printf("Connected to %s\n", serviceName)
	err := res.ProbeRPC(ctx)
	if err != nil {
		fmt.Printf("Could not probe target %s: %s\n", serviceName, err)
		return nil
	}

	// reacquire lock, make sure nobody has got this service in between and store it
	tlock.Lock()
	// check if another thread managed to insert one whilst we were probing
	for _, t := range rpctargets {
		if t.serviceName == serviceName {
			tlock.Unlock()
			t.lastUsed = time.Now()
			return t
		}
	}
	res.closed = false
	rpctargets = append(rpctargets, res)
	tlock.Unlock()

	return res
}
func (t *RPCTarget) close() error {
	tlock.Lock()
	defer tlock.Unlock()
	c := t.con
	if c != nil {
		c.Close()
	}
	t.closed = true
	return nil
}
func (t *RPCTarget) Remove() {
	var rn []*RPCTarget
	tlock.Lock()
	for _, r := range rpctargets {
		if r == t {
			continue
		}
		rn = append(rn, r)
	}
	rpctargets = rn
	tlock.Unlock()
	t.close()
}

// probe RPC (get method descriptors)
func (t *RPCTarget) ProbeRPC(ctx context.Context) error {
	c := rf.NewServerReflectionClient(t.con)
	rc := grpcreflect.NewClient(ctx, c)

	ls, err := rc.ListServices()
	if err != nil {
		return fmt.Errorf("Failed to query serverreflectioninfo: %s", err)
	}
	for _, svc := range ls {
		if svc == "grpc.reflection.v1alpha.ServerReflection" {
			// do not resolve reflection service itself
			continue
		}
		sd, err := rc.ResolveService(svc)
		if err != nil {
			return fmt.Errorf("Failed to resolve service (%s): %s", svc, err)
		}
		for _, method := range sd.GetMethods() {
			t.methods = append(t.methods, method)
		}
	}
	t.lastProbed = time.Now()
	return nil
}

