// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/shinobi"
	"github.com/nanoninja/shinobi/middleware"
)

func TestBodyLimit_WithinLimit(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.BodyLimit(shinobi.MB)) // 1 MB
	r.Post("/", func(c shinobi.Ctx) error {
		_, _ = io.ReadAll(c.Request().Body)
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("hello"))

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBodyLimit_ExceedsLimit(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.BodyLimit(4))
	r.Post("/", func(c shinobi.Ctx) error {
		_, err := io.ReadAll(c.Request().Body)
		return err
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("hello world"))

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

func TestBodyLimit_NextCalled(t *testing.T) {
	nextCalled := false

	r := shinobi.NewRouter()
	r.Use(middleware.BodyLimit(shinobi.MB))
	r.Post("/", func(c shinobi.Ctx) error {
		nextCalled = true
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("hello"))

	r.ServeHTTP(rec, req)

	assert.True(t, nextCalled)
}

func TestBodyLimit_NoBody(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.BodyLimit(shinobi.MB))
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
