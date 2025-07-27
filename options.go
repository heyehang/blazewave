package blazewave

import (
	"time"

	"github.com/heyehang/blazewave/core/pool"
	"github.com/heyehang/blazewave/core/timer"
)

// CommonOptions contains common options for both client and server
type CommonOptions struct {
	// Subprotocols lists the WebSocket subprotocols to negotiate with the server.
	Subprotocols []string

	// CompressionMode controls the compression mode.
	// Defaults to CompressionDisabled.
	//
	// See docs on CompressionMode for details.
	CompressionMode CompressionMode

	// CompressionThreshold controls the minimum size of a message before compression is applied.
	//
	// Defaults to 512 bytes for CompressionNoContextTakeover and 128 bytes
	// for CompressionContextTakeover.
	CompressionThreshold int

	// ReaderPool is the reader of the ring pool.
	ReaderPool pool.BufferPool

	// WriterPool is the writer of the ring pool.
	WriterPool pool.BufferPool

	// HeartbeatInterval sets the interval for heartbeat ping messages.
	// If set to 0, heartbeat is disabled.
	// Default is 0 (disabled).
	HeartbeatInterval time.Duration

	// HeartbeatTimeout sets the timeout for heartbeat pong responses.
	// If no pong is received within this timeout, the connection is closed.
	// Default is 10 seconds.
	HeartbeatTimeout time.Duration

	// HeartbeatTimer is the timer pool for heartbeat management.
	// If not provided, a new timer will be created for each connection.
	HeartbeatTimer timer.TimerPool
}

// ServerOptions contains options for creating a server
type ServerOption struct {
	CommonOptions
}

type DialOption struct {
	CommonOptions
}
