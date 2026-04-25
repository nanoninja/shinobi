# Shinobi

**Shinobi** is a lightweight HTTP micro-framework for **Go 1.22+**, built on top of the standard library. It wraps `net/http` with a rich request context, expressive routing, and a clean middleware API — without sacrificing compatibility with the Go ecosystem.

[![Go Version](https://img.shields.io/badge/go-1.22%2B-00ADD8.svg?style=flat&logo=go)](https://github.com/nanoninja/shinobi)
[![Go Reference](https://pkg.go.dev/badge/github.com/nanoninja/shinobi.svg)](https://pkg.go.dev/github.com/nanoninja/shinobi)
[![Go Report Card](https://goreportcard.com/badge/github.com/nanoninja/shinobi)](https://goreportcard.com/report/github.com/nanoninja/shinobi)
[![CI](https://github.com/nanoninja/shinobi/actions/workflows/ci.yaml/badge.svg)](https://github.com/nanoninja/shinobi/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/nanoninja/shinobi/branch/main/graph/badge.svg)](https://codecov.io/gh/nanoninja/shinobi)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue.svg)](LICENSE)

> **v0.x** — Shinobi is in active development. The API is stabilizing but breaking changes may still occur between minor versions. It can be used in production — pin your version in `go.mod` and review the [CHANGELOG](CHANGELOG.md) before upgrading.

## Features

- **Rich Context (`Ctx`)** — Unified API for request, response, params, cookies, rendering, and more.
- **Error-returning Handlers** — Handlers return `error`, enabling clean centralized error handling.
- **Expressive Routing** — Groups, multi-method routes, prefix mounting, and route introspection.
- **Middleware** — Global, group-level, or per-route middleware with `Use`, `With`, and `With().Route()`.
- **Rate Limiting** — Per-IP sliding window rate limiter with configurable limit and window.
- **Real IP** — Extract the true client IP behind reverse proxies with optional trusted CIDR enforcement.
- **WebSocket** — Adapter interface to integrate any WebSocket library (gorilla, nhooyr, …) without coupling to a specific implementation.
- **stdlib Adapter** — Bridge `http.Handler` and `func(http.Handler) http.Handler` into shinobi with `AdaptHTTP` and `Adapt`.
- **Static File Serving** — Serve local directories or embedded filesystems with `FileServer` and `FileServerFS`.
- **Built-in Rendering** — JSON, XML, plain text, HTML templates, binary blobs, and file serving.
- **Server Lifecycle** — `Listen`, `ListenTLS`, `ListenGraceful`, `ListenTLSGraceful`, and a custom `Server()` builder.
- **Functional Options** — Clean configuration via `WithPrefix`, `WithRenderer`, `WithErrorHandler`, and more.

## Installation

```bash
go get github.com/nanoninja/shinobi
```

## Quick Start

```go
package main

import (
    "log"
    "log/slog"
    "net/http"
    "os"
    "time"

    "github.com/nanoninja/shinobi"
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    app := shinobi.New(
        shinobi.WithPrefix("/api"),
        shinobi.WithLogger(logger),
    )

    app.Get("/health", func(c shinobi.Ctx) error {
        return c.String(http.StatusOK, "OK")
    })

    app.Group("/v1", func(r shinobi.Router) {
        r.Get("/users/{id}", getUser)
        r.Post("/users", createUser)
    })

    log.Fatal(app.ListenGraceful(":8080", 10*time.Second))
}
```

## Configuration

Shinobi uses the **Functional Options** pattern. All options configure the underlying router.

```go
app := shinobi.New(
    shinobi.WithPrefix("/api/v1"),
    shinobi.WithLogger(logger),
    shinobi.WithDebug(true),
    shinobi.WithRenderer(tmplRenderer),
    shinobi.WithBinder(shinobi.JSONBinder(shinobi.JSONStrict())),
    shinobi.WithValidator(myValidator),
    shinobi.WithErrorHandler(myErrorHandler),
    shinobi.WithNotFound(notFoundHandler),
    shinobi.WithMux(http.NewServeMux()),
)
```

| Option | Description |
|---|---|
| `WithPrefix(string)` | Sets a global base path for all routes. |
| `WithLogger(*slog.Logger)` | Sets the logger injected into each `Ctx`. Defaults to `slog.Default()`. |
| `WithDebug(bool)` | Enables debug mode — error responses include the original error message. |
| `WithRenderer(render.Renderer)` | Sets the template renderer injected into each `Ctx`. |
| `WithBinder(Binder)` | Sets the binder used to decode request bodies in `Ctx.Bind`. |
| `WithValidator(Validator)` | Sets the validator used in `Ctx.Validate`. |
| `WithErrorHandler(ErrorHandler)` | Overrides the default 500 error handler. |
| `WithNotFound(HandlerFunc)` | Sets a custom 404 handler. |
| `WithMux(*http.ServeMux)` | Injects a custom `ServeMux` instance. |

## Size Constants

Shinobi exports `KB`, `MB`, and `GB` constants for use with any byte-size parameter (`MaxSize`, `BodyLimit`, `JSONMaxBytes`, …):

```go
shinobi.KB // 1 024 bytes
shinobi.MB // 1 048 576 bytes
shinobi.GB // 1 073 741 824 bytes

// Examples
shinobi.NewImageUpload(f, h, 5*shinobi.MB)
app.Use(middleware.BodyLimit(10 * shinobi.MB))
shinobi.JSONMaxBytes(shinobi.MB)
```

## Routing

### HTTP Verb Shortcuts

```go
app.Get("/users", listUsers)
app.Post("/users", createUser)
app.Put("/users/{id}", updateUser)
app.Patch("/users/{id}", patchUser)
app.Delete("/users/{id}", deleteUser)
```

### Path Parameters

```go
app.Get("/users/{id}", func(c shinobi.Ctx) error {
    id := c.Param("id")
    return c.String(http.StatusOK, "User: %s", id)
})
```

### Multi-Method Routes

Register multiple methods on a single path without repetition:

```go
app.Route("/users/{id}", func(rt shinobi.Route) {
    rt.Get(getUser)
    rt.Put(updateUser)
    rt.Delete(deleteUser)
})
```

To apply a middleware to all methods of a route, use `With` before `Route`:

```go
app.With(authMiddleware).Route("/users/{id}", func(rt shinobi.Route) {
    rt.Get(getUser)
    rt.Put(updateUser)
    rt.Delete(deleteUser)
})
```

### Groups

Organize routes under a shared prefix. Each group inherits the parent middleware stack independently:

```go
app.Group("/api", func(r shinobi.Router) {
    r.Use(authMiddleware)

    r.Group("/v1", func(v1 shinobi.Router) {
        v1.Get("/users", listUsers)
        v1.Post("/users", createUser)
    })
})
```

### Mount

Attach a shinobi `Handler` at a given prefix. Use `AdaptHTTP` to mount stdlib handlers:

```go
// Mount a shinobi handler
app.Mount("/admin", adminHandler)

// Mount a stdlib http.Handler (third-party router, custom handler...)
app.Mount("/metrics", shinobi.AdaptHTTP(promhttp.Handler()))
```

### Static Files

`FileServer` and `FileServerFS` return a shinobi `Handler` ready to use with `Mount`:

```go
// Serve from a local directory
app.Mount("/assets", shinobi.FileServer("./public"))

// Serve from an embedded filesystem (Go embed)
//go:embed public
var embedded embed.FS

app.Mount("/assets", shinobi.FileServerFS(embedded))

// Use fs.Sub to rebase the root — files at public/style.css are served as /assets/style.css
sub, _ := fs.Sub(embedded, "public")
app.Mount("/assets", shinobi.FileServerFS(sub))
```

### Route Introspection

```go
for _, route := range app.Routes() {
    fmt.Printf("%s %s\n", route.Method, route.Pattern)
}
```

## Handler

Shinobi handlers return an `error`, enabling centralized error handling at the framework level:

```go
type Handler interface {
    Handle(c Ctx) error
}

// Use a plain function with HandlerFunc
app.Get("/ping", shinobi.HandlerFunc(func(c shinobi.Ctx) error {
    return c.String(http.StatusOK, "pong")
}))
```

## Context (`Ctx`)

Every handler receives a `Ctx` that wraps the HTTP request and response.

### Request

```go
c.Request()                  // *http.Request
c.Method()                   // "GET", "POST", ...
c.Path()                     // "/users/42"
c.Host()                     // "example.com"
c.Scheme()                   // "https" or "http"
c.Param("id")                // path parameter
c.Query("page")              // query string value
c.QueryValues()              // url.Values
c.FormValue("name")          // form field
c.FormFile("avatar")         // multipart file upload
c.ContentType()              // "application/json"
c.UserAgent()                // User-Agent header
c.Referer()                  // Referer header
c.IsMethod("POST")           // case-insensitive method check
c.IsSecure()                 // HTTPS check
c.IsXHR()                    // XMLHttpRequest check
c.IsWebSocket()              // WebSocket upgrade check
```

### Bind

`Bind` decodes the request body into a struct based on the `Content-Type` header. JSON and XML are supported out of the box via `DefaultBinder`.

```go
app.Post("/users", func(c shinobi.Ctx) error {
    var u CreateUser
    if err := c.Bind(&u); err != nil {
        return err
    }
    return c.JSON(http.StatusCreated, u)
})
```

Use `WithBinder` to customize the binder:

```go
app := shinobi.New(
    shinobi.WithBinder(shinobi.BinderRegistry{
        "application/json": shinobi.JSONBinder(
            shinobi.JSONStrict(),
            shinobi.JSONMaxBytes(shinobi.MB),
        ),
        "application/xml": shinobi.XMLBinder(),
    }),
)
```

### BindQuery

`BindQuery` decodes URL query parameters into a struct using `query:""` field tags. Pointer fields are optional — `nil` when the parameter is absent.

```go
type SearchParams struct {
    Query     string    `query:"q"`
    Page      int       `query:"page"`
    Active    bool      `query:"active"`
    CreatedAt time.Time `query:"created_at"`                      // RFC3339, "2006-01-02 15:04:05" or "2006-01-02"
    BirthDate time.Time `query:"birth_date" format:"2006-01-02"`  // custom format
}

app.Get("/search", func(c shinobi.Ctx) error {
    var p SearchParams
    if err := c.BindQuery(&p); err != nil {
        return err
    }
    return c.JSON(http.StatusOK, p)
})
```

Supported types: `string`, `int`, `int64`, `float64`, `bool`, `time.Time`, and their pointer variants.

Use `format:""` to specify a custom [Go time layout](https://pkg.go.dev/time#Layout). When absent, the following formats are tried in order: `time.RFC3339`, `time.DateTime`, `time.DateOnly`.

### BindForm

`BindForm` decodes `application/x-www-form-urlencoded` fields into a struct using `form:""` field tags. Same rules as `BindQuery` — pointer fields are optional, `time.Time` and `format:""` are supported.

```go
type RegisterForm struct {
    Username  string    `form:"username"`
    BirthDate time.Time `form:"birth_date" format:"2006-01-02"`
}

app.Post("/register", func(c shinobi.Ctx) error {
    var f RegisterForm
    if err := c.BindForm(&f); err != nil {
        return err
    }
    return c.JSON(http.StatusOK, f)
})
```

`BindForm` is also available via `c.Bind` when the `Content-Type` is `application/x-www-form-urlencoded`.

### Extending with a custom Binder

`BinderRegistry` is open for extension — add any MIME type without modifying Shinobi.

The built-in `FormBinder` supports `string`, `int`, `int64`, `float64`, `bool`, and `time.Time`. For more advanced use cases — nested structs, slices, custom types — you can replace it with a third-party decoder such as [`gorilla/schema`](https://github.com/gorilla/schema):

```go
import "github.com/gorilla/schema"

type schemaFormBinder struct {
    decoder *schema.Decoder
}

func (b *schemaFormBinder) Bind(r *http.Request, v any) error {
    if err := r.ParseForm(); err != nil {
        return err
    }
    return b.decoder.Decode(v, r.PostForm)
}

app := shinobi.New(
    shinobi.WithBinder(shinobi.BinderRegistry{
        "application/json":                  shinobi.JSONBinder(shinobi.JSONStrict()),
        "application/xml":                   shinobi.XMLBinder(),
        "application/x-www-form-urlencoded": &schemaFormBinder{decoder: schema.NewDecoder()},
    }),
)
```

### Validate

`Validate` validates a value using the configured `Validator`. No default implementation is provided — plug in the library of your choice via `WithValidator`. Calling `c.Validate` without a configured validator returns `ErrValidatorNotSet`.

Implement the `Validator` interface:

```go
type Validator interface {
    Validate(v any) error
}
```

Example with [`go-playground/validator`](https://github.com/go-playground/validator):

```go
import "github.com/go-playground/validator/v10"

type customValidator struct {
    v *validator.Validate
}

func (cv *customValidator) Validate(v any) error {
    return cv.v.Struct(v)
}

app := shinobi.New(
    shinobi.WithValidator(&customValidator{
        v: validator.New(),
    }),
)

app.Post("/users", func(c shinobi.Ctx) error {
    var u CreateUser
    if err := c.Bind(&u); err != nil {
        return err
    }
    if err := c.Validate(&u); err != nil {
        return err
    }
    return c.JSON(http.StatusCreated, u)
})
```

### Self-validating types

Any type that implements the `Validatable` interface is validated directly by `c.Validate`, bypassing the globally configured `Validator`. This is useful for types that carry their own validation logic.

```go
type Validatable interface {
    Validate() error
}
```

### File Upload Validation

`FileUpload` implements `Validatable` and validates a multipart file against size, MIME type, and extension constraints. Pass it to `c.Validate` after calling `c.FormFile`.

Three constructors are available depending on the use case:

```go
// Preset for web images (JPEG, PNG, GIF, WebP)
upload := shinobi.NewImageUpload(f, header, 2*shinobi.MB)

// Preset for PDF documents
upload := shinobi.NewDocumentUpload(f, header, 10*shinobi.MB)

// Bare constructor — configure constraints manually
upload := shinobi.NewFileUploadValidator(f, header)
upload.MaxSize = 2 << 20
upload.AllowedTypes = []string{"image/jpeg", "image/png"}
upload.AllowedExtensions = []string{".jpg", ".png"}
```

Use `AddType` and `AddExtension` to extend a preset without replacing it:

```go
app.Post("/upload", func(c shinobi.Ctx) error {
    f, header, err := c.FormFile("avatar")
    if err != nil {
        return shinobi.HTTPError(http.StatusBadRequest, err.Error())
    }
    defer f.Close()

    upload := shinobi.NewImageUpload(f, header, 2*shinobi.MB).
        AddType("image/webp").
        AddExtension(".webp")
    if err := c.Validate(upload); err != nil {
        return err
    }
    // f is ready to read — Seek(0) was called after each check
    return c.NoContent()
})
```

**Size** is measured by reading the actual file body, not from the client-declared `Header.Size`. For a hard cap at the transport level, combine with `middleware.BodyLimit`:

```go
app.Use(middleware.BodyLimit(10 << 20)) // 10 MB max across all routes
```

**MIME type** is detected from the first 512 bytes of the file content (`http.DetectContentType`), not from the `Content-Type` header sent by the client. This covers common formats (images, PDF, ZIP…) but not all — SVG is detected as `text/xml`. Provide a custom `DetectFunc` to use a more complete library:

```go
import "github.com/gabriel-vasile/mimetype"

upload := shinobi.NewImageUpload(f, header, 2*shinobi.MB).AddType("image/svg+xml")
upload.DetectFunc = func(b []byte) string { return mimetype.Detect(b).String() }
```

**Extension** is checked against the filename declared by the client. It is spoofable on its own — always combine with `AllowedTypes` for reliable validation.

### Context Values

Values are stored in the request's context chain (`context.WithValue` under the hood):

```go
c.Set("user", user)          // store
v, ok := c.Get("user")       // retrieve

// Stdlib context access
c.Context()                  // context.Context from the request
c.WithContext(newCtx)        // returns a new Ctx with updated context
```

### Response

```go
c.Response()                 // *Response (wrapped ResponseWriter)
c.SetHeader("X-Custom", "v")
c.AddHeader("Vary", "Accept")
```

### Rendering

```go
c.JSON(http.StatusOK, data)
c.XML(http.StatusOK, data)
c.CSV(http.StatusOK, [][]string{{"name", "age"}, {"alice", "30"}})
c.String(http.StatusOK, "Hello, %s", name)
c.HTML(http.StatusOK, "index.html", data)   // requires WithRenderer
c.Blob(http.StatusOK, pdfBytes, render.MimePDF())
c.NoContent()
```

### Files

```go
c.File("/path/to/file.pdf")
c.Attachment("/path/to/report.pdf")   // Content-Disposition: attachment
c.Inline("/path/to/preview.pdf")      // Content-Disposition: inline
```

### Navigation

```go
return c.Redirect("/login", http.StatusFound)
```

### Cookies

```go
c.SetCookie(&http.Cookie{Name: "session", Value: "abc"})
c.DeleteCookie("session", "/")
```

## Middleware

Middlewares wrap a `Handler` to extend or intercept request processing:

```go
type Middleware func(Handler) Handler
```

### Applying Middleware

```go
// Global
app.Use(loggerMiddleware, authMiddleware)

// Group-level
app.Group("/admin", func(r shinobi.Router) {
    r.Use(requireAdmin)
    r.Get("/dashboard", dashboard)
})

// Per-route scoping
app.With(rateLimitMiddleware).Post("/login", handleLogin)
```

### Writing a Middleware

```go
func Logger(next shinobi.Handler) shinobi.Handler {
    return shinobi.HandlerFunc(func(c shinobi.Ctx) error {
        start := time.Now()
        err := next.Handle(c)
        slog.Info("request",
            "method", c.Method(),
            "path", c.Path(),
            "duration", time.Since(start),
        )
        return err
    })
}
```

### Adapting stdlib Middleware

```go
// func(http.Handler) http.Handler → shinobi Middleware
app.Use(shinobi.Adapt(corsMiddleware))
app.Use(shinobi.Adapt(tracingMiddleware))

// http.Handler → shinobi Handler
app.Mount("/metrics", shinobi.AdaptHTTP(promhttp.Handler()))
```

`Adapt` propagates any context changes made by the stdlib middleware (e.g. `r.WithContext(...)`) back into the shinobi `Ctx`.

## Built-in Middlewares

Shinobi provides built-in middlewares in the `middleware` sub-package.

### RequestID

`RequestID` assigns a unique ID to each request via the `X-Request-ID` header. If the incoming request already carries the header, it is reused — useful for distributed tracing. The ID is also stored in the request context.

```go
app.Use(middleware.RequestID())

// retrieve in a handler
app.Get("/", func(c shinobi.Ctx) error {
    id, _ := c.Get(middleware.RequestIDKey)
    return c.String(http.StatusOK, "request id: %s", id)
})
```

### Logger

`Logger` logs each request using `slog`. It records the HTTP method, path, status code, duration, and request ID if present.

```go
app.Use(middleware.RequestID())
app.Use(middleware.Logger())
```

Example output:

```json
{"time":"2026-04-19T10:00:00Z","level":"INFO","msg":"request","method":"GET","path":"/users","status":200,"duration":"1.2ms","request_id":"a1b2c3d4-..."}
```

### Recoverer

`Recoverer` catches panics in handlers, logs the error and stack trace via `slog`, and returns a `500 Internal Server Error` through the configured error handler.

```go
import "github.com/nanoninja/shinobi/middleware"

app.Use(middleware.Recoverer())
```

The panic is wrapped in a `*StatusError` with the original cause available for logging:

```go
app := shinobi.New(
    shinobi.WithErrorHandler(func(err error, c shinobi.Ctx) {
        if e, ok := err.(*shinobi.StatusError); ok {
            if e.Cause != nil {
                slog.Error("panic recovered", "cause", e.Cause)
            }
            _ = c.JSON(e.Code, map[string]any{"error": "internal server error"})
            return
        }
        _ = c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
    }),
)
app.Use(middleware.Recoverer())
```

### Compress

`Compress` compresses the response body using gzip or deflate, selected from the client's `Accept-Encoding` header. It sets `Content-Encoding` and `Vary: Accept-Encoding` headers and removes `Content-Length`.

```go
import (
    "compress/gzip"
    "github.com/nanoninja/shinobi/middleware"
)

app.Use(middleware.Compress(gzip.DefaultCompression))
```

The `level` parameter controls compression strength. Use constants from `compress/gzip` or `compress/flate` (e.g. `gzip.BestSpeed`, `flate.BestCompression`). An invalid level falls back to no compression.

### Timeout

`Timeout` cancels the request context after the given duration. If the deadline is exceeded before the handler writes a response, a `503 Service Unavailable` is returned through the error handler.

```go
app.Use(middleware.Timeout(5 * time.Second))
```

> **Note:** `Timeout` only works if the handler respects context cancellation — for example by passing the context to database queries or outgoing HTTP calls. CPU-bound work that never checks `ctx.Err()` will not be interrupted.

### CORS

`CORS` adds Cross-Origin Resource Sharing headers to every response. Preflight requests (`OPTIONS`) are handled automatically and short-circuited with a `204 No Content`.

```go
// Permissive defaults — suitable for development
app.Use(middleware.CORS(middleware.DefaultCORSConfig()))

// Restrict to specific origins
app.Use(middleware.CORS(middleware.DefaultCORSConfig("https://myapp.com", "https://staging.myapp.com")))

// Fully custom
app.Use(middleware.CORS(middleware.CORSConfig{
    AllowedOrigins:   []string{"https://myapp.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge:           3600,
}))
```

When multiple specific origins are provided, the middleware reflects only the matched request `Origin` in the response and adds `Vary: Origin` automatically.

> **Note:** `AllowCredentials: true` is incompatible with a wildcard origin (`*`). Browsers will reject such a combination — always specify explicit origins when credentials are needed.

### SecureHeaders

`SecureHeaders` sets common security-related HTTP response headers with safe, opinionated defaults.

```go
app.Use(middleware.SecureHeaders())
```

| Header | Value |
|---|---|
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `X-XSS-Protection` | `0` |
| `Referrer-Policy` | `strict-origin-when-cross-origin` |

`SecureHeadersWithHSTS` adds `Strict-Transport-Security` on top of the standard headers. Only use it in production behind HTTPS — enabling HSTS on a plain HTTP server makes the site unreachable over HTTP for the duration of `maxAge`.

```go
// 1 year, current domain only
app.Use(middleware.SecureHeadersWithHSTS(31536000, false))

// 1 year, current domain + all subdomains
app.Use(middleware.SecureHeadersWithHSTS(31536000, true))
```

### BodyLimit

`BodyLimit` caps the request body size. Requests exceeding the limit receive a `413 Request Entity Too Large` through the error handler.

```go
app.Use(middleware.BodyLimit(shinobi.MB))
```

### BasicAuth

`BasicAuth` enforces HTTP Basic Authentication. Unauthenticated requests receive a `401` with a `WWW-Authenticate` challenge. On success, the credential is stored in the context.

> **WARNING:** Basic Auth credentials are base64-encoded, not encrypted. Always use this middleware behind HTTPS in production.

```go
// Single user
app.Use(middleware.BasicAuth(middleware.BasicAuthConfig{
    Validator: middleware.Auth("alice", "s3cr3t"),
}))

// Multiple users
app.Use(middleware.BasicAuth(middleware.BasicAuthConfig{
    Validator: middleware.User{"alice": "s3cr3t", "bob": "p4ssw0rd"},
}))

// Custom validator (e.g. database lookup)
app.Use(middleware.BasicAuth(middleware.BasicAuthConfig{
    Realm: "Admin Area",
    Validator: middleware.ValidateFunc(func(_ shinobi.Ctx, c middleware.BasicAuthCredential) bool {
        return middleware.SecureCompare(c.Username, "alice") &&
            middleware.SecureCompare(c.Password, "s3cr3t")
    }),
}))
```

Retrieve the authenticated user in a handler with `BasicAuthCredentialFrom`:

```go
app.Get("/profile", func(c shinobi.Ctx) error {
    cred, _ := middleware.BasicAuthCredentialFrom(c)
    return c.String(http.StatusOK, "hello, %s", cred.Username)
})
```

### RateLimit

`RateLimit` limits the number of requests per IP using a sliding window algorithm. Requests that exceed the limit receive a `429 Too Many Requests` through the error handler.

```go
app.Use(middleware.RateLimit(100, time.Minute))
```

### RealIP

`RealIP` sets the request's `RemoteAddr` to the real client IP extracted from the `True-Client-IP`, `X-Real-IP`, or `X-Forwarded-For` headers (in that order). The original port is preserved.

> **WARNING:** Only use this middleware when requests come through a trusted reverse proxy. Without `TrustedProxies`, any client can forge these headers.

```go
// Trust headers unconditionally — suitable only behind a controlled proxy
app.Use(middleware.RealIP())

// Restrict to known proxy CIDR ranges — recommended for production
app.Use(middleware.RealIPWithConfig(middleware.RealIPConfig{
    TrustedProxies: []string{"10.0.0.0/8", "192.168.1.0/24"},
}))
```

Place `RealIP` early in the middleware stack — before `Logger`, `BasicAuth`, and `RateLimit` — so that subsequent layers see the correct client IP.

## WebSocket

Shinobi does not implement the WebSocket protocol. Instead it provides a minimal adapter interface so you can integrate any WebSocket library without coupling your application to a specific implementation.

```go
go get github.com/nanoninja/shinobi/websocket
```

### Interfaces

```go
type Upgrader interface {
    Upgrade(w http.ResponseWriter, r *http.Request) (Conn, error)
}

type Conn interface {
    ReadMessage() (messageType int, p []byte, err error)
    WriteMessage(messageType int, data []byte) error
    Close() error
}
```

### Message Type Constants

```go
websocket.TextMessage   // 1
websocket.BinaryMessage // 2
websocket.CloseMessage  // 8
websocket.PingMessage   // 9
websocket.PongMessage   // 10
```

### Usage with gorilla/websocket

Implement the `Upgrader` interface once for your chosen library:

```go
import (
    "net/http"

    "github.com/gorilla/websocket"
    ws "github.com/nanoninja/shinobi/websocket"
)

type GorillaUpgrader struct {
    u *websocket.Upgrader
}

func NewGorillaUpgrader() *GorillaUpgrader {
    return &GorillaUpgrader{
        u: &websocket.Upgrader{
            CheckOrigin: func(r *http.Request) bool { return true },
        },
    }
}

func (g *GorillaUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (ws.Conn, error) {
    return g.u.Upgrade(w, r, nil)
}
```

Register the route with `ws.Handle`. The connection is automatically closed when the handler returns:

```go
upgrader := NewGorillaUpgrader()

app.Get("/ws", ws.Handle(upgrader, func(c shinobi.Ctx, conn ws.Conn) error {
    for {
        msgType, msg, err := conn.ReadMessage()
        if err != nil {
            return err
        }
        if err := conn.WriteMessage(msgType, msg); err != nil {
            return err
        }
    }
}))
```

## Error Handling

Handlers return errors that are caught by the framework. The default handler writes a `500 Internal Server Error`. Override it with `WithErrorHandler`:

```go
app := shinobi.New(
    shinobi.WithErrorHandler(func(err error, c shinobi.Ctx) {
        c.Response().Header().Set("Content-Type", "application/json")
        _ = c.JSON(http.StatusInternalServerError, map[string]string{
            "error": err.Error(),
        })
    }),
)
```

### HTTPError

Use `HTTPError` to return a structured HTTP error directly from a handler. The default error handler will respond with the appropriate status code automatically.

```go
app.Get("/users/{id}", func(c shinobi.Ctx) error {
    user, err := getUser(c.Param("id"))
    if err != nil {
        return shinobi.HTTPError(http.StatusNotFound, "user not found")
    }
    return c.JSON(http.StatusOK, user)
})

// without message — uses http.StatusText as fallback
return shinobi.HTTPError(http.StatusUnauthorized)
```

Handle `*StatusError` in a custom error handler to customize the response format:

```go
shinobi.WithErrorHandler(func(err error, c shinobi.Ctx) {
    if e, ok := err.(*shinobi.StatusError); ok {
        _ = c.JSON(e.Code, map[string]any{"error": e.Message})
        return
    }
    _ = c.JSON(http.StatusInternalServerError, map[string]any{"error": "internal server error"})
})
```

Use `WithInternal` to wrap the original error for logging without exposing it to the client:

```go
user, err := db.Find(c.Param("id"))
if err != nil {
    return shinobi.HTTPError(http.StatusNotFound, "user not found").WithInternal(err)
}
```

The internal cause is accessible via `errors.Is` and `errors.As`, and available on `StatusError.Cause` for logging in the error handler:

```go
shinobi.WithErrorHandler(func(err error, c shinobi.Ctx) {
    if e, ok := err.(*shinobi.StatusError); ok {
        if e.Cause != nil {
            slog.Error("internal error", "cause", e.Cause)
        }
        _ = c.JSON(e.Code, map[string]any{"error": e.Message})
        return
    }
    _ = c.JSON(http.StatusInternalServerError, map[string]any{"error": "internal server error"})
})
```

You can define your own helpers on top of `HTTPError`:

```go
func NotFound(msg ...any) error     { return shinobi.HTTPError(http.StatusNotFound, msg...) }
func Unauthorized(msg ...any) error { return shinobi.HTTPError(http.StatusUnauthorized, msg...) }
func BadRequest(msg ...any) error   { return shinobi.HTTPError(http.StatusBadRequest, msg...) }
```

## HTML Templates

Provide a pre-loaded template renderer at startup. Shinobi integrates with [`github.com/nanoninja/render`](https://github.com/nanoninja/render):

```go
import (
    "github.com/nanoninja/render/tmpl"
    "github.com/nanoninja/render/tmpl/loader"
)

l := loader.NewFS(loader.LoaderConfig{Root: "templates", Extension: ".html"})

t := tmpl.NewHTML()
t.Load(l)

app := shinobi.New(shinobi.WithRenderer(t))

app.Get("/", func(c shinobi.Ctx) error {
    return c.HTML(http.StatusOK, "index", map[string]any{
        "Title": "Home",
    })
})
```

## Server Lifecycle

```go
// HTTP — blocking
app.Listen(":8080")

// HTTPS — blocking
app.ListenTLS(":443", "cert.pem", "key.pem")

// HTTP with graceful shutdown on SIGINT/SIGTERM
log.Fatal(app.ListenGraceful(":8080", 10*time.Second))

// HTTPS with graceful shutdown
log.Fatal(app.ListenTLSGraceful(":443", "cert.pem", "key.pem", 10*time.Second))

// Custom server — full control over TLS config, timeouts, etc.
srv := app.Server(":443")
srv.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS13}
srv.ReadTimeout = 5 * time.Second
log.Fatal(srv.ListenAndServeTLS("cert.pem", "key.pem"))
```

## Versioning

Shinobi follows [Semantic Versioning](https://semver.org/). While at `v0.x`, breaking changes may occur between minor versions — see the [CHANGELOG](CHANGELOG.md) for details before upgrading.

## License

This project is licensed under the BSD 3-Clause License.
See the [LICENSE](LICENSE) file for details.
