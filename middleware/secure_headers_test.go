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

func TestSecureHeaders_DefaultHeaders(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.SecureHeaders())
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "0", rec.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", rec.Header().Get("Referrer-Policy"))
}

func TestSecureHeaders_NextCalled(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.SecureHeaders())
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestSecureHeadersWithHSTS_Header(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.SecureHeadersWithHSTS(31536000, false))
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, "max-age=31536000", rec.Header().Get("Strict-Transport-Security"))
}

func TestSecureHeadersWithHSTS_IncludeSubdomains(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.SecureHeadersWithHSTS(31536000, true))
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, "max-age=31536000; includeSubDomains", rec.Header().Get("Strict-Transport-Security"))
}

func TestSecureHeadersWithHSTS_IncludesBaseHeaders(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.SecureHeadersWithHSTS(31536000, false))
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "0", rec.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", rec.Header().Get("Referrer-Policy"))
}
