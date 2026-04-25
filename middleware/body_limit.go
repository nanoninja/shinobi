// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"errors"
	"net/http"

	"github.com/nanoninja/shinobi"
)

// BodyLimit returns a middleware that caps the request body size to limit bytes.
// If the body exceeds the limit, a 413 Request Entity Too Large is returned
// through the error handler.
//
// Example:
//
//	app.Use(middleware.BodyLimit(shinobi.MB))
//	app.Use(middleware.BodyLimit(10 * shinobi.MB))
func BodyLimit(limit int64) shinobi.Middleware {
	return func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			r := c.Request()
			r.Body = http.MaxBytesReader(c.Response(), r.Body, limit)

			err := next.Handle(c)

			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				return shinobi.HTTPError(http.StatusRequestEntityTooLarge)
			}

			return err
		})
	}
}
