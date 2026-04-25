// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package websocket provides an interface for integrating any WebSocket
// library with Shinobi. It does not implement the WebSocket protocol itself —
// use a third-party library such as gorilla/websocket or nhooyr.io/websocket
// and wrap it with the Upgrader interface.
package websocket

import (
	"net/http"

	"github.com/nanoninja/shinobi"
)

// Message type constants matching the WebSocket protocol wire format.
const (
	TextMessage   = 1
	BinaryMessage = 2
	CloseMessage  = 8
	PingMessage   = 9
	PongMessage   = 10
)

// Conn represents an active WebSocket connection.
type Conn interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	Close() error
}

// Upgrader upgrades an HTTP connection to a WebSocket connection.
type Upgrader interface {
	Upgrade(w http.ResponseWriter, r *http.Request) (Conn, error)
}

// Handler handles an active WebSocket connection alongside the request context.
type Handler func(c shinobi.Ctx, conn Conn) (err error)

// Handle returns a shinobi.Handler that upgrades the HTTP connection to
// WebSocket and delegates to the provided handler. The connection is
// automatically closed when the handler returns.
func Handle(u Upgrader, h Handler) shinobi.Handler {
	return shinobi.HandlerFunc(func(c shinobi.Ctx) (err error) {
		var conn Conn
		conn, err = u.Upgrade(c.Response(), c.Request())
		if err != nil {
			return
		}
		defer func() {
			if cerr := conn.Close(); cerr != nil && err == nil {
				err = cerr
			}
		}()
		err = h(c, conn)
		return
	})
}
