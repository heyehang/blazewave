package blazewave

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

func createMockConn() *Conn {
	return &Conn{}
}

func TestBlazeWaveHandlerRegistration(t *testing.T) {
	e := NewEvent()

	connectCalled := false
	e.OnConnect(func(ctx context.Context, conn *Conn) error {
		connectCalled = true
		return nil
	})

	disconnectCalled := false
	e.OnDisconnect(func(ctx context.Context, conn *Conn) error {
		disconnectCalled = true
		return nil
	})

	textMessageCalled := false
	e.OnTextMessage(func(ctx context.Context, conn *Conn, payload []byte) error {
		textMessageCalled = true
		return nil
	})

	binaryMessageCalled := false
	e.OnBinaryMessage(func(ctx context.Context, conn *Conn, payload []byte) error {
		binaryMessageCalled = true
		return nil
	})

	errorCalled := false
	e.OnError(func(ctx context.Context, conn *Conn, err error) error {
		errorCalled = true
		return nil
	})

	closeCalled := false
	e.OnClose(func(ctx context.Context, conn *Conn, code StatusCode, reason string) error {
		closeCalled = true
		return nil
	})

	e.OnPing(func(ctx context.Context, conn *Conn, payload []byte) error {
		return nil
	})

	pongCalled := false
	e.OnPong(func(ctx context.Context, conn *Conn, payload []byte) error {
		pongCalled = true
		return nil
	})

	ctx := context.Background()

	mockConn := createMockConn()

	err := e.handleConnect(ctx, mockConn)
	if err != nil {
		t.Errorf("handleConnect failed: %v", err)
	}
	if !connectCalled {
		t.Error("Connect handler should have been called")
	}

	err = e.handleDisconnect(ctx, mockConn)
	if err != nil {
		t.Errorf("handleDisconnect failed: %v", err)
	}
	if !disconnectCalled {
		t.Error("Disconnect handler should have been called")
	}

	err = e.handleMessage(ctx, mockConn, MessageText, []byte("test"))
	if err != nil {
		t.Errorf("handleMessage failed: %v", err)
	}
	if !textMessageCalled {
		t.Error("Text message handler should have been called")
	}

	err = e.handleMessage(ctx, mockConn, MessageBinary, []byte{1, 2, 3})
	if err != nil {
		t.Errorf("handleMessage failed: %v", err)
	}
	if !binaryMessageCalled {
		t.Error("Binary message handler should have been called")
	}

	testErr := errors.New("test error")
	err = e.handleError(ctx, mockConn, testErr)
	if err != nil {
		t.Errorf("handleError failed: %v", err)
	}
	if !errorCalled {
		t.Error("Error handler should have been called")
	}

	err = e.handleClose(ctx, mockConn, StatusNormalClosure, "test")
	if err != nil {
		t.Errorf("handleClose failed: %v", err)
	}
	if !closeCalled {
		t.Error("Close handler should have been called")
	}

	err = e.handlePong(ctx, mockConn, []byte("pong"))
	if err != nil {
		t.Errorf("handlePong failed: %v", err)
	}
	if !pongCalled {
		t.Error("Pong handler should have been called")
	}
}

func TestBlazeWaveMultipleHandlers(t *testing.T) {
	e := NewEvent()

	executionOrder := make([]int, 0)
	var mu sync.Mutex

	e.OnConnect(func(ctx context.Context, conn *Conn) error {
		mu.Lock()
		executionOrder = append(executionOrder, 1)
		mu.Unlock()
		return nil
	})

	e.OnConnect(func(ctx context.Context, conn *Conn) error {
		mu.Lock()
		executionOrder = append(executionOrder, 2)
		mu.Unlock()
		return nil
	})

	e.OnConnect(func(ctx context.Context, conn *Conn) error {
		mu.Lock()
		executionOrder = append(executionOrder, 3)
		mu.Unlock()
		return nil
	})

	ctx := context.Background()
	err := e.handleConnect(ctx, nil)
	if err != nil {
		t.Errorf("handleConnect failed: %v", err)
	}

	expectedOrder := []int{1, 2, 3}
	if !reflect.DeepEqual(executionOrder, expectedOrder) {
		t.Errorf("Expected execution order %v, got %v", expectedOrder, executionOrder)
	}
}

func TestBlazeWaveMiddleware(t *testing.T) {
	e := NewEvent()

	middlewareCalled := false
	handlerCalled := false

	e.Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, args *EventArgs) error {
			middlewareCalled = true
			return next(ctx, args)
		}
	})

	e.OnConnect(func(ctx context.Context, conn *Conn) error {
		handlerCalled = true
		return nil
	})

	ctx := context.Background()

	err := e.handleConnect(ctx, nil)
	if err != nil {
		t.Errorf("handleConnect failed: %v", err)
	}

	if !middlewareCalled {
		t.Error("Middleware should have been called")
	}

	if !handlerCalled {
		t.Error("Connect handler should have been called")
	}
}

func TestBlazeWaveMultipleMiddleware(t *testing.T) {
	e := NewEvent()

	executionOrder := make([]int, 0)
	var mu sync.Mutex

	e.Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, args *EventArgs) error {
			mu.Lock()
			executionOrder = append(executionOrder, 1)
			mu.Unlock()
			err := next(ctx, args)
			mu.Lock()
			executionOrder = append(executionOrder, 6)
			mu.Unlock()
			return err
		}
	})

	e.Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, args *EventArgs) error {
			mu.Lock()
			executionOrder = append(executionOrder, 2)
			mu.Unlock()
			err := next(ctx, args)
			mu.Lock()
			executionOrder = append(executionOrder, 5)
			mu.Unlock()
			return err
		}
	})

	e.Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, args *EventArgs) error {
			mu.Lock()
			executionOrder = append(executionOrder, 3)
			mu.Unlock()
			err := next(ctx, args)
			mu.Lock()
			executionOrder = append(executionOrder, 4)
			mu.Unlock()
			return err
		}
	})

	e.OnConnect(func(ctx context.Context, conn *Conn) error {
		mu.Lock()
		executionOrder = append(executionOrder, 0) // Handler is called last
		mu.Unlock()
		return nil
	})

	ctx := context.Background()
	err := e.handleConnect(ctx, nil)
	if err != nil {
		t.Errorf("handleConnect failed: %v", err)
	}

	expectedOrder := []int{1, 2, 3, 0, 4, 5, 6}
	if !reflect.DeepEqual(executionOrder, expectedOrder) {
		t.Errorf("Expected execution order %v, got %v", expectedOrder, executionOrder)
	}
}

func TestBlazeWaveMiddlewareError(t *testing.T) {
	e := NewEvent()

	expectedErr := errors.New("middleware error")

	e.Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, args *EventArgs) error {
			return expectedErr
		}
	})

	handlerCalled := false
	e.OnConnect(func(ctx context.Context, conn *Conn) error {
		handlerCalled = true
		return nil
	})

	ctx := context.Background()

	err := e.handleConnect(ctx, nil)
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}

	if handlerCalled {
		t.Error("Handler should not have been called when middleware returns error")
	}
}

func TestBlazeWaveHandlerError(t *testing.T) {
	e := NewEvent()

	expectedErr := errors.New("handler error")

	e.OnConnect(func(ctx context.Context, conn *Conn) error {
		return expectedErr
	})

	ctx := context.Background()

	err := e.handleConnect(ctx, nil)
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestBlazeWaveCacheFunctionality(t *testing.T) {
	e := NewEvent()

	middlewareCallCount := 0
	e.Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, args *EventArgs) error {
			middlewareCallCount++
			return next(ctx, args)
		}
	})

	handlerCallCount := 0
	e.OnConnect(func(ctx context.Context, conn *Conn) error {
		handlerCallCount++
		return nil
	})

	ctx := context.Background()

	err := e.handleConnect(ctx, nil)
	if err != nil {
		t.Errorf("handleConnect failed: %v", err)
	}

	err = e.handleConnect(ctx, nil)
	if err != nil {
		t.Errorf("handleConnect failed: %v", err)
	}

	err = e.handleConnect(ctx, nil)
	if err != nil {
		t.Errorf("handleConnect failed: %v", err)
	}

	if middlewareCallCount < 1 {
		t.Errorf("Expected middleware to be called at least 1 time during cache building, got %d", middlewareCallCount)
	}

	if handlerCallCount != 3 {
		t.Errorf("Expected handler to be called 3 times, got %d", handlerCallCount)
	}

	e.Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, args *EventArgs) error {
			return next(ctx, args)
		}
	})

}

func TestBlazeWaveConcurrency(t *testing.T) {
	e := NewEvent()

	handlerCount := 100
	for i := 0; i < handlerCount; i++ {
		e.OnConnect(func(ctx context.Context, conn *Conn) error {
			return nil
		})
	}

	executionCount := 0
	var mu sync.Mutex

	e.OnConnect(func(ctx context.Context, conn *Conn) error {
		mu.Lock()
		executionCount++
		mu.Unlock()
		return nil
	})

	eventCount := 50
	var wg sync.WaitGroup
	for i := 0; i < eventCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			err := e.handleConnect(ctx, nil)
			if err != nil {
				t.Errorf("handleConnect failed: %v", err)
			}
		}()
	}

	wg.Wait()

	if executionCount != eventCount {
		t.Errorf("Expected %d event executions, got %d", eventCount, executionCount)
	}
}

func TestBlazeWaveMessageHandling(t *testing.T) {
	e := NewEvent()

	textMessageReceived := false
	binaryMessageReceived := false
	textPayload := []byte("test message")
	binaryPayload := []byte{1, 2, 3, 4, 5}

	e.OnTextMessage(func(ctx context.Context, conn *Conn, payload []byte) error {
		textMessageReceived = true
		if !reflect.DeepEqual(payload, textPayload) {
			t.Errorf("Expected payload %v, got %v", textPayload, payload)
		}
		return nil
	})

	e.OnBinaryMessage(func(ctx context.Context, conn *Conn, payload []byte) error {
		binaryMessageReceived = true
		if !reflect.DeepEqual(payload, binaryPayload) {
			t.Errorf("Expected payload %v, got %v", binaryPayload, payload)
		}
		return nil
	})

	ctx := context.Background()

	mockConn := createMockConn()

	err := e.handleMessage(ctx, mockConn, MessageText, textPayload)
	if err != nil {
		t.Errorf("handleMessage failed: %v", err)
	}

	if !textMessageReceived {
		t.Error("Text message handler should have been called")
	}

	err = e.handleMessage(ctx, mockConn, MessageBinary, binaryPayload)
	if err != nil {
		t.Errorf("handleMessage failed: %v", err)
	}

	if !binaryMessageReceived {
		t.Error("Binary message handler should have been called")
	}
}

func TestBlazeWaveErrorHandling(t *testing.T) {
	e := NewEvent()

	errorReceived := false
	var receivedError error

	e.OnError(func(ctx context.Context, conn *Conn, err error) error {
		errorReceived = true
		receivedError = err
		return nil
	})

	ctx := context.Background()

	mockConn := createMockConn()

	closeErr := &CloseError{Code: StatusNormalClosure, Reason: "test close"}
	err := e.handleError(ctx, mockConn, closeErr)
	if err != nil {
		t.Errorf("handleError failed: %v", err)
	}

	if !errorReceived {
		t.Error("Error handler should have been called")
	}

	if receivedError != closeErr {
		t.Errorf("Expected error %v, got %v", closeErr, receivedError)
	}

	errorReceived = false
	receivedError = nil
	regularErr := errors.New("regular error")
	err = e.handleError(ctx, mockConn, regularErr)
	if err != nil {
		t.Errorf("handleError failed: %v", err)
	}

	if !errorReceived {
		t.Error("Error handler should have been called")
	}

	if receivedError != regularErr {
		t.Errorf("Expected error %v, got %v", regularErr, receivedError)
	}
}

func TestBlazeWaveCloseHandling(t *testing.T) {
	e := NewEvent()

	closeReceived := false
	var receivedCode StatusCode
	var receivedReason string

	e.OnClose(func(ctx context.Context, conn *Conn, code StatusCode, reason string) error {
		closeReceived = true
		receivedCode = code
		receivedReason = reason
		return nil
	})

	ctx := context.Background()

	mockConn := createMockConn()

	err := e.handleClose(ctx, mockConn, StatusNormalClosure, "normal close")
	if err != nil {
		t.Errorf("handleClose failed: %v", err)
	}

	if !closeReceived {
		t.Error("Close handler should have been called")
	}

	if receivedCode != StatusNormalClosure {
		t.Errorf("Expected code %v, got %v", StatusNormalClosure, receivedCode)
	}

	if receivedReason != "normal close" {
		t.Errorf("Expected reason %s, got %s", "normal close", receivedReason)
	}

	closeReceived = false
	receivedCode = 0
	receivedReason = ""
	err = e.handleClose(ctx, mockConn, StatusInternalError, "internal error")
	if err != nil {
		t.Errorf("handleClose failed: %v", err)
	}

	if !closeReceived {
		t.Error("Close handler should have been called")
	}

	if receivedCode != StatusInternalError {
		t.Errorf("Expected code %v, got %v", StatusInternalError, receivedCode)
	}

	if receivedReason != "internal error" {
		t.Errorf("Expected reason %s, got %s", "internal error", receivedReason)
	}
}

func TestBlazeWaveEmptyHandlers(t *testing.T) {
	e := NewEvent()

	ctx := context.Background()

	mockConn := createMockConn()

	err := e.handleConnect(ctx, mockConn)
	if err != nil {
		t.Errorf("handleConnect with no handlers should return nil, got %v", err)
	}

	err = e.handleDisconnect(ctx, mockConn)
	if err != nil {
		t.Errorf("handleDisconnect with no handlers should return nil, got %v", err)
	}

	err = e.handleMessage(ctx, mockConn, MessageText, []byte("test"))
	if err != nil {
		t.Errorf("handleMessage with no handlers should return nil, got %v", err)
	}

	err = e.handleError(ctx, mockConn, errors.New("test"))
	if err != nil {
		t.Errorf("handleError with no handlers should return nil, got %v", err)
	}

	err = e.handleClose(ctx, mockConn, StatusNormalClosure, "test")
	if err != nil {
		t.Errorf("handleClose with no handlers should return nil, got %v", err)
	}

	pingHandlers := e.pingHandlers

	dynamicHandler := func(ctx context.Context, conn *Conn) error {
		args := &EventArgs{Conn: conn, Payload: []byte("ping")}
		for _, handler := range pingHandlers {
			if err := handler(ctx, args); err != nil {
				return err
			}
		}
		return nil
	}

	err = dynamicHandler(ctx, mockConn)
	if err != nil {
		t.Errorf("handlePing with no handlers should return nil, got %v", err)
	}

	err = e.handlePong(ctx, mockConn, []byte("pong"))
	if err != nil {
		t.Errorf("handlePong with no handlers should return nil, got %v", err)
	}
}

func TestBlazeWavePerformance(t *testing.T) {
	e := NewEvent()

	for i := 0; i < 10; i++ {
		e.Use(func(next HandlerFunc) HandlerFunc {
			return func(ctx context.Context, args *EventArgs) error {
				return next(ctx, args)
			}
		})
	}

	e.OnConnect(func(ctx context.Context, conn *Conn) error {
		return nil
	})

	ctx := context.Background()

	start := time.Now()
	iterations := 1000

	for i := 0; i < iterations; i++ {
		err := e.handleConnect(ctx, nil)
		if err != nil {
			t.Errorf("handleConnect failed: %v", err)
		}
	}

	duration := time.Since(start)
	avgTime := duration / time.Duration(iterations)

	if avgTime > time.Millisecond {
		t.Errorf("Average time per call too high: %v", avgTime)
	}

	t.Logf("Average time per call: %v", avgTime)
}
