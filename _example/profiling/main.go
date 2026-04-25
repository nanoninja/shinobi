// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Profiling example — exposes pprof endpoints behind BasicAuth.
//
// Usage:
//
//	go run ./profiling
//
// Access the profiler (password: secret):
//
//	curl -u admin:secret http://localhost:8080/debug/pprof/
//	go tool pprof http://admin:secret@localhost:8080/debug/pprof/heap
package main

import (
	"log"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/nanoninja/shinobi"
	"github.com/nanoninja/shinobi/middleware"
)

func main() {
	app := shinobi.New()

	app.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	pprofMux := http.NewServeMux()
	pprofMux.HandleFunc("/", pprof.Index)
	pprofMux.HandleFunc("/cmdline", pprof.Cmdline)
	pprofMux.HandleFunc("/profile", pprof.Profile)
	pprofMux.HandleFunc("/symbol", pprof.Symbol)
	pprofMux.HandleFunc("/trace", pprof.Trace)
	pprofMux.Handle("/goroutine", pprof.Handler("goroutine"))
	pprofMux.Handle("/heap", pprof.Handler("heap"))
	pprofMux.Handle("/block", pprof.Handler("block"))
	pprofMux.Handle("/allocs", pprof.Handler("allocs"))
	pprofMux.Handle("/mutex", pprof.Handler("mutex"))
	pprofMux.Handle("/threadcreate", pprof.Handler("threadcreate"))

	// Mount strips the prefix before passing to the handler —
	// register pprof handlers without the /debug/pprof prefix.
	app.With(middleware.BasicAuth(middleware.BasicAuthConfig{
		Realm:     "Debug",
		Validator: middleware.Auth("admin", "secret"),
	})).Mount("/debug/pprof", shinobi.AdaptHTTP(pprofMux))

	log.Fatal(app.ListenGraceful(":8080", 30*time.Second))
}
