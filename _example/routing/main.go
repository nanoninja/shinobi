// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"net/http"
	"time"

	"github.com/nanoninja/shinobi"
)

func main() {
	app := shinobi.New()

	// Path parameters
	app.Get("/users/{id}", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "user: %s", c.Param("id"))
	})

	// Multi-method route
	app.Route("/posts/{id}", func(rt shinobi.Route) {
		rt.Get(func(c shinobi.Ctx) error {
			return c.JSON(http.StatusOK, map[string]string{"id": c.Param("id")})
		})
		rt.Delete(func(c shinobi.Ctx) error {
			return c.NoContent()
		})
	})

	// Groups
	app.Group("/api/v1", func(r shinobi.Router) {
		r.Get("/users", func(c shinobi.Ctx) error {
			return c.JSON(http.StatusOK, []string{"alice", "bob"})
		})
		r.Post("/users", func(c shinobi.Ctx) error {
			return c.String(http.StatusCreated, "created")
		})
	})

	// Mount a sub-handler
	app.Mount("/files", shinobi.FileServer("./public"))

	// Route introspection
	for _, route := range app.Routes() {
		log.Printf("%s %s", route.Method, route.Pattern)
	}

	log.Fatal(app.ListenGraceful(":8080", 10*time.Second))
}
