// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shinobi

import (
	"io"
	"net/http"
)

// Response wraps http.ResponseWriter to capture the HTTP status code
// and guard against writing headers more than once.
type Response struct {
	http.ResponseWriter
	status  int
	written bool
}

// NewResponse returns a Response with a default 200 OK status.
func NewResponse(w http.ResponseWriter) *Response {
	return &Response{
		ResponseWriter: w,
		status:         http.StatusOK,
	}
}

// Status returns the captured HTTP status code.
func (r *Response) Status() int {
	return r.status
}

// Written reports whether the response headers have already been sent.
func (r *Response) Written() bool {
	return r.written
}

// WriteHeader sends the HTTP status code once, ignoring subsequent calls.
func (r *Response) WriteHeader(code int) {
	if !r.written {
		r.status = code
		r.written = true
		r.ResponseWriter.WriteHeader(code)
	}
}

// Write sends the response body, writing a 200 OK header first if not yet sent.
func (r *Response) Write(b []byte) (int, error) {
	if !r.written {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}

// WriteString writes a string to the response body.
func (r *Response) WriteString(s string) (int, error) {
	if !r.written {
		r.WriteHeader(http.StatusOK)
	}
	return io.WriteString(r.ResponseWriter, s)
}

// Unwrap returns the underlying http.ResponseWriter.
func (r *Response) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// SetResponseWriter replaces the underlying ResponseWriter using a wrapping function.
// The function receives the current writer, ensuring the chain is never broken.
func (r *Response) SetResponseWriter(fn func(http.ResponseWriter) http.ResponseWriter) {
	r.ResponseWriter = fn(r.ResponseWriter)
}
