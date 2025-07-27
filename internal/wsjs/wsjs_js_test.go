//go:build js
// +build js

package wsjs

import (
	"syscall/js"
	"testing"
)

func TestHandleJSError_NoPanic(t *testing.T) {
	var err error
	handleJSError(&err, nil) // Should not panic or set err
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestWebSocketAPI(t *testing.T) {
	if js.Global().Get("WebSocket").IsUndefined() {
		t.Skip("WebSocket API not available in this environment")
	}

	ws, err := New("ws://localhost:12345", nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	_ = ws.Subprotocol()

	removeOpen := ws.OnOpen(func(e js.Value) {})
	removeMessage := ws.OnMessage(func(m MessageEvent) {})
	removeError := ws.OnError(func(e js.Value) {})
	removeClose := ws.OnClose(func(ce CloseEvent) {})

	removeOpen()
	removeMessage()
	removeError()
	removeClose()

	_ = ws.SendText("hello")
	_ = ws.SendBytes([]byte{1, 2, 3})
	_ = ws.Close(1000, "bye")
}

func TestExtractArrayBuffer(t *testing.T) {
	if js.Global().Get("Uint8Array").IsUndefined() {
		t.Skip("Uint8Array not available in this environment")
	}
	arr := []byte{1, 2, 3, 4}
	jsArr := uint8Array(arr)
	goArr := extractArrayBuffer(jsArr)
	if len(goArr) != len(arr) {
		t.Errorf("expected length %d, got %d", len(arr), len(goArr))
	}
}
