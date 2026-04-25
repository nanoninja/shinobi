// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/nanoninja/shinobi"
)

func main() {
	app := shinobi.New(
		shinobi.WithErrorHandler(func(err error, c shinobi.Ctx) {
			if e, ok := err.(*shinobi.StatusError); ok {
				if e.Cause != nil {
					slog.Error("internal error", "cause", e.Cause)
				}
				_ = c.JSON(e.Code, map[string]any{"error": e.Message})
				return
			}
			_ = c.JSON(http.StatusInternalServerError, map[string]any{
				"error": "internal server error",
			})
		}),
	)

	app.Get("/users/{id}", func(c shinobi.Ctx) error {
		id := c.Param("id")
		if id == "0" {
			return shinobi.HTTPError(http.StatusNotFound, "user not found")
		}
		return c.JSON(http.StatusOK, map[string]string{"id": id})
	})

	app.Get("/internal", func(shinobi.Ctx) error {
		return shinobi.HTTPError(http.StatusInternalServerError).
			WithInternal(http.ErrAbortHandler)
	})

	log.Fatal(app.ListenGraceful(":8080", 10*time.Second))
}
