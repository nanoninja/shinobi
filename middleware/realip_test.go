// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nanoninja/shinobi"
	"github.com/nanoninja/shinobi/middleware"
)

func TestRealIP_TrueClientIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("True-Client-IP", "203.0.113.1")

	var got string
	h := middleware.RealIP()(shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		got = c.Request().RemoteAddr
		return nil
	}))
	_ = h.Handle(shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig()))

	if got != "203.0.113.1:1234" {
		t.Errorf("got %q, want %q", got, "203.0.113.1:1234")
	}
}

func TestRealIP_XRealIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("X-Real-IP", "203.0.113.2")

	var got string
	h := middleware.RealIP()(shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		got = c.Request().RemoteAddr
		return nil
	}))
	_ = h.Handle(shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig()))

	if got != "203.0.113.2:1234" {
		t.Errorf("got %q, want %q", got, "203.0.113.2:1234")
	}
}

func TestRealIP_XForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.3, 10.0.0.1")

	var got string
	h := middleware.RealIP()(shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		got = c.Request().RemoteAddr
		return nil
	}))
	_ = h.Handle(shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig()))

	if got != "203.0.113.3:1234" {
		t.Errorf("got %q, want %q", got, "203.0.113.3:1234")
	}
}

func TestRealIP_TrueClientIPTakesPrecedence(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("True-Client-IP", "203.0.113.1")
	r.Header.Set("X-Real-IP", "203.0.113.2")
	r.Header.Set("X-Forwarded-For", "203.0.113.3")

	var got string
	h := middleware.RealIP()(shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		got = c.Request().RemoteAddr
		return nil
	}))
	_ = h.Handle(shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig()))

	if got != "203.0.113.1:1234" {
		t.Errorf("got %q, want %q", got, "203.0.113.1:1234")
	}
}

func TestRealIP_InvalidIPIgnored(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("X-Real-IP", "not-an-ip")

	var got string
	h := middleware.RealIP()(shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		got = c.Request().RemoteAddr
		return nil
	}))
	_ = h.Handle(shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig()))

	if got != "10.0.0.1:1234" {
		t.Errorf("got %q, want %q", got, "10.0.0.1:1234")
	}
}

func TestRealIP_NoHeadersLeaveRemoteAddrUnchanged(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"

	var got string
	h := middleware.RealIP()(shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		got = c.Request().RemoteAddr
		return nil
	}))
	_ = h.Handle(shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig()))

	if got != "10.0.0.1:1234" {
		t.Errorf("got %q, want %q", got, "10.0.0.1:1234")
	}
}

func TestRealIP_IPv6(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "[::1]:1234"
	r.Header.Set("X-Real-IP", "2001:db8::1")

	var got string
	h := middleware.RealIP()(shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		got = c.Request().RemoteAddr
		return nil
	}))
	_ = h.Handle(shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig()))

	if got != "[2001:db8::1]:1234" {
		t.Errorf("got %q, want %q", got, "[2001:db8::1]:1234")
	}
}

func TestRealIPWithConfig_TrustedProxy(t *testing.T) {
	cfg := middleware.RealIPConfig{TrustedProxies: []string{"10.0.0.0/8"}}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("X-Real-IP", "203.0.113.5")

	var got string
	h := middleware.RealIPWithConfig(cfg)(shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		got = c.Request().RemoteAddr
		return nil
	}))
	_ = h.Handle(shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig()))

	if got != "203.0.113.5:1234" {
		t.Errorf("got %q, want %q", got, "203.0.113.5:1234")
	}
}

func TestRealIPWithConfig_UntrustedProxyIgnored(t *testing.T) {
	cfg := middleware.RealIPConfig{TrustedProxies: []string{"10.0.0.0/8"}}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "1.2.3.4:1234"
	r.Header.Set("X-Real-IP", "203.0.113.5")

	var got string
	h := middleware.RealIPWithConfig(cfg)(shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		got = c.Request().RemoteAddr
		return nil
	}))
	_ = h.Handle(shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig()))

	if got != "1.2.3.4:1234" {
		t.Errorf("got %q, want %q", got, "1.2.3.4:1234")
	}
}

func TestRealIP_RemoteAddrWithoutPort(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1"
	r.Header.Set("X-Real-IP", "203.0.113.1")

	var got string
	h := middleware.RealIP()(shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		got = c.Request().RemoteAddr
		return nil
	}))
	_ = h.Handle(shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig()))

	if got != "203.0.113.1" {
		t.Errorf("got %q, want %q", got, "203.0.113.1")
	}
}

func TestRealIPWithConfig_TrustedProxyWithoutPort(t *testing.T) {
	cfg := middleware.RealIPConfig{TrustedProxies: []string{"10.0.0.0/8"}}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1"
	r.Header.Set("X-Real-IP", "203.0.113.5")

	var got string
	h := middleware.RealIPWithConfig(cfg)(shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		got = c.Request().RemoteAddr
		return nil
	}))
	_ = h.Handle(shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig()))

	if got != "203.0.113.5" {
		t.Errorf("got %q, want %q", got, "203.0.113.5")
	}
}

func TestRealIPWithConfig_UnparsableRemoteAddr(t *testing.T) {
	cfg := middleware.RealIPConfig{TrustedProxies: []string{"10.0.0.0/8"}}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "invalid"
	r.Header.Set("X-Real-IP", "203.0.113.5")

	var got string
	h := middleware.RealIPWithConfig(cfg)(shinobi.HandlerFunc(func(c shinobi.Ctx) error {
		got = c.Request().RemoteAddr
		return nil
	}))
	_ = h.Handle(shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig()))

	if got != "invalid" {
		t.Errorf("got %q, want %q", got, "invalid")
	}
}

func TestRealIPWithConfig_InvalidCIDRPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid CIDR")
		}
	}()
	middleware.RealIPWithConfig(middleware.RealIPConfig{
		TrustedProxies: []string{"not-a-cidr"},
	})
}
