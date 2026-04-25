// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"crypto/rand"
	"fmt"

	"github.com/nanoninja/shinobi"
)

// RequestIDKey is the context key used to store and retrieve the request ID.
const RequestIDKey = "X-Request-ID"

// RequestID returns a middleware that assigns a unique ID to each request.
// If the incoming request already has an X-Request-ID header, it is reused.
// The ID is set on both the response header and the request context.
func RequestID() shinobi.Middleware {
	return func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			id := c.Request().Header.Get(RequestIDKey)
			if id == "" {
				id = generateID()
			}
			c.SetHeader(RequestIDKey, id)
			c.Set(RequestIDKey, id)
			return next.Handle(c)
		})
	}
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
