// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/nanoninja/shinobi"
	"github.com/nanoninja/shinobi/middleware"
)

const uploadDir = "./uploads"

func main() {
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		log.Fatalf("cannot create upload directory: %v", err)
	}

	app := shinobi.New()

	// Cap the entire request body at the transport level before any handler runs.
	app.Use(middleware.BodyLimit(10 * shinobi.MB))

	app.Post("/upload", handleUpload)

	log.Fatal(app.ListenGraceful(":8080", 10*time.Second))
}

func handleUpload(c shinobi.Ctx) error {
	f, header, err := c.FormFile("file")
	if err != nil {
		return shinobi.HTTPError(http.StatusBadRequest, "missing file field")
	}
	defer func() { _ = f.Close() }()

	// Validate size, MIME type, and extension before touching the filesystem.
	upload := shinobi.NewImageUpload(f, header, 5*shinobi.MB)
	if err := c.Validate(upload); err != nil {
		return err
	}

	// Build a safe destination path — never trust header.Filename directly.
	name := filepath.Base(filepath.Clean(header.Filename))
	dst := filepath.Join(uploadDir, fmt.Sprintf("%d-%s", time.Now().UnixNano(), name))

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("cannot create destination file: %w", err)
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, f); err != nil {
		return fmt.Errorf("cannot write file: %w", err)
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"file": filepath.Base(dst),
	})
}
