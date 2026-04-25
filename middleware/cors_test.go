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

func TestCORS_WildcardOrigin(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.CORS(middleware.DefaultCORSConfig()))
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.NotEqual(t, "", rec.Header().Get("Access-Control-Allow-Methods"))
	assert.NotEqual(t, "", rec.Header().Get("Access-Control-Allow-Headers"))
}

func TestCORS_PreflightWildcard(t *testing.T) {
	nextCalled := false

	r := shinobi.NewRouter()
	r.Use(middleware.CORS(middleware.DefaultCORSConfig()))
	r.Options("/", func(c shinobi.Ctx) error {
		nextCalled = true
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "86400", rec.Header().Get("Access-Control-Max-Age"))
	assert.False(t, nextCalled)
}

func TestCORS_AllowedOriginReflected(t *testing.T) {
	cfg := middleware.DefaultCORSConfig("https://a.com", "https://b.com")

	r := shinobi.NewRouter()
	r.Use(middleware.CORS(cfg))
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://a.com")

	r.ServeHTTP(rec, req)

	assert.Equal(t, "https://a.com", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Origin", rec.Header().Get("Vary"))
}

func TestCORS_UnknownOriginIgnored(t *testing.T) {
	cfg := middleware.DefaultCORSConfig("https://a.com")

	r := shinobi.NewRouter()
	r.Use(middleware.CORS(cfg))
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.com")

	r.ServeHTTP(rec, req)

	assert.Equal(t, "", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_AllowCredentials(t *testing.T) {
	cfg := middleware.DefaultCORSConfig("https://a.com")
	cfg.AllowCredentials = true

	r := shinobi.NewRouter()
	r.Use(middleware.CORS(cfg))
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://a.com")

	r.ServeHTTP(rec, req)

	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORS_NoCredentialsByDefault(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.CORS(middleware.DefaultCORSConfig()))
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, "", rec.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORS_MaxAgeOnPreflightOnly(t *testing.T) {
	cfg := middleware.DefaultCORSConfig()
	cfg.MaxAge = 3600

	r := shinobi.NewRouter()
	r.Use(middleware.CORS(cfg))
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, "", rec.Header().Get("Access-Control-Max-Age"))
}

func TestCORS_WildcardWithCredentialsPanics(t *testing.T) {
	assert.Panics(t, func() {
		middleware.CORS(middleware.CORSConfig{
			AllowedOrigins:   []string{"*"},
			AllowCredentials: true,
		})
	}, "cors: AllowCredentials requires explicit origins, wildcard (*) is not allowed")
}

func TestCORS_NoMaxAgeWhenZero(t *testing.T) {
	cfg := middleware.DefaultCORSConfig()
	cfg.MaxAge = 0

	r := shinobi.NewRouter()
	r.Use(middleware.CORS(cfg))
	r.Options("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")

	r.ServeHTTP(rec, req)

	assert.Equal(t, "", rec.Header().Get("Access-Control-Max-Age"))
}
