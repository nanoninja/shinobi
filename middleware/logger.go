// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"log/slog"
	"time"

	"github.com/nanoninja/shinobi"
)

// Logger returns a middleware that logs each request using slog.
// It records the HTTP method, path, status code, duration, and request ID if present.
func Logger() shinobi.Middleware {
	return func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			start := time.Now()
			err := next.Handle(c)
			attrs := []slog.Attr{
				slog.String("method", c.Method()),
				slog.String("path", c.Path()),
				slog.Int("status", c.Response().Status()),
				slog.Duration("duration", time.Since(start)),
			}
			if id, ok := c.Get(RequestIDKey); ok {
				attrs = append(attrs, slog.Any("request_id", id))
			}
			c.Logger().LogAttrs(c.Context(), slog.LevelInfo, "request", attrs...)
			return err
		})
	}
}
