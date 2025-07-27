//go:build js

package blazewave

import "context"

// EventArgs struct for all event types (exported)
type EventArgs struct {
	Conn    *Conn
	Payload []byte
	Error   error
	Code    StatusCode
	Reason  string
}

type HandlerFunc func(ctx context.Context, args *EventArgs) error

type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// Event is the event system for js/wasm
type Event struct {
	textMessageHandlers   []HandlerFunc
	binaryMessageHandlers []HandlerFunc
	errorHandlers         []HandlerFunc
	closeHandlers         []HandlerFunc
	connectHandlers       []HandlerFunc
	disconnectHandlers    []HandlerFunc
	pingHandlers          []HandlerFunc
	pongHandlers          []HandlerFunc

	mws []MiddlewareFunc
}

func NewEvent() *Event {
	return &Event{
		textMessageHandlers:   make([]HandlerFunc, 0),
		binaryMessageHandlers: make([]HandlerFunc, 0),
		errorHandlers:         make([]HandlerFunc, 0),
		closeHandlers:         make([]HandlerFunc, 0),
		connectHandlers:       make([]HandlerFunc, 0),
		disconnectHandlers:    make([]HandlerFunc, 0),
		pingHandlers:          make([]HandlerFunc, 0),
		pongHandlers:          make([]HandlerFunc, 0),
		mws:                   make([]MiddlewareFunc, 0),
	}
}

// Middleware registration
func (e *Event) Use(mw MiddlewareFunc) {
	e.mws = append(e.mws, mw)
}

// Registration methods (type-safe user API)
func (e *Event) OnTextMessage(h ...func(ctx context.Context, conn *Conn, payload []byte) error) {
	for _, handler := range h {
		e.textMessageHandlers = append(e.textMessageHandlers, func(ctx context.Context, args *EventArgs) error {
			return handler(ctx, args.Conn, args.Payload)
		})
	}
}
func (e *Event) OnBinaryMessage(h ...func(ctx context.Context, conn *Conn, payload []byte) error) {
	for _, handler := range h {
		e.binaryMessageHandlers = append(e.binaryMessageHandlers, func(ctx context.Context, args *EventArgs) error {
			return handler(ctx, args.Conn, args.Payload)
		})
	}
}
func (e *Event) OnError(h ...func(ctx context.Context, conn *Conn, err error) error) {
	for _, handler := range h {
		e.errorHandlers = append(e.errorHandlers, func(ctx context.Context, args *EventArgs) error {
			return handler(ctx, args.Conn, args.Error)
		})
	}
}
func (e *Event) OnClose(h ...func(ctx context.Context, conn *Conn, code StatusCode, reason string) error) {
	for _, handler := range h {
		e.closeHandlers = append(e.closeHandlers, func(ctx context.Context, args *EventArgs) error {
			return handler(ctx, args.Conn, args.Code, args.Reason)
		})
	}
}
func (e *Event) OnConnect(h ...func(ctx context.Context, conn *Conn) error) {
	for _, handler := range h {
		e.connectHandlers = append(e.connectHandlers, func(ctx context.Context, args *EventArgs) error {
			return handler(ctx, args.Conn)
		})
	}
}
func (e *Event) OnDisconnect(h ...func(ctx context.Context, conn *Conn) error) {
	for _, handler := range h {
		e.disconnectHandlers = append(e.disconnectHandlers, func(ctx context.Context, args *EventArgs) error {
			return handler(ctx, args.Conn)
		})
	}
}
func (e *Event) OnPing(h ...func(ctx context.Context, conn *Conn, payload []byte) error) {
	for _, handler := range h {
		e.pingHandlers = append(e.pingHandlers, func(ctx context.Context, args *EventArgs) error {
			return handler(ctx, args.Conn, args.Payload)
		})
	}
}
func (e *Event) OnPong(h ...func(ctx context.Context, conn *Conn, payload []byte) error) {
	for _, handler := range h {
		e.pongHandlers = append(e.pongHandlers, func(ctx context.Context, args *EventArgs) error {
			return handler(ctx, args.Conn, args.Payload)
		})
	}
}

// Helper to wrap a handler with all middleware
func (e *Event) wrapWithMiddleware(h HandlerFunc) HandlerFunc {
	wrapped := h
	for i := len(e.mws) - 1; i >= 0; i-- {
		wrapped = e.mws[i](wrapped)
	}
	return wrapped
}

// Event dispatch method (can be called by js/wasm main loop)
func (e *Event) handleMessage(ctx context.Context, conn *Conn, messageType MessageType, payload []byte) error {
	args := &EventArgs{Conn: conn, Payload: payload}
	switch messageType {
	case MessageText:
		for _, h := range e.textMessageHandlers {
			wrapped := e.wrapWithMiddleware(h)
			if err := wrapped(ctx, args); err != nil {
				return err
			}
		}
	case MessageBinary:
		for _, h := range e.binaryMessageHandlers {
			wrapped := e.wrapWithMiddleware(h)
			if err := wrapped(ctx, args); err != nil {
				return err
			}
		}
	}
	return nil
}
func (e *Event) handleError(ctx context.Context, conn *Conn, err error) error {
	args := &EventArgs{Conn: conn, Error: err}
	for _, h := range e.errorHandlers {
		wrapped := e.wrapWithMiddleware(h)
		if err2 := wrapped(ctx, args); err2 != nil {
			return err2
		}
	}
	return nil
}
func (e *Event) handleClose(ctx context.Context, conn *Conn, code StatusCode, reason string) error {
	args := &EventArgs{Conn: conn, Code: code, Reason: reason}
	for _, h := range e.closeHandlers {
		wrapped := e.wrapWithMiddleware(h)
		if err := wrapped(ctx, args); err != nil {
			return err
		}
	}
	return nil
}
func (e *Event) handleConnect(ctx context.Context, conn *Conn) error {
	args := &EventArgs{Conn: conn}
	for _, h := range e.connectHandlers {
		wrapped := e.wrapWithMiddleware(h)
		if err := wrapped(ctx, args); err != nil {
			return err
		}
	}
	return nil
}
func (e *Event) handleDisconnect(ctx context.Context, conn *Conn) error {
	args := &EventArgs{Conn: conn}
	for _, h := range e.disconnectHandlers {
		wrapped := e.wrapWithMiddleware(h)
		if err := wrapped(ctx, args); err != nil {
			return err
		}
	}
	return nil
}

// handlePing and handlePong are no-ops in js/wasm because native WebSocket API does not expose ping/pong frames.
//
//lint:ignore U1000 This method is intentionally kept for API compatibility; js/wasm cannot receive ping frames.
func (e *Event) handlePing(ctx context.Context, conn *Conn, payload []byte) error {
	// no-op: browser WebSocket API does not expose ping frames
	return nil
}

//lint:ignore U1000 This method is intentionally kept for API compatibility; js/wasm cannot receive pong frames.
func (e *Event) handlePong(ctx context.Context, conn *Conn, payload []byte) error {
	// no-op: browser WebSocket API does not expose pong frames
	return nil
}
