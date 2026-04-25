// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shinobi_test

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/render"
	"github.com/nanoninja/shinobi"
)

func TestRouter_WithBinder(t *testing.T) {
	r := shinobi.NewRouter(shinobi.WithBinder(shinobi.JSONBinder()))
	r.Post("/bind", func(c shinobi.Ctx) error {
		var got struct{ Name string }
		if err := c.Bind(&got); err != nil {
			return err
		}
		return c.String(http.StatusOK, got.Name)
	})

	body := strings.NewReader(`{"name":"alice"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bind", body)
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "alice", rec.Body.String())
}

func TestRouter_WithMux(t *testing.T) {
	mux := http.NewServeMux()
	r := shinobi.NewRouter(shinobi.WithMux(mux))
	r.Get("/hello", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "custom mux")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "custom mux", rec.Body.String())
}

func TestRouter_WithPrefix(t *testing.T) {
	r := shinobi.NewRouter(shinobi.WithPrefix("/api"))
	r.Get("/users", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "users")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "users", rec.Body.String())
}

func TestRouter_WithRenderer(t *testing.T) {
	r := shinobi.NewRouter(shinobi.WithRenderer(&mockRenderer{}))
	r.Get("/html", func(c shinobi.Ctx) error {
		return c.HTML(http.StatusOK, "index.html", nil, render.NoOptions)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/html", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "rendered", rec.Body.String())
}

func TestRouter_Handle(t *testing.T) {
	t.Run("registers plain path handler", func(t *testing.T) {
		r := shinobi.NewRouter()
		r.Handle("/plain", shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			return c.String(http.StatusOK, "plain")
		}))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/plain", nil)

		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "plain", rec.Body.String())
	})

	t.Run("normalizes lowercase method in pattern", func(t *testing.T) {
		r := shinobi.NewRouter()
		r.Handle("post /lower", shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			return c.String(http.StatusAccepted, "ok")
		}))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/lower", nil)

		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusAccepted, rec.Code)
	})
}

func TestRouter_HandleFunc(t *testing.T) {
	r := shinobi.NewRouter()
	r.HandleFunc("GET /hf", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "handlefunc")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/hf", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "handlefunc", rec.Body.String())
}

func TestRouter_MethodFunc(t *testing.T) {
	r := shinobi.NewRouter()
	r.MethodFunc(http.MethodGet, "/mf", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "methodfunc")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mf", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "methodfunc", rec.Body.String())
}

func TestRouter_Get(t *testing.T) {
	r := shinobi.NewRouter()
	r.Get("/hello", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "hello")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello", rec.Body.String())
}

func TestRouter_HTTPVerb(t *testing.T) {
	cases := []struct {
		method   string
		register func(r shinobi.Router, h shinobi.HandlerFunc)
	}{
		{"GET", func(r shinobi.Router, h shinobi.HandlerFunc) { r.Get("/x", h) }},
		{"PUT", func(r shinobi.Router, h shinobi.HandlerFunc) { r.Put("/x", h) }},
		{"PATCH", func(r shinobi.Router, h shinobi.HandlerFunc) { r.Patch("/x", h) }},
		{"HEAD", func(r shinobi.Router, h shinobi.HandlerFunc) { r.Head("/x", h) }},
		{"OPTIONS", func(r shinobi.Router, h shinobi.HandlerFunc) { r.Options("/x", h) }},
		{"DELETE", func(r shinobi.Router, h shinobi.HandlerFunc) { r.Delete("/x", h) }},
		{"CONNECT", func(r shinobi.Router, h shinobi.HandlerFunc) { r.Connect("/x", h) }},
		{"TRACE", func(r shinobi.Router, h shinobi.HandlerFunc) { r.Trace("/x", h) }},
	}

	for _, tc := range cases {
		t.Run(tc.method, func(t *testing.T) {
			r := shinobi.NewRouter()
			tc.register(r, func(c shinobi.Ctx) error {
				return c.String(http.StatusOK, tc.method)
			})

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "/x", nil)

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestRouter_Param(t *testing.T) {
	r := shinobi.NewRouter()
	r.Get("/users/{id}", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, c.Param("id"))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "42", rec.Body.String())
}

func TestRouter_Routes(t *testing.T) {
	r := shinobi.NewRouter()

	r.Get("/users", func(shinobi.Ctx) error { return nil })
	r.Post("/users", func(shinobi.Ctx) error { return nil })
	r.Delete("/users/{id}", func(shinobi.Ctx) error { return nil })

	routes := r.Routes()

	assert.Len(t, routes, 3)
	assert.Equal(t, "GET", routes[0].Method)
	assert.Equal(t, "/users", routes[0].Pattern)
	assert.Equal(t, "POST", routes[1].Method)
	assert.Equal(t, "DELETE", routes[2].Method)
	assert.Equal(t, "/users/{id}", routes[2].Pattern)
}

func TestRouter_NotFound(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		r := shinobi.NewRouter()
		r.Get("/", func(shinobi.Ctx) error {
			return nil
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("custom not found", func(t *testing.T) {
		notFoundHandler := func(c shinobi.Ctx) error {
			return c.String(http.StatusNotFound, "Not Found")
		}

		r := shinobi.NewRouter(shinobi.WithNotFound(notFoundHandler))
		r.Get("/", func(shinobi.Ctx) error {
			return nil
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/hello", nil)

		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "Not Found", rec.Body.String())
	})
}

func TestRouter_RootExact(t *testing.T) {
	notFoundHandler := func(c shinobi.Ctx) error {
		return c.String(http.StatusNotFound, "not found")
	}

	r := shinobi.NewRouter(shinobi.WithNotFound(notFoundHandler))
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "root")
	})

	t.Run("root matches exactly", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "root", rec.Body.String())
	})

	t.Run("unknown path goes to not found", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "not found", rec.Body.String())
	})
}

func TestRouter_Group(t *testing.T) {
	r := shinobi.NewRouter()
	r.Group("/api", func(g shinobi.Router) {
		g.Get("/users", func(c shinobi.Ctx) error {
			return c.String(http.StatusOK, "users")
		})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "users", rec.Body.String())
}

func TestRouter_Mount(t *testing.T) {
	r := shinobi.NewRouter()
	r.Mount("/static", shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "mounted")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/static/file.txt", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "mounted", rec.Body.String())
}

func TestRouter_Route(t *testing.T) {
	r := shinobi.NewRouter()
	r.Route("/users", func(rt shinobi.Route) {
		rt.Get(func(c shinobi.Ctx) error {
			return c.String(http.StatusOK, "get users")
		})
		rt.Post(func(c shinobi.Ctx) error {
			return c.String(http.StatusCreated, "created")
		})
	})

	t.Run("GET /users", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/users", nil)

		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "get users", rec.Body.String())
	})

	t.Run("POST /users", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/users", nil)

		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, "created", rec.Body.String())
	})

	t.Run("With applies middleware to all Route methods", func(t *testing.T) {
		var mwCallCount int
		mw := func(next shinobi.Handler) shinobi.Handler {
			return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
				mwCallCount++
				return next.Handle(c)
			})
		}

		r := shinobi.NewRouter()
		r.With(mw).Route("/guarded", func(rt shinobi.Route) {
			rt.Get(func(c shinobi.Ctx) error {
				return c.String(http.StatusOK, "ok")
			})
			rt.Post(func(c shinobi.Ctx) error {
				return c.String(http.StatusCreated, "created")
			})
		})

		for _, method := range []string{http.MethodGet, http.MethodPost} {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(method, "/guarded", nil)
			r.ServeHTTP(rec, req)
		}

		assert.Equal(t, 2, mwCallCount)
	})
}

func TestRouter_RouteVerbs(t *testing.T) {
	cases := []struct {
		method   string
		register func(rt shinobi.Route, h shinobi.HandlerFunc)
	}{
		{"PUT", func(rt shinobi.Route, h shinobi.HandlerFunc) { rt.Put(h) }},
		{"PATCH", func(rt shinobi.Route, h shinobi.HandlerFunc) { rt.Patch(h) }},
		{"HEAD", func(rt shinobi.Route, h shinobi.HandlerFunc) { rt.Head(h) }},
		{"OPTIONS", func(rt shinobi.Route, h shinobi.HandlerFunc) { rt.Options(h) }},
		{"DELETE", func(rt shinobi.Route, h shinobi.HandlerFunc) { rt.Delete(h) }},
		{"CONNECT", func(rt shinobi.Route, h shinobi.HandlerFunc) { rt.Connect(h) }},
		{"TRACE", func(rt shinobi.Route, h shinobi.HandlerFunc) { rt.Trace(h) }},
	}

	for _, tc := range cases {
		t.Run(tc.method, func(t *testing.T) {
			r := shinobi.NewRouter()
			r.Route("/x", func(rt shinobi.Route) {
				tc.register(rt, func(c shinobi.Ctx) error {
					return c.String(http.StatusOK, tc.method)
				})
			})

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "/x", nil)
			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestRouter_ChainBuiltAtRegistration(t *testing.T) {
	var callCount int
	mw := func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			callCount++
			return next.Handle(c)
		})
	}

	r := shinobi.NewRouter()
	r.Get("/ping", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "pong")
	})
	r.Use(mw)

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))

	assert.Equal(t, 0, callCount)
}

func TestRouter_Use(t *testing.T) {
	called := false

	mw := func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			called = true
			return next.Handle(c)
		})
	}

	r := shinobi.NewRouter()
	r.Use(mw)
	r.Get("/ping", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "pong")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	r.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRouter_With(t *testing.T) {
	called := false
	mw := func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			called = true
			return next.Handle(c)
		})
	}

	r := shinobi.NewRouter()
	r.With(mw).Get("/scoped", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	r.Get("/other", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "other")
	})

	t.Run("scoped route executes middleware", func(t *testing.T) {
		called = false

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/scoped", nil)

		r.ServeHTTP(rec, req)

		assert.True(t, called)
	})

	t.Run("other route skips middleware", func(t *testing.T) {
		called = false

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/other", nil)

		r.ServeHTTP(rec, req)

		assert.False(t, called)
	})
}

func TestRouter_ErrorHandler(t *testing.T) {
	t.Run("default error handler returns 500", func(t *testing.T) {
		r := shinobi.NewRouter()
		r.Get("/fail", func(shinobi.Ctx) error {
			return errors.New("boom")
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/fail", nil)

		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("custom error handler writes error body", func(t *testing.T) {
		customErr := errors.New("something went wrong")

		r := shinobi.NewRouter(
			shinobi.WithErrorHandler(func(err error, c shinobi.Ctx) {
				_ = c.String(http.StatusInternalServerError, err.Error())
			}),
		)
		r.Get("/fail", func(shinobi.Ctx) error {
			return customErr
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/fail", nil)

		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "something went wrong", rec.Body.String())
	})
}

func TestRouter_WrapPath_NoLeadingSlash(t *testing.T) {
	r := shinobi.NewRouter(shinobi.WithPrefix("api"))
	r.Get("/users", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "users")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "users", rec.Body.String())
}

func TestRouter_WrapPath_TrailingSlash(t *testing.T) {
	r := shinobi.NewRouter()
	r.Handle("/api/", shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "api")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1", nil)
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRouter_WithDebug(t *testing.T) {
	t.Run("propagates debug flag to Ctx", func(t *testing.T) {
		var got bool
		r := shinobi.NewRouter(shinobi.WithDebug(true))
		r.Get("/", func(c shinobi.Ctx) error {
			got = c.Debug()
			return nil
		})

		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

		assert.Equal(t, true, got)
	})

	t.Run("default is false", func(t *testing.T) {
		var got bool
		r := shinobi.NewRouter()
		r.Get("/", func(c shinobi.Ctx) error {
			got = c.Debug()
			return nil
		})

		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

		assert.Equal(t, false, got)
	})
}

func TestRouter_WithLogger(t *testing.T) {
	t.Run("injects custom logger into Ctx", func(t *testing.T) {
		custom := slog.New(slog.NewTextHandler(io.Discard, nil))
		var got *slog.Logger

		r := shinobi.NewRouter(shinobi.WithLogger(custom))
		r.Get("/", func(c shinobi.Ctx) error {
			got = c.Logger()
			return nil
		})

		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

		assert.Equal(t, custom, got)
	})

	t.Run("ignores nil logger", func(t *testing.T) {
		var got *slog.Logger

		r := shinobi.NewRouter(shinobi.WithLogger(nil))
		r.Get("/", func(c shinobi.Ctx) error {
			got = c.Logger()
			return nil
		})

		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

		assert.NotNil(t, got)
	})
}
