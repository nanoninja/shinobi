// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/nanoninja/shinobi"
)

type CreateUser struct {
	Name  string `json:"name"  validate:"required,min=2"`
	Email string `json:"email" validate:"required,email"`
}

type customValidator struct {
	v *validator.Validate
}

func (cv *customValidator) Validate(v any) error {
	return cv.v.Struct(v)
}

func main() {
	app := shinobi.New(
		shinobi.WithValidator(&customValidator{
			v: validator.New(),
		}),
	)

	app.Post("/users", func(c shinobi.Ctx) error {
		var u CreateUser
		if err := c.Bind(&u); err != nil {
			return shinobi.HTTPError(http.StatusBadRequest, err.Error())
		}
		if err := c.Validate(&u); err != nil {
			return shinobi.HTTPError(http.StatusUnprocessableEntity, err.Error())
		}
		return c.JSON(http.StatusCreated, u)
	})

	log.Fatal(app.ListenGraceful(":8080", 10*time.Second))
}
