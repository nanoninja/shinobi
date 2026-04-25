// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shinobi

import (
	"log/slog"
	"net/http"
	"path"
	"strings"

	"github.com/nanoninja/render"
)

// WithBinder sets a custom Binder used to decode request bodies in Ctx.Bind.
func WithBinder(b Binder) Option {
	return func(c *Config) {
		if b != nil {
			c.binder = b
		}
	}
}

// WithErrorHandler sets a custom handler for errors returned by route handlers.
func WithErrorHandler(h ErrorHandler) Option {
	return func(c *Config) {
		if h != nil {
			c.errorHandler = h
		}
	}
}

// WithMux allows providing a custom http.ServeMux instance.
func WithMux(mux *http.ServeMux) Option {
	return func(c *Config) {
		if mux != nil {
			c.mux = mux
		}
	}
}

// WithNotFound sets a custom handler for 404 Not Found errors during initialization.
func WithNotFound(h HandlerFunc) Option {
	return func(c *Config) {
		if h != nil {
			c.notFound = h
		}
	}
}

// WithPrefix sets an initial global prefix for all routes.
func WithPrefix(prefix string) Option {
	return func(c *Config) {
		c.prefix = prefix
	}
}

// WithRenderer sets the template renderer injected into each Ctx.
func WithRenderer(r render.Renderer) Option {
	return func(c *Config) {
		if r != nil {
			c.tmpl = r
		}
	}
}

// WithDebug enables or disables debug mode injected into each Ctx.
// In debug mode, error responses include the original error message.
func WithDebug(v bool) Option {
	return func(c *Config) {
		c.debug = v
	}
}

// WithLogger sets the logger injected into each Ctx.
// Ignored if l is nil. Defaults to slog.Default().
func WithLogger(l *slog.Logger) Option {
	return func(c *Config) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithValidator sets the validator injected into each Ctx.
func WithValidator(v Validator) Option {
	return func(c *Config) {
		if v != nil {
			c.validator = v
		}
	}
}

// Route defines multi-method registration on a single pattern.
type Route interface {
	Connect(h HandlerFunc)
	Delete(h HandlerFunc)
	Get(h HandlerFunc)
	Head(h HandlerFunc)
	Options(h HandlerFunc)
	Patch(h HandlerFunc)
	Post(h HandlerFunc)
	Put(h HandlerFunc)
	Trace(h HandlerFunc)
}

type route struct {
	pattern string
	r       *router
}

func (r *route) Connect(h HandlerFunc) {
	r.r.Method(http.MethodConnect, r.pattern, h)
}

func (r *route) Delete(h HandlerFunc) {
	r.r.Method(http.MethodDelete, r.pattern, h)
}

func (r *route) Get(h HandlerFunc) {
	r.r.Method(http.MethodGet, r.pattern, h)
}

func (r *route) Head(h HandlerFunc) {
	r.r.Method(http.MethodHead, r.pattern, h)
}

func (r *route) Options(h HandlerFunc) {
	r.r.Method(http.MethodOptions, r.pattern, h)
}

func (r *route) Patch(h HandlerFunc) {
	r.r.Method(http.MethodPatch, r.pattern, h)
}

func (r *route) Post(h HandlerFunc) {
	r.r.Method(http.MethodPost, r.pattern, h)
}

func (r *route) Put(h HandlerFunc) {
	r.r.Method(http.MethodPut, r.pattern, h)
}

func (r *route) Trace(h HandlerFunc) {
	r.r.Method(http.MethodTrace, r.pattern, h)
}

// RouteInfo holds the HTTP method and fully resolved pattern of a registered route.
// It is returned by Router.Routes() for introspection and debugging purposes.
type RouteInfo struct {
	Method  string // HTTP method (e.g. "GET", "POST")
	Pattern string // Full resolved path (e.g. "/api/v1/users/{id}")
}

// Router defines the interface for a layered HTTP router.
type Router interface {
	http.Handler

	// Handle registers a new route with a handler for the given pattern.
	Handle(pattern string, h Handler)

	// HandleFunc registers a new route with a handler function.
	HandleFunc(pattern string, h HandlerFunc)

	// Method registers a new route with a specific HTTP method and handler.
	Method(method, pattern string, h Handler)

	// MethodFunc registers a new route with a specific HTTP method and handler function.
	MethodFunc(method, pattern string, h HandlerFunc)

	// HTTP Verb shortcuts.
	Connect(pattern string, h HandlerFunc)
	Delete(pattern string, h HandlerFunc)
	Get(pattern string, h HandlerFunc)
	Head(pattern string, h HandlerFunc)
	Options(pattern string, h HandlerFunc)
	Patch(pattern string, h HandlerFunc)
	Post(pattern string, h HandlerFunc)
	Put(pattern string, h HandlerFunc)
	Trace(pattern string, h HandlerFunc)

	// Group creates a new sub-router with a prefix.
	// All routes registered within the provided function will inherit this prefix.
	Group(prefix string, fn func(Router))

	// Mount attaches a shinobi Handler at the given prefix.
	// The mounted handler receives requests with the prefix stripped from the path.
	// Use AdaptHTTP to mount stdlib http.Handler implementations.
	Mount(prefix string, h Handler)

	// Use appends one or more middlewares to the router's global middleware stack.
	Use(...Middleware)

	Route(pattern string, fn func(Route))

	// Routes returns all registered routes.
	Routes() []RouteInfo

	// With returns a new Router instance that includes the provided middlewares
	// in addition to the existing ones. Perfect for method chaining.
	With(middlewares ...Middleware) Router

	// NotFound registers a custom handler for 404 Not Found errors.
	NotFound(h HandlerFunc)
}

type router struct {
	prefix          string
	mux             *http.ServeMux
	routes          *[]RouteInfo
	middlewares     []Middleware
	config          *Config
	errorHandler    ErrorHandler
	notFoundHandler HandlerFunc
}

// NewRouter creates a new Router instance with optional configurations.
func NewRouter(opts ...Option) Router {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	routes := make([]RouteInfo, 0)
	r := &router{
		mux:          cfg.mux,
		prefix:       cfg.prefix,
		config:       cfg,
		routes:       &routes,
		errorHandler: cfg.errorHandler,
	}
	if cfg.notFound != nil {
		r.NotFound(cfg.notFound)
	}
	return r
}

func (r *router) Handle(pattern string, h Handler) {
	if method, p, ok := strings.Cut(pattern, " "); ok {
		r.Method(strings.ToUpper(method), p, h)
		return
	}
	r.mux.Handle(r.wrapPath(pattern), r.chain(h))
}

func (r *router) HandleFunc(pattern string, h HandlerFunc) {
	r.Handle(pattern, h)
}

func (r *router) Method(method, pattern string, h Handler) {
	p := r.wrapPath(pattern)
	if p == "/" {
		p = "/{$}"
	}
	r.mux.Handle(method+" "+p, r.chain(h))
	*r.routes = append(*r.routes, RouteInfo{Method: method, Pattern: p})
}

func (r *router) MethodFunc(method, pattern string, h HandlerFunc) {
	r.Method(method, pattern, h)
}

func (r *router) Connect(pattern string, h HandlerFunc) {
	r.Method(http.MethodConnect, pattern, h)
}

func (r *router) Delete(pattern string, h HandlerFunc) {
	r.Method(http.MethodDelete, pattern, h)
}

func (r *router) Get(pattern string, h HandlerFunc) {
	r.Method(http.MethodGet, pattern, h)
}

func (r *router) Head(pattern string, h HandlerFunc) {
	r.Method(http.MethodHead, pattern, h)
}

func (r *router) Options(pattern string, h HandlerFunc) {
	r.Method(http.MethodOptions, pattern, h)
}

func (r *router) Patch(pattern string, h HandlerFunc) {
	r.Method(http.MethodPatch, pattern, h)
}

func (r *router) Post(pattern string, h HandlerFunc) {
	r.Method(http.MethodPost, pattern, h)
}

func (r *router) Put(pattern string, h HandlerFunc) {
	r.Method(http.MethodPut, pattern, h)
}

func (r *router) Trace(pattern string, h HandlerFunc) {
	r.Method(http.MethodTrace, pattern, h)
}

func (r *router) Routes() []RouteInfo {
	return *r.routes
}

func (r *router) NotFound(h HandlerFunc) {
	if h != nil {
		r.notFoundHandler = h
	}
}

func (r *router) Group(prefix string, fn func(Router)) {
	g := r.clone()
	g.prefix = r.wrapPath(prefix)

	if fn != nil {
		fn(g)
	}
}

func (r *router) Mount(prefix string, h Handler) {
	fullPath := r.wrapPath(prefix)
	pattern := strings.TrimRight(fullPath, "/") + "/"

	r.mux.Handle(pattern, http.StripPrefix(strings.TrimRight(fullPath, "/"), r.chain(h)))
}

func (r *router) Route(pattern string, fn func(Route)) {
	rt := &route{
		pattern: pattern,
		r:       r.clone(),
	}
	if fn != nil {
		fn(rt)
	}
}

func (r *router) Use(middlewares ...Middleware) {
	r.middlewares = append(r.middlewares, middlewares...)
}

func (r *router) With(middlewares ...Middleware) Router {
	g := r.clone()
	g.Use(middlewares...)
	return g
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.notFoundHandler != nil {
		h, pattern := r.mux.Handler(req)
		if pattern == "" {
			r.chain(r.notFoundHandler).ServeHTTP(w, req)
			return
		}
		h.ServeHTTP(w, req)
		return
	}
	r.mux.ServeHTTP(w, req)
}

func (r *router) chain(h Handler) http.Handler {
	final := h
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		final = r.middlewares[i](final)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		c := NewCtx(w, req, r.config)
		if err := final.Handle(c); err != nil {
			r.errorHandler(err, c)
		}
	})
}

// clone creates a shallow copy of the router, including its prefix,
// mux, and a copy of the current middleware stack.
func (r *router) clone() *router {
	mws := make([]Middleware, len(r.middlewares))
	copy(mws, r.middlewares)

	return &router{
		prefix:          r.prefix,
		mux:             r.mux,
		routes:          r.routes,
		middlewares:     mws,
		errorHandler:    r.errorHandler,
		notFoundHandler: r.notFoundHandler,
		config:          r.config,
	}
}

// wrapPath combines the router's prefix with the given pattern.
// It ensures the path is clean, starts with a slash, and preserves
// the trailing slash if the pattern specifically requested it.
func (r *router) wrapPath(pattern string) string {
	p := path.Join(r.prefix, pattern)

	if strings.HasSuffix(pattern, "/") && !strings.HasSuffix(p, "/") {
		p += "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}
