// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/nanoninja/shinobi"
)

// compressWriter wraps http.ResponseWriter to compress the response body.
type compressWriter struct {
	http.ResponseWriter
	writer io.Writer
}

func (w *compressWriter) Write(b []byte) (int, error) {
	return w.writer.Write(b)
}

// Compress returns a middleware that compresses the response body using gzip or
// deflate, selected from the client's Accept-Encoding header. If the header is
// absent or unsupported, the response is passed through unmodified.
//
// The level parameter controls compression and must be a valid value for the
// chosen algorithm (e.g. gzip.DefaultCompression, flate.BestSpeed). An invalid
// level falls back to no compression.
//
// The middleware sets Content-Encoding and Vary: Accept-Encoding headers, and
// removes Content-Length since the compressed size differs from the original.
//
// Example:
//
//	app.Use(middleware.Compress(gzip.DefaultCompression))
func Compress(level int) shinobi.Middleware {
	return func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			encoding := c.Request().Header.Get("Accept-Encoding")
			switch {
			case strings.Contains(encoding, "gzip"):
				gz, err := gzip.NewWriterLevel(c.Response().ResponseWriter, level)
				if err != nil {
					return next.Handle(c)
				}
				defer func() { _ = gz.Close() }()
				c.Response().SetResponseWriter(func(w http.ResponseWriter) http.ResponseWriter {
					return &compressWriter{ResponseWriter: w, writer: gz}
				})
				c.Response().Header().Set("Content-Encoding", "gzip")
				c.Response().Header().Del("Content-Length")
				c.Response().Header().Add("Vary", "Accept-Encoding")

			case strings.Contains(encoding, "deflate"):
				fl, err := flate.NewWriter(c.Response().ResponseWriter, level)
				if err != nil {
					return next.Handle(c)
				}
				defer func() { _ = fl.Close() }()
				c.Response().SetResponseWriter(func(w http.ResponseWriter) http.ResponseWriter {
					return &compressWriter{ResponseWriter: w, writer: fl}
				})
				c.Response().Header().Set("Content-Encoding", "deflate")
				c.Response().Header().Del("Content-Length")
				c.Response().Header().Add("Vary", "Accept-Encoding")
			}

			return next.Handle(c)
		})
	}
}
