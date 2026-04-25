// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shinobi_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/shinobi"
)

func TestHTTPError(t *testing.T) {
	err := shinobi.HTTPError(http.StatusNotFound, "user not found")

	assert.Equal(t, http.StatusNotFound, err.Code)
	assert.Equal(t, "user not found", err.Message)
}

func TestHTTPError_WithoutMessage(t *testing.T) {
	err := shinobi.HTTPError(http.StatusNotFound)

	assert.Equal(t, http.StatusNotFound, err.Code)
	assert.Nil(t, err.Message)
}

func TestStatusError_Error_WithMessage(t *testing.T) {
	err := shinobi.HTTPError(http.StatusNotFound, "user not found")

	assert.Equal(t, "404: user not found", err.Error())
}

func TestStatusError_Error_WithoutMessage(t *testing.T) {
	err := shinobi.HTTPError(http.StatusNotFound)

	assert.Equal(t, "404: Not Found", err.Error())
}

func TestStatusError_WithInternal(t *testing.T) {
	cause := errors.New("sql: no rows")
	err := shinobi.HTTPError(http.StatusNotFound, "user not found").WithInternal(cause)

	assert.Equal(t, http.StatusNotFound, err.Code)
	assert.ErrorIs(t, err, cause)
}

func TestStatusError_Unwrap(t *testing.T) {
	cause := errors.New("sql: no rows")
	err := shinobi.HTTPError(http.StatusNotFound).WithInternal(cause)

	assert.ErrorIs(t, err, cause)
}
