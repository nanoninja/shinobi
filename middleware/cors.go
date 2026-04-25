// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/nanoninja/shinobi"
)

// CORSConfig holds the configuration for the CORS middleware.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns a CORSConfig with permissive defaults suitable for development.
// Specific origins can be provided to restrict access in production.
//
// Example:
//
//	middleware.CORS(middleware.DefaultCORSConfig())
//	middleware.CORS(middleware.DefaultCORSConfig("https://myapp.com"))
func DefaultCORSConfig(origins ...string) CORSConfig {
	allowedOrigins := []string{"*"}
	if len(origins) > 0 {
		allowedOrigins = origins
	}
	return CORSConfig{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Content-Type", "Authorization"},
		MaxAge:         86400,
	}
}

// CORS returns a middleware that adds Cross-Origin Resource Sharing headers
// to every response based on the provided configuration.
//
// Preflight requests (OPTIONS) are handled automatically and short-circuited
// with a 204 No Content response.
//
// Example:
//
//	r.Use(middleware.CORS(middleware.DefaultCORSConfig()))
//
//	// With specific origins:
//	r.Use(middleware.CORS(middleware.DefaultCORSConfig("https://myapp.com")))
//
//	// Fully custom:
//	r.Use(middleware.CORS(middleware.CORSConfig{
//	    AllowedOrigins: []string{"https://example.com"},
//	    AllowedMethods: []string{"GET", "POST"},
//	    AllowedHeaders: []string{"Content-Type", "Authorization"},
//	}))
func CORS(cfg CORSConfig) shinobi.Middleware {
	if cfg.AllowCredentials && len(cfg.AllowedOrigins) == 1 &&
		cfg.AllowedOrigins[0] == "*" {
		panic("cors: AllowCredentials requires explicit origins, wildcard (*) is not allowed")
	}

	methods := strings.Join(cfg.AllowedMethods, ", ")
	headers := strings.Join(cfg.AllowedHeaders, ", ")

	allowAll := len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*"
	allowed := make(map[string]bool, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		allowed[o] = true
	}

	return func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			origin := c.Request().Header.Get("Origin")

			if allowAll {
				c.SetHeader("Access-Control-Allow-Origin", "*")
			} else if origin != "" && allowed[origin] {
				c.SetHeader("Access-Control-Allow-Origin", origin)
				c.AddHeader("Vary", "Origin")
			}

			c.SetHeader("Access-Control-Allow-Methods", methods)
			c.SetHeader("Access-Control-Allow-Headers", headers)

			if cfg.AllowCredentials {
				c.SetHeader("Access-Control-Allow-Credentials", "true")
			}
			if c.IsMethod(http.MethodOptions) {
				if cfg.MaxAge > 0 {
					c.SetHeader("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
				}
				return c.NoContent()
			}
			return next.Handle(c)
		})
	}
}
