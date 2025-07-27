//go:build !js

package blazewave

import (
	"context"
	"net/http"
	"time"

	"github.com/heyehang/blazewave/core/pool"
	"github.com/heyehang/blazewave/core/timer"
)

// Client manages WebSocket event handlers
type Client struct {
	// Client options
	opts *clientOption

	// Global Event
	*Event
}

// NewClient creates a new client
func NewClient(options ...ClientOptions) *Client {
	opts := &clientOption{
		DialOptions: &DialOptions{},
	}
	for _, option := range options {
		option(opts)
	}

	e := NewEvent()

	if opts.DialOptions == nil {
		opts.DialOptions = &DialOptions{}
	}

	opts.DialOptions.Event = e

	var c = &Client{
		opts:  opts,
		Event: e,
	}

	return c
}

type clientOption struct {
	*DialOptions
}

type ClientOptions func(*clientOption)

// WithClientHTTPClient sets the HTTP client for the event manager
func WithClientHTTPClient(httpClient *http.Client) ClientOptions {
	return func(o *clientOption) {
		o.HTTPClient = httpClient
	}
}

// WithClientHTTPHeader sets the HTTP header for the event manager
func WithClientHTTPHeader(httpHeader http.Header) ClientOptions {
	return func(o *clientOption) {
		o.HTTPHeader = httpHeader
	}
}

// WithClientHost sets the host for the event manager
func WithClientHost(host string) ClientOptions {
	return func(o *clientOption) {
		o.Host = host
	}
}

// WithClientSubprotocols sets the subprotocols for the event manager
func WithClientSubprotocols(subprotocols []string) ClientOptions {
	return func(o *clientOption) {
		o.Subprotocols = subprotocols
	}
}

// WithClientCompressionMode sets the compression mode for the event manager
func WithClientCompressionMode(compressionMode CompressionMode) ClientOptions {
	return func(o *clientOption) {
		o.CompressionMode = compressionMode
	}
}

// WithClientCompressionThreshold sets the compression threshold for the event manager
func WithClientCompressionThreshold(compressionThreshold int) ClientOptions {
	return func(o *clientOption) {
		o.CompressionThreshold = compressionThreshold
	}
}

// WithClientReaderPool sets the reader pool for the event manager
func WithClientReaderPool(readerPool *pool.Pool) ClientOptions {
	return func(o *clientOption) {
		o.ReaderPool = readerPool
	}
}

// WithClientWriterPool sets the writer pool for the event manager
func WithClientWriterPool(writerPool *pool.Pool) ClientOptions {
	return func(o *clientOption) {
		o.WriterPool = writerPool
	}
}

// WithClientHeartbeatInterval sets the heartbeat interval for the event manager
func WithClientHeartbeatInterval(heartbeatInterval time.Duration) ClientOptions {
	return func(o *clientOption) {
		o.HeartbeatInterval = heartbeatInterval
	}
}

// WithClientHeartbeatTimeout sets the heartbeat timeout for the event manager
func WithClientHeartbeatTimeout(heartbeatTimeout time.Duration) ClientOptions {
	return func(o *clientOption) {
		o.HeartbeatTimeout = heartbeatTimeout
	}
}

// WithClientHeartbeatTimer sets the heartbeat timer for the event manager
func WithClientHeartbeatTimer(heartbeatTimer *timer.Timer) ClientOptions {
	return func(o *clientOption) {
		o.HeartbeatTimer = heartbeatTimer
	}
}

// Dial accepts a WebSocket connection with event-driven capabilities
func (c *Client) Dial(ctx context.Context, url string) (*Conn, *http.Response, error) {
	conn, resp, err := Dial(ctx, url, c.opts.DialOptions)

	if err != nil {
		return nil, resp, err
	}

	return conn, resp, nil
}
