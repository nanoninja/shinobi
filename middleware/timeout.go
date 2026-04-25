// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/nanoninja/shinobi"
)

// Timeout returns a middleware that cancels the request context after the given
// duration. If the deadline is exceeded before the handler writes a response,
// a 503 Service Unavailable error is returned through the error handler.
//
// Timeout only works if the handler respects context cancellation — for example
// by passing the context to database queries or outgoing HTTP calls. CPU-bound
// work that never checks ctx.Err() will not be interrupted.
//
// Example:
//
//	app.Use(middleware.Timeout(5 * time.Second))
func Timeout(duration time.Duration) shinobi.Middleware {
	return func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			ctx, cancel := context.WithTimeout(c.Context(), duration)
			defer cancel()

			c = c.WithContext(ctx)
			err := next.Handle(c)

			if ctx.Err() == context.DeadlineExceeded && !c.Response().Written() {
				return shinobi.HTTPError(http.StatusServiceUnavailable)
			}

			return err
		})
	}
}
