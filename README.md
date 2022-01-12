# JSON API Multiplexer

## Purpose of this module

There is a lot of complicated stuff that needs
to be considered when exposing apis. Neither hes nor pinpoint
have APIs our clients find satisfying (slow, unpredictable, complicated
are some descriptions we here). We (Guru) also find the APIs awkward because
they are hard to extend. (It involves updating either HES or Pinpoint
rather than adding a new module)

Thus this module solves some of those issues:

## It is single forwarding target for LBProxy

As per the stated goal of LBProxy for it to not become
more complicated as is, this provides a single grpc service
which lbproxy can call (using go-framework) for all
json/rest apis we want to expose. Thus, all URLs flagged
as "jsonapi" shall be forwarded to this module as GRPC

## LBProxy does not need to use http to forward calls
  
Instead, it can use GRPC with its benefits, including:

  1. load balancing
  2. fail-over
  3. metrics on duration/performance
  4. scale-out (multiplexed TCP connections)

## This module shall lookup individual modules

It uses a number of keys (foremost the url) and forward the api
request via grpc to a module.

## This module shall authorize access to an api

Using the bearer token thing mechanism or whatever is cool
at the time. (it expects lbproxy to authenticate!)

## This module shall throttle requests

To a quota appropriate for the call

## "maybes"

* check for well-formed json
* parse json and hand it over in a nice format
* validate the content of the request

## Two styles are available:
1. "old-style" routing.yaml (deprecated)
  This requires a special RPC call to be exposed in the
  module. The RPC Call signature must match JsonAPI(JsonRequest)
  the jsonapimultiplexer will call that RPC call for all urls
  mapping into the module
  
2. "new-style" autorouting.yaml
 This requires no special RPC calls to the module.
 The gRPC service must be mapped to a url prefix
 All gRPC calls will be available as  "/[serviceprefix]/[methodname]"
 A basic enquiry mechanism is also available:
 "/[serviceprefix]/ServiceInfo" lists the available RPC calls
 "/[serviceprefix]/[methodname]/Info" lists the input and output json of a method

## Adding a new module

* add the service and pathprefix to autorouting.yaml


