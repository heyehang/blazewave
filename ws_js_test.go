package blazewave_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/heyehang/blazewave"
	"github.com/heyehang/blazewave/internal/test/assert"
	"github.com/heyehang/blazewave/internal/test/wstest"
)

func TestWasmEcho(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	c, resp, err := blazewave.Dial(ctx, os.Getenv("WS_ECHO_SERVER_URL"), &blazewave.DialOptions{
		CommonOptions: blazewave.CommonOptions{
			Subprotocols: []string{"echo"},
		},
	})
	assert.Success(t, err)
	defer c.Close(blazewave.StatusInternalError, "")

	assert.Equal(t, "subprotocol", "echo", c.Subprotocol())
	assert.Equal(t, "response code", http.StatusSwitchingProtocols, resp.StatusCode)

	c.SetReadLimit(65536)
	for range 10 {
		err = wstest.Echo(ctx, c, 65536)
		assert.Success(t, err)
	}

	err = c.Close(blazewave.StatusNormalClosure, "")
	assert.Success(t, err)
}

func TestWasmDialTimeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	beforeDial := time.Now()
	_, _, err := blazewave.Dial(ctx, "ws://example.com:9893", &blazewave.DialOptions{
		CommonOptions: blazewave.CommonOptions{
			Subprotocols: []string{"echo"},
		},
	})
	assert.Error(t, err)
	if time.Since(beforeDial) >= time.Second {
		t.Fatal("wasm context dial timeout is not working", time.Since(beforeDial))
	}
}

func TestWasmEventSystem(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var (
		connectCalled    bool
		disconnectCalled bool
		messageReceived  bool
		errorReceived    bool
		middlewareCalled bool
	)

	event := &blazewave.Event{} // use direct struct literal for test
	// Register a middleware to check if it is called
	event.Use(func(next blazewave.HandlerFunc) blazewave.HandlerFunc {
		return func(ctx context.Context, args *blazewave.EventArgs) error {
			middlewareCalled = true
			return next(ctx, args)
		}
	})
	// Register OnConnect event
	event.OnConnect(func(ctx context.Context, conn *blazewave.Conn) error {
		connectCalled = true
		return nil
	})
	// Register OnDisconnect event
	event.OnDisconnect(func(ctx context.Context, conn *blazewave.Conn) error {
		disconnectCalled = true
		return nil
	})
	// Register OnTextMessage event
	event.OnTextMessage(func(ctx context.Context, conn *blazewave.Conn, payload []byte) error {
		messageReceived = true
		return nil
	})
	// Register OnError event
	event.OnError(func(ctx context.Context, conn *blazewave.Conn, err error) error {
		errorReceived = true
		return nil
	})

	c, resp, err := blazewave.Dial(ctx, os.Getenv("WS_ECHO_SERVER_URL"), &blazewave.DialOptions{
		CommonOptions: blazewave.CommonOptions{
			Subprotocols: []string{"echo"},
		},
		Event: event,
	})
	assert.Success(t, err)
	defer c.Close(blazewave.StatusInternalError, "")

	assert.Equal(t, "subprotocol", "echo", c.Subprotocol())
	assert.Equal(t, "response code", http.StatusSwitchingProtocols, resp.StatusCode)

	// Send a message to trigger OnTextMessage
	err = c.Write(ctx, blazewave.MessageText, []byte("hello wasm event"))
	assert.Success(t, err)

	// Read a message to ensure event is triggered
	_, _, _ = c.Read(ctx)
	time.Sleep(100 * time.Millisecond) // Wait for event callback

	assert.Equal(t, "OnConnect should be called", true, connectCalled)
	assert.Equal(t, "Middleware should be called", true, middlewareCalled)
	assert.Equal(t, "OnTextMessage should be called", true, messageReceived)

	// Close the connection to trigger OnDisconnect
	err = c.Close(blazewave.StatusNormalClosure, "")
	assert.Success(t, err)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, "OnDisconnect should be called", true, disconnectCalled)

	// Trigger error event by closing again
	_ = c.Close(blazewave.StatusNormalClosure, "")
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, "OnError should be called", true, errorReceived)
}
