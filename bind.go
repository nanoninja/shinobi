// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shinobi

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Binder decodes an HTTP request body into a value.
type Binder interface {
	Bind(r *http.Request, v any) error
}

var timeType = reflect.TypeOf(time.Time{})

type queryBinder struct{}

// QueryBinder returns a Binder that decodes URL query parameters into a struct
// using `query:""` field tags. Supported types: string, int, int64, float64,
// bool, time.Time. Use `format:""` to specify a custom time layout.
// Pointer fields are treated as optional — nil when the parameter is absent.
func QueryBinder() Binder {
	return queryBinder{}
}

func (queryBinder) Bind(r *http.Request, v any) error {
	values := r.URL.Query()
	return bindStruct(v, "query", func(key string) (string, bool) {
		raw, ok := values[key]
		if !ok || len(raw) == 0 {
			return "", false
		}
		return raw[0], true
	})
}

type formBinder struct{}

// FormBinder returns a Binder that decodes form fields into a struct
// using `form:""` field tags. Supported types: string, int, int64, float64,
// bool, time.Time. Use `format:""` to specify a custom time layout.
// Pointer fields are treated as optional — nil when the field is absent.
func FormBinder() Binder {
	return formBinder{}
}

func (formBinder) Bind(r *http.Request, v any) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	return bindStruct(v, "form", func(key string) (string, bool) {
		raw := r.PostForm[key]
		if len(raw) == 0 {
			return "", false
		}
		return raw[0], true
	})
}

// bindStruct is the shared reflection loop used by QueryBinder and FormBinder.
// tagName selects which struct tag to read ("query" or "form").
// getValue resolves the raw string value for a given tag key.
func bindStruct(v any, tagName string, getValue func(key string) (string, bool)) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("bind requires a pointer to a struct, got %T", v)
	}
	rv = rv.Elem()
	rt := rv.Type()

	for i := range rt.NumField() {
		field := rt.Field(i)
		tag := field.Tag.Get(tagName)
		if tag == "" || tag == "-" {
			continue
		}
		raw, ok := getValue(tag)
		if !ok {
			continue
		}
		format := field.Tag.Get("format")
		fv := rv.Field(i)
		if fv.Kind() == reflect.Pointer {
			ptr := reflect.New(fv.Type().Elem())
			if err := setFieldFromString(ptr.Elem(), raw, format); err != nil {
				return fmt.Errorf("field %q: %w", field.Name, err)
			}
			fv.Set(ptr)
		} else {
			if err := setFieldFromString(fv, raw, format); err != nil {
				return fmt.Errorf("field %q: %w", field.Name, err)
			}
		}
	}
	return nil
}

type jsonBinder struct {
	disallowUnknownFields bool
	useNumber             bool
	maxBytes              int64
}

// JSONBinder returns a Binder that decodes JSON request bodies.
func JSONBinder(opts ...func(*jsonBinder)) Binder {
	b := &jsonBinder{}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// JSONNumber configures JSONBinder to decode numbers as json.Number instead of float64.
func JSONNumber() func(*jsonBinder) {
	return func(b *jsonBinder) { b.useNumber = true }
}

// JSONMaxBytes configures JSONBinder to limit the request body size to n bytes.
func JSONMaxBytes(n int64) func(*jsonBinder) {
	return func(b *jsonBinder) { b.maxBytes = n }
}

// JSONStrict configures JSONBinder to reject unknown fields.
func JSONStrict() func(*jsonBinder) {
	return func(b *jsonBinder) { b.disallowUnknownFields = true }
}

func (b *jsonBinder) Bind(r *http.Request, v any) error {
	body := io.Reader(r.Body)
	if b.maxBytes > 0 {
		body = io.LimitReader(r.Body, b.maxBytes)
	}
	d := json.NewDecoder(body)
	if b.disallowUnknownFields {
		d.DisallowUnknownFields()
	}
	if b.useNumber {
		d.UseNumber()
	}
	return d.Decode(v)
}

type xmlBinder struct{}

// XMLBinder returns a Binder that decodes XML request bodies.
func XMLBinder() Binder {
	return &xmlBinder{}
}

func (xmlBinder) Bind(r *http.Request, v any) error {
	return xml.NewDecoder(r.Body).Decode(v)
}

// BinderRegistry maps MIME type prefixes to their corresponding Binder.
// It implements Binder itself, selecting the appropriate decoder based on Content-Type.
type BinderRegistry map[string]Binder

// Bind selects the appropriate Binder based on the request Content-Type and decodes the body into v.
func (bg BinderRegistry) Bind(r *http.Request, v any) error {
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	for mime, b := range bg {
		if strings.Contains(ct, mime) {
			return b.Bind(r, v)
		}
	}
	return ErrUnsupportedContentType
}

// DefaultBinder is the default BinderRegistry used when no custom Binder is configured.
// It supports application/json, application/xml, and application/x-www-form-urlencoded out of the box.
var DefaultBinder = BinderRegistry{
	"application/json":                  JSONBinder(),
	"application/xml":                   XMLBinder(),
	"application/x-www-form-urlencoded": FormBinder(),
}

var defaultTimeFormats = []string{
	time.RFC3339,
	time.DateTime, // "2006-01-02 15:04:05"
	time.DateOnly, // "2006-01-02"
}

func setFieldFromString(v reflect.Value, s, format string) error {
	if v.Type() == timeType {
		formats := defaultTimeFormats
		if format != "" {
			formats = []string{format}
		}
		for _, f := range formats {
			if t, err := time.Parse(f, s); err == nil {
				v.Set(reflect.ValueOf(t))
				return nil
			}
		}
		return fmt.Errorf("cannot parse %q as time.Time", s)
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetInt(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(s, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetFloat(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		v.SetBool(b)
	}
	return nil
}
