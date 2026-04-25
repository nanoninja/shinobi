// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import "github.com/nanoninja/shinobi"

// noCacheControl disables caching at all layers: clients, proxies, and CDNs.
// no-store: do not store the response anywhere.
// no-cache: always revalidate even if stored (belt-and-suspenders with no-store).
// no-transform: prevent proxies from modifying the response body.
const noCacheControl = "no-store, no-cache, no-transform"

// conditionalHeaders are stripped from the request to prevent intermediate
// proxies from serving a cached 304 Not Modified instead of a fresh response.
var conditionalHeaders = []string{
	"ETag",
	"If-Modified-Since",
	"If-Match",
	"If-None-Match",
	"If-Range",
	"If-Unmodified-Since",
}

// NoCache sets HTTP headers to prevent caching by clients, proxies and CDNs.
// It also strips conditional request headers to ensure a fresh response is
// always returned. Pragma is included for HTTP/1.0 proxy compatibility.
func NoCache() shinobi.Middleware {
	return func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			r := c.Request()
			for _, h := range conditionalHeaders {
				r.Header.Del(h)
			}
			c.SetHeader("Cache-Control", noCacheControl)
			c.SetHeader("Pragma", "no-cache")
			return next.Handle(c)
		})
	}
}
