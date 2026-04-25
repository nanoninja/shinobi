// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shinobi_test

import (
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/render"
	"github.com/nanoninja/shinobi"
)

func TestApp_New(t *testing.T) {
	app := shinobi.New()

	app.Get("/ping", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "pong")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "pong", rec.Body.String())
}

func TestApp_Router(t *testing.T) {
	app := shinobi.New()
	r := app.Router()

	assert.NotNil(t, r)

	r.Get("/via-router", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/via-router", nil)
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestApp_Name(t *testing.T) {
	t.Run("default name is shinobi", func(t *testing.T) {
		assert.Equal(t, "shinobi", shinobi.New().Name())
	})

	t.Run("SetName updates the name", func(t *testing.T) {
		app := shinobi.New()
		app.SetName("myapp")
		assert.Equal(t, "myapp", app.Name())
	})

	t.Run("SetName ignores empty string", func(t *testing.T) {
		app := shinobi.New()
		app.SetName("")
		assert.Equal(t, "shinobi", app.Name())
	})
}

func TestApp_Server(t *testing.T) {
	app := shinobi.New()
	srv := app.Server(":9090")

	assert.Equal(t, ":9090", srv.Addr)
	assert.Equal[http.Handler](t, app, srv.Handler)
	assert.Equal(t, 5*time.Second, srv.ReadHeaderTimeout)
	assert.Equal(t, 10*time.Second, srv.ReadTimeout)
	assert.Equal(t, 30*time.Second, srv.WriteTimeout)
	assert.Equal(t, 120*time.Second, srv.IdleTimeout)
}

func TestApp_Listen_Error(t *testing.T) {
	assert.Error(t, shinobi.New().Listen("invalid-addr"))
}

func TestApp_ListenTLS_Error(t *testing.T) {
	err := shinobi.New().ListenTLS("invalid-addr", "cert.pem", "key.pem")

	assert.Error(t, err)
}

func TestApp_ListenGraceful_Error(t *testing.T) {
	err := shinobi.New().ListenGraceful("invalid-addr", time.Second)
	assert.Error(t, err)
}

func TestApp_ListenTLSGraceful_Error(t *testing.T) {
	err := shinobi.New().ListenTLSGraceful("invalid-addr", "cert.pem", "key.pem", time.Second)

	assert.Error(t, err)
}

func TestApp_ListenGraceful(t *testing.T) {
	ln, err := net.Listen("tcp", "localhost:0")
	assert.NoError(t, err)

	addr := ln.Addr().String()
	_ = ln.Close()

	app := shinobi.New()

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.ListenGraceful(addr, time.Second)
	}()

	time.Sleep(50 * time.Millisecond)

	p, _ := os.FindProcess(os.Getpid())
	_ = p.Signal(syscall.SIGINT)

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: server did not shut down")
	}
}

func TestApp_FileServer(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0o644)

	assert.NoError(t, err)

	app := shinobi.New()
	app.Mount("/static", shinobi.FileServer(dir))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/static/hello.txt", nil)

	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello", rec.Body.String())
}

func TestApp_FileServerFS(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0o644)

	assert.NoError(t, err)

	app := shinobi.New()
	app.Mount("/static", shinobi.FileServerFS(os.DirFS(dir)))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/static/hello.txt", nil)

	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello", rec.Body.String())
}

func TestApp_Handle(t *testing.T) {
	app := shinobi.New()
	app.Handle("GET /plain", shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "plain")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/plain", nil)
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "plain", rec.Body.String())
}

func TestApp_HandleFunc(t *testing.T) {
	app := shinobi.New()
	app.HandleFunc("GET /hf", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "handlefunc")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/hf", nil)
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "handlefunc", rec.Body.String())
}

func TestApp_Method(t *testing.T) {
	app := shinobi.New()
	app.Method(http.MethodGet, "/method", shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "method")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/method", nil)
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "method", rec.Body.String())
}

func TestApp_MethodFunc(t *testing.T) {
	app := shinobi.New()
	app.MethodFunc(http.MethodGet, "/mf", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "methodfunc")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mf", nil)
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "methodfunc", rec.Body.String())
}

func TestApp_HTTPVerb(t *testing.T) {
	cases := []struct {
		method   string
		register func(app *shinobi.App, h shinobi.HandlerFunc)
	}{
		{"GET", func(app *shinobi.App, h shinobi.HandlerFunc) { app.Get("/x", h) }},
		{"POST", func(app *shinobi.App, h shinobi.HandlerFunc) { app.Post("/x", h) }},
		{"PUT", func(app *shinobi.App, h shinobi.HandlerFunc) { app.Put("/x", h) }},
		{"PATCH", func(app *shinobi.App, h shinobi.HandlerFunc) { app.Patch("/x", h) }},
		{"HEAD", func(app *shinobi.App, h shinobi.HandlerFunc) { app.Head("/x", h) }},
		{"OPTIONS", func(app *shinobi.App, h shinobi.HandlerFunc) { app.Options("/x", h) }},
		{"DELETE", func(app *shinobi.App, h shinobi.HandlerFunc) { app.Delete("/x", h) }},
		{"CONNECT", func(app *shinobi.App, h shinobi.HandlerFunc) { app.Connect("/x", h) }},
		{"TRACE", func(app *shinobi.App, h shinobi.HandlerFunc) { app.Trace("/x", h) }},
	}

	for _, tc := range cases {
		t.Run(tc.method, func(t *testing.T) {
			app := shinobi.New()
			tc.register(app, func(c shinobi.Ctx) error {
				return c.String(http.StatusOK, tc.method)
			})

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "/x", nil)
			app.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestApp_Group(t *testing.T) {
	app := shinobi.New()
	app.Group("/api", func(r shinobi.Router) {
		r.Get("/ping", func(c shinobi.Ctx) error {
			return c.String(http.StatusOK, "pong")
		})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "pong", rec.Body.String())
}

func TestApp_Route(t *testing.T) {
	app := shinobi.New()
	app.Route("/users", func(rt shinobi.Route) {
		rt.Get(func(c shinobi.Ctx) error {
			return c.String(http.StatusOK, "get")
		})
		rt.Post(func(c shinobi.Ctx) error {
			return c.String(http.StatusCreated, "post")
		})
	})

	t.Run("GET", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		app.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("POST", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/users", nil)
		app.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code)
	})
}

func TestApp_Mount(t *testing.T) {
	app := shinobi.New()
	app.Mount("/api", shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "mounted")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1", nil)
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "mounted", rec.Body.String())
}

func TestApp_Mount_AdaptHTTP(t *testing.T) {
	app := shinobi.New()
	stdlib := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("stdlib"))
	})
	app.Mount("/ext", shinobi.AdaptHTTP(stdlib))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ext/anything", nil)
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "stdlib", rec.Body.String())
}

func TestApp_Mount_SubRouter(t *testing.T) {
	app := shinobi.New()
	sub := shinobi.NewRouter()
	sub.Get("/health", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})
	app.Mount("/v1", shinobi.AdaptHTTP(sub))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestApp_Delegation(t *testing.T) {
	called := false

	mw := func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			called = true
			return next.Handle(c)
		})
	}

	app := shinobi.New()
	app.Use(mw)
	app.Get("/check", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/check", nil)
	app.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestApp_With(t *testing.T) {
	called := false
	mw := func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			called = true
			return next.Handle(c)
		})
	}

	app := shinobi.New()
	app.With(mw).Get("/scoped", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/scoped", nil)
	app.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestApp_NotFound(t *testing.T) {
	app := shinobi.New()
	app.NotFound(func(c shinobi.Ctx) error {
		return c.String(http.StatusNotFound, "custom 404")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "custom 404", rec.Body.String())
}

func TestApp_Routes(t *testing.T) {
	app := shinobi.New()
	app.Get("/users", func(shinobi.Ctx) error { return nil })
	app.Post("/users", func(shinobi.Ctx) error { return nil })

	routes := app.Routes()

	assert.Len(t, routes, 2)
	assert.Equal(t, "GET", routes[0].Method)
	assert.Equal(t, "POST", routes[1].Method)
}

func TestApp_Debug(t *testing.T) {
	t.Run("IsDebug defaults to false", func(t *testing.T) {
		assert.Equal(t, false, shinobi.New().IsDebug())
	})

	t.Run("SetDebug enables debug mode", func(t *testing.T) {
		app := shinobi.New()
		app.SetDebug(true)
		assert.Equal(t, true, app.IsDebug())
	})

	t.Run("WithDebug enables debug mode", func(t *testing.T) {
		app := shinobi.New(shinobi.WithDebug(true))
		assert.Equal(t, true, app.IsDebug())
	})
}

func TestApp_SetLogger(t *testing.T) {
	t.Run("injects custom logger into Ctx", func(t *testing.T) {
		custom := slog.New(slog.NewTextHandler(io.Discard, nil))
		app := shinobi.New()
		app.SetLogger(custom)

		var got *slog.Logger
		app.Get("/log", func(c shinobi.Ctx) error {
			got = c.Logger()
			return nil
		})

		app.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/log", nil))

		assert.Equal(t, custom, got)
	})

	t.Run("ignores nil logger", func(t *testing.T) {
		app := shinobi.New()
		app.SetLogger(nil)

		var got *slog.Logger
		app.Get("/log2", func(c shinobi.Ctx) error {
			got = c.Logger()
			return nil
		})

		app.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/log2", nil))

		assert.NotNil(t, got)
	})
}

func TestApp_New_WithNotFound(t *testing.T) {
	app := shinobi.New(shinobi.WithNotFound(func(c shinobi.Ctx) error {
		return c.String(http.StatusNotFound, "not here")
	}))

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/unknown", nil))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "not here", rec.Body.String())
}

func TestDefaultErrorHandler_Debug_PlainError(t *testing.T) {
	app := shinobi.New(shinobi.WithDebug(true))
	app.Get("/fail", func(_ shinobi.Ctx) error {
		return errors.New("something went wrong")
	})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/fail", nil))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.StringContains(t, rec.Body.String(), "something went wrong")
}

func BenchmarkApp_ServeHTTP(b *testing.B) {
	app := shinobi.New()
	app.Get("/ping", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "pong")
	})
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	b.ResetTimer()
	for b.Loop() {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

func BenchmarkApp_ServeHTTP_PathParam(b *testing.B) {
	app := shinobi.New()
	app.Get("/users/{id}", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, c.Param("id"))
	})
	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)

	b.ResetTimer()
	for b.Loop() {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

func BenchmarkApp_ServeHTTP_Middleware(b *testing.B) {
	noop := func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			return next.Handle(c)
		})
	}
	app := shinobi.New()
	app.Use(noop, noop, noop)
	app.Get("/ping", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "pong")
	})
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	b.ResetTimer()
	for b.Loop() {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

// benchPayload is the shared struct payload for all response benchmarks.
// A struct avoids reflect.MapIter allocs that a map[string]any would add.
type benchPayload struct {
	ID    int    `json:"id"    xml:"id"`
	Name  string `json:"name"  xml:"name"`
	Email string `json:"email" xml:"email"`
}

var sharedPayload = benchPayload{ID: 1, Name: "alice", Email: "alice@example.com"}

// ── Response format benchmarks ────────────────────────────────────────────────

func BenchmarkCtx_String(b *testing.B) {
	app := shinobi.New()
	app.Get("/text", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "pong")
	})
	req := httptest.NewRequest(http.MethodGet, "/text", nil)
	for b.Loop() {
		app.ServeHTTP(httptest.NewRecorder(), req)
	}
}

func BenchmarkCtx_JSON(b *testing.B) {
	app := shinobi.New()
	app.Get("/json", func(c shinobi.Ctx) error {
		return c.JSON(http.StatusOK, sharedPayload, render.NoOptions)
	})
	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	for b.Loop() {
		app.ServeHTTP(httptest.NewRecorder(), req)
	}
}

func BenchmarkCtx_XML(b *testing.B) {
	app := shinobi.New()
	app.Get("/xml", func(c shinobi.Ctx) error {
		return c.XML(http.StatusOK, sharedPayload, render.NoOptions)
	})
	req := httptest.NewRequest(http.MethodGet, "/xml", nil)
	for b.Loop() {
		app.ServeHTTP(httptest.NewRecorder(), req)
	}
}

// ── Parallel benchmarks — expose contention on NewCtx/NewResponse ─────────────

func BenchmarkApp_ServeHTTP_Parallel(b *testing.B) {
	app := shinobi.New()
	app.Get("/ping", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "pong")
	})
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			app.ServeHTTP(httptest.NewRecorder(), req)
		}
	})
}

func BenchmarkCtx_JSON_Parallel(b *testing.B) {
	app := shinobi.New()
	app.Get("/json", func(c shinobi.Ctx) error {
		return c.JSON(http.StatusOK, sharedPayload, render.NoOptions)
	})
	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			app.ServeHTTP(httptest.NewRecorder(), req)
		}
	})
}
