// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"testing"
	"time"

	"github.com/nanoninja/assert"
)

func TestRateLimiter_Allow_SlidingWindow(t *testing.T) {
	window := 50 * time.Millisecond
	rl := &rateLimiter{
		requests: make(map[string]*ipBucket),
		limit:    2,
		window:   window,
	}

	// Fill the buffer.
	rl.allow("1.2.3.4")
	rl.allow("1.2.3.4")

	// Buffer full — should be blocked.
	assert.False(t, rl.allow("1.2.3.4"))

	// Wait for the window to expire so oldest entry is before cutoff.
	time.Sleep(2 * window)

	// Oldest entry expired — sliding window reuses the slot.
	assert.True(t, rl.allow("1.2.3.4"))
}

func TestRateLimiter_Cleanup_EmptyBucket(t *testing.T) {
	rl := &rateLimiter{
		requests: map[string]*ipBucket{
			"ghost": {timestamps: make([]time.Time, 1), size: 0},
		},
		limit:  1,
		window: time.Millisecond,
	}

	rl.cleanup()

	_, exists := rl.requests["ghost"]
	assert.False(t, exists)
}
