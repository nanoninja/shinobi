// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package shinobi is a lightweight HTTP micro-framework built on top of the Go standard library.
package shinobi

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nanoninja/render"
)

// Size constants for use with MaxSize, BodyLimit, and JSONMaxBytes.
// Example: shinobi.NewImageUpload(f, h, 5*shinobi.MB)
const (
	KB int64 = 1 << 10 // 1 024 bytes
	MB int64 = 1 << 20 // 1 048 576 bytes
	GB int64 = 1 << 30 // 1 073 741 824 bytes
)

// Config holds the configuration for an App and its Router.
// Use functional Options to customize it; most callers rely on DefaultConfig.
type Config struct {
	// name is used as a prefix in log messages.
	name string

	// binder handles request body and query decoding.
	binder Binder

	// validator validates decoded request values.
	validator Validator

	// tmpl is the renderer used by Ctx.HTML.
	tmpl render.Renderer

	// logger is the structured logger injected into every Ctx.
	logger *slog.Logger

	// debug enables verbose logging when true.
	debug bool

	// mux is the underlying HTTP multiplexer.
	mux *http.ServeMux

	// prefix is the base path prepended to all registered routes.
	prefix string

	// errorHandler is called when a handler returns an error.
	errorHandler ErrorHandler

	// notFound is the handler invoked for unmatched routes.
	notFound HandlerFunc
}

// Option defines a function type for configuring the Config.
type Option func(*Config)

// DefaultConfig returns a Config with production-safe defaults.
func DefaultConfig() *Config { return defaultConfig() }

func defaultConfig() *Config {
	return &Config{
		name:         "shinobi",
		binder:       DefaultBinder,
		logger:       slog.Default(),
		mux:          http.NewServeMux(),
		errorHandler: defaultErrorHandler,
	}
}

// App is the top-level shinobi application.
// It wraps a Router and manages the HTTP server lifecycle.
type App struct {
	config *Config
	router *router
}

// New creates a new App with the given options.
// All options configure the underlying Router.
func New(opts ...Option) *App {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	routes := make([]RouteInfo, 0)
	app := &App{
		config: cfg,
		router: &router{
			mux:          cfg.mux,
			prefix:       cfg.prefix,
			config:       cfg,
			routes:       &routes,
			errorHandler: cfg.errorHandler,
		},
	}
	if cfg.notFound != nil {
		app.router.NotFound(cfg.notFound)
	}
	return app
}

// SetDebug enables or disables debug mode. In debug mode, routes are logged at
// startup and error responses include the original error message.
func (a *App) SetDebug(v bool) {
	a.config.debug = v
}

// SetLogger sets the logger used by the application and injected into each request context.
// Ignored if l is nil. Defaults to slog.Default().
func (a *App) SetLogger(l *slog.Logger) {
	if l != nil {
		a.config.logger = l
	}
}

// SetName sets the application name used in log messages.
// Ignored if name is empty. Defaults to "shinobi".
func (a *App) SetName(name string) {
	if name != "" {
		a.config.name = name
	}
}

// IsDebug reports whether the application is running in debug mode.
func (a *App) IsDebug() bool {
	return a.config.debug
}

// Name returns the application name used in log messages.
func (a *App) Name() string {
	return a.config.name
}

// Router returns the underlying Router for advanced configuration.
func (a *App) Router() Router {
	return a.router
}

// Server returns an http.Server pre-configured with production-safe defaults:
// ReadHeaderTimeout 5s, ReadTimeout 10s, WriteTimeout 30s, IdleTimeout 120s.
// Any field can be overridden after calling this method.
func (a *App) Server(addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           a,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

// ServeHTTP implements http.Handler, delegating to the underlying router.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

// Listen starts an HTTP server on addr and blocks until it stops.
func (a *App) Listen(addr string) error {
	a.config.logger.Info(a.config.name+" listening", "addr", addr)
	return http.ListenAndServe(addr, a)
}

// ListenTLS starts an HTTPS server with the given certificate and key files.
func (a *App) ListenTLS(addr, certFile, keyFile string) error {
	srv := a.Server(addr)
	srv.TLSConfig = defaultTLSConfig()

	a.config.logger.Info(a.config.name+" listening TLS", "addr", addr)

	return srv.ListenAndServeTLS(certFile, keyFile)
}

// ListenGraceful starts an HTTP server and shuts it down gracefully on SIGINT or SIGTERM.
func (a *App) ListenGraceful(addr string, timeout time.Duration) error {
	srv := a.Server(addr)

	a.config.logger.Info(a.config.name+" listening", "addr", addr)

	return a.serveGraceful(srv, func() error {
		return srv.ListenAndServe()
	}, timeout)
}

// ListenTLSGraceful starts an HTTPS server and shuts it down gracefully on SIGINT or SIGTERM.
func (a *App) ListenTLSGraceful(addr, certFile, keyFile string, timeout time.Duration) error {
	srv := a.Server(addr)
	srv.TLSConfig = defaultTLSConfig()

	a.config.logger.Info(a.config.name+" listening TLS", "addr", addr)

	return a.serveGraceful(srv, func() error {
		return srv.ListenAndServeTLS(certFile, keyFile)
	}, timeout)
}

// Handle registers a new route with a handler for the given pattern.
func (a *App) Handle(pattern string, h Handler) { a.router.Handle(pattern, h) }

// HandleFunc registers a new route with a handler function for the given pattern.
func (a *App) HandleFunc(pattern string, h HandlerFunc) { a.router.HandleFunc(pattern, h) }

// Method registers a new route with a specific HTTP method and handler.
func (a *App) Method(method, pattern string, h Handler) { a.router.Method(method, pattern, h) }

// MethodFunc registers a new route with a specific HTTP method and handler function.
func (a *App) MethodFunc(method, pattern string, h HandlerFunc) {
	a.router.MethodFunc(method, pattern, h)
}

// Connect registers a CONNECT route.
func (a *App) Connect(pattern string, h HandlerFunc) { a.router.Connect(pattern, h) }

// Delete registers a DELETE route.
func (a *App) Delete(pattern string, h HandlerFunc) { a.router.Delete(pattern, h) }

// Get registers a GET route.
func (a *App) Get(pattern string, h HandlerFunc) { a.router.Get(pattern, h) }

// Head registers a HEAD route.
func (a *App) Head(pattern string, h HandlerFunc) { a.router.Head(pattern, h) }

// Options registers an OPTIONS route.
func (a *App) Options(pattern string, h HandlerFunc) { a.router.Options(pattern, h) }

// Patch registers a PATCH route.
func (a *App) Patch(pattern string, h HandlerFunc) { a.router.Patch(pattern, h) }

// Post registers a POST route.
func (a *App) Post(pattern string, h HandlerFunc) { a.router.Post(pattern, h) }

// Put registers a PUT route.
func (a *App) Put(pattern string, h HandlerFunc) { a.router.Put(pattern, h) }

// Trace registers a TRACE route.
func (a *App) Trace(pattern string, h HandlerFunc) { a.router.Trace(pattern, h) }

// Group creates a sub-router with a prefix for all routes registered within fn.
func (a *App) Group(prefix string, fn func(Router)) { a.router.Group(prefix, fn) }

// Route registers multiple HTTP methods on a single pattern.
func (a *App) Route(pattern string, fn func(Route)) { a.router.Route(pattern, fn) }

// Mount attaches a Handler at the given prefix.
func (a *App) Mount(prefix string, h Handler) { a.router.Mount(prefix, h) }

// Use appends one or more middlewares to the global middleware stack.
func (a *App) Use(middlewares ...Middleware) { a.router.Use(middlewares...) }

// With returns a new Router scoped with the provided middlewares.
func (a *App) With(middlewares ...Middleware) Router { return a.router.With(middlewares...) }

// NotFound registers a custom handler for 404 responses.
func (a *App) NotFound(h HandlerFunc) { a.router.NotFound(h) }

// Routes returns all registered routes for introspection.
func (a *App) Routes() []RouteInfo { return a.router.Routes() }

// defaultTLSConfig returns a TLS configuration with TLS 1.2 as the minimum version.
// Go automatically negotiates TLS 1.3 when supported by the client.
func defaultTLSConfig() *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12}
}

// serveGraceful runs start in a goroutine and waits for a signal to shut down.
func (a *App) serveGraceful(srv *http.Server, start func() error, timeout time.Duration) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	errCh := make(chan error, 1)
	go func() {
		if err := start(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-quit:
		a.config.logger.Info("shutting down", "timeout", timeout)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	a.config.logger.Info("server stopped")
	return nil
}
