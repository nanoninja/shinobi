// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"strconv"

	"github.com/nanoninja/shinobi"
)

// SecureHeaders returns a middleware that sets security-related HTTP response headers.
//
// Example:
//
//	app.Use(middleware.SecureHeaders())
func SecureHeaders() shinobi.Middleware {
	return func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			c.SetHeader("X-Content-Type-Options", "nosniff")
			c.SetHeader("X-Frame-Options", "DENY")
			c.SetHeader("X-XSS-Protection", "0")
			c.SetHeader("Referrer-Policy", "strict-origin-when-cross-origin")

			return next.Handle(c)
		})
	}
}

// SecureHeadersWithHSTS returns a middleware that applies all SecureHeaders
// and additionally sets Strict-Transport-Security.
// Only use this in production behind HTTPS — enabling HSTS on HTTP will break the app.
//
// Example:
//
//	app.Use(middleware.SecureHeadersWithHSTS(31536000, false))
func SecureHeadersWithHSTS(maxAge int, includeSubdomains bool) shinobi.Middleware {
	hsts := "max-age=" + strconv.Itoa(maxAge)
	if includeSubdomains {
		hsts += "; includeSubDomains"
	}
	secure := SecureHeaders()
	return func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			c.SetHeader("Strict-Transport-Security", hsts)
			return secure(next).Handle(c)
		})
	}
}
