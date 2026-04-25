// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shinobi

import (
	"fmt"
	"io/fs"
	"net/http"
)

// Handler is the core interface for handling an HTTP request within a shinobi context.
type Handler interface {
	Handle(c Ctx) error
}

// HandlerFunc is a function adapter that implements Handler.
type HandlerFunc func(c Ctx) error

// Handle calls f(c) and returns its error.
func (f HandlerFunc) Handle(c Ctx) error {
	return f(c)
}

// Middleware wraps a Handler to extend or intercept request processing.
type Middleware func(Handler) Handler

// AdaptHTTP wraps a stdlib http.Handler as a shinobi Handler.
// Use it to mount external handlers (file servers, third-party routers) via Mount.
func AdaptHTTP(h http.Handler) Handler {
	return HandlerFunc(func(c Ctx) error {
		h.ServeHTTP(c.Response(), c.Request())
		return nil
	})
}

// Adapt converts a stdlib middleware (func(http.Handler) http.Handler) into a shinobi Middleware.
// Context changes made by the stdlib middleware (e.g. r.WithContext) are propagated to the shinobi Ctx.
func Adapt(m func(http.Handler) http.Handler) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(c Ctx) error {
			var handlerErr error
			m(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				handlerErr = next.Handle(c.WithContext(r.Context()))
			})).ServeHTTP(c.Response(), c.Request())
			return handlerErr
		})
	}
}

// FileServer returns a Handler that serves static files from the given directory.
// Designed to be used with Mount.
//
// Example:
//
//	r.Mount("/assets", shinobi.FileServer("./public"))
func FileServer(dir string) Handler {
	return AdaptHTTP(http.FileServer(http.Dir(dir)))
}

// FileServerFS returns a Handler that serves static files from the given fs.FS.
// Useful with Go's embed package.
//
// Example:
//
//	//go:embed public
//	var public embed.FS
//
//	r.Mount("/assets", shinobi.FileServerFS(public))
func FileServerFS(fsys fs.FS) Handler {
	return AdaptHTTP(http.FileServerFS(fsys))
}

// ErrorHandler handles errors returned by a Handler.
type ErrorHandler func(err error, c Ctx)

// defaultErrorHandler handles errors returned by route handlers.
// If the error is a *StatusError, it responds with the corresponding HTTP status code.
// Otherwise, it falls back to a 500 Internal Server Error.
func defaultErrorHandler(err error, c Ctx) {
	if e, ok := err.(*StatusError); ok {
		if e.Cause != nil {
			c.Logger().Error("request error", "status", e.Code, "error", e.Cause)
		} else if e.Code >= http.StatusInternalServerError {
			c.Logger().Error("request error", "status", e.Code, "error", e.Message)
		}
		msg := http.StatusText(e.Code)
		if c.Debug() {
			msg = fmt.Sprintf("%v", e.Message)
		}
		http.Error(c.Response(), msg, e.Code)
		return
	}
	c.Logger().Error("request error", "error", err)
	msg := http.StatusText(http.StatusInternalServerError)
	if c.Debug() {
		msg = err.Error()
	}
	http.Error(c.Response(), msg, http.StatusInternalServerError)
}
