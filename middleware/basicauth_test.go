// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/shinobi"
	"github.com/nanoninja/shinobi/middleware"
)

type logRecord struct {
	level   slog.Level
	message string
	attrs   map[string]string
}

type captureHandler struct {
	records []logRecord
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler         { return h }
func (h *captureHandler) WithGroup(_ string) slog.Handler              { return h }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	attrs := make(map[string]string)
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.String()
		return true
	})
	h.records = append(h.records, logRecord{
		level:   r.Level,
		message: r.Message,
		attrs:   attrs,
	})
	return nil
}

func withCaptureLogger(t *testing.T) *captureHandler {
	t.Helper()
	h := &captureHandler{}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })
	return h
}

func newBasicAuthRouter(cfg middleware.BasicAuthConfig) (shinobi.Router, func() bool) {
	nextCalled := false
	r := shinobi.NewRouter()
	r.Use(middleware.BasicAuth(cfg))
	r.Get("/", func(c shinobi.Ctx) error {
		nextCalled = true
		return c.String(http.StatusOK, "ok")
	})
	return r, func() bool { return nextCalled }
}

func TestBasicAuth_ValidCredentials(t *testing.T) {
	r, called := newBasicAuthRouter(middleware.BasicAuthConfig{
		Validator: middleware.Auth("alice", "s3cr3t"),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("alice", "s3cr3t")

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called())
}

func TestBasicAuth_WrongPassword(t *testing.T) {
	r, called := newBasicAuthRouter(middleware.BasicAuthConfig{
		Validator: middleware.Auth("alice", "s3cr3t"),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("alice", "wrong")

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called())
}

func TestBasicAuth_MissingHeader(t *testing.T) {
	r, called := newBasicAuthRouter(middleware.BasicAuthConfig{
		Validator: middleware.Auth("alice", "s3cr3t"),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called())
}

func TestBasicAuth_NilValidator(t *testing.T) {
	r, called := newBasicAuthRouter(middleware.BasicAuthConfig{
		Validator: nil,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("alice", "s3cr3t")

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called())
}

func TestBasicAuth_WWWAuthenticateHeader(t *testing.T) {
	r, _ := newBasicAuthRouter(middleware.BasicAuthConfig{
		Validator: middleware.Auth("alice", "s3cr3t"),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.StringContains(t, rec.Header().Get("WWW-Authenticate"), "Basic realm=")
	assert.StringContains(t, rec.Header().Get("WWW-Authenticate"), middleware.BasicAuthRealm)
}

func TestBasicAuth_CustomRealm(t *testing.T) {
	r, _ := newBasicAuthRouter(middleware.BasicAuthConfig{
		Realm:     "Admin Area",
		Validator: middleware.Auth("alice", "s3cr3t"),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.StringContains(t, rec.Header().Get("WWW-Authenticate"), "Admin Area")
}

func TestBasicAuth_Charset(t *testing.T) {
	r, _ := newBasicAuthRouter(middleware.BasicAuthConfig{
		Charset:   "UTF-8",
		Validator: middleware.Auth("alice", "s3cr3t"),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	r.ServeHTTP(rec, req)

	assert.StringContains(t, rec.Header().Get("WWW-Authenticate"), `charset="UTF-8"`)
}

func TestBasicAuth_CredentialInContext(t *testing.T) {
	var captured middleware.BasicAuthCredential

	r := shinobi.NewRouter()
	r.Use(middleware.BasicAuth(middleware.BasicAuthConfig{
		Validator: middleware.Auth("alice", "s3cr3t"),
	}))
	r.Get("/", func(c shinobi.Ctx) error {
		captured, _ = middleware.BasicAuthCredentialFrom(c)
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("alice", "s3cr3t")

	r.ServeHTTP(rec, req)

	assert.Equal(t, "alice", captured.Username)
	assert.Equal(t, "s3cr3t", captured.Password)
	assert.True(t, captured.OK)
}

func TestBasicAuth_UserMap(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.BasicAuth(middleware.BasicAuthConfig{
		Validator: middleware.User{"alice": "s3cr3t", "bob": "p4ssw0rd"},
	}))
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	t.Run("known user", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetBasicAuth("bob", "p4ssw0rd")
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("unknown user", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetBasicAuth("eve", "hack")
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestBasicAuth_ValidateFunc(t *testing.T) {
	r := shinobi.NewRouter()
	r.Use(middleware.BasicAuth(middleware.BasicAuthConfig{
		Validator: middleware.ValidateFunc(func(_ shinobi.Ctx, c middleware.BasicAuthCredential) bool {
			return middleware.SecureCompare(c.Username, "alice") &&
				middleware.SecureCompare(c.Password, "s3cr3t")
		}),
	}))
	r.Get("/", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("alice", "s3cr3t")

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSecureCompare(t *testing.T) {
	assert.True(t, middleware.SecureCompare("abc", "abc"))
	assert.False(t, middleware.SecureCompare("abc", "xyz"))
	assert.False(t, middleware.SecureCompare("abc", "ab"))
	assert.False(t, middleware.SecureCompare("", "abc"))
	assert.True(t, middleware.SecureCompare("", ""))
}

func TestBasicAuth_LogsSuccessfulAuth(t *testing.T) {
	h := withCaptureLogger(t)

	r, _ := newBasicAuthRouter(middleware.BasicAuthConfig{
		Validator: middleware.Auth("alice", "s3cr3t"),
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("alice", "s3cr3t")
	r.ServeHTTP(httptest.NewRecorder(), req)

	assert.Len(t, h.records, 1)
	assert.Equal(t, slog.LevelInfo, h.records[0].level)
	assert.Equal(t, "basicauth: success", h.records[0].message)
	assert.Equal(t, "alice", h.records[0].attrs["username"])
}

func TestBasicAuth_LogsFailedAuth(t *testing.T) {
	h := withCaptureLogger(t)

	r, _ := newBasicAuthRouter(middleware.BasicAuthConfig{
		Validator: middleware.Auth("alice", "s3cr3t"),
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("alice", "wrong")
	r.ServeHTTP(httptest.NewRecorder(), req)

	assert.Len(t, h.records, 1)
	assert.Equal(t, slog.LevelWarn, h.records[0].level)
	assert.Equal(t, "basicauth: unauthorized", h.records[0].message)
	assert.Equal(t, "alice", h.records[0].attrs["username"])
}

func TestBasicAuthCredentialFrom_NotPresent(t *testing.T) {
	c := shinobi.NewCtx(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil), shinobi.DefaultConfig())
	_, ok := middleware.BasicAuthCredentialFrom(c)
	assert.False(t, ok)
}
