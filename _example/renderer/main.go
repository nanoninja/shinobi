// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"net/http"
	"time"

	"github.com/nanoninja/render/tmpl"
	"github.com/nanoninja/render/tmpl/loader"
	"github.com/nanoninja/shinobi"
)

type User struct {
	Name string
	Role string
}

func main() {
	l := loader.NewString(map[string]string{
		"index.html": `<!DOCTYPE html>
<html>
<head><title>{{ .Name }}</title></head>
<body>
  <h1>Hello, {{ .Name }}!</h1>
  <p>Role: {{ .Role }}</p>
</body>
</html>`,
	}, tmpl.LoaderConfig{Extension: ".html"})

	t := tmpl.HTML("shinobi")
	if err := t.Load(l); err != nil {
		log.Fatal(err)
	}

	app := shinobi.New(
		shinobi.WithRenderer(t),
	)

	app.Get("/", func(c shinobi.Ctx) error {
		return c.HTML(http.StatusOK, "index.html", User{
			Name: "Alice",
			Role: "admin",
		})
	})

	log.Fatal(app.ListenGraceful(":8080", 10*time.Second))
}
