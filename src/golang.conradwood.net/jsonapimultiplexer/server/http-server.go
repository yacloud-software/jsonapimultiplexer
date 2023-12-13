package main

import (
	"fmt"
	"golang.conradwood.net/apis/auth"
	lb "golang.conradwood.net/apis/h2gproxy"
	"golang.conradwood.net/go-easyops/authremote"
	"golang.conradwood.net/go-easyops/server"
	"golang.conradwood.net/go-easyops/utils"
	//	"golang.org/x/net/context"
	//	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"io/ioutil"
	"net/http"
)

// *****************************************************************
// internal http server
// for testing and as long as LBProxy doesn't support
// calling us via gRPC
// *****************************************************************
func startHTTP() {
	if *httpPort == 0 {
		return
	}
	fmt.Printf("Starting internal debug http server on port %d\n", *httpPort)
	http.HandleFunc("/", mainRequest)
	sd := server.NewHTMLServerDef("jsonapimultiplexer.htmlinterface")
	sd.SetPort(*httpPort)
	server.AddRegistry(sd)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *httpPort), nil)
	utils.Bail("Failed to listen", err)
}

func mainRequest(w http.ResponseWriter, r *http.Request) {
	/*
		user, err := getUser()
		if err != nil {
			fmt.Printf("%s\n", err)
			return
		}
	*/
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s := fmt.Sprintf("Error serving html: %s", err)
		w.Write([]byte(s + "\n"))
		fmt.Println(s)
		return
	}
	sr := &lb.ServeRequest{
		Path:   r.URL.Path,
		Body:   string(body),
		Method: r.Method,
	}
	for k, v := range r.Header {
		fmt.Printf("Header: %v,%v\n", k, v)
		sr.Headers = append(sr.Headers, &lb.Header{Name: k, Values: v})
	}

	// add the query parameters
	r.ParseForm()
	for k, va := range r.Form {
		if (va == nil) || (len(va) == 0) {
			continue
		}
		v := va[0]
		fmt.Printf("KV: %s = %s\n", k, v)
		p := &lb.Parameter{Name: k, Value: v}
		sr.Parameters = append(sr.Parameters, p)
	}
	// end query parameters

	// creating token because this is called during a http request (no context in http)
	tctx := authremote.Context()
	md, _ := metadata.FromOutgoingContext(tctx)
	tctx = metadata.NewIncomingContext(tctx, md)

	fmt.Printf("internal http request: Achtung, Achtung\n")
	response, e := srv.Serve(tctx, sr)
	if e != nil {
		s := fmt.Sprintf("Error serving html: %s", e)
		w.Write([]byte(s + "\n"))
		fmt.Println(s)
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		// For local development using http server, set 200 response
		if (*r).Method == "OPTIONS" {
			w.WriteHeader(int(200))
			return
		}

		w.Write(response.Body)
		w.Write([]byte("\n"))
		w.WriteHeader(int(response.HTTPResponseCode))

	}

}
func getUser() (*auth.User, error) {
	res := &auth.User{
		ID:        "95",
		Email:     "cnw@conradwood.net", // not guru on purpose, it needs auth layer!!
		FirstName: "fixme",
		LastName:  "jsonapimultiplexer",
		Abbrev:    "cnw",
	}
	return res, nil
}

// *****************************************************************
// end internal test server
// *****************************************************************





