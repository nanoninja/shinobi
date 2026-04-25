// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shinobi_test

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/shinobi"
)

type testQueryParams struct {
	Query     string  `query:"q"`
	Page      int     `query:"page"`
	Score     int64   `query:"score"`
	Rate      float64 `query:"rate"`
	Active    bool    `query:"active"`
	Label     *string `query:"label"`
	Limit     *int    `query:"limit"`
	Skip      string  `query:"-"`
	NoTag     string
	CreatedAt time.Time `query:"created_at"`
	BirthDate time.Time `query:"birth_date" format:"2006-01-02"`
}

func TestQueryBinder_StringField(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/?q=hello", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Equal(t, "hello", p.Query)
}

func TestQueryBinder_IntField(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/?page=3", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Equal(t, 3, p.Page)
}

func TestQueryBinder_Int64Field(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/?score=9", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Equal(t, int64(9), p.Score)
}

func TestQueryBinder_FloatField(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/?rate=3.14", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Equal(t, 3.14, p.Rate)
}

func TestQueryBinder_BoolField(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/?active=true", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.True(t, p.Active)
}

func TestQueryBinder_PointerPresent(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/?label=foo", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.NotNil(t, p.Label)
	assert.Equal(t, "foo", *p.Label)
}

func TestQueryBinder_PointerAbsent(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Nil(t, p.Label)
	assert.Nil(t, p.Limit)
}

func TestQueryBinder_SkipDashTag(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/?Skip=x", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Equal(t, "", p.Skip)
}

func TestQueryBinder_SkipNoTag(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/?NoTag=x", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Equal(t, "", p.NoTag)
}

func TestQueryBinder_AbsentField(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Equal(t, 0, p.Page)
}

func TestQueryBinder_InvalidInt(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/?page=abc", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.Error(t, err)
}

func TestQueryBinder_InvalidFloat(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/?rate=abc", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.Error(t, err)
}

func TestQueryBinder_InvalidBool(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/?active=notabool", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.Error(t, err)
}

func TestQueryBinder_InvalidPointerField(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/?limit=abc", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.Error(t, err)
	assert.Nil(t, p.Limit)
}

func TestQueryBinder_NotAPointer(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	err := shinobi.QueryBinder().Bind(req, p)

	assert.Error(t, err)
}

func TestCtx_BindQuery(t *testing.T) {
	app := shinobi.New()

	var got testQueryParams
	app.Get("/search", func(c shinobi.Ctx) error {
		return c.BindQuery(&got)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/search?q=shinobi&page=2&active=true", nil)

	app.ServeHTTP(rec, req)

	assert.Equal(t, "shinobi", got.Query)
	assert.Equal(t, 2, got.Page)
	assert.True(t, got.Active)
}

type testFormParams struct {
	Name      string  `form:"name"`
	Age       int     `form:"age"`
	Score     int64   `form:"score"`
	Rate      float64 `form:"rate"`
	Active    bool    `form:"active"`
	Label     *string `form:"label"`
	Limit     *int    `form:"limit"`
	Skip      string  `form:"-"`
	NoTag     string
	CreatedAt time.Time `form:"created_at"`
	BirthDate time.Time `form:"birth_date" format:"2006-01-02"`
}

func TestFormBinder_StringField(t *testing.T) {
	var p testFormParams
	body := strings.NewReader("name=alice")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := shinobi.FormBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Equal(t, "alice", p.Name)
}

func TestFormBinder_IntField(t *testing.T) {
	var p testFormParams
	body := strings.NewReader("age=30")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := shinobi.FormBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Equal(t, 30, p.Age)
}

func TestFormBinder_FloatField(t *testing.T) {
	var p testFormParams
	body := strings.NewReader("rate=3.14")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := shinobi.FormBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Equal(t, 3.14, p.Rate)
}

func TestFormBinder_BoolField(t *testing.T) {
	var p testFormParams
	body := strings.NewReader("active=true")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := shinobi.FormBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.True(t, p.Active)
}

func TestFormBinder_PointerPresent(t *testing.T) {
	var p testFormParams
	body := strings.NewReader("label=foo")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := shinobi.FormBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.NotNil(t, p.Label)
	assert.Equal(t, "foo", *p.Label)
}

func TestFormBinder_PointerAbsent(t *testing.T) {
	var p testFormParams
	body := strings.NewReader("")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := shinobi.FormBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Nil(t, p.Label)
	assert.Nil(t, p.Limit)
}

func TestFormBinder_SkipDashTag(t *testing.T) {
	var p testFormParams
	body := strings.NewReader("Skip=x")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := shinobi.FormBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Equal(t, "", p.Skip)
}

func TestFormBinder_SkipNoTag(t *testing.T) {
	var p testFormParams
	body := strings.NewReader("NoTag=x")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := shinobi.FormBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Equal(t, "", p.NoTag)
}

func TestJSONBinder(t *testing.T) {
	var got struct{ Name string }

	body := strings.NewReader(`{"name":"alice"}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/json")

	err := shinobi.JSONBinder().Bind(req, &got)

	assert.NoError(t, err)
	assert.Equal(t, "alice", got.Name)
}

func TestJSONBinder_Strict(t *testing.T) {
	var got struct{ Name string }

	body := strings.NewReader(`{"name":"bob","unknown":"field"}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)

	err := shinobi.JSONBinder(shinobi.JSONStrict()).Bind(req, &got)

	assert.Error(t, err)
}

func TestJSONBinder_Number(t *testing.T) {
	var got struct{ Value json.Number }

	body := strings.NewReader(`{"value":42}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)

	err := shinobi.JSONBinder(shinobi.JSONNumber()).Bind(req, &got)

	assert.NoError(t, err)
	assert.Equal(t, json.Number("42"), got.Value)
}

func TestJSONBinder_MaxBytes(t *testing.T) {
	var got struct{ Name string }

	body := strings.NewReader(`{"name":"gopher"}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)

	err := shinobi.JSONBinder(shinobi.JSONMaxBytes(3)).Bind(req, &got)

	assert.Error(t, err)
}

func TestXMLBinder(t *testing.T) {
	var got struct {
		XMLName xml.Name `xml:"User"`
		Name    string   `xml:"name"`
	}

	body := strings.NewReader(`<User><name>alice</name></User>`)
	req := httptest.NewRequest(http.MethodPost, "/", body)

	err := shinobi.XMLBinder().Bind(req, &got)

	assert.NoError(t, err)
	assert.Equal(t, "alice", got.Name)
}

func TestBinderRegistry_JSON(t *testing.T) {
	var got struct{ Name string }

	body := strings.NewReader(`{"name":"bob"}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/json")

	err := shinobi.DefaultBinder.Bind(req, &got)

	assert.NoError(t, err)
	assert.Equal(t, "bob", got.Name)
}

func TestBinderRegistry(t *testing.T) {
	var got struct {
		XMLName xml.Name `xml:"User"`
		Name    string   `xml:"name"`
	}

	body := strings.NewReader(`<User><name>gopher</name></User>`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/xml")

	err := shinobi.DefaultBinder.Bind(req, &got)

	assert.NoError(t, err)
	assert.Equal(t, "gopher", got.Name)
}

func TestBinderRegistry_Form(t *testing.T) {
	var got testFormParams

	body := strings.NewReader("name=gopher&age=7")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := shinobi.DefaultBinder.Bind(req, &got)

	assert.NoError(t, err)
	assert.Equal(t, "gopher", got.Name)
	assert.Equal(t, 7, got.Age)
}

func TestBinderRegistry_UnsupportedContentType(t *testing.T) {
	var got testFormParams

	body := strings.NewReader("name=alice")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "text/plain")

	err := shinobi.DefaultBinder.Bind(req, &got)

	assert.Error(t, err)
}

func TestFormBinder_InvalidInt(t *testing.T) {
	var p testFormParams
	body := strings.NewReader("age=abc")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := shinobi.FormBinder().Bind(req, &p)

	assert.Error(t, err)
}

func TestFormBinder_InvalidFloat(t *testing.T) {
	var p testFormParams
	body := strings.NewReader("rate=abc")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := shinobi.FormBinder().Bind(req, &p)

	assert.Error(t, err)
}

func TestFormBinder_InvalidBool(t *testing.T) {
	var p testFormParams
	body := strings.NewReader("active=notabool")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := shinobi.FormBinder().Bind(req, &p)

	assert.Error(t, err)
}

func TestFormBinder_InvalidPointerField(t *testing.T) {
	var p testFormParams
	body := strings.NewReader("limit=abc")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := shinobi.FormBinder().Bind(req, &p)

	assert.Error(t, err)
	assert.Nil(t, p.Limit)
}

func TestFormBinder_NotAPointer(t *testing.T) {
	var p testFormParams
	body := strings.NewReader("")
	req := httptest.NewRequest(http.MethodPost, "/", body)

	err := shinobi.FormBinder().Bind(req, p)

	assert.Error(t, err)
}

func TestFormBinder_ParseFormError(t *testing.T) {
	var p testFormParams
	req := httptest.NewRequest(http.MethodPost, "/", io.NopCloser(errReader{}))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := shinobi.FormBinder().Bind(req, &p)

	assert.Error(t, err)
}

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, errors.New("read error")
}

func TestCtx_BindForm(t *testing.T) {
	app := shinobi.New()

	var got testFormParams
	app.Post("/submit", func(c shinobi.Ctx) error {
		return c.BindForm(&got)
	})

	body := strings.NewReader("name=shinobi&age=3&active=true")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/submit", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	app.ServeHTTP(rec, req)

	assert.Equal(t, "shinobi", got.Name)
	assert.Equal(t, 3, got.Age)
	assert.True(t, got.Active)
}

func TestQueryBinder_TimeRFC3339(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/?created_at=2026-04-20T10:00:00Z", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Equal(t, 2026, p.CreatedAt.Year())
	assert.Equal(t, time.April, p.CreatedAt.Month())
	assert.Equal(t, 20, p.CreatedAt.Day())
}

func TestQueryBinder_TimeDateOnly(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/?created_at=2026-04-20", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Equal(t, 2026, p.CreatedAt.Year())
	assert.Equal(t, 20, p.CreatedAt.Day())
}

func TestQueryBinder_TimeCustomFormat(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/?birth_date=1990-06-15", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Equal(t, 1990, p.BirthDate.Year())
	assert.Equal(t, time.June, p.BirthDate.Month())
	assert.Equal(t, 15, p.BirthDate.Day())
}

func TestQueryBinder_TimeInvalidValue(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/?created_at=not-a-date", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.Error(t, err)
}

func TestQueryBinder_TimeInvalidCustomFormat(t *testing.T) {
	var p testQueryParams
	req := httptest.NewRequest(http.MethodGet, "/?birth_date=20-06-1990", nil)

	err := shinobi.QueryBinder().Bind(req, &p)

	assert.Error(t, err)
}

func TestFormBinder_TimeRFC3339(t *testing.T) {
	var p testFormParams
	body := strings.NewReader("created_at=2026-04-20T10:00:00Z")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := shinobi.FormBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Equal(t, 2026, p.CreatedAt.Year())
	assert.Equal(t, time.April, p.CreatedAt.Month())
	assert.Equal(t, 20, p.CreatedAt.Day())
}

func TestFormBinder_TimeCustomFormat(t *testing.T) {
	var p testFormParams
	body := strings.NewReader("birth_date=1990-06-15")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := shinobi.FormBinder().Bind(req, &p)

	assert.NoError(t, err)
	assert.Equal(t, 1990, p.BirthDate.Year())
	assert.Equal(t, time.June, p.BirthDate.Month())
	assert.Equal(t, 15, p.BirthDate.Day())
}

func TestFormBinder_TimeInvalidValue(t *testing.T) {
	var p testFormParams
	body := strings.NewReader("created_at=not-a-date")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	err := shinobi.FormBinder().Bind(req, &p)

	assert.Error(t, err)
}

func BenchmarkQueryBinder(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/?q=hello&page=2&score=99&rate=3.14&active=true", nil)
	binder := shinobi.QueryBinder()

	b.ResetTimer()
	for b.Loop() {
		var p testQueryParams
		_ = binder.Bind(req, &p)
	}
}

func BenchmarkFormBinder(b *testing.B) {
	binder := shinobi.FormBinder()

	b.ResetTimer()
	for b.Loop() {
		body := strings.NewReader("name=shinobi&age=3&score=99&rate=3.14&active=true")
		req := httptest.NewRequest(http.MethodPost, "/", body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		var p testFormParams
		_ = binder.Bind(req, &p)
	}
}

func BenchmarkQueryBinder_WithTime(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/?q=hello&page=2&created_at=2026-04-20T10:00:00Z", nil)
	binder := shinobi.QueryBinder()

	b.ResetTimer()
	for b.Loop() {
		var p testQueryParams
		_ = binder.Bind(req, &p)
	}
}
