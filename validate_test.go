// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shinobi_test

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/shinobi"
)

// multipartFile wraps *bytes.Reader and adds the Close method required by multipart.File.
type multipartFile struct {
	*bytes.Reader
}

func (multipartFile) Close() error { return nil }

// errReadFile simulates a file whose Read method always returns an error.
type errReadFile struct {
	*bytes.Reader
}

func (errReadFile) Read(_ []byte) (int, error) { return 0, errors.New("read error") }
func (errReadFile) Close() error               { return nil }

// errSeekFile simulates a file whose Seek method always returns an error.
type errSeekFile struct {
	*bytes.Reader
}

func (errSeekFile) Seek(_ int64, _ int) (int64, error) { return 0, errors.New("seek error") }
func (errSeekFile) Close() error                       { return nil }

func newMultipartFile(t testing.TB, filename string, content []byte) (multipart.File, *multipart.FileHeader) {
	t.Helper()
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", http.DetectContentType(content))
	header := &multipart.FileHeader{
		Filename: filename,
		Header:   h,
		Size:     int64(len(content)),
	}
	return multipartFile{bytes.NewReader(content)}, header
}

// --- NewFileUploadValidator ---

func TestNewFileUploadValidator_NoConstraints(t *testing.T) {
	f, h := newMultipartFile(t, "file.txt", []byte("hello"))

	err := shinobi.NewFileUploadValidator(f, h).Validate()

	assert.NoError(t, err)
}

// --- MaxSize ---

func TestFileUpload_MaxSize_Within(t *testing.T) {
	f, h := newMultipartFile(t, "file.txt", []byte("hello"))
	fu := shinobi.NewFileUploadValidator(f, h)
	fu.MaxSize = 10

	assert.NoError(t, fu.Validate())
}

func TestFileUpload_MaxSize_Exceeded(t *testing.T) {
	f, h := newMultipartFile(t, "file.txt", bytes.Repeat([]byte("x"), 100))
	fu := shinobi.NewFileUploadValidator(f, h)
	fu.MaxSize = 10

	err := fu.Validate()
	assert.Error(t, err)

	var se *shinobi.StatusError
	assert.ErrorAs(t, err, &se)
	assert.Equal(t, http.StatusRequestEntityTooLarge, se.Code)
}

func TestFileUpload_MaxSize_Zero_Disabled(t *testing.T) {
	f, h := newMultipartFile(t, "file.txt", bytes.Repeat([]byte("x"), 1000))
	fu := shinobi.NewFileUploadValidator(f, h)
	fu.MaxSize = 0

	assert.NoError(t, fu.Validate())
}

func TestFileUpload_MaxSize_ReadError(t *testing.T) {
	_, h := newMultipartFile(t, "file.txt", []byte("hello"))
	fu := shinobi.NewFileUploadValidator(errReadFile{bytes.NewReader([]byte("hello"))}, h)
	fu.MaxSize = 100

	assert.Error(t, fu.Validate())
}

func TestFileUpload_MaxSize_SeekError(t *testing.T) {
	_, h := newMultipartFile(t, "file.txt", []byte("hello"))
	fu := shinobi.NewFileUploadValidator(errSeekFile{bytes.NewReader([]byte("hello"))}, h)
	fu.MaxSize = 100

	assert.Error(t, fu.Validate())
}

// --- AllowedTypes ---

func TestFileUpload_AllowedTypes_Match(t *testing.T) {
	content := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	f, h := newMultipartFile(t, "photo.jpg", content)
	fu := shinobi.NewFileUploadValidator(f, h)
	fu.AllowedTypes = []string{"image/jpeg"}

	assert.NoError(t, fu.Validate())
}

func TestFileUpload_AllowedTypes_NoMatch(t *testing.T) {
	f, h := newMultipartFile(t, "file.txt", []byte("hello world"))
	fu := shinobi.NewFileUploadValidator(f, h)
	fu.AllowedTypes = []string{"image/jpeg", "image/png"}

	err := fu.Validate()
	assert.Error(t, err)

	var se *shinobi.StatusError
	assert.ErrorAs(t, err, &se)
	assert.Equal(t, http.StatusUnsupportedMediaType, se.Code)
}

func TestFileUpload_AllowedTypes_ReadError(t *testing.T) {
	_, h := newMultipartFile(t, "file.txt", []byte("hello"))
	fu := shinobi.NewFileUploadValidator(errReadFile{bytes.NewReader([]byte("hello"))}, h)
	fu.AllowedTypes = []string{"image/jpeg"}

	assert.Error(t, fu.Validate())
}

func TestFileUpload_AllowedTypes_SeekError(t *testing.T) {
	_, h := newMultipartFile(t, "file.txt", []byte("hello"))
	fu := shinobi.NewFileUploadValidator(errSeekFile{bytes.NewReader([]byte("hello"))}, h)
	fu.AllowedTypes = []string{"text/plain"}

	assert.Error(t, fu.Validate())
}

func TestFileUpload_AllowedTypes_CustomDetectFunc(t *testing.T) {
	f, h := newMultipartFile(t, "image.svg", []byte("<svg></svg>"))
	fu := shinobi.NewFileUploadValidator(f, h)
	fu.AllowedTypes = []string{"image/svg+xml"}
	fu.DetectFunc = func(_ []byte) string { return "image/svg+xml" }

	assert.NoError(t, fu.Validate())
}

// --- AllowedExtensions ---

func TestFileUpload_AllowedExtensions_Match(t *testing.T) {
	f, h := newMultipartFile(t, "photo.jpg", []byte("data"))
	fu := shinobi.NewFileUploadValidator(f, h)
	fu.AllowedExtensions = []string{".jpg", ".png"}

	assert.NoError(t, fu.Validate())
}

func TestFileUpload_AllowedExtensions_NoMatch(t *testing.T) {
	f, h := newMultipartFile(t, "script.exe", []byte("data"))
	fu := shinobi.NewFileUploadValidator(f, h)
	fu.AllowedExtensions = []string{".jpg", ".png"}

	err := fu.Validate()
	assert.Error(t, err)

	var se *shinobi.StatusError
	assert.ErrorAs(t, err, &se)
	assert.Equal(t, http.StatusUnsupportedMediaType, se.Code)
}

func TestFileUpload_AllowedExtensions_CaseInsensitive(t *testing.T) {
	f, h := newMultipartFile(t, "photo.JPG", []byte("data"))
	fu := shinobi.NewFileUploadValidator(f, h)
	fu.AllowedExtensions = []string{".jpg"}

	assert.NoError(t, fu.Validate())
}

// --- Presets ---

func TestNewImageUpload_ValidateJPEG(t *testing.T) {
	content := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	f, h := newMultipartFile(t, "photo.jpg", content)

	err := shinobi.NewImageUpload(f, h, 5*shinobi.MB).Validate()

	assert.NoError(t, err)
}

func TestNewImageUpload_WrongType(t *testing.T) {
	f, h := newMultipartFile(t, "doc.pdf", []byte("%PDF-1.4"))

	err := shinobi.NewImageUpload(f, h, 5*shinobi.MB).Validate()

	assert.Error(t, err)
}

func TestNewDocumentUpload_ValidPDF(t *testing.T) {
	content := []byte("%PDF-1.4 fake pdf content")
	f, h := newMultipartFile(t, "report.pdf", content)

	err := shinobi.NewDocumentUpload(f, h, 10*shinobi.MB).Validate()

	assert.NoError(t, err)
}

func TestNewDocumentUpload_WrongExtension(t *testing.T) {
	content := []byte("%PDF-1.4 fake pdf content")
	f, h := newMultipartFile(t, "report.txt", content)

	err := shinobi.NewDocumentUpload(f, h, 10*shinobi.MB).Validate()

	assert.Error(t, err)
}

// --- AddType / AddExtension ---

func TestFileUpload_AddType_Chaining(t *testing.T) {
	f, h := newMultipartFile(t, "image.svg", []byte("<svg></svg>"))
	fu := shinobi.NewImageUpload(f, h, 5*shinobi.MB).
		AddType("image/svg+xml").
		AddExtension(".svg")
	fu.DetectFunc = func(_ []byte) string { return "image/svg+xml" }

	assert.NoError(t, fu.Validate())
}

// --- Ctx.Validate avec Validatable ---

func TestCtx_Validate_Validatable(t *testing.T) {
	app := shinobi.New()

	content := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	f, h := newMultipartFile(t, "photo.jpg", content)

	var got error
	app.Post("/upload", func(c shinobi.Ctx) error {
		got = c.Validate(shinobi.NewImageUpload(f, h, 5*shinobi.MB))
		return got
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/upload", nil)
	app.ServeHTTP(rec, req)

	assert.NoError(t, got)
}
