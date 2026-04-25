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
	app := shinobi.New(
		shinobi.WithPrefix("/api"),
	)

	app.Get("/health", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "OK")
	})

	log.Fatal(app.ListenGraceful(":8080", 10*time.Second))
}
