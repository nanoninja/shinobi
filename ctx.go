// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shinobi

import (
	"context"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/nanoninja/render"
)

// Ctx represents the context of an HTTP request/response cycle.
// It wraps http.Request and http.ResponseWriter and provides a unified
// API for handlers and middleware.
type Ctx interface {
	// Request returns the underlying HTTP request.
	Request() *http.Request

	// Response returns the wrapped HTTP response writer.
	Response() *Response

	// Context returns the request's stdlib context.
	Context() context.Context

	// WithContext returns a shallow copy of Ctx with the request context replaced.
	WithContext(ctx context.Context) Ctx

	// Set stores a value in the request context chain.
	Set(key, value any)

	// Get retrieves a value from the request context chain.
	Get(key any) (any, bool)

	// Logger returns the logger associated with this request context.
	Logger() *slog.Logger

	// Debug reports whether the application is running in debug mode.
	Debug() bool

	// IsLoopback reports whether addr is a loopback address.
	IsLoopback(addr string) bool

	// IsSecure reports whether the request was made over HTTPS.
	IsSecure() bool

	// IsXHR reports whether the request was made via XMLHttpRequest.
	IsXHR() bool

	// IsWebSocket reports whether the request is a WebSocket upgrade.
	IsWebSocket() bool

	// IsMethod reports whether the request method matches m (case-insensitive).
	IsMethod(s string) bool

	// Method returns the HTTP method of the request (e.g. "GET", "POST").
	Method() string

	// Path returns the URL path of the request (e.g. "/users/42").
	Path() string

	// Host returns the host from the request (e.g. "example.com").
	Host() string

	// Scheme returns "https" if the request is secure, "http" otherwise.
	Scheme() string

	// Query returns the first value for the named query parameter.
	Query(key string) string

	// QueryValues returns all query parameters as url.Values.
	QueryValues() url.Values

	// FormValue returns the first value for the named form field.
	FormValue(key string) string

	// ParseForm populates Request.Form and Request.PostForm.
	ParseForm() error

	// Param returns the path parameter value for the given key.
	Param(key string) string

	// ContentType returns the Content-Type header of the request.
	ContentType() string

	// UserAgent returns the User-Agent header of the request.
	UserAgent() string

	// Referer returns the Referer header of the request.
	Referer() string

	// FormFile returns the uploaded file for the named form key.
	FormFile(key string) (multipart.File, *multipart.FileHeader, error)

	// Bind decodes the request body into v based on the Content-Type header.
	Bind(v any) error

	// BindQuery decodes URL query parameters into v using struct field tags.
	// Fields are mapped via the `query:""` tag. Supported types: string, int,
	// int64, float64, bool, time.Time. Use `format:""` to specify a custom time
	// layout; otherwise RFC3339, DateTime and DateOnly are tried in order.
	// Pointer fields are optional — nil when the parameter is absent.
	// Fields without a tag or with tag `query:"-"` are ignored.
	//
	// Example:
	//
	//	type SearchParams struct {
	//	    Query string    `query:"q"`
	//	    Page  int       `query:"page"`
	//	    Since time.Time `query:"since"`
	//	}
	//	var p SearchParams
	//	if err := c.BindQuery(&p); err != nil {
	//	    return err
	//	}
	BindQuery(v any) error

	// BindForm decodes form fields from the request body into v using `form:""` struct tags.
	// Supported types: string, int, int64, float64, bool, time.Time. Use `format:""`
	// to specify a custom time layout; otherwise RFC3339, DateTime and DateOnly are tried in order.
	// Pointer fields are optional — nil when the field is absent.
	// Fields without a tag or with tag `form:"-"` are ignored.
	BindForm(v any) error

	// Validate validates v using the configured Validator.
	Validate(v any) error

	// SetCookie adds a Set-Cookie header to the response.
	SetCookie(cookie *http.Cookie)

	// DeleteCookie expires a cookie by name and path.
	DeleteCookie(name, path string)

	// SetHeader sets a response header, replacing any existing value.
	SetHeader(key, value string)

	// AddHeader adds a response header value without replacing existing ones.
	AddHeader(key, value string)

	// Redirect sends an HTTP redirect. Returns an error if code is not 3xx.
	Redirect(url string, code int) error

	// NoContent sends a 204 No Content response.
	NoContent() error

	// Render writes data to the response using the given renderer.
	Render(code int, r render.Renderer, data any, opts render.Options) error

	// String sends a plain text response, formatted if args are provided.
	String(code int, s string, args ...any) error

	// JSON sends a JSON-encoded response.
	JSON(code int, data any, opts render.Options) error

	// XML sends an XML-encoded response.
	XML(code int, data any, opts render.Options) error

	// HTML renders a named template with the pre-loaded template renderer.
	HTML(code int, name string, data any, opts render.Options) error

	// CSV renders a 2D string slice as a CSV response.
	CSV(code int, data [][]string, opts render.Options) error

	// Blob sends raw binary data with the given content type via the render package.
	Blob(code int, data []byte, opts render.Options) error

	// File serves a file from the given path using http.ServeContent.
	File(path string) error

	// Attachment serves a file as a downloadable attachment with Content-Disposition.
	Attachment(path string) error

	// Inline serves a file inline in the browser with Content-Disposition.
	Inline(path string) error
}

type ctx struct {
	response  Response
	request   *http.Request
	tmpl      render.Renderer
	binder    Binder
	validator Validator
	logger    *slog.Logger
	debug     bool
	query     url.Values
}

// NewCtx creates a new Ctx wrapping the given ResponseWriter, Request and Config.
func NewCtx(w http.ResponseWriter, req *http.Request, cfg *Config) Ctx {
	c := &ctx{
		request:   req,
		tmpl:      cfg.tmpl,
		binder:    cfg.binder,
		validator: cfg.validator,
		logger:    cfg.logger,
		debug:     cfg.debug,
	}
	c.response = Response{ResponseWriter: w, status: http.StatusOK}
	return c
}

func (c *ctx) Logger() *slog.Logger {
	return c.logger
}

func (c *ctx) Debug() bool {
	return c.debug
}

func (c *ctx) Request() *http.Request {
	return c.request
}

func (c *ctx) Response() *Response {
	return &c.response
}

func (c *ctx) Context() context.Context {
	return c.request.Context()
}

func (c *ctx) WithContext(newCtx context.Context) Ctx {
	return &ctx{
		request:   c.request.WithContext(newCtx),
		response:  c.response,
		tmpl:      c.tmpl,
		binder:    c.binder,
		validator: c.validator,
		logger:    c.logger,
		debug:     c.debug,
	}
}

func (c *ctx) Set(key, value any) {
	c.request = c.request.WithContext(
		context.WithValue(c.request.Context(), key, value),
	)
}

func (c *ctx) Get(key any) (any, bool) {
	v := c.request.Context().Value(key)
	return v, v != nil
}

func (*ctx) IsLoopback(addr string) bool {
	ip, err := netip.ParseAddr(addr)
	if err != nil {
		return false
	}
	return ip.IsLoopback()
}

func (c *ctx) IsSecure() bool {
	return c.Scheme() == "https"
}

func (c *ctx) IsXHR() bool {
	return c.request.Header.Get("X-Requested-With") == "XMLHttpRequest"
}

func (c *ctx) IsWebSocket() bool {
	return strings.EqualFold(c.request.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(c.request.Header.Get("Connection")), "upgrade")
}

func (c *ctx) IsMethod(m string) bool {
	return strings.EqualFold(m, c.request.Method)
}

func (c *ctx) Method() string {
	return c.request.Method
}

func (c *ctx) Path() string {
	return c.request.URL.Path
}

func (c *ctx) Host() string {
	return c.request.Host
}

func (c *ctx) Scheme() string {
	if c.request.TLS != nil ||
		c.request.Header.Get("X-Forwarded-Proto") == "https" {
		return "https"
	}
	return "http"
}

func (c *ctx) Query(key string) string {
	return c.QueryValues().Get(key)
}

func (c *ctx) QueryValues() url.Values {
	if c.query == nil {
		c.query = c.request.URL.Query()
	}
	return c.query
}

func (c *ctx) FormValue(key string) string {
	return c.request.FormValue(key)
}

func (c *ctx) ParseForm() error {
	return c.request.ParseForm()
}

func (c *ctx) Param(key string) string {
	return c.request.PathValue(key)
}

func (c *ctx) ContentType() string {
	return c.request.Header.Get("Content-Type")
}

func (c *ctx) UserAgent() string {
	return c.request.UserAgent()
}

func (c *ctx) Referer() string {
	return c.request.Referer()
}

func (c *ctx) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return c.request.FormFile(key)
}

func (c *ctx) Bind(v any) error {
	return c.binder.Bind(c.request, v)
}

func (c *ctx) BindQuery(v any) error {
	return QueryBinder().Bind(c.request, v)
}

func (c *ctx) BindForm(v any) error {
	return FormBinder().Bind(c.request, v)
}

func (c *ctx) Validate(v any) error {
	if sv, ok := v.(Validatable); ok {
		return sv.Validate()
	}
	if c.validator == nil {
		return ErrValidatorNotSet
	}
	return c.validator.Validate(v)
}

func (c *ctx) SetCookie(cookie *http.Cookie) {
	http.SetCookie(&c.response, cookie)
}

func (c *ctx) DeleteCookie(name, path string) {
	http.SetCookie(&c.response, &http.Cookie{
		Name:   name,
		Value:  "",
		Path:   path,
		MaxAge: -1,
	})
}

func (c *ctx) SetHeader(key, value string) {
	c.response.Header().Set(key, value)
}

func (c *ctx) AddHeader(key, value string) {
	c.response.Header().Add(key, value)
}

func (c *ctx) Redirect(url string, code int) error {
	if code < 300 || code >= 400 {
		return ErrInvalidRedirectStatusCode
	}
	http.Redirect(&c.response, c.request, url, code)
	return nil
}

func (c *ctx) NoContent() error {
	c.response.WriteHeader(http.StatusNoContent)
	return nil
}

var (
	defaultJSONRenderer = render.JSON()
	defaultXMLRenderer  = render.XML()
	defaultTextRenderer = render.Text()
	defaultCSVRenderer  = render.CSV()
	defaultBlobRenderer = render.Binary()
)

func (c *ctx) Render(code int, r render.Renderer, data any, opts render.Options) error {
	if r.ContentType() != "" {
		c.SetHeader("Content-Type", r.ContentType())
	}
	c.Response().WriteHeader(code)
	return r.Render(c.Context(), &c.response, data, opts)
}

func (c *ctx) String(code int, s string, args ...any) error {
	if len(args) == 0 {
		return c.Render(code, render.Text(), s, render.NoOptions)
	}
	return c.Render(code, defaultTextRenderer, s, render.Options{Args: args})
}

func (c *ctx) JSON(code int, data any, opts render.Options) error {
	return c.Render(code, defaultJSONRenderer, data, opts)
}

func (c *ctx) XML(code int, data any, opts render.Options) error {
	return c.Render(code, defaultXMLRenderer, data, opts)
}

func (c *ctx) HTML(code int, name string, data any, opts render.Options) error {
	if c.tmpl == nil {
		return ErrTemplateRendererNotSet
	}
	opts.Name = name
	return c.Render(code, c.tmpl, data, opts)
}

func (c *ctx) CSV(code int, data [][]string, opts render.Options) error {
	return c.Render(code, defaultCSVRenderer, data, opts)
}

func (c *ctx) Blob(code int, data []byte, opts render.Options) error {
	return c.Render(code, defaultBlobRenderer, data, opts)
}

func (c *ctx) File(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		return err
	}
	http.ServeContent(&c.response, c.request, fi.Name(), fi.ModTime(), f)
	return nil
}

func (c *ctx) Attachment(path string) error {
	c.response.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(path)),
	)
	return c.File(path)
}

func (c *ctx) Inline(path string) error {
	c.response.Header().Set("Content-Disposition",
		fmt.Sprintf(`inline; filename="%s"`, filepath.Base(path)),
	)
	return c.File(path)
}
