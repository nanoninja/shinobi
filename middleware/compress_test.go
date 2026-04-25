// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware_test

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/shinobi"
	"github.com/nanoninja/shinobi/middleware"
)

func newCompressRouter(level int) shinobi.Router {
	r := shinobi.NewRouter()
	r.Use(middleware.Compress(level))
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "hello world")
	})
	return r
}

func TestCompress_Gzip(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	newCompressRouter(gzip.DefaultCompression).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
	assert.Equal(t, "Accept-Encoding", rec.Header().Get("Vary"))

	gr, err := gzip.NewReader(rec.Body)
	assert.NoError(t, err)
	defer func() { _ = gr.Close() }()

	body, err := io.ReadAll(gr)
	assert.NoError(t, err)
	assert.Equal(t, "hello world", string(body))
}

func TestCompress_Deflate(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "deflate")

	newCompressRouter(flate.DefaultCompression).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "deflate", rec.Header().Get("Content-Encoding"))
	assert.Equal(t, "Accept-Encoding", rec.Header().Get("Vary"))

	fr := flate.NewReader(rec.Body)
	defer func() { _ = fr.Close() }()

	body, err := io.ReadAll(fr)
	assert.NoError(t, err)
	assert.Equal(t, "hello world", string(body))
}

func TestCompress_NoEncoding(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	newCompressRouter(gzip.DefaultCompression).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "", rec.Header().Get("Content-Encoding"))
	assert.Equal(t, "hello world", rec.Body.String())
}

func TestCompress_ContentLengthRemoved(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	newCompressRouter(gzip.DefaultCompression).ServeHTTP(rec, req)

	assert.Equal(t, "", rec.Header().Get("Content-Length"))
}

func TestCompress_InvalidLevel_Gzip(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	newCompressRouter(999).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello world", rec.Body.String())
}

func TestCompress_InvalidLevel_Deflate(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "deflate")

	newCompressRouter(999).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello world", rec.Body.String())
}
