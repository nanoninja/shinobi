// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/shinobi"
	"github.com/nanoninja/shinobi/middleware"
)

func TestTimeout_WithinDeadline(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.Timeout(100 * time.Millisecond))
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestTimeout_Exceeded_NoResponse(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.Timeout(10 * time.Millisecond))
	r.Get("/", func(c shinobi.Ctx) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return c.String(http.StatusOK, "ok")
		case <-c.Context().Done():
			return nil
		}
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestTimeout_Exceeded_ResponseAlreadyWritten(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.Timeout(50 * time.Millisecond))
	r.Get("/", func(c shinobi.Ctx) error {
		_ = c.String(http.StatusOK, "partial")
		select {
		case <-time.After(200 * time.Millisecond):
		case <-c.Context().Done():
		}
		return nil
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTimeout_ContextPropagated(t *testing.T) {
	var ctxErr error

	r := shinobi.NewRouter()
	r.Use(middleware.Timeout(10 * time.Millisecond))
	r.Get("/", func(c shinobi.Ctx) error {
		<-c.Context().Done()
		ctxErr = c.Context().Err()
		return nil
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, true, ctxErr != nil)
}
