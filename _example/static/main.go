// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"
	"io/fs"
	"log"
	"time"

	"github.com/nanoninja/shinobi"
)

//go:embed public
var embedded embed.FS

func main() {
	app := shinobi.New()

	// Serve from a local directory
	app.Mount("/assets", shinobi.FileServer("./public"))

	// Serve from an embedded filesystem
	app.Mount("/embed", shinobi.FileServerFS(embedded))

	// Rebase the root — files at public/style.css served as /static/style.css
	sub, err := fs.Sub(embedded, "public")
	if err != nil {
		log.Fatal(err)
	}
	app.Mount("/static", shinobi.FileServerFS(sub))

	log.Fatal(app.ListenGraceful(":8080", 10*time.Second))
}
