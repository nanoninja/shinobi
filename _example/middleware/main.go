// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"compress/gzip"
	"log"
	"net/http"
	"time"

	"github.com/nanoninja/shinobi"
	"github.com/nanoninja/shinobi/middleware"
)

func main() {
	app := shinobi.New()

	// Middleware order matters:
	// 1. RealIP   — resolve client IP before anything reads RemoteAddr
	// 2. RequestID — assign trace ID before Logger captures it
	// 3. Logger   — log with correct IP and request ID
	// 4. Recoverer — catch panics from all subsequent handlers
	app.Use(middleware.RealIP())
	app.Use(middleware.RequestID())
	app.Use(middleware.Logger())
	app.Use(middleware.Recoverer())
	app.Use(middleware.Compress(gzip.DefaultCompression))
	app.Use(middleware.Timeout(10 * time.Second))

	app.Get("/", func(c shinobi.Ctx) error {
		id, _ := c.Get(middleware.RequestIDKey)
		return c.String(http.StatusOK, "request id: %s", id)
	})

	app.Get("/panic", func(shinobi.Ctx) error {
		panic("oops")
	})

	// Rate limiting and body limit applied per-route or per-group
	app.With(
		middleware.RateLimit(100, time.Minute),
		middleware.BodyLimit(1<<20),
	).Post("/upload", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "uploaded")
	})

	// NoCache on routes that must never be cached — auth, user data, etc.
	app.With(middleware.NoCache()).Get("/profile", func(c shinobi.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"user": "alice"})
	})

	log.Fatal(app.ListenGraceful(":8080", 10*time.Second))
}
