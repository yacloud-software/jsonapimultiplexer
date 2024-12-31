// client create: JSONApiMultiplexerClient
/*
  Created by /home/cnw/devel/go/yatools/src/golang.yacloud.eu/yatools/protoc-gen-cnw/protoc-gen-cnw.go
*/

/* geninfo:
   rendererv : 2
   filename  : golang.conradwood.net/apis/jsonapimultiplexer/jsonapimultiplexer.proto
   gopackage : golang.conradwood.net/apis/jsonapimultiplexer
   importname: ai_0
   clientfunc: GetJSONApiMultiplexer
   serverfunc: NewJSONApiMultiplexer
   lookupfunc: JSONApiMultiplexerLookupID
   varname   : client_JSONApiMultiplexerClient_0
   clientname: JSONApiMultiplexerClient
   servername: JSONApiMultiplexerServer
   gsvcname  : jsonapimultiplexer.JSONApiMultiplexer
   lockname  : lock_JSONApiMultiplexerClient_0
   activename: active_JSONApiMultiplexerClient_0
*/

package jsonapimultiplexer

import (
   "sync"
   "golang.conradwood.net/go-easyops/client"
)
var (
  lock_JSONApiMultiplexerClient_0 sync.Mutex
  client_JSONApiMultiplexerClient_0 JSONApiMultiplexerClient
)

func GetJSONApiMultiplexerClient() JSONApiMultiplexerClient { 
    if client_JSONApiMultiplexerClient_0 != nil {
        return client_JSONApiMultiplexerClient_0
    }

    lock_JSONApiMultiplexerClient_0.Lock() 
    if client_JSONApiMultiplexerClient_0 != nil {
       lock_JSONApiMultiplexerClient_0.Unlock()
       return client_JSONApiMultiplexerClient_0
    }

    client_JSONApiMultiplexerClient_0 = NewJSONApiMultiplexerClient(client.Connect(JSONApiMultiplexerLookupID()))
    lock_JSONApiMultiplexerClient_0.Unlock()
    return client_JSONApiMultiplexerClient_0
}

func JSONApiMultiplexerLookupID() string { return "jsonapimultiplexer.JSONApiMultiplexer" } // returns the ID suitable for lookup in the registry. treat as opaque, subject to change.

func init() {
   client.RegisterDependency("jsonapimultiplexer.JSONApiMultiplexer")
   AddService("jsonapimultiplexer.JSONApiMultiplexer")
}
