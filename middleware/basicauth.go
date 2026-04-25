// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/nanoninja/shinobi"
)

// BasicAuthRealm is the default protection space sent in the WWW-Authenticate
// challenge when no realm is configured in BasicAuthConfig.
// See RFC 7617, section 2.
const BasicAuthRealm = "Authorization Required"

// basicAuthKey is the context key used to store the authenticated credential.
type basicAuthKey struct{}

// BasicAuthConfig holds the configuration for the BasicAuth middleware.
type BasicAuthConfig struct {
	// Realm is the protection space presented to the client in the
	// WWW-Authenticate header. Defaults to BasicAuthRealm if empty.
	// See RFC 7617, section 2.
	Realm string

	// Charset hints the client to encode credentials in the specified charset
	// before base64 encoding. RFC 7617 defines "UTF-8" as the only valid value.
	// If empty, no charset parameter is sent.
	// See RFC 7617, section 2.1.
	Charset string

	// Validator checks the credentials provided by the client.
	// If nil, all requests are rejected.
	Validator BasicAuthValidator
}

// BasicAuthCredential holds the credentials extracted from an HTTP Basic Auth request.
type BasicAuthCredential struct {
	// Username is the username decoded from the Authorization header.
	Username string

	// Password is the password decoded from the Authorization header.
	Password string

	// OK reports whether the Authorization header was present and parseable.
	OK bool
}

// BasicAuthValidator is the interface implemented by types that can authenticate
// a BasicAuthCredential. Implementations must not distinguish between
// "wrong password" and "user not found" — both in the response and in response
// time — to prevent username enumeration via timing attacks.
type BasicAuthValidator interface {
	Validate(c shinobi.Ctx, credential BasicAuthCredential) bool
}

// ValidateFunc is a function adapter that implements BasicAuthValidator.
// It lets you use an inline function wherever a BasicAuthValidator is expected:
//
//	middleware.BasicAuth(middleware.BasicAuthConfig{
//	    Validator: middleware.ValidateFunc(func(_ shinobi.Ctx, c middleware.BasicAuthCredential) bool {
//	        return middleware.SecureCompare(c.Username, "alice") &&
//	            middleware.SecureCompare(c.Password, "s3cr3t")
//	    }),
//	})
type ValidateFunc func(shinobi.Ctx, BasicAuthCredential) bool

// Validate calls f with the given Ctx and credential.
func (f ValidateFunc) Validate(c shinobi.Ctx, credential BasicAuthCredential) bool {
	return f(c, credential)
}

// User is a map of username to plain-text password that implements BasicAuthValidator.
// It is convenient for development and simple use cases, but plain-text passwords
// should not be used in production — prefer a custom BasicAuthValidator backed
// by bcrypt or argon2.
//
// Example:
//
//	middleware.BasicAuth(middleware.BasicAuthConfig{
//	    Validator: middleware.User{"alice": "s3cr3t", "bob": "p4ssw0rd"},
//	})
type User map[string]string

// Validate returns true if the credential matches an entry in the map.
// The password comparison is performed in constant time via SecureCompare.
func (u User) Validate(_ shinobi.Ctx, c BasicAuthCredential) bool {
	password, ok := u[c.Username]
	return ok && SecureCompare(c.Password, password)
}

// Auth returns a BasicAuthValidator that accepts exactly one username/password pair.
// Both comparisons are performed in constant time via SecureCompare.
//
// Example:
//
//	middleware.BasicAuth(middleware.BasicAuthConfig{
//	    Validator: middleware.Auth("alice", "s3cr3t"),
//	})
func Auth(username, password string) BasicAuthValidator {
	return ValidateFunc(func(_ shinobi.Ctx, c BasicAuthCredential) bool {
		return SecureCompare(c.Username, username) &&
			SecureCompare(c.Password, password)
	})
}

// SecureCompare reports whether a and b are equal using constant-time comparison
// to prevent timing attacks. Use it in BasicAuthValidator implementations
// instead of == when comparing credentials.
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// BasicAuthCredentialFrom retrieves the authenticated credential stored in the
// context by the BasicAuth middleware. Returns false if no credential is present.
//
// Example:
//
//	app.Get("/profile", func(c shinobi.Ctx) error {
//	    cred, ok := middleware.BasicAuthCredentialFrom(c)
//	    if !ok {
//	        return shinobi.HTTPError(http.StatusUnauthorized)
//	    }
//	    return c.String(http.StatusOK, "hello, %s", cred.Username)
//	})
func BasicAuthCredentialFrom(c shinobi.Ctx) (BasicAuthCredential, bool) {
	v, ok := c.Get(basicAuthKey{})
	if !ok {
		return BasicAuthCredential{}, false
	}
	cred, ok := v.(BasicAuthCredential)
	return cred, ok
}

// BasicAuth returns a middleware that enforces HTTP Basic Authentication.
// On success, the authenticated credential is stored in the context and
// retrievable via BasicAuthCredentialFrom. Unauthenticated requests receive
// a 401 with a WWW-Authenticate challenge.
//
// WARNING: Basic Auth credentials are base64-encoded, not encrypted.
// Always use this middleware behind HTTPS in production.
//
// Example:
//
//	app.Use(middleware.BasicAuth(middleware.BasicAuthConfig{
//	    Validator: middleware.Auth("alice", "s3cr3t"),
//	}))
//
//	// Multiple users:
//	app.Use(middleware.BasicAuth(middleware.BasicAuthConfig{
//	    Validator: middleware.User{"alice": "s3cr3t", "bob": "p4ssw0rd"},
//	}))
func BasicAuth(cfg BasicAuthConfig) shinobi.Middleware {
	realm := cfg.Realm
	if realm == "" {
		realm = BasicAuthRealm
	}
	if cfg.Validator == nil {
		return func(_ shinobi.Handler) shinobi.Handler {
			return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
				basicAuthChallenge(c.Response(), realm, cfg.Charset)
				return nil
			})
		}
	}

	return func(next shinobi.Handler) shinobi.Handler {
		return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			username, password, ok := c.Request().BasicAuth()
			credential := BasicAuthCredential{
				Username: username,
				Password: password,
				OK:       ok,
			}
			if credential.OK && cfg.Validator.Validate(c, credential) {
				c.Logger().Info("basicauth: success",
					"username", credential.Username,
					"ip", c.Request().RemoteAddr,
				)
				c.Set(basicAuthKey{}, credential)
				return next.Handle(c)
			}
			c.Logger().Warn("basicauth: unauthorized",
				"username", credential.Username,
				"ip", c.Request().RemoteAddr,
			)
			basicAuthChallenge(c.Response(), realm, cfg.Charset)
			return nil
		})
	}
}

// basicAuthChallenge sets the WWW-Authenticate header and writes a 401 status.
// The realm and charset values are sanitized to prevent header injection.
var basicAuthSanitizer = strings.NewReplacer(`"`, "", "\r", "", "\n", "")

func basicAuthChallenge(w http.ResponseWriter, realm, charset string) {
	header := `Basic realm="` + basicAuthSanitizer.Replace(realm) + `"`
	if charset != "" {
		header += `, charset="` + basicAuthSanitizer.Replace(charset) + `"`
	}
	w.Header().Set("WWW-Authenticate", header)
	w.WriteHeader(http.StatusUnauthorized)
}
