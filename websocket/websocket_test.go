// Copyright 2026 The Shinobi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/shinobi"
	ws "github.com/nanoninja/shinobi/websocket"
)

// --- mocks ---

type mockConn struct {
	closeErr error
	closed   bool
}

func (m *mockConn) ReadMessage() (int, []byte, error) {
	return ws.TextMessage, []byte("hello"), nil
}

func (m *mockConn) WriteMessage(_ int, _ []byte) error {
	return nil
}

func (m *mockConn) Close() error {
	m.closed = true
	return m.closeErr
}

type mockUpgrader struct {
	conn ws.Conn
	err  error
}

func (m *mockUpgrader) Upgrade(_ http.ResponseWriter, _ *http.Request) (ws.Conn, error) {
	return m.conn, m.err
}

func handle(u ws.Upgrader, h ws.Handler) error {
	r := httptest.NewRequest(http.MethodGet, "/ws", nil)
	handler := ws.Handle(u, h)
	return handler.Handle(shinobi.NewCtx(httptest.NewRecorder(), r, shinobi.DefaultConfig()))
}

// --- tests ---

func TestHandle_UpgradeError(t *testing.T) {
	upgradeErr := errors.New("upgrade failed")
	u := &mockUpgrader{err: upgradeErr}

	err := handle(u, func(_ shinobi.Ctx, _ ws.Conn) error {
		t.Fatal("handler should not be called when upgrade fails")
		return nil
	})

	assert.ErrorIs(t, err, upgradeErr)
}

func TestHandle_HandlerCalled(t *testing.T) {
	called := false
	u := &mockUpgrader{conn: &mockConn{}}

	err := handle(u, func(_ shinobi.Ctx, _ ws.Conn) error {
		called = true
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, called)
}

func TestHandle_ConnClosedAfterHandler(t *testing.T) {
	conn := &mockConn{}
	u := &mockUpgrader{conn: conn}

	_ = handle(u, func(_ shinobi.Ctx, _ ws.Conn) error {
		return nil
	})

	assert.True(t, conn.closed)
}

func TestHandle_HandlerError(t *testing.T) {
	handlerErr := errors.New("handler error")
	u := &mockUpgrader{conn: &mockConn{}}

	err := handle(u, func(_ shinobi.Ctx, _ ws.Conn) error {
		return handlerErr
	})

	assert.ErrorIs(t, err, handlerErr)
}

func TestHandle_CloseErrorPropagatesWhenHandlerSucceeds(t *testing.T) {
	closeErr := errors.New("close error")
	conn := &mockConn{closeErr: closeErr}
	u := &mockUpgrader{conn: conn}

	err := handle(u, func(_ shinobi.Ctx, _ ws.Conn) error {
		return nil
	})

	assert.ErrorIs(t, err, closeErr)
}

func TestHandle_HandlerErrorTakesPrecedenceOverCloseError(t *testing.T) {
	handlerErr := errors.New("handler error")
	conn := &mockConn{closeErr: errors.New("close error")}
	u := &mockUpgrader{conn: conn}

	err := handle(u, func(_ shinobi.Ctx, _ ws.Conn) error {
		return handlerErr
	})

	assert.ErrorIs(t, err, handlerErr)
}

func TestHandle_CtxPassedToHandler(t *testing.T) {
	u := &mockUpgrader{conn: &mockConn{}}
	var gotCtx shinobi.Ctx

	_ = handle(u, func(c shinobi.Ctx, _ ws.Conn) error {
		gotCtx = c
		return nil
	})

	assert.NotNil(t, gotCtx)
}

func TestHandle_ConnPassedToHandler(t *testing.T) {
	conn := &mockConn{}
	u := &mockUpgrader{conn: conn}
	var gotConn ws.Conn

	_ = handle(u, func(_ shinobi.Ctx, c ws.Conn) error {
		gotConn = c
		return nil
	})

	assert.Equal(t, conn, gotConn.(*mockConn))
}
