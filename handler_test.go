// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shinobi_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/shinobi"
)

func TestHandlerFunc(t *testing.T) {
	t.Run("calls wrapped function", func(t *testing.T) {
		called := false

		fn := shinobi.HandlerFunc(func(shinobi.Ctx) error {
			called = true
			return nil
		})

		c := shinobi.NewCtx(
			httptest.NewRecorder(),
			httptest.NewRequest(http.MethodGet, "/", nil),
			shinobi.DefaultConfig(),
		)
		err := fn.Handle(c)

		assert.NoError(t, err)
		assert.Equal(t, true, called)
	})
}

func TestAdaptHTTP(t *testing.T) {
	stdlib := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	w := httptest.NewRecorder()
	c := shinobi.NewCtx(w, httptest.NewRequest(http.MethodGet, "/", nil), shinobi.DefaultConfig())

	err := shinobi.AdaptHTTP(stdlib).Handle(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, w.Code)
}

func TestAdapt(t *testing.T) {
	t.Run("executes stdlib middleware", func(t *testing.T) {
		called := false

		mw := shinobi.Adapt(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				next.ServeHTTP(w, r)
			})
		})

		c := shinobi.NewCtx(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil), shinobi.DefaultConfig())
		err := mw(shinobi.HandlerFunc(func(shinobi.Ctx) error { return nil })).Handle(c)

		assert.NoError(t, err)
		assert.Equal(t, true, called)
	})

	t.Run("propagates context changes", func(t *testing.T) {
		type key struct{}

		mw := shinobi.Adapt(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := context.WithValue(r.Context(), key{}, "injected")
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})

		var got any

		c := shinobi.NewCtx(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil), shinobi.DefaultConfig())
		_ = mw(shinobi.HandlerFunc(func(c shinobi.Ctx) error {
			got, _ = c.Get(key{})
			return nil
		})).Handle(c)

		assert.Equal(t, "injected", got)
	})

	t.Run("returns error from next handler", func(t *testing.T) {
		sentinel := errors.New("handler error")
		mw := shinobi.Adapt(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		})

		c := shinobi.NewCtx(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil), shinobi.DefaultConfig())
		err := mw(shinobi.HandlerFunc(func(shinobi.Ctx) error { return sentinel })).Handle(c)

		assert.ErrorIs(t, err, sentinel)
	})
}

func TestDefaultErrorHandler(t *testing.T) {
	t.Run("StatusError responds with status text not message", func(t *testing.T) {
		r := shinobi.NewRouter()
		r.Get("/err", func(shinobi.Ctx) error {
			return shinobi.HTTPError(http.StatusNotFound, "user not found")
		})

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/err", nil))

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "Not Found\n", rec.Body.String())
	})

	t.Run("generic error responds with 500 and status text", func(t *testing.T) {
		r := shinobi.NewRouter()
		r.Get("/fail", func(shinobi.Ctx) error {
			return errors.New("secret internal error")
		})

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/fail", nil))

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "Internal Server Error\n", rec.Body.String())
	})

	t.Run("StatusError with internal cause does not expose cause", func(t *testing.T) {
		r := shinobi.NewRouter()
		r.Get("/fail", func(shinobi.Ctx) error {
			return shinobi.HTTPError(http.StatusInternalServerError).WithInternal(errors.New("db connection failed"))
		})

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/fail", nil))

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "Internal Server Error\n", rec.Body.String())
	})

	t.Run("StatusError exposes message in debug mode", func(t *testing.T) {
		r := shinobi.NewRouter(shinobi.WithDebug(true))
		r.Get("/err", func(shinobi.Ctx) error {
			return shinobi.HTTPError(http.StatusBadRequest, "user not found")
		})

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/err", nil))

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "user not found\n", rec.Body.String())
	})

	t.Run("generic error exposes message in debug mode", func(t *testing.T) {
		r := shinobi.NewRouter(shinobi.WithDebug(true))
		r.Get("/fail", func(shinobi.Ctx) error {
			return errors.New("secret detail")
		})

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/fail", nil))

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "secret detail\n", rec.Body.String())
	})
}
