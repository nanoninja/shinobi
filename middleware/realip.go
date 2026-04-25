// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"net/http"
	"net/netip"
	"strings"

	"github.com/nanoninja/shinobi"
)

var (
	trueClientIP  = http.CanonicalHeaderKey("True-Client-IP")
	xForwardedFor = http.CanonicalHeaderKey("X-Forwarded-For")
	xRealIP       = http.CanonicalHeaderKey("X-Real-IP")
)

// RealIPConfig holds configuration for the RealIP middleware.
type RealIPConfig struct {
	// TrustedProxies is a list of CIDR ranges of trusted reverse proxies.
	// When non-empty, IP headers are only applied if the direct connection
	// originates from one of these ranges — prevents client IP spoofing.
	// When empty, headers are always trusted.
	// Panics at startup if any CIDR is invalid.
	TrustedProxies []string
}

// RealIP sets the request's RemoteAddr to the real client IP extracted from
// the True-Client-IP, X-Real-IP, or X-Forwarded-For headers (in that order).
// Only use this middleware when requests come through a trusted reverse proxy.
func RealIP() shinobi.Middleware {
	return RealIPWithConfig(RealIPConfig{})
}

// RealIPWithConfig returns a RealIP middleware with a custom configuration.
func RealIPWithConfig(cfg RealIPConfig) shinobi.Middleware {
	trusted := parsePrefixes(cfg.TrustedProxies)

	return func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			r := c.Request()
			if len(trusted) > 0 && !isTrustedProxy(r.RemoteAddr, trusted) {
				return next.Handle(c)
			}
			if addr, ok := realIP(r); ok {
				r.RemoteAddr = joinWithPort(r.RemoteAddr, addr)
			}
			return next.Handle(c)
		})
	}
}

// realIP extracts the client IP from the request headers, checking
// True-Client-IP, X-Real-IP, and X-Forwarded-For in that order.
// Returns the parsed address and false if no valid IP is found.
func realIP(r *http.Request) (netip.Addr, bool) {
	var raw string

	if tcip := r.Header.Get(trueClientIP); tcip != "" {
		raw = tcip
	} else if xrip := r.Header.Get(xRealIP); xrip != "" {
		raw = xrip
	} else if xff := r.Header.Get(xForwardedFor); xff != "" {
		raw, _, _ = strings.Cut(xff, ",")
	}
	raw = strings.TrimSpace(raw)
	addr, err := netip.ParseAddr(raw)
	if err != nil {
		return netip.Addr{}, false
	}
	return addr, true
}

// parsePrefixes parses a list of CIDR strings into netip.Prefix values.
// Prefixes are masked to their canonical form (e.g. 192.168.1.5/24 → 192.168.1.0/24).
// Panics if any CIDR is invalid.
func parsePrefixes(cidrs []string) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(cidrs))
	for _, cidr := range cidrs {
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			panic("realip: invalid CIDR " + cidr + ": " + err.Error())
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return prefixes
}

// isTrustedProxy reports whether remoteAddr falls within one of the trusted CIDR ranges.
func isTrustedProxy(remoteAddr string, trusted []netip.Prefix) bool {
	addr, ok := parseAddr(remoteAddr)
	if !ok {
		return false
	}
	for _, prefix := range trusted {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// joinWithPort replaces the host part of remoteAddr with ip, preserving the original port.
// Falls back to ip.String() alone if remoteAddr has no port.
func joinWithPort(remoteAddr string, ip netip.Addr) string {
	addrPort, err := netip.ParseAddrPort(remoteAddr)
	if err != nil {
		return ip.String()
	}
	return netip.AddrPortFrom(ip, addrPort.Port()).String()
}

// parseAddr parses remoteAddr as either a host:port pair or a bare IP address.
func parseAddr(remoteAddr string) (netip.Addr, bool) {
	if ap, err := netip.ParseAddrPort(remoteAddr); err == nil {
		return ap.Addr(), true
	}
	if a, err := netip.ParseAddr(remoteAddr); err == nil {
		return a, true
	}
	return netip.Addr{}, false
}
