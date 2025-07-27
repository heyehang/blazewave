//go:build !js

package blazewave

import (
	"net/http"
	"time"

	"github.com/heyehang/blazewave/core/pool"
	"github.com/heyehang/blazewave/core/timer"
)

// Server manages WebSocket event handlers
type Server struct {
	// Server options
	opts *serverOption

	// Global Event
	*Event
}

// NewServer creates a new server
func NewServer(options ...ServerOptions) *Server {
	opts := &serverOption{
		AcceptOptions: &AcceptOptions{},
	}
	for _, option := range options {
		option(opts)
	}

	e := NewEvent()

	if opts.AcceptOptions == nil {
		opts.AcceptOptions = &AcceptOptions{}
	}

	opts.AcceptOptions.Event = e

	var s = &Server{
		opts:  opts,
		Event: e,
	}

	return s
}

type serverOption struct {
	*AcceptOptions
}

type ServerOptions func(*serverOption)

// WithServerSubprotocols sets the subprotocols for the event manager
func WithServerSubprotocols(subprotocols []string) ServerOptions {
	return func(o *serverOption) {
		o.Subprotocols = subprotocols
	}
}

// WithServerInsecureSkipVerify sets the insecure skip verify for the event manager
func WithServerInsecureSkipVerify(insecureSkipVerify bool) ServerOptions {
	return func(o *serverOption) {
		o.InsecureSkipVerify = insecureSkipVerify
	}
}

// WithServerCompressionMode sets the compression mode for the event manager
func WithServerCompressionMode(compressionMode CompressionMode) ServerOptions {
	return func(o *serverOption) {
		o.CompressionMode = compressionMode
	}
}

// WithServerCompressionThreshold sets the compression threshold for the event manager
func WithServerCompressionThreshold(compressionThreshold int) ServerOptions {
	return func(o *serverOption) {
		o.CompressionThreshold = compressionThreshold
	}
}

// WithServerOriginPatterns sets the origin patterns for the event manager
func WithServerOriginPatterns(originPatterns []string) ServerOptions {
	return func(o *serverOption) {
		o.OriginPatterns = originPatterns
	}
}

// WithServerReaderPool sets the reader pool for the event manager
func WithServerReaderPool(readerPool *pool.Pool) ServerOptions {
	return func(o *serverOption) {
		o.ReaderPool = readerPool
	}
}

// WithServerWriterPool sets the writer pool for the event manager
func WithServerWriterPool(writerPool *pool.Pool) ServerOptions {
	return func(o *serverOption) {
		o.WriterPool = writerPool
	}
}

// WithServerHeartbeatInterval sets the heartbeat interval for the event manager
func WithServerHeartbeatInterval(heartbeatInterval time.Duration) ServerOptions {
	return func(o *serverOption) {
		o.HeartbeatInterval = heartbeatInterval
	}
}

// WithServerHeartbeatTimeout sets the heartbeat timeout for the event manager
func WithServerHeartbeatTimeout(heartbeatTimeout time.Duration) ServerOptions {
	return func(o *serverOption) {
		o.HeartbeatTimeout = heartbeatTimeout
	}
}

// WithServerHeartbeatTimer sets the heartbeat timer for the event manager
func WithServerHeartbeatTimer(heartbeatTimer *timer.Timer) ServerOptions {
	return func(o *serverOption) {
		o.HeartbeatTimer = heartbeatTimer
	}
}

// Accept accepts a WebSocket connection with event-driven capabilities
func (s *Server) Accept(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	conn, err := Accept(w, r, s.opts.AcceptOptions)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
