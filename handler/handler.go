package handler

import "github.com/RottenNinja-Go/framework"

type EndpointOptions interface {
	SetSummary(summary string)
	SetDescription(description string)
	SetTags(tags ...string)
}

// EndpointBuilder provides a fluent API for building endpoints with optional metadata
type EndpointBuilder struct {
	endpoint framework.Endpoint
}

// Summary sets the endpoint summary
func (b *EndpointBuilder) SetSummary(summary string) {
	b.endpoint.SetSummary(summary)
}

// Description sets the endpoint description
func (b *EndpointBuilder) SetDescription(description string) {
	b.endpoint.SetDescription(description)
}

// Tags sets the endpoint tags
func (b *EndpointBuilder) SetTags(tags ...string) {
	b.endpoint.SetTags(tags...)
}

// Use adds one or more middleware functions to the endpoint
// Middleware is applied in the order it's added (first added = outermost wrapper)
// Example: Use(logging, auth, ratelimit) wraps as logging(auth(ratelimit(handler)))
func (b *EndpointBuilder) Use(middleware ...framework.Middleware) {
	b.endpoint.Use(middleware...)
}

// GET registers a GET endpoint with type-safe handler using a fluent API
// Works with both Framework and Group through the Router interface
// Can be used directly: GET(f, path, handler).Summary("...").Tags("...").Register()
// Or without metadata: GET(f, path, handler).Register()
func GET[Req any, Resp any](r framework.Router, path string, handler framework.Handler[Req, Resp], optFn func(EndpointOptions)) {
	hRoute := &EndpointBuilder{endpoint: framework.CreateEndpoint("GET", path, handler)}
	optFn(hRoute)
	framework.RegisterEndpoint(r, hRoute.endpoint)
}

// POST registers a POST endpoint with type-safe handler using a fluent API
// Works with both Framework and Group through the Router interface
// Can be used directly: POST(f, path, handler).Summary("...").Tags("...").Register()
// Or without metadata: POST(f, path, handler).Register()
func POST[Req any, Resp any](r framework.Router, path string, handler framework.Handler[Req, Resp], optFn func(EndpointOptions)) {
	hRoute := &EndpointBuilder{endpoint: framework.CreateEndpoint("POST", path, handler)}
	optFn(hRoute)
	framework.RegisterEndpoint(r, hRoute.endpoint)
}

// PUT registers a PUT endpoint with type-safe handler using a fluent API
// Works with both Framework and Group through the Router interface
// Can be used directly: PUT(f, path, handler).Summary("...").Tags("...").Register()
// Or without metadata: PUT(f, path, handler).Register()
func PUT[Req any, Resp any](r framework.Router, path string, handler framework.Handler[Req, Resp], optFn func(EndpointOptions)) {
	hRoute := &EndpointBuilder{endpoint: framework.CreateEndpoint("PUT", path, handler)}
	optFn(hRoute)
	framework.RegisterEndpoint(r, hRoute.endpoint)
}

// PATCH registers a PATCH endpoint with type-safe handler using a fluent API
// Works with both Framework and Group through the Router interface
// Can be used directly: PATCH(f, path, handler).Summary("...").Tags("...").Register()
// Or without metadata: PATCH(f, path, handler).Register()
func PATCH[Req any, Resp any](r framework.Router, path string, handler framework.Handler[Req, Resp], optFn func(EndpointOptions)) {
	hRoute := &EndpointBuilder{endpoint: framework.CreateEndpoint("PATH", path, handler)}
	optFn(hRoute)
	framework.RegisterEndpoint(r, hRoute.endpoint)
}

// DELETE registers a DELETE endpoint with type-safe handler using a fluent API
// Works with both Framework and Group through the Router interface
// Can be used directly: DELETE(f, path, handler).Summary("...").Tags("...").Register()
// Or without metadata: DELETE(f, path, handler).Register()
func DELETE[Req any, Resp any](r framework.Router, path string, handler framework.Handler[Req, Resp], optFn func(EndpointOptions)) {
	hRoute := &EndpointBuilder{endpoint: framework.CreateEndpoint("DELETE", path, handler)}
	optFn(hRoute)
	framework.RegisterEndpoint(r, hRoute.endpoint)
}
