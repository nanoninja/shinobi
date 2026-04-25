// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"net/http"
	"time"

	"github.com/nanoninja/shinobi"
)

// CreateUser is used for JSON/XML body binding.
type CreateUser struct {
	Name  string `json:"name"  xml:"name"`
	Email string `json:"email" xml:"email"`
}

// SearchParams is used for query string binding.
// GET /search?q=go&page=2&since=2026-01-01
type SearchParams struct {
	Query string    `query:"q"`
	Page  int       `query:"page"`
	Since time.Time `query:"since"`
}

// EventForm is used for form binding.
// POST /events with application/x-www-form-urlencoded body.
type EventForm struct {
	Title   string    `form:"title"`
	Date    time.Time `form:"date" format:"2006-01-02"`
	Private bool      `form:"private"`
}

func main() {
	app := shinobi.New(
		shinobi.WithBinder(shinobi.BinderRegistry{
			"application/json": shinobi.JSONBinder(
				shinobi.JSONStrict(),
				shinobi.JSONMaxBytes(1<<20), // 1 MB
			),
			"application/xml": shinobi.XMLBinder(),
		}),
	)

	// Body binding — JSON or XML based on Content-Type.
	app.Post("/users", func(c shinobi.Ctx) error {
		var u CreateUser
		if err := c.Bind(&u); err != nil {
			return shinobi.HTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusCreated, u)
	})

	// Query string binding with time.Time.
	app.Get("/search", func(c shinobi.Ctx) error {
		var p SearchParams
		if err := c.BindQuery(&p); err != nil {
			return shinobi.HTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusOK, p)
	})

	// Form binding with a custom date format.
	app.Post("/events", func(c shinobi.Ctx) error {
		var f EventForm
		if err := c.BindForm(&f); err != nil {
			return shinobi.HTTPError(http.StatusBadRequest, err.Error())
		}
		return c.JSON(http.StatusCreated, f)
	})

	log.Fatal(app.ListenGraceful(":8080", 10*time.Second))
}
