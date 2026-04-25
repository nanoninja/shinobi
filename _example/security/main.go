// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Security example — demonstrates BasicAuth, SecureHeaders, CORS and BodyLimit.
//
// Run:
//
//	go run ./_example/security/main.go
//
// To test with HTTPS, generate a self-signed certificate first:
//
//	go run $(go env GOROOT)/src/crypto/tls/generate_cert.go --host localhost
//
// This creates cert.pem and key.pem in the current directory.
// Then replace ListenGraceful with ListenTLSGraceful in main().
//
// Test with curl:
//
//	# 401 — no credentials
//	curl -i http://localhost:8080/admin/profile
//
//	# 200 — valid credentials
//	curl -i -u alice:s3cr3t http://localhost:8080/admin/profile
//
//	# 403 — wrong credentials
//	curl -i -u alice:wrong http://localhost:8080/admin/profile
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/nanoninja/shinobi"
	"github.com/nanoninja/shinobi/middleware"
)

func main() {
	app := shinobi.New()

	// Secure headers on all routes.
	app.Use(middleware.SecureHeaders())

	// CORS for all routes — restrict to a specific origin in production.
	app.Use(middleware.CORS(middleware.DefaultCORSConfig()))

	// Limit request body to 1 MB on all routes.
	app.Use(middleware.BodyLimit(1 << 20))

	// Public route — no authentication required.
	app.Get("/health", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "OK")
	})

	// Protected group — BasicAuth enforced on every route inside.
	app.Group("/admin", func(r shinobi.Router) {
		r.Use(middleware.BasicAuth(middleware.BasicAuthConfig{
			Realm: "Admin Area",
			Validator: middleware.User{
				"alice": "s3cr3t",
				"bob":   "p4ssw0rd",
			},
		}))

		r.Get("/profile", func(c shinobi.Ctx) error {
			cred, _ := middleware.BasicAuthCredentialFrom(c)
			return c.JSON(http.StatusOK, map[string]string{
				"user": cred.Username,
			})
		})

		r.Get("/dashboard", func(c shinobi.Ctx) error {
			cred, _ := middleware.BasicAuthCredentialFrom(c)
			return c.JSON(http.StatusOK, map[string]any{
				"user":    cred.Username,
				"message": "welcome to the dashboard",
			})
		})
	})

	log.Fatal(app.ListenGraceful(":8080", 10*time.Second))
}
