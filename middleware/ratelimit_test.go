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

func newRateLimitRouter(limit int, window time.Duration) shinobi.Router {
	r := shinobi.NewRouter()
	r.Use(middleware.RateLimit(limit, window))
	r.Get("/ping", func(c shinobi.Ctx) error {
		return c.String(http.StatusOK, "pong")
	})
	return r
}

func TestRateLimit_AllowsUnderLimit(t *testing.T) {
	r := newRateLimitRouter(3, time.Minute)

	for range 3 {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "1.2.3.4:1000"
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestRateLimit_BlocksOverLimit(t *testing.T) {
	r := newRateLimitRouter(2, time.Minute)

	for range 2 {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "1.2.3.4:1000"
		r.ServeHTTP(rec, req)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "1.2.3.4:1000"
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimit_IsolatesPerIP(t *testing.T) {
	r := newRateLimitRouter(1, time.Minute)

	for _, ip := range []string{"1.1.1.1:1000", "2.2.2.2:1000", "3.3.3.3:1000"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = ip
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestRateLimit_SameIPDifferentPorts(t *testing.T) {
	r := newRateLimitRouter(1, time.Minute)

	req1 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req1.RemoteAddr = "1.2.3.4:1000"
	r.ServeHTTP(httptest.NewRecorder(), req1)

	rec := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req2.RemoteAddr = "1.2.3.4:9999"
	r.ServeHTTP(rec, req2)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimit_ResetsAfterWindow(t *testing.T) {
	r := newRateLimitRouter(1, 50*time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "1.2.3.4:1000"
	r.ServeHTTP(httptest.NewRecorder(), req)

	time.Sleep(60 * time.Millisecond)

	rec := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req2.RemoteAddr = "1.2.3.4:1000"
	r.ServeHTTP(rec, req2)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_InvalidRemoteAddr(t *testing.T) {
	r := newRateLimitRouter(5, time.Minute)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "not-a-valid-addr"
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_CleanupRemovesExpiredEntries(t *testing.T) {
	window := 50 * time.Millisecond
	r := newRateLimitRouter(5, window)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "9.9.9.9:1000"
	r.ServeHTTP(httptest.NewRecorder(), req)

	// Wait for window + cleanup tick to fire.
	time.Sleep(3 * window)

	// A new request from the same IP must be allowed — bucket was cleaned up.
	rec := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req2.RemoteAddr = "9.9.9.9:1000"
	r.ServeHTTP(rec, req2)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func BenchmarkRateLimit_Allow(b *testing.B) {
	r := newRateLimitRouter(1000, time.Minute)
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "1.2.3.4:1000"

	b.ResetTimer()
	for b.Loop() {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
	}
}

func BenchmarkRateLimit_AllowParallel(b *testing.B) {
	r := newRateLimitRouter(1000, time.Minute)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "1.2.3.4:1000"
		rec := httptest.NewRecorder()
		for pb.Next() {
			r.ServeHTTP(rec, req)
		}
	})
}

func TestRateLimit_SlidingWindow(t *testing.T) {
	window := 50 * time.Millisecond
	r := newRateLimitRouter(2, window)

	for range 2 {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "5.5.5.5:1000"

		r.ServeHTTP(httptest.NewRecorder(), req)
	}

	time.Sleep(3 * window)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	req.RemoteAddr = "5.5.5.5:1000"
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// func TestRateLimiter_Cleanup_EmptyBucket(t *testing.T) {
// 	rl := &rateLimiter{
// 		requests: map[string]*ipBucket{
// 			"ghost": {timestamps: make([]time.Time, 1), size: 0},
// 		},
// 		limit:   1,
// 		windows: time.Millisecond,
// 	}

// 	rl.cleanup()
// }
