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

func TestNoCache_SetsHeaders(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h := middleware.NoCache()(shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	}))
	_ = h.Handle(shinobi.NewCtx(rec, r, shinobi.DefaultConfig()))

	assert.Equal(t, "no-store, no-cache, no-transform", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "no-cache", rec.Header().Get("Pragma"))
}

func TestNoCache_StripsConditionalHeaders(t *testing.T) {
	headers := []string{
		"ETag",
		"If-Modified-Since",
		"If-Match",
		"If-None-Match",
		"If-Range",
		"If-Unmodified-Since",
	}

	for _, header := range headers {
		t.Run(header, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set(header, "some-value")

			var got string
			h := middleware.NoCache()(shinobi.HandlerFunc(func(c shinobi.Ctx) error {
				got = c.Request().Header.Get(header)
				return nil
			}))
			_ = h.Handle(shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig()))

			assert.Equal(t, "", got)
		})
	}
}

func TestNoCache_CallsNext(t *testing.T) {
	called := false
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	h := middleware.NoCache()(shinobi.HandlerFunc(func(shinobi.Ctx) error {
		called = true
		return nil
	}))
	_ = h.Handle(shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig()))

	assert.True(t, called)
}

func TestNoCache_DoesNotStripUnrelatedHeaders(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer token")
	r.Header.Set("Content-Type", "application/json")

	var auth, ct string
	h := middleware.NoCache()(shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		auth = c.Request().Header.Get("Authorization")
		ct = c.Request().Header.Get("Content-Type")
		return nil
	}))
	_ = h.Handle(shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig()))

	assert.Equal(t, "Bearer token", auth)
	assert.Equal(t, "application/json", ct)
}
