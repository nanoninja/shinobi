// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shinobi

import (
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

// Validator validates a decoded value.
// No default implementation is provided — plug in the library of your choice
// (e.g. go-playground/validator) via WithValidator.
type Validator interface {
	Validate(v any) error
}

// Validatable may be implemented by any value that knows how to validate itself.
// If a value passed to Ctx.Validate implements this interface, its Validate method
// is called directly, bypassing the globally configured Validator.
// This is useful for types that carry their own validation logic (e.g. FileUpload)
// without depending on the application-level validator.
type Validatable interface {
	Validate() error
}

// FileUpload holds a multipart file and its metadata, and implements Validatable.
// Pass it to Ctx.Validate to enforce size, MIME type, and extension constraints.
//
// Size is measured by reading the actual file body — not from Header.Size which
// is declared by the client and cannot be trusted. The file is seeked back to
// the start after measurement.
//
// MIME type is detected from the first 512 bytes of the file content.
// By default http.DetectContentType is used (W3C MIME sniffing), which covers
// common formats but not all — SVG for instance is detected as text/xml.
// Provide a custom DetectFunc to use a more complete detection library.
//
// AllowedExtensions is checked against the filename declared by the client.
// It is spoofable on its own — always combine with AllowedTypes for reliable validation.
type FileUpload struct {
	File              multipart.File
	Header            *multipart.FileHeader
	MaxSize           int64               // maximum allowed size in bytes; 0 disables the check
	AllowedTypes      []string            // accepted MIME type prefixes; nil disables the check
	AllowedExtensions []string            // accepted extensions e.g. ".jpg"; nil disables the check
	DetectFunc        func([]byte) string // custom MIME detector; nil falls back to http.DetectContentType
}

// NewFileUploadValidator creates a bare FileUpload from the result of Ctx.FormFile.
// No constraints are set — configure MaxSize, AllowedTypes, AllowedExtensions,
// and DetectFunc directly on the returned value, or use a preset constructor.
func NewFileUploadValidator(file multipart.File, header *multipart.FileHeader) *FileUpload {
	return &FileUpload{
		File:   file,
		Header: header,
	}
}

// NewImageUpload creates a FileUpload preset for common web image formats
// (JPEG, PNG, GIF, WebP). maxSize is the maximum allowed file size in bytes.
func NewImageUpload(file multipart.File, header *multipart.FileHeader, maxSize int64) *FileUpload {
	return &FileUpload{
		File:              file,
		Header:            header,
		MaxSize:           maxSize,
		AllowedTypes:      []string{"image/jpeg", "image/png", "image/gif", "image/webp"},
		AllowedExtensions: []string{".jpg", ".jpeg", ".png", ".gif", ".webp"},
	}
}

// NewDocumentUpload creates a FileUpload preset for PDF documents.
// maxSize is the maximum allowed file size in bytes.
func NewDocumentUpload(file multipart.File, header *multipart.FileHeader, maxSize int64) *FileUpload {
	return &FileUpload{
		File:              file,
		Header:            header,
		MaxSize:           maxSize,
		AllowedTypes:      []string{"application/pdf"},
		AllowedExtensions: []string{".pdf"},
	}
}

// AddType appends one or more MIME type prefixes to the allowed types list.
// Returns the FileUpload to allow method chaining.
func (fu *FileUpload) AddType(t ...string) *FileUpload {
	fu.AllowedTypes = append(fu.AllowedTypes, t...)
	return fu
}

// AddExtension appends one or more file extensions to the allowed extensions list.
// Extensions should include the leading dot (e.g. ".svg").
// Returns the FileUpload to allow method chaining.
func (fu *FileUpload) AddExtension(exts ...string) *FileUpload {
	fu.AllowedExtensions = append(fu.AllowedExtensions, exts...)
	return fu
}

// Validate checks the file against the configured constraints in order:
// actual size, MIME type, and file extension. It implements Validatable
// and is called automatically by Ctx.Validate.
func (fu *FileUpload) Validate() error {
	if fu.MaxSize > 0 {
		lr := io.LimitReader(fu.File, fu.MaxSize+1)
		n, err := io.Copy(io.Discard, lr)
		if err != nil {
			return err
		}
		if _, err := fu.File.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if n > fu.MaxSize {
			return HTTPError(http.StatusRequestEntityTooLarge, "file too large")
		}
	}
	if len(fu.AllowedTypes) > 0 {
		buf := make([]byte, 512)
		n, err := fu.File.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if _, err := fu.File.Seek(0, io.SeekStart); err != nil {
			return err
		}
		detect := http.DetectContentType
		if fu.DetectFunc != nil {
			detect = fu.DetectFunc
		}
		detected := detect(buf[:n])
		allowed := false
		for _, t := range fu.AllowedTypes {
			if strings.HasPrefix(detected, t) {
				allowed = true
				break
			}
		}
		if !allowed {
			return HTTPError(http.StatusUnsupportedMediaType, "file type not allowed")
		}
	}
	if len(fu.AllowedExtensions) > 0 {
		ext := strings.ToLower(filepath.Ext(fu.Header.Filename))
		for _, e := range fu.AllowedExtensions {
			if strings.ToLower(e) == ext {
				return nil
			}
		}
		return HTTPError(http.StatusUnsupportedMediaType, "file extension not allowed")
	}
	return nil
}
