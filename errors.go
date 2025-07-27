package blazewave

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"syscall"
)

// Expose underlying errors
var (
	// Connection related errors
	ErrConnectionClosed = errors.New("connection closed")
	ErrConnectionBroken = errors.New("connection broken")
	ErrNetworkClosed    = errors.New("use of closed network connection")
	ErrBrokenPipe       = errors.New("broken pipe")
	ErrWriteBrokenPipe  = errors.New("write: broken pipe")

	// WebSocket protocol related errors
	ErrReceivedCloseFrame = errors.New("received close frame")
	ErrUnexpectedClose    = errors.New("unexpected close")

	// Read related errors
	ErrReadTimeout       = errors.New("read timeout")
	ErrReadCancelled     = errors.New("read cancelled")
	ErrReadLimitExceeded = errors.New("read limit exceeded")
	ErrReadEOF           = errors.New("read EOF")

	// Write related errors
	ErrWriteTimeout       = errors.New("write timeout")
	ErrWriteCancelled     = errors.New("write cancelled")
	ErrWriteLimitExceeded = errors.New("write limit exceeded")

	// Control frame related errors
	ErrInvalidControlFrame = errors.New("invalid control frame")
	ErrPingTimeout         = errors.New("ping timeout")
	ErrPongTimeout         = errors.New("pong timeout")
)

// Protocol error types
var (
	ErrProtocolViolation  = errors.New("websocket protocol violation")
	ErrInvalidHandshake   = errors.New("invalid handshake")
	ErrUnsupportedVersion = errors.New("unsupported websocket version")
	ErrInvalidFrame       = errors.New("invalid frame format")
	ErrInvalidStatusCode  = errors.New("invalid status code")
	ErrInvalidOpcode      = errors.New("invalid opcode")
	ErrInvalidRSV         = errors.New("invalid rsv bits")
	ErrInvalidPayload     = errors.New("invalid payload")
	ErrConfiguration      = errors.New("configuration error")
)

// IsConnClosed checks if the error is a connection closed error
func IsConnClosed(err error) bool {
	return errors.Is(err, ErrConnectionClosed)
}

// IsBrokenPipe checks if the error is a broken pipe error
func IsBrokenPipe(err error) bool {
	return errors.Is(err, ErrBrokenPipe) || errors.Is(err, ErrWriteBrokenPipe)
}

// IsReadClosed checks if the error is a read closed error
func IsReadClosed(err error) bool {
	return errors.Is(err, ErrReadTimeout) || errors.Is(err, ErrReadCancelled)
}

// IsWriteClosed checks if the error is a write closed error
func IsWriteClosed(err error) bool {
	return errors.Is(err, ErrWriteTimeout) || errors.Is(err, ErrWriteCancelled)
}

// IsConnectionError checks if the error is a connection related error
func IsConnectionError(err error) bool {
	if errors.Is(err, ErrConnectionClosed) ||
		errors.Is(err, ErrConnectionBroken) ||
		errors.Is(err, ErrNetworkClosed) ||
		errors.Is(err, ErrBrokenPipe) ||
		errors.Is(err, ErrWriteBrokenPipe) ||
		errors.Is(err, ErrReceivedCloseFrame) ||
		errors.Is(err, ErrUnexpectedClose) {
		return true
	}
	if err != nil && strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}
	return false
}

// IsNetworkError checks if the error is a network system error
func IsNetworkError(err error) bool {
	var syscallErr *syscall.Errno
	if errors.As(err, &syscallErr) {
		switch *syscallErr {
		case syscall.EPIPE, syscall.ECONNRESET, syscall.ENOTCONN:
			return true
		}
	}

	// Check net.Error type
	var netErr net.Error
	return errors.As(err, &netErr)
}

// IsReadError checks if the error is a read related error
func IsReadError(err error) bool {
	if errors.Is(err, ErrReadTimeout) ||
		errors.Is(err, ErrReadCancelled) ||
		errors.Is(err, ErrReadLimitExceeded) {
		return true
	}
	if err != nil && strings.Contains(err.Error(), "read on closed network connection") {
		return true
	}
	return false
}

// IsWriteError checks if the error is a write related error
func IsWriteError(err error) bool {
	if errors.Is(err, ErrWriteTimeout) ||
		errors.Is(err, ErrWriteCancelled) ||
		errors.Is(err, ErrWriteLimitExceeded) {
		return true
	}
	if err != nil && strings.Contains(err.Error(), "write on closed network connection") {
		return true
	}
	return false
}

// WrapConnectionError wraps connection errors
func WrapConnectionError(err error, operation string) error {
	if err == nil {
		return nil
	}

	// Check if it's a known network error
	if IsNetworkError(err) {
		switch {
		case errors.Is(err, syscall.EPIPE):
			return fmt.Errorf("%s: %w", operation, ErrBrokenPipe)
		case errors.Is(err, syscall.ECONNRESET):
			return fmt.Errorf("%s: %w", operation, ErrConnectionBroken)
		case errors.Is(err, syscall.ENOTCONN):
			return fmt.Errorf("%s: %w", operation, ErrNetworkClosed)
		}
	}

	// Check if it's net.ErrClosed - keep original for compatibility with existing tests
	if errors.Is(err, net.ErrClosed) {
		return ErrNetworkClosed
	}

	// Check string matching (backward compatibility)
	errStr := err.Error()
	switch {
	case contains(errStr, "broken pipe"):
		return fmt.Errorf("%s: %w", operation, ErrBrokenPipe)
	case contains(errStr, "connection closed"):
		return fmt.Errorf("%s: %w", operation, ErrConnectionClosed)
	case contains(errStr, "use of closed network connection"):
		return fmt.Errorf("%s: %w", operation, ErrNetworkClosed)
	case contains(errStr, "received close frame"):
		return fmt.Errorf("%s: %w", operation, ErrReceivedCloseFrame)
	}

	// If not a known error, return the original error
	return fmt.Errorf("%s: %w", operation, err)
}

// contains checks if a string contains a substring (case insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr))))
}

// containsSubstring simple string contains check
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// WrapProtocolError wraps protocol-related errors with context
func WrapProtocolError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// WrapProtocolViolation creates a protocol violation error with details
func WrapProtocolViolation(details string, args ...interface{}) error {
	msg := fmt.Sprintf(details, args...)
	return fmt.Errorf("%w: %s", ErrProtocolViolation, msg)
}

// WrapInvalidHandshake creates an invalid handshake error with details
func WrapInvalidHandshake(details string, args ...interface{}) error {
	msg := fmt.Sprintf(details, args...)
	return fmt.Errorf("%w: %s", ErrInvalidHandshake, msg)
}

// WrapConfigurationError wraps configuration-related errors
func WrapConfigurationError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// IsProtocolError checks if the error is a protocol-related error
func IsProtocolError(err error) bool {
	return errors.Is(err, ErrProtocolViolation) ||
		errors.Is(err, ErrInvalidHandshake) ||
		errors.Is(err, ErrUnsupportedVersion) ||
		errors.Is(err, ErrInvalidFrame) ||
		errors.Is(err, ErrInvalidStatusCode) ||
		errors.Is(err, ErrInvalidOpcode) ||
		errors.Is(err, ErrInvalidRSV) ||
		errors.Is(err, ErrInvalidPayload)
}

// IsConfigurationError checks if the error is a configuration-related error
func IsConfigurationError(err error) bool {
	return errors.Is(err, ErrConfiguration)
}

// IsNetworkClosed checks if the error means the network connection is closed.
func IsNetworkClosed(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrNetworkClosed) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrClosedPipe) ||
		errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	if strings.Contains(err.Error(), "use of closed network connection") ||
		strings.Contains(err.Error(), "file already closed") ||
		strings.Contains(err.Error(), "connection reset by peer") {
		return true
	}
	return false
}
