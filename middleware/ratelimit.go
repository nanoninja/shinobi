// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"net/http"
	"net/netip"
	"sync"
	"time"

	"github.com/nanoninja/shinobi"
)

// RateLimit returns a middleware that limits the number of requests per IP
// address within the given time window. Requests exceeding the limit receive
// a 429 Too Many Requests response via the router's error handler.
func RateLimit(limit int, window time.Duration) shinobi.Middleware {
	rl := &rateLimiter{
		requests: make(map[string]*ipBucket),
		limit:    limit,
		window:   window,
	}

	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()

		for range ticker.C {
			rl.cleanup()
		}
	}()

	return func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			if !rl.allow(ipFromRequest(c.Request())) {
				return shinobi.HTTPError(http.StatusTooManyRequests)
			}
			return next.Handle(c)
		})
	}
}

type rateLimiter struct {
	mu       sync.Mutex
	requests map[string]*ipBucket
	limit    int
	window   time.Duration
}

// ipBucket is a fixed-size ring buffer that stores the timestamps of the last
// limit requests for a given IP. It avoids growing the slice unboundedly and
// keeps allow() at O(1) instead of O(n).
type ipBucket struct {
	timestamps []time.Time
	head       int // index of the oldest entry
	size       int // number of entries currently in use
}

func (rl *rateLimiter) allow(ip string) bool {
	now := time.Now()
	cutoff := now.Add(-rl.window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.requests[ip]
	if !ok {
		b = &ipBucket{timestamps: make([]time.Time, rl.limit)}
		rl.requests[ip] = b
	}

	// Buffer not yet full — add the timestamp in the next available slot.
	if b.size < rl.limit {
		b.timestamps[(b.head+b.size)%rl.limit] = now
		b.size++
		return true
	}

	// Buffer full — allow only if the oldest entry has expired.
	if b.timestamps[b.head].Before(cutoff) {
		b.timestamps[b.head] = now
		b.head = (b.head + 1) % rl.limit
		return true
	}

	return false
}

func (rl *rateLimiter) cleanup() {
	cutoff := time.Now().Add(-rl.window)
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for ip, b := range rl.requests {
		if b.size == 0 {
			delete(rl.requests, ip)
			continue
		}
		newest := (b.head + b.size - 1) % rl.limit
		if b.timestamps[newest].Before(cutoff) {
			delete(rl.requests, ip)
		}
	}
}

func ipFromRequest(r *http.Request) string {
	addr, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return addr.Addr().String()
}
