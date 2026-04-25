# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [0.1.0] - 2026-04-25

### Added
- **Core Engine**: Lightweight HTTP router with support for groups, sub-routers (Mount), and custom error handling.
- **Unified Context**: `Ctx` interface for simplified request/response management, cookie handling, and parameter extraction.
- **Rendering**: Multi-format rendering support including JSON, XML, HTML, Text, CSV, and Binary.
- **Middleware Suite**: Logger, Recoverer, CORS, RateLimit, RealIP, NoCache, SecureHeaders, Timeout, and RequestID.
- **Data Binding**: Struct-based binding for Query and Form data with support for `time.Time`.
- **WebSocket**: Support for WebSocket handlers through an upgrader interface.
- **Validation**: `Validatable` interface and `FileUpload` component for secure multipart file handling and validation.
- **Server**: Production-safe defaults for `http.Server` and support for graceful shutdown.