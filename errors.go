// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shinobi

import (
	"errors"
	"fmt"
	"net/http"
)

// Renderer errors
var (
	// ErrTemplateRendererNotSet is returned when HTML is called without a configured template renderer.
	ErrTemplateRendererNotSet = errors.New("template renderer not configured")
)

// HTTP Errors
var (
	// ErrInvalidRedirectStatusCode is returned when an invalid HTTP status code is provided for redirection.
	ErrInvalidRedirectStatusCode = errors.New("invalid redirect status code")
)

// Binder errors
var (
	// ErrUnsupportedContentType is returned when no Binder is registered for the request Content-Type.
	ErrUnsupportedContentType = errors.New("unsupported content type")
)

// Validate errors
var (
	// ErrValidatorNotSet is returned when Validate is called without a configured Validator.
	ErrValidatorNotSet = errors.New("validator not configured")
)

// StatusError represents an HTTP error with a status code and message.
type StatusError struct {
	Code    int
	Message any
	Cause   error
}

// WithInternal wraps an internal error for logging without exposing it to the client.
func (e *StatusError) WithInternal(err error) *StatusError {
	e.Cause = err
	return e
}

// Unwrap returns the internal cause, enabling errors.Is and errors.As traversal.
func (e *StatusError) Unwrap() error {
	return e.Cause
}

// HTTPError creates a new StatusError with the given HTTP status code and optional message.
func HTTPError(code int, message ...any) *StatusError {
	e := &StatusError{Code: code}
	if len(message) > 0 {
		e.Message = message[0]
	}
	return e
}

func (e *StatusError) Error() string {
	if e.Message != nil {
		return fmt.Sprintf("%d: %v", e.Code, e.Message)
	}
	return fmt.Sprintf("%d: %s", e.Code, http.StatusText(e.Code))
}
