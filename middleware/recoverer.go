// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/nanoninja/shinobi"
)

// Recoverer returns a middleware that recovers from panics, logs the error and stack trace,
// and returns a 500 Internal Server Error via the configured error handler.
func Recoverer() shinobi.Middleware {
	return func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) (err error) {
			defer func() {
				if r := recover(); r != nil {
					c.Logger().Error("panic recovered",
						"error", r,
						"stack", string(debug.Stack()),
					)
					err = shinobi.HTTPError(http.StatusInternalServerError).
						WithInternal(fmt.Errorf("%v", r))
				}
			}()
			return next.Handle(c)
		})
	}
}
