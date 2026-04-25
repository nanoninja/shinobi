// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shinobi_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/render"
	"github.com/nanoninja/shinobi"
)

func newTestCtx(
	t testing.TB,
	method,
	target string,
) (shinobi.Ctx, *httptest.ResponseRecorder) {
	t.Helper()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, target, nil)

	return shinobi.NewCtx(w, r, shinobi.DefaultConfig()), w
}

func TestNewCtx(t *testing.T) {
	c, w := newTestCtx(t, http.MethodGet, "/")

	assert.NotNil(t, c.Request())
	assert.NotNil(t, c.Response())
	assert.Equal[http.ResponseWriter](t, w, c.Response().Unwrap())
}

func TestCtx_Request_Response_Context(t *testing.T) {
	c, w := newTestCtx(t, http.MethodGet, "/")

	assert.Equal(t, http.MethodGet, c.Request().Method)
	assert.Equal[http.ResponseWriter](t, w, c.Response().Unwrap())
	assert.NotNil(t, c.Context())
	assert.Equal(t, c.Request().Context(), c.Context())
}

func TestCtx_WithContext(t *testing.T) {
	type key struct{}
	c, _ := newTestCtx(t, http.MethodGet, "/")
	newCtx := context.WithValue(context.Background(), key{}, "new")

	c2 := c.WithContext(newCtx)

	assert.Equal(t, newCtx, c2.Context())
	assert.NotEqual(t, c.Context(), c2.Context())
}

func TestCtx_Set_Get(t *testing.T) {
	t.Run("returns stored value", func(t *testing.T) {
		type key struct{}
		c, _ := newTestCtx(t, http.MethodGet, "/")

		c.Set(key{}, "value")
		got, ok := c.Get(key{})

		assert.Equal(t, true, ok)
		assert.Equal(t, "value", got)
	})

	t.Run("returns false for missing key", func(t *testing.T) {
		type key struct{}
		c, _ := newTestCtx(t, http.MethodGet, "/")

		_, ok := c.Get(key{})

		assert.Equal(t, false, ok)
	})
}

func TestCtx_IsLoopback(t *testing.T) {
	c, _ := newTestCtx(t, http.MethodGet, "/")

	assert.Equal(t, true, c.IsLoopback("127.0.0.1"))
	assert.Equal(t, true, c.IsLoopback("::1"))
	assert.Equal(t, false, c.IsLoopback("8.8.8.8"))
	assert.Equal(t, false, c.IsLoopback("no-an-ip"))
}

func TestCtx_IsMethod(t *testing.T) {
	c, _ := newTestCtx(t, http.MethodPost, "/")

	assert.Equal(t, true, c.IsMethod("POST"))
	assert.Equal(t, true, c.IsMethod("post"))
	assert.Equal(t, false, c.IsMethod("GET"))
}

func TestCtx_IsSecure(t *testing.T) {
	t.Run("false by default", func(t *testing.T) {
		c, _ := newTestCtx(t, http.MethodGet, "/")

		assert.Equal(t, false, c.IsSecure())
	})

	t.Run("true with X-Forwarded-Proto header", func(t *testing.T) {
		c, _ := newTestCtx(t, http.MethodGet, "/")
		c.Request().Header.Set("X-Forwarded-Proto", "https")

		assert.Equal(t, true, c.IsSecure())
	})
}

func TestCtx_XHR(t *testing.T) {
	t.Run("false by default", func(t *testing.T) {
		c, _ := newTestCtx(t, http.MethodGet, "/")

		assert.Equal(t, false, c.IsXHR())
	})

	t.Run("true with X-Requested-With header", func(t *testing.T) {
		c, _ := newTestCtx(t, http.MethodGet, "/")
		c.Request().Header.Set("X-Requested-With", "XMLHttpRequest")

		assert.Equal(t, true, c.IsXHR())
	})
}

func TestCtx_IsWebsocket(t *testing.T) {
	t.Run("false by default", func(t *testing.T) {
		c, _ := newTestCtx(t, http.MethodGet, "/")

		assert.Equal(t, false, c.IsWebSocket())
	})

	t.Run("true with Upgrade and Connection headers", func(t *testing.T) {
		c, _ := newTestCtx(t, http.MethodGet, "/")
		c.Request().Header.Set("Upgrade", "websocket")
		c.Request().Header.Set("Connection", "Upgrade")

		assert.Equal(t, true, c.IsWebSocket())
	})
}

func TestCtx_Method(t *testing.T) {
	c, _ := newTestCtx(t, http.MethodPost, "/")

	assert.Equal(t, http.MethodPost, c.Method())
}

func TestCtx_Path(t *testing.T) {
	c, _ := newTestCtx(t, http.MethodGet, "/users/42")

	assert.Equal(t, "/users/42", c.Path())
}

func TestCtx_Host(t *testing.T) {
	c, _ := newTestCtx(t, http.MethodGet, "/")
	c.Request().Host = "example.com"

	assert.Equal(t, "example.com", c.Host())
}

func TestCtx_Scheme(t *testing.T) {
	t.Run("http by default", func(t *testing.T) {
		c, _ := newTestCtx(t, http.MethodGet, "/")

		assert.Equal(t, "http", c.Scheme())
	})

	t.Run("https with X-Forwarded-Proto header", func(t *testing.T) {
		c, _ := newTestCtx(t, http.MethodGet, "/")
		c.Request().Header.Set("X-Forwarded-Proto", "https")

		assert.Equal(t, "https", c.Scheme())
	})

	t.Run("https with TLS", func(t *testing.T) {
		c, _ := newTestCtx(t, http.MethodGet, "/")
		c.Request().TLS = &tls.ConnectionState{}

		assert.Equal(t, "https", c.Scheme())
	})
}

func TestCtx_Query(t *testing.T) {
	c, _ := newTestCtx(t, http.MethodGet, "/?name=shinobi&env=prod&env=dev")

	assert.Equal(t, "shinobi", c.Query("name"))
	assert.Equal(t, "prod", c.Query("env"))
	assert.Equal(t, "", c.Query("missing"))
	assert.Equal(t, []string{"prod", "dev"}, c.QueryValues()["env"])
}

func TestCtx_FormValue(t *testing.T) {
	body := strings.NewReader("name=shinobi")
	r := httptest.NewRequest(http.MethodPost, "/", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c := shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig())

	assert.Equal(t, "shinobi", c.FormValue("name"))
	assert.Equal(t, "", c.FormValue("missing"))
}

func TestCtx_ParseForm(t *testing.T) {
	body := strings.NewReader("name=shinobi")
	r := httptest.NewRequest(http.MethodPost, "/", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c := shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig())

	err := c.ParseForm()

	assert.NoError(t, err)
	assert.Equal(t, "shinobi", c.Request().Form.Get("name"))
}

func TestCtx_Param(t *testing.T) {
	app := shinobi.New()

	var got string
	app.Get("/users/{id}", func(c shinobi.Ctx) error {
		got = c.Param("id")
		return nil
	})

	app.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/users/42", nil),
	)

	assert.Equal(t, "42", got)
}

func TestCtx_ContentType(t *testing.T) {
	c, _ := newTestCtx(t, http.MethodPost, "/")
	c.Request().Header.Set("Content-Type", "application/json")

	assert.Equal(t, "application/json", c.ContentType())
}

func TestCtx_UserAgent(t *testing.T) {
	c, _ := newTestCtx(t, http.MethodGet, "/")
	c.Request().Header.Set("User-Agent", "shinobi-test/1.0")

	assert.Equal(t, "shinobi-test/1.0", c.UserAgent())
}

func TestCtx_Referer(t *testing.T) {
	c, _ := newTestCtx(t, http.MethodGet, "/")
	c.Request().Header.Set("Referer", "https://example.com")

	assert.Equal(t, "https://example.com", c.Referer())
}

func TestCtx_FormFile(t *testing.T) {
	t.Run("returns uploaded file", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("upload", "hello.txt")
		assert.NoError(t, err)

		_, err = part.Write([]byte("file content"))
		assert.NoError(t, err)
		assert.NoError(t, writer.Close())

		r := httptest.NewRequest(http.MethodPost, "/", body)
		r.Header.Set("Content-Type", writer.FormDataContentType())
		c := shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig())

		f, header, err := c.FormFile("upload")
		assert.NoError(t, err)
		assert.Equal(t, "hello.txt", header.Filename)

		content, err := io.ReadAll(f)
		assert.NoError(t, err)
		assert.Equal(t, "file content", string(content))
	})

	t.Run("returns error for missing key", func(t *testing.T) {
		c, _ := newTestCtx(t, http.MethodPost, "/")
		_, _, err := c.FormFile("missing")

		assert.Error(t, err)
	})
}

func TestCtx_Bind(t *testing.T) {
	app := shinobi.New()
	app.Post("/bind", func(c shinobi.Ctx) error {
		var got struct{ Name string }
		if err := c.Bind(&got); err != nil {
			return err
		}
		return c.String(http.StatusOK, got.Name)
	})

	body := strings.NewReader(`{"name":"alice"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bind", body)
	req.Header.Set("Content-Type", "application/json")

	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "alice", rec.Body.String())
}

type mockValidator struct{}

func (mockValidator) Validate(any) error { return nil }

func TestCtx_Validate(t *testing.T) {
	app := shinobi.New(
		shinobi.WithValidator(&mockValidator{}),
	)
	app.Post("/validate", func(c shinobi.Ctx) error {
		return c.Validate(&struct{}{})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/validate", nil)
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCtx_Validate_NotSet(t *testing.T) {
	c, _ := newTestCtx(t, http.MethodPost, "/")
	err := c.Validate(&struct{}{})

	assert.ErrorIs(t, err, shinobi.ErrValidatorNotSet)
}

func TestCtx_SetCookie(t *testing.T) {
	c, w := newTestCtx(t, http.MethodGet, "/")
	c.SetCookie(&http.Cookie{Name: "session", Value: "abc123"})

	assert.Equal(t, "session=abc123", w.Header().Get("Set-Cookie"))
}

func TestCtx_DeleteCookie(t *testing.T) {
	c, w := newTestCtx(t, http.MethodGet, "/")
	c.DeleteCookie("session", "/")

	cookie := w.Header().Get("Set-Cookie")

	assert.StringContains(t, cookie, "session=")
	assert.StringContains(t, cookie, "Max-Age=0")
}

func TestCtx_SetHeader(t *testing.T) {
	c, w := newTestCtx(t, http.MethodGet, "/")
	c.SetHeader("X-Custom", "first")
	c.SetHeader("X-Custom", "second")

	assert.Equal(t, "second", w.Header().Get("X-Custom"))
}

func TestCtx_AddHeader(t *testing.T) {
	c, w := newTestCtx(t, http.MethodGet, "/")
	c.AddHeader("X-Custom", "first")
	c.AddHeader("X-Custom", "second")

	assert.Equal(t, []string{"first", "second"}, w.Header().Values("X-Custom"))
}

func TestCtx_Redirect(t *testing.T) {
	t.Run("valid 3xx code", func(t *testing.T) {
		c, w := newTestCtx(t, http.MethodGet, "/")
		err := c.Redirect("/new", http.StatusMovedPermanently)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusMovedPermanently, w.Code)
		assert.Equal(t, "/new", w.Header().Get("Location"))
	})

	t.Run("invalid code returns error", func(t *testing.T) {
		c, _ := newTestCtx(t, http.MethodGet, "/")
		err := c.Redirect("/new", http.StatusOK)

		assert.ErrorIs(t, err, shinobi.ErrInvalidRedirectStatusCode)
	})
}

func TestCtx_NoContent(t *testing.T) {
	c, w := newTestCtx(t, http.MethodGet, "/")
	err := c.NoContent()

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCtx_String(t *testing.T) {
	t.Run("with format args", func(t *testing.T) {
		c, w := newTestCtx(t, http.MethodGet, "/")
		err := c.String(http.StatusOK, "hello %s", "world")

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "hello world", w.Body.String())
	})

	t.Run("without format args", func(t *testing.T) {
		c, w := newTestCtx(t, http.MethodGet, "/")
		err := c.String(http.StatusOK, "hello")

		assert.NoError(t, err)
		assert.Equal(t, "hello", w.Body.String())
	})
}

func TestCtx_JSON(t *testing.T) {
	c, w := newTestCtx(t, http.MethodGet, "/")
	err := c.JSON(http.StatusOK, map[string]string{"key": "value"}, render.NoOptions)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.StringContains(t, w.Body.String(), `"key":"value`)
}

func TestCtx_XML(t *testing.T) {
	type item struct {
		Name string `xml:"name"`
	}

	c, w := newTestCtx(t, http.MethodGet, "/")
	err := c.XML(http.StatusOK, item{Name: "shinobi"}, render.NoOptions)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.StringContains(t, w.Header().Get("Content-Type"), "application/xml")
	assert.StringContains(t, w.Body.String(), "<name>shinobi</name>")
}

type mockRenderer struct{}

func (m *mockRenderer) ContentType() string {
	return ""
}

func (m *mockRenderer) Render(_ context.Context, w io.Writer, _ any, _ render.Options) error {
	_, err := io.WriteString(w, "rendered")
	return err
}

func TestCtx_HTML(t *testing.T) {
	t.Run("returns error when renderer not set", func(t *testing.T) {
		c, _ := newTestCtx(t, http.MethodGet, "/")
		err := c.HTML(http.StatusOK, "index.html", nil, render.NoOptions)

		assert.ErrorIs(t, err, shinobi.ErrTemplateRendererNotSet)
	})

	t.Run("renders with renderer", func(t *testing.T) {
		app := shinobi.New(
			shinobi.WithRenderer(&mockRenderer{}),
		)
		app.Get("/html", func(c shinobi.Ctx) error {
			return c.HTML(http.StatusOK, "index.html", nil, render.NoOptions)
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/html", nil)
		app.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "rendered", rec.Body.String())
	})
}

func TestCtx_CSV(t *testing.T) {
	c, w := newTestCtx(t, http.MethodGet, "/")
	err := c.CSV(http.StatusOK, [][]string{
		{"name", "age"},
		{"alice", "30"},
	}, render.NoOptions)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.StringContains(t, w.Header().Get("Content-Type"), "text/csv")
	assert.StringContains(t, w.Body.String(), "name,age")
	assert.StringContains(t, w.Body.String(), "alice,30")
}

func TestCtx_Blob(t *testing.T) {
	c, w := newTestCtx(t, http.MethodGet, "/")
	err := c.Blob(http.StatusOK, []byte("binary data"), render.NoOptions)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "binary data", w.Body.String())
}

func TestCtx_File(t *testing.T) {
	t.Run("serves existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "hello.txt")
		err := os.WriteFile(path, []byte("file content"), 0o644)
		assert.NoError(t, err)

		c, w := newTestCtx(t, http.MethodGet, "/")
		err = c.File(path)

		assert.NoError(t, err)
		assert.Equal(t, "file content", w.Body.String())
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		c, _ := newTestCtx(t, http.MethodGet, "/")
		err := c.File("/nonexistent/file.txt")

		assert.Error(t, err)
	})
}

func TestCtx_Attachment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.txt")
	err := os.WriteFile(path, []byte("data"), 0o644)
	assert.NoError(t, err)

	c, w := newTestCtx(t, http.MethodGet, "/")
	err = c.Attachment(path)

	assert.NoError(t, err)
	assert.StringContains(t, w.Header().Get("Content-Disposition"), `attachment`)
	assert.StringContains(t, w.Header().Get("Content-Disposition"), "report.txt")
}

func TestCtx_Inline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "image.png")
	err := os.WriteFile(path, []byte("imgdata"), 0o644)
	assert.NoError(t, err)

	c, w := newTestCtx(t, http.MethodGet, "/")
	err = c.Inline(path)

	assert.NoError(t, err)
	assert.StringContains(t, w.Header().Get("Content-Disposition"), `inline`)
	assert.StringContains(t, w.Header().Get("Content-Disposition"), `image.png`)
}
