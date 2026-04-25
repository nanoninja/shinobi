// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shinobi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/shinobi"
)

func TestNewResponse(t *testing.T) {
	w := httptest.NewRecorder()
	r := shinobi.NewResponse(w)

	assert.Equal(t, http.StatusOK, r.Status())
	assert.Equal(t, false, r.Written())
}

func TestResponse_WriteHeader(t *testing.T) {
	t.Run("sets status and marks written", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := shinobi.NewResponse(w)

		r.WriteHeader(http.StatusNotFound)

		assert.Equal(t, http.StatusNotFound, r.Status())
		assert.Equal(t, true, r.Written())
	})

	t.Run("ignores subsequent calls", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := shinobi.NewResponse(w)

		r.WriteHeader(http.StatusCreated)
		r.WriteHeader(http.StatusInternalServerError)

		assert.Equal(t, http.StatusCreated, r.Status())
	})
}

func TestResponse_Written(t *testing.T) {
	w := httptest.NewRecorder()
	r := shinobi.NewResponse(w)

	assert.Equal(t, false, r.Written())

	r.WriteHeader(http.StatusOK)
	assert.Equal(t, true, r.Written())
}

func TestResponse_Write(t *testing.T) {
	w := httptest.NewRecorder()
	r := shinobi.NewResponse(w)

	n, err := r.Write([]byte("hello"))

	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, true, r.Written())
	assert.Equal(t, http.StatusOK, r.Status())
	assert.Equal(t, "hello", w.Body.String())
}

func TestResponse_WriteString(t *testing.T) {
	w := httptest.NewRecorder()
	r := shinobi.NewResponse(w)

	n, err := r.WriteString("hello")

	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, true, r.Written())
	assert.Equal(t, "hello", w.Body.String())
}

func TestResponse_Unwrap(t *testing.T) {
	w := httptest.NewRecorder()
	r := shinobi.NewResponse(w)

	assert.Equal[http.ResponseWriter](t, w, r.Unwrap())
}

func TestResponse_SetResponseWriter(t *testing.T) {
	t.Run("wraps the current writer", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := shinobi.NewResponse(w)

		var received http.ResponseWriter
		r.SetResponseWriter(func(current http.ResponseWriter) http.ResponseWriter {
			received = current
			return current
		})

		assert.Equal[http.ResponseWriter](t, w, received)
	})

	t.Run("replaces the underlying writer", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := shinobi.NewResponse(w)

		other := httptest.NewRecorder()
		r.SetResponseWriter(func(http.ResponseWriter) http.ResponseWriter {
			return other
		})

		_, err := r.Write([]byte("hello"))
		assert.NoError(t, err)

		assert.Equal(t, "hello", other.Body.String())
		assert.Equal(t, "", w.Body.String())
	})

	t.Run("chains multiple calls", func(t *testing.T) {
		var order []string
		w := httptest.NewRecorder()
		r := shinobi.NewResponse(w)

		r.SetResponseWriter(func(current http.ResponseWriter) http.ResponseWriter {
			order = append(order, "first")
			return current
		})
		r.SetResponseWriter(func(current http.ResponseWriter) http.ResponseWriter {
			order = append(order, "second")
			return current
		})

		assert.Equal(t, []string{"first", "second"}, order)
	})
}
