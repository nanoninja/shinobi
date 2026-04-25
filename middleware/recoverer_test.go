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

func TestRecoverer_Panic(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.Recoverer())
	r.Get("/panic", func(shinobi.Ctx) error {
		panic("something went wrong")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRecoverer_NoPanic(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.Recoverer())
	r.Get("/ok", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestRecoverer_WithInternal(t *testing.T) {
	var captured error

	r := shinobi.NewRouter(
		shinobi.WithErrorHandler(func(err error, _ shinobi.Ctx) {
			captured = err
		}),
	)
	r.Use(middleware.Recoverer())
	r.Get("/panic", func(shinobi.Ctx) error {
		panic("boom")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)

	r.ServeHTTP(rec, req)

	e, ok := captured.(*shinobi.StatusError)

	assert.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, e.Code)
	assert.NotNil(t, e.Cause)
}
