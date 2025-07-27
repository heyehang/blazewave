//go:build !js

package blazewave

import (
	"context"
	"io"
	"net"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/heyehang/blazewave/core/pool"
	"github.com/heyehang/blazewave/core/timer"
	"github.com/heyehang/blazewave/internal/bufio"
)

// MessageType represents the type of a WebSocket message.
// See https://tools.ietf.org/html/rfc6455#section-5.6
type MessageType int

// MessageType constants.
const (
	// MessageText is for UTF-8 encoded text messages like JSON.
	MessageText MessageType = iota + 1
	// MessageBinary is for binary messages like protobufs.
	MessageBinary
)

// Conn represents a WebSocket connection.
// All methods may be called concurrently except for Reader and Read.
//
// You must always read from the connection. Otherwise control
// frames will not be handled. See Reader and CloseRead.
//
// Be sure to call Close on the connection when you
// are finished with it to release associated resources.
//
// On any error from any method, the connection is closed
// with an appropriate reason.
//
// This applies to context expirations as well unfortunately.
// See https://github.com/nhooyr/websocket/issues/242#issuecomment-633182220
type Conn struct {
	noCopy noCopy

	subprotocol    string
	rwc            io.ReadWriteCloser
	client         bool
	copts          *compressionOptions
	flateThreshold int

	brPool pool.BufferPool
	br     *bufio.Reader
	brBuf  *pool.Buffer

	bwPool pool.BufferPool
	bw     *bufio.Writer
	bwBuf  *pool.Buffer

	readTimeout     chan context.Context
	writeTimeout    chan context.Context
	timeoutLoopDone chan struct{}

	// Read state.
	readMu         *mu
	readHeaderBuf  [8]byte
	readControlBuf [maxControlPayload]byte
	msgReader      *msgReader

	// Write state.
	msgWriter      *msgWriter
	writeFrameMu   *mu
	writeBuf       []byte
	writeHeaderBuf [8]byte
	writeHeader    header

	// Close handshake state.
	closeStateMu     sync.RWMutex
	closeReceivedErr error
	closeSentErr     error

	// CloseRead state.
	closeReadMu   sync.Mutex
	closeReadCtx  context.Context
	closeReadDone chan struct{}

	closeMu    sync.Mutex // Protects following.
	closed     chan struct{}
	closeOnce  sync.Once
	closedFlag *atomic.Bool

	pingCounter   *atomic.Int64
	activePingsMu sync.Mutex
	activePings   map[string]chan<- struct{}

	// Event-driven capabilities
	event *Event

	localAddr  net.Addr
	remoteAddr net.Addr

	// Heartbeat detection
	heartbeatTimer     timer.TimerPool
	heartbeatTimerData *timer.TimerData
	heartbeatConfig    struct {
		interval time.Duration
		timeout  time.Duration
	}
	lastPongTime        *atomic.Int64 // Unix timestamp of last pong received
	heartbeatFailed     *atomic.Bool  // Whether heartbeat has failed
	heartbeatTimerOwned bool          // Whether this connection owns the timer (created locally)
}

type connConfig struct {
	subprotocol    string
	rwc            io.ReadWriteCloser
	client         bool
	copts          *compressionOptions
	flateThreshold int

	event *Event

	brPool pool.BufferPool
	br     *bufio.Reader
	brBuf  *pool.Buffer

	bwPool pool.BufferPool
	bw     *bufio.Writer
	bwBuf  *pool.Buffer

	localAddr  net.Addr
	remoteAddr net.Addr

	// Heartbeat configuration
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
	heartbeatTimer    timer.TimerPool
}

type safeRWC struct {
	rwc    io.ReadWriteCloser
	closed *atomic.Bool
}

func (s *safeRWC) Read(p []byte) (int, error) {
	if s.closed.Load() {
		return 0, ErrNetworkClosed
	}
	n, err := s.rwc.Read(p)
	if err == nil && s.closed.Load() {
		return 0, ErrNetworkClosed
	}
	return n, err
}

func (s *safeRWC) Write(p []byte) (int, error) {
	if s.closed.Load() {
		return 0, ErrNetworkClosed
	}
	n, err := s.rwc.Write(p)
	if err == nil && s.closed.Load() {
		return 0, ErrNetworkClosed
	}
	return n, err
}

func (s *safeRWC) Close() error {
	s.closed.Store(true)
	return s.rwc.Close()
}

func newConn(cfg connConfig) *Conn {
	var (
		closedFlag      = &atomic.Bool{}
		lastPongTime    = &atomic.Int64{}
		heartbeatFailed = &atomic.Bool{}
		pingCounter     = &atomic.Int64{}
	)
	c := &Conn{
		subprotocol:    cfg.subprotocol,
		rwc:            &safeRWC{rwc: cfg.rwc, closed: closedFlag},
		client:         cfg.client,
		copts:          cfg.copts,
		flateThreshold: cfg.flateThreshold,

		brPool: cfg.brPool,
		br:     cfg.br,
		brBuf:  cfg.brBuf,

		bwPool: cfg.bwPool,
		bw:     cfg.bw,
		bwBuf:  cfg.bwBuf,

		readTimeout:     make(chan context.Context),
		writeTimeout:    make(chan context.Context),
		timeoutLoopDone: make(chan struct{}),

		closed:      make(chan struct{}),
		activePings: make(map[string]chan<- struct{}),

		event: cfg.event,

		localAddr:  cfg.localAddr,
		remoteAddr: cfg.remoteAddr,

		heartbeatConfig: struct {
			interval time.Duration
			timeout  time.Duration
		}{
			interval: cfg.heartbeatInterval,
			timeout:  cfg.heartbeatTimeout,
		},
		heartbeatTimer:  cfg.heartbeatTimer,
		closedFlag:      closedFlag,
		lastPongTime:    lastPongTime,
		heartbeatFailed: heartbeatFailed,
		pingCounter:     pingCounter,
	}

	c.readMu = newMu(c)
	c.writeFrameMu = newMu(c)

	c.msgReader = newMsgReader(c)

	c.msgWriter = newMsgWriter(c)
	if c.client {
		c.writeBuf = extractBufioWriterBuf(c.bw, c.rwc)
	}

	if c.flate() && c.flateThreshold == 0 {
		c.flateThreshold = 128
		if !c.msgWriter.flateContextTakeover() {
			c.flateThreshold = 512
		}
	}

	runtime.SetFinalizer(c, func(c *Conn) {
		c.close()
	})

	go c.timeoutLoop()

	go c.startReadLoop()

	c.initHeartbeat()

	// If Event is set, trigger connect event
	if c.event != nil {
		ctx := context.Background()
		_ = c.event.handleConnect(ctx, c)
	}

	return c
}

func (c *Conn) startReadLoop() {
	if c.event != nil && !c.client {
		for {
			_, _, err := c.Reader(context.Background())
			if err != nil {
				return
			}
		}
	}
}

// Subprotocol returns the negotiated subprotocol.
// An empty string means the default protocol.
func (c *Conn) Subprotocol() string {
	return c.subprotocol
}

func (c *Conn) close() error {
	var err error
	c.closeOnce.Do(func() {
		c.closeMu.Lock()
		defer c.closeMu.Unlock()

		// Set closed flag first to ensure all I/O operations return net.ErrClosed
		if c.closedFlag != nil {
			c.closedFlag.Store(true)
		}

		runtime.SetFinalizer(c, nil)
		close(c.closed)

		// Stop heartbeat if enabled
		if c.heartbeatTimer != nil && c.heartbeatTimerData != nil {
			c.heartbeatTimer.Del(c.heartbeatTimerData)
			c.heartbeatTimerData = nil
		}

		// Stop locally owned timer
		if c.heartbeatTimerOwned && c.heartbeatTimer != nil {
			c.heartbeatTimer.Stop()
			c.heartbeatTimer = nil
			c.heartbeatTimerOwned = false
		}

		// If Event is set, trigger disconnect event
		if c.event != nil {
			ctx := context.Background()
			_ = c.event.handleDisconnect(ctx, c)
		}

		if c.brPool != nil && c.brBuf != nil {
			c.brPool.Put(c.brBuf)
			c.brPool = nil
			c.brBuf = nil
		}

		if c.bwPool != nil && c.bwBuf != nil {
			c.bwPool.Put(c.bwBuf)
			c.bwPool = nil
			c.bwBuf = nil
		}

		// With the close of rwc, these become safe to close.
		err = c.rwc.Close()
		c.msgWriter.close()
		c.msgReader.close()
	})
	return err
}

func (c *Conn) timeoutLoop() {
	defer close(c.timeoutLoopDone)

	readCtx := context.Background()
	writeCtx := context.Background()

	for {
		select {
		case <-c.closed:
			return

		case writeCtx = <-c.writeTimeout:
		case readCtx = <-c.readTimeout:

		case <-readCtx.Done():
			c.close()
			return
		case <-writeCtx.Done():
			c.close()
			return
		}
	}
}

// Ping sends a ping to the peer and waits for a pong.
// Use this to measure latency or ensure the peer is responsive.
// Ping must be called concurrently with Reader as it does
// not read from the connection but instead waits for a Reader call
// to read the pong.
//
// TCP Keepalives should suffice for most use cases.
func (c *Conn) Ping(ctx context.Context) error {
	p := c.pingCounter.Add(1)

	err := c.ping(ctx, strconv.FormatInt(p, 10))
	if err != nil {
		return WrapConnectionError(err, "ping")
	}
	return nil
}

func (c *Conn) ping(ctx context.Context, p string) error {
	pong := make(chan struct{}, 1)

	c.activePingsMu.Lock()
	c.activePings[p] = pong
	c.activePingsMu.Unlock()

	defer func() {
		c.activePingsMu.Lock()
		delete(c.activePings, p)
		c.activePingsMu.Unlock()
	}()

	err := c.writeControl(ctx, opPing, []byte(p))
	if err != nil {
		return err
	}

	select {
	case <-c.closed:
		return ErrNetworkClosed
	case <-ctx.Done():
		return WrapConnectionError(ctx.Err(), "wait for pong")
	case <-pong:
		return nil
	}
}

type mu struct {
	c  *Conn
	ch chan struct{}
}

func newMu(c *Conn) *mu {
	return &mu{
		c:  c,
		ch: make(chan struct{}, 1),
	}
}

func (m *mu) forceLock() {
	m.ch <- struct{}{}
}

func (m *mu) tryLock() bool {
	select {
	case m.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

func (m *mu) lock(ctx context.Context) error {
	select {
	case <-m.c.closed:
		return ErrNetworkClosed
	case <-ctx.Done():
		return WrapConnectionError(ctx.Err(), "acquire lock")
	case m.ch <- struct{}{}:
		// To make sure the connection is certainly alive.
		// As it's possible the send on m.ch was selected
		// over the receive on closed.
		select {
		case <-m.c.closed:
			// Make sure to release.
			m.unlock()
			return ErrNetworkClosed
		default:
		}
		return nil
	}
}

func (m *mu) unlock() {
	select {
	case <-m.ch:
	default:
	}
}

type noCopy struct{}

func (*noCopy) Lock() {}
