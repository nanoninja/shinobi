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

func TestLogger_LogsRequest(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.Logger())
	r.Get("/hello", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestLogger_WithRequestID(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEqual(t, "", rec.Header().Get("X-Request-ID"))
}

func TestLogger_PropagatesError(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.Logger())
	r.Get("/err", func(shinobi.Ctx) error {
		return shinobi.HTTPError(http.StatusBadRequest, "bad request")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/err", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
