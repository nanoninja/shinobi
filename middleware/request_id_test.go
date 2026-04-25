// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/shinobi"
	"github.com/nanoninja/shinobi/middleware"
)

func TestRequestID_Generated(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.RequestID())
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.NotEqual(t, "", rec.Header().Get("X-Request-ID"))
}

func TestRequestID_Reused(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.RequestID())
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "my-custom-id")

	r.ServeHTTP(rec, req)

	assert.Equal(t, "my-custom-id", rec.Header().Get("X-Request-ID"))
}

func TestRequestID_StoredInContext(t *testing.T) {
	var captured any

	r := shinobi.NewRouter()
	r.Use(middleware.RequestID())
	r.Get("/", func(c shinobi.Ctx) error {
		captured, _ = c.Get(middleware.RequestIDKey)
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.NotNil(t, captured)
	assert.Equal[any](t, rec.Header().Get("X-Request-ID"), captured)
}

func TestRequestID_Unique(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.RequestID())
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec1 := httptest.NewRecorder()
	rec2 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec1, req1)
	r.ServeHTTP(rec2, req2)

	assert.NotEqual(t,
		rec1.Header().Get("X-Request-ID"),
		rec2.Header().Get("X-Request-ID"),
	)
}
