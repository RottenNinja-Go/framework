package handler

import "github.com/RottenNinja-Go/framework"

type EndpointOptions interface {
	SetSummary(summary string)
	SetDescription(description string)
	SetTags(tags ...string)
}

// EndpointBuilder provides a fluent API for building endpoints with optional metadata
type EndpointBuilder[Req any, Resp any] struct {
	*framework.HandlerRoute[Req, Resp]
}

// Summary sets the endpoint summary
func (b *EndpointBuilder[Req, Resp]) SetSummary(summary string) {
	b.Summary = summary
}

// Description sets the endpoint description
func (b *EndpointBuilder[Req, Resp]) SetDescription(description string) {
	b.Description = description
}

// Tags sets the endpoint tags
func (b *EndpointBuilder[Req, Resp]) SetTags(tags ...string) {
	b.Tags = tags
}

// Use adds one or more middleware functions to the endpoint
// Middleware is applied in the order it's added (first added = outermost wrapper)
// Example: Use(logging, auth, ratelimit) wraps as logging(auth(ratelimit(handler)))
func (b *EndpointBuilder[Req, Resp]) Use(middleware ...framework.Middleware) {
	b.Middlewares = append(b.Middlewares, middleware...)
}


// GET registers a GET endpoint with type-safe handler using a fluent API
// Works with both Framework and Group through the Router interface
// Can be used directly: GET(f, path, handler).Summary("...").Tags("...").Register()
// Or without metadata: GET(f, path, handler).Register()
func GET[Req any, Resp any](r framework.Router, path string, handler framework.Handler[Req, Resp], optFn func(EndpointOptions)) {
	hRoute := &EndpointBuilder[Req, Resp]{
		HandlerRoute: &framework.HandlerRoute[Req, Resp]{
			EndpointSpec: &framework.EndpointSpec{
				Method:       "GET",
				RelativePath: path,
			},
			Handler: handler,
		},
	}
	optFn(hRoute)
	framework.RegisterHandlerRoute(r, hRoute.HandlerRoute)
}

// POST registers a POST endpoint with type-safe handler using a fluent API
// Works with both Framework and Group through the Router interface
// Can be used directly: POST(f, path, handler).Summary("...").Tags("...").Register()
// Or without metadata: POST(f, path, handler).Register()
func POST[Req any, Resp any](r framework.Router, path string, handler framework.Handler[Req, Resp], optFn func(EndpointOptions)) {
	hRoute := &EndpointBuilder[Req, Resp]{
		HandlerRoute: &framework.HandlerRoute[Req, Resp]{
			EndpointSpec: &framework.EndpointSpec{
				Method:       "POST",
				RelativePath: path,
			},
			Handler: handler,
		},
	}
	optFn(hRoute)
	framework.RegisterHandlerRoute(r, hRoute.HandlerRoute)
}

// PUT registers a PUT endpoint with type-safe handler using a fluent API
// Works with both Framework and Group through the Router interface
// Can be used directly: PUT(f, path, handler).Summary("...").Tags("...").Register()
// Or without metadata: PUT(f, path, handler).Register()
func PUT[Req any, Resp any](r framework.Router, path string, handler framework.Handler[Req, Resp], optFn func(EndpointOptions)) {
	hRoute := &EndpointBuilder[Req, Resp]{
		HandlerRoute: &framework.HandlerRoute[Req, Resp]{
			EndpointSpec: &framework.EndpointSpec{
				Method:       "PUT",
				RelativePath: path,
			},
			Handler: handler,
		},
	}
	optFn(hRoute)
	framework.RegisterHandlerRoute(r, hRoute.HandlerRoute)
}

// PATCH registers a PATCH endpoint with type-safe handler using a fluent API
// Works with both Framework and Group through the Router interface
// Can be used directly: PATCH(f, path, handler).Summary("...").Tags("...").Register()
// Or without metadata: PATCH(f, path, handler).Register()
func PATCH[Req any, Resp any](r framework.Router, path string, handler framework.Handler[Req, Resp], optFn func(EndpointOptions)) {
	hRoute := &EndpointBuilder[Req, Resp]{
		HandlerRoute: &framework.HandlerRoute[Req, Resp]{
			EndpointSpec: &framework.EndpointSpec{
				Method:       "GET",
				RelativePath: path,
			},
			Handler: handler,
		},
	}
	optFn(hRoute)
	framework.RegisterHandlerRoute(r, hRoute.HandlerRoute)
}

// DELETE registers a DELETE endpoint with type-safe handler using a fluent API
// Works with both Framework and Group through the Router interface
// Can be used directly: DELETE(f, path, handler).Summary("...").Tags("...").Register()
// Or without metadata: DELETE(f, path, handler).Register()
func DELETE[Req any, Resp any](r framework.Router, path string, handler framework.Handler[Req, Resp], optFn func(EndpointOptions)) {
	hRoute := &EndpointBuilder[Req, Resp]{
		HandlerRoute: &framework.HandlerRoute[Req, Resp]{
			EndpointSpec: &framework.EndpointSpec{
				Method:       "DELETE",
				RelativePath: path,
			},
			Handler: handler,
		},
	}
	optFn(hRoute)
	framework.RegisterHandlerRoute(r, hRoute.HandlerRoute)
}
