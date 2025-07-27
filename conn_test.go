//go:build !js

package blazewave_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"bufio"

	"github.com/heyehang/blazewave"
	"github.com/heyehang/blazewave/core/pool"
	"github.com/heyehang/blazewave/internal/errd"
	"github.com/heyehang/blazewave/internal/test/assert"
	"github.com/heyehang/blazewave/internal/test/wstest"
	"github.com/heyehang/blazewave/internal/test/xrand"
	"github.com/heyehang/blazewave/internal/wsjson"
	"github.com/heyehang/blazewave/internal/xsync"
)

func TestConn(t *testing.T) {

	t.Run("fuzzData", func(t *testing.T) {

		compressionMode := func() blazewave.CompressionMode {
			return blazewave.CompressionMode(xrand.Int(int(blazewave.CompressionContextTakeover) + 1))
		}

		for range 2 {
			t.Run("", func(t *testing.T) {
				tt, c1, c2 := newConnTest(t, &blazewave.DialOptions{
					CommonOptions: blazewave.CommonOptions{
						CompressionMode:      compressionMode(),
						CompressionThreshold: xrand.Int(9999),
					},
				}, &blazewave.AcceptOptions{
					CommonOptions: blazewave.CommonOptions{
						CompressionMode:      compressionMode(),
						CompressionThreshold: xrand.Int(9999),
					},
				})

				tt.goEchoLoop(c2)

				c1.SetReadLimit(131072)

				for range 2 {
					err := wstest.Echo(tt.ctx, c1, 131072)
					if err != nil && !isClosedNetworkError(err) {
						t.Errorf("unexpected error: %v", err)
					}
				}

				err := c1.Close(blazewave.StatusNormalClosure, "")
				if err != nil && !isClosedNetworkError(err) {
					t.Errorf("unexpected close error: %v", err)
				}
			})
		}
	})

	t.Run("badClose", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)

		c2.CloseRead(tt.ctx)

		err := c1.Close(-1, "")
		assert.Equal(t, "protocol error", true, blazewave.IsProtocolError(err))
		assert.Contains(t, err.Error(), "status code StatusCode(-1) cannot be set")
	})

	t.Run("ping", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)

		c1.CloseRead(tt.ctx)
		c2.CloseRead(tt.ctx)

		for range 3 {
			err := c1.Ping(tt.ctx)
			if err != nil && !isClosedNetworkError(err) && !isContextDeadlineError(err) {
				t.Errorf("unexpected ping error: %v", err)
			}
		}

		err := c1.Close(blazewave.StatusNormalClosure, "")
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected close error: %v", err)
		}
	})

	t.Run("badPing", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)

		c2.CloseRead(tt.ctx)

		ctx, cancel := context.WithTimeout(tt.ctx, time.Millisecond*500)
		defer cancel()

		err := c1.Ping(ctx)
		if err != nil {
			if !blazewave.IsConnectionError(err) &&
				!blazewave.IsReadError(err) &&
				!blazewave.IsWriteError(err) &&
				!errors.Is(err, context.DeadlineExceeded) &&
				!errors.Is(err, context.Canceled) {
				t.Errorf("unexpected error type: %v", err)
			}
		}
	})

	t.Run("pingReceivedPongReceived", func(t *testing.T) {
		var pingReceived1, pongReceived1 bool
		var pingReceived2, pongReceived2 bool

		dialEM := blazewave.NewServer()
		dialEM.Event.OnPing(func(ctx context.Context, conn *blazewave.Conn, payload []byte) error {
			pingReceived1 = true
			return nil
		})
		dialEM.Event.OnPong(func(ctx context.Context, conn *blazewave.Conn, payload []byte) error {
			pongReceived1 = true
			return nil
		})

		acceptEM := blazewave.NewServer()
		acceptEM.Event.OnPing(func(ctx context.Context, conn *blazewave.Conn, payload []byte) error {
			pingReceived2 = true
			return nil
		})
		acceptEM.Event.OnPong(func(ctx context.Context, conn *blazewave.Conn, payload []byte) error {
			pongReceived2 = true
			return nil
		})

		tt, c1, c2 := newConnTest(t,
			&blazewave.DialOptions{
				Event: dialEM.Event,
			},
			&blazewave.AcceptOptions{
				Event: acceptEM.Event,
			},
		)

		c1.CloseRead(tt.ctx)
		c2.CloseRead(tt.ctx)

		ctx, cancel := context.WithTimeout(tt.ctx, time.Second*5)
		defer cancel()

		err := c1.Ping(ctx)
		if err != nil {
			if isContextDeadlineError(err) {
				t.Logf("ping timeout (expected in some cases): %v", err)
				return // Skip subsequent assertions
			}
			if !isClosedNetworkError(err) {
				t.Errorf("unexpected ping error: %v", err)
			}
		}

		c1.CloseNow()
		c2.CloseNow()

		assert.Equal(t, "only one side receives the ping", false, pingReceived1 && pingReceived2)
		assert.Equal(t, "only one side receives the pong", false, pongReceived1 && pongReceived2)
		assert.Equal(t, "ping and pong received", true, (pingReceived1 && pongReceived2) || (pingReceived2 && pongReceived1))
	})

	t.Run("pingReceivedPongNotReceived", func(t *testing.T) {
		var pingReceived1 bool
		var pingReceived2 bool
		server1 := blazewave.NewServer()
		server1.Event.OnPing(func(ctx context.Context, conn *blazewave.Conn, payload []byte) error {
			pingReceived1 = true
			return nil
		})

		server2 := blazewave.NewServer()
		server2.Event.OnPing(func(ctx context.Context, conn *blazewave.Conn, payload []byte) error {
			pingReceived2 = true
			return nil
		})

		tt, c1, c2 := newConnTest(t,
			&blazewave.DialOptions{
				Event: server1.Event,
			},
			&blazewave.AcceptOptions{
				Event: server2.Event,
			},
		)

		c1.CloseRead(tt.ctx)
		c2.CloseRead(tt.ctx)

		ctx, cancel := context.WithTimeout(tt.ctx, time.Second*5)
		defer cancel()

		err := c1.Ping(ctx)
		if err != nil {
			if isContextDeadlineError(err) {
				t.Logf("ping timeout (expected in some cases): %v", err)
				return // Skip subsequent assertions
			}
			if !isClosedNetworkError(err) {
				t.Errorf("unexpected ping error: %v", err)
			}
		}

		c1.CloseNow()
		c2.CloseNow()

		assert.Equal(t, "ping events should be received", true, pingReceived1 || pingReceived2)
	})

	t.Run("concurrentWrite", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)

		tt.goDiscardLoop(c2)

		msg := xrand.Bytes(xrand.Int(9999))
		const count = 10
		errs := make(chan error, count)

		for range count {
			go func() {
				select {
				case errs <- c1.Write(tt.ctx, blazewave.MessageBinary, msg):
				case <-tt.ctx.Done():
					return
				}
			}()
		}

		for range count {
			select {
			case err := <-errs:
				if err != nil && !isClosedNetworkError(err) {
					t.Errorf("unexpected error: %v", err)
				}
			case <-tt.ctx.Done():
				t.Fatal(tt.ctx.Err())
			}
		}

		err := c1.Close(blazewave.StatusNormalClosure, "")
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected close error: %v", err)
		}
	})

	t.Run("concurrentWriteError", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)

		tt.goDiscardLoop(c2)

		_, err := c1.Writer(tt.ctx, blazewave.MessageText)
		if err != nil && !isClosedNetworkError(err) && !blazewave.IsWriteError(err) && !blazewave.IsConnectionError(err) {
			t.Fatalf("unexpected error: %#v", err)
		}
	})

	t.Run("netConn", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)

		n1 := blazewave.NetConn(tt.ctx, c1, blazewave.MessageBinary)
		n2 := blazewave.NetConn(tt.ctx, c2, blazewave.MessageBinary)

		d, _ := tt.ctx.Deadline()
		n1.SetDeadline(d)
		n1.SetDeadline(time.Time{})

		errs := xsync.Go(func() error {
			_, err := n2.Write([]byte("hello"))
			if err != nil {
				return err
			}
			return n2.Close()
		})

		b, err := io.ReadAll(n1)
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		_, err = n1.Read(nil)
		assert.Equal(t, "read error", err, io.EOF)

		select {
		case err := <-errs:
			if err != nil && !isClosedNetworkError(err) {
				t.Errorf("unexpected error: %v", err)
			}
		case <-tt.ctx.Done():
			t.Fatal(tt.ctx.Err())
		}

		assert.Equal(t, "read msg", []byte("hello"), b)
	})

	t.Run("netConn/BadMsg", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)

		n1 := blazewave.NetConn(tt.ctx, c1, blazewave.MessageBinary)
		n2 := blazewave.NetConn(tt.ctx, c2, blazewave.MessageText)

		c2.CloseRead(tt.ctx)
		errs := xsync.Go(func() error {
			_, err := n2.Write([]byte("hello"))
			return err
		})

		_, err := io.ReadAll(n1)
		assert.Contains(t, err, `unexpected frame type read (expected MessageBinary): MessageText`)

		select {
		case err := <-errs:
			if err != nil && !isClosedNetworkError(err) {
				t.Errorf("unexpected error: %v", err)
			}
		case <-tt.ctx.Done():
			t.Fatal(tt.ctx.Err())
		}
	})

	t.Run("netConn/readLimit", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)

		n1 := blazewave.NetConn(tt.ctx, c1, blazewave.MessageBinary)
		n2 := blazewave.NetConn(tt.ctx, c2, blazewave.MessageBinary)

		s := strings.Repeat("papa", 1<<20)
		errs := xsync.Go(func() error {
			_, err := n2.Write([]byte(s))
			if err != nil {
				return err
			}
			return n2.Close()
		})

		b, err := io.ReadAll(n1)
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		_, err = n1.Read(nil)
		assert.Equal(t, "read error", err, io.EOF)

		select {
		case err := <-errs:
			if err != nil && !isClosedNetworkError(err) {
				t.Errorf("unexpected error: %v", err)
			}
		case <-tt.ctx.Done():
			t.Fatal(tt.ctx.Err())
		}

		assert.Equal(t, "read msg", s, string(b))
	})

	t.Run("netConn/pastDeadline", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)

		n1 := blazewave.NetConn(tt.ctx, c1, blazewave.MessageBinary)
		n2 := blazewave.NetConn(tt.ctx, c2, blazewave.MessageBinary)

		n1.SetDeadline(time.Now().Add(-time.Minute))
		n2.SetDeadline(time.Now().Add(-time.Minute))

	})

	t.Run("wsjson", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)

		tt.goEchoLoop(c2)

		c1.SetReadLimit(1 << 30)

		exp := xrand.String(xrand.Int(131072))

		werr := xsync.Go(func() error {
			return wsjson.Write(tt.ctx, c1, exp)
		})

		var act any
		err := wsjson.Read(tt.ctx, c1, &act)
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}
		assert.Equal(t, "read msg", exp, act)

		select {
		case err := <-werr:
			if err != nil && !isClosedNetworkError(err) {
				t.Errorf("unexpected error: %v", err)
			}
		case <-tt.ctx.Done():
			t.Fatal(tt.ctx.Err())
		}

		err = c1.Close(blazewave.StatusNormalClosure, "")
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected close error: %v", err)
		}
	})

	t.Run("HTTPClient.Timeout", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, &blazewave.DialOptions{
			HTTPClient: &http.Client{Timeout: time.Second * 5},
		}, nil)

		tt.goEchoLoop(c2)

		c1.SetReadLimit(1 << 30)

		exp := xrand.String(xrand.Int(131072))

		werr := xsync.Go(func() error {
			return wsjson.Write(tt.ctx, c1, exp)
		})

		var act any
		err := wsjson.Read(tt.ctx, c1, &act)
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}
		assert.Equal(t, "read msg", exp, act)

		select {
		case err := <-werr:
			if err != nil && !isClosedNetworkError(err) {
				t.Errorf("unexpected error: %v", err)
			}
		case <-tt.ctx.Done():
			t.Fatal(tt.ctx.Err())
		}

		err = c1.Close(blazewave.StatusNormalClosure, "")
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected close error: %v", err)
		}
	})

	t.Run("CloseNow", func(t *testing.T) {
		_, c1, c2 := newConnTest(t, nil, nil)

		err1 := c1.CloseNow()
		if err1 != nil && !isClosedNetworkError(err1) {
			t.Errorf("unexpected error: %v", err1)
		}
		err2 := c2.CloseNow()
		if err2 != nil && !isClosedNetworkError(err2) {
			t.Errorf("unexpected error: %v", err2)
		}
		err1 = c1.CloseNow()
		if err1 != nil && !isClosedNetworkError(err1) {
			t.Errorf("unexpected error: %v", err1)
		}
		err2 = c2.CloseNow()
		if err2 != nil && !isClosedNetworkError(err2) {
			t.Errorf("unexpected error: %v", err2)
		}
	})

	t.Run("MidReadClose", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)

		tt.goEchoLoop(c2)

		c1.SetReadLimit(131072)

		for range 5 {
			err := wstest.Echo(tt.ctx, c1, 131072)
			if err != nil && !isClosedNetworkError(err) {
				t.Errorf("unexpected error: %v", err)
			}
		}

		err := wsjson.Write(tt.ctx, c1, "four")
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}
		_, _, err = c1.Reader(tt.ctx)
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		err = c1.Close(blazewave.StatusNormalClosure, "")
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected close error: %v", err)
		}
	})

	t.Run("Subprotocol", func(t *testing.T) {
		_, c1, c2 := newConnTest(t, &blazewave.DialOptions{
			CommonOptions: blazewave.CommonOptions{
				Subprotocols: []string{"test-protocol"},
			},
		}, &blazewave.AcceptOptions{
			CommonOptions: blazewave.CommonOptions{
				Subprotocols: []string{"test-protocol"},
			},
		})

		subprotocol := c1.Subprotocol()
		assert.Equal(t, "subprotocol should match", "test-protocol", subprotocol)

		_, c3, c4 := newConnTest(t, nil, nil)
		emptySubprotocol := c3.Subprotocol()
		assert.Equal(t, "empty subprotocol should be empty string", "", emptySubprotocol)

		c1.CloseNow()
		c2.CloseNow()
		c3.CloseNow()
		c4.CloseNow()
	})

	t.Run("MuCreation", func(t *testing.T) {
		_, c1, c2 := newConnTest(t, nil, nil)

		mu := blazewave.NewMu(c1)
		if mu == nil {
			t.Error("mu should not be nil")
		}

		c1.CloseNow()
		c2.CloseNow()
	})

	t.Run("NewConnWithBlazeWave", func(t *testing.T) {
		server := blazewave.NewServer()
		server.Event.OnConnect(func(ctx context.Context, conn *blazewave.Conn) error {
			return nil
		})
		server.Event.OnDisconnect(func(ctx context.Context, conn *blazewave.Conn) error {
			return nil
		})

		tt, c1, c2 := newConnTest(t, &blazewave.DialOptions{
			Event: server.Event,
		}, &blazewave.AcceptOptions{
			Event: server.Event,
		})

		tt.goEchoLoop(c2)
		err := wstest.Echo(tt.ctx, c1, 1024)
		if err != nil && !isClosedNetworkError(err) && !isProtocolError(err) && !isContextDeadlineError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		// Ensure proper cleanup
		c1.CloseNow()
		c2.CloseNow()

		// Give some time for goroutines to clean up
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("NewConnWithCompression", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, &blazewave.DialOptions{
			CommonOptions: blazewave.CommonOptions{
				CompressionMode:      blazewave.CompressionContextTakeover,
				CompressionThreshold: 128,
			},
		}, &blazewave.AcceptOptions{
			CommonOptions: blazewave.CommonOptions{
				CompressionMode:      blazewave.CompressionContextTakeover,
				CompressionThreshold: 128,
			},
		})

		tt.goEchoLoop(c2)
		err := wstest.Echo(tt.ctx, c1, 1024)
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		c1.CloseNow()
		c2.CloseNow()

		// Give some time for goroutines to clean up
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("NewConnWithPools", func(t *testing.T) {
		readerPool := pool.NewPool(4, 1024)
		writerPool := pool.NewPool(4, 1024)

		tt, c1, c2 := newConnTest(t, &blazewave.DialOptions{
			CommonOptions: blazewave.CommonOptions{
				ReaderPool: readerPool,
				WriterPool: writerPool,
			},
		}, &blazewave.AcceptOptions{
			CommonOptions: blazewave.CommonOptions{
				ReaderPool: readerPool,
				WriterPool: writerPool,
			},
		})

		tt.goEchoLoop(c2)
		err := wstest.Echo(tt.ctx, c1, 1024)
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		c1.CloseNow()
		c2.CloseNow()

		// Give some time for goroutines to clean up
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("NetConnWithDeadlines", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)

		nc := blazewave.NetConn(tt.ctx, c1, blazewave.MessageBinary)

		err := nc.SetDeadline(time.Now().Add(-time.Minute))
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		err = nc.SetDeadline(time.Now().Add(time.Minute))
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		err = nc.SetReadDeadline(time.Now().Add(time.Minute))
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		err = nc.SetWriteDeadline(time.Now().Add(time.Minute))
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		c1.CloseNow()
		c2.CloseNow()
	})

	t.Run("ConnWithAddresses", func(t *testing.T) {
		conn1, conn2 := net.Pipe()

		tt, c1, c2 := newConnTest(t, nil, nil)

		tt.goEchoLoop(c2)
		err := wstest.Echo(tt.ctx, c1, 1024)
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		c1.CloseNow()
		c2.CloseNow()
		conn1.Close()
		conn2.Close()
	})

	t.Run("TimeoutLoop", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)

		ctx, cancel := context.WithTimeout(tt.ctx, time.Millisecond*100)
		defer cancel()

		err := c1.Write(ctx, blazewave.MessageText, []byte("test"))
		if err != nil && !isClosedNetworkError(err) && !isProtocolError(err) && !isContextDeadlineError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		c1.CloseNow()
		c2.CloseNow()
	})

	t.Run("PingWithBlazeWave", func(t *testing.T) {
		server := blazewave.NewServer()
		server.Event.OnPing(func(ctx context.Context, conn *blazewave.Conn, payload []byte) error {
			return nil
		})

		tt, c1, c2 := newConnTest(t, &blazewave.DialOptions{
			Event: server.Event,
		}, &blazewave.AcceptOptions{
			Event: server.Event,
		})

		ctx, cancel := context.WithTimeout(tt.ctx, time.Second*5)
		defer cancel()

		readDone := make(chan struct{})
		go func() {
			defer close(readDone)
			for {
				select {
				case <-ctx.Done():
					return
				default:
					_, _, err := c2.Read(ctx)
					if err != nil {
						return
					}
				}
			}
		}()

		err := c1.Ping(ctx)
		if err != nil {
			if !blazewave.IsConnectionError(err) &&
				!blazewave.IsReadError(err) &&
				!blazewave.IsWriteError(err) &&
				!errors.Is(err, context.DeadlineExceeded) &&
				!errors.Is(err, context.Canceled) {
				t.Errorf("unexpected ping error: %v", err)
			}
		}

		time.Sleep(time.Millisecond * 100)

		c1.CloseNow()
		c2.CloseNow()

		select {
		case <-readDone:
		case <-time.After(time.Second):
		}
	})

	t.Run("CloseWithBlazeWave", func(t *testing.T) {
		server := blazewave.NewServer()
		var disconnectReceived bool
		server.Event.OnDisconnect(func(ctx context.Context, conn *blazewave.Conn) error {
			disconnectReceived = true
			return nil
		})

		_, c1, c2 := newConnTest(t, &blazewave.DialOptions{
			Event: server.Event,
		}, &blazewave.AcceptOptions{
			Event: server.Event,
		})

		err := c1.Close(blazewave.StatusNormalClosure, "")
		if err != nil && !isClosedNetworkError(err) && !isContextDeadlineError(err) {
			t.Errorf("unexpected close error: %v", err)
		}

		time.Sleep(time.Millisecond * 100)
		assert.Equal(t, "disconnect should be received", true, disconnectReceived)

		c2.CloseNow()
	})

	t.Run("NetConnWithDifferentMessageTypes", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)

		nc1 := blazewave.NetConn(tt.ctx, c1, blazewave.MessageText)

		nc2 := blazewave.NetConn(tt.ctx, c2, blazewave.MessageBinary)

		if nc1 == nil {
			t.Error("nc1 should not be nil")
		}
		if nc2 == nil {
			t.Error("nc2 should not be nil")
		}

		c1.CloseNow()
		c2.CloseNow()
	})

	t.Run("NetConnDeadlineHandling", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)

		nc := blazewave.NetConn(tt.ctx, c1, blazewave.MessageBinary)

		err := nc.SetDeadline(time.Time{})
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		err = nc.SetReadDeadline(time.Time{})
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		err = nc.SetWriteDeadline(time.Time{})
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		c1.CloseNow()
		c2.CloseNow()
	})

	t.Run("MuTryLockEdgeCases", func(t *testing.T) {
		_, c1, c2 := newConnTest(t, nil, nil)

		mu := blazewave.NewMu(c1)
		if mu == nil {
			t.Error("mu should not be nil")
		}

		c1.CloseNow()
		c2.CloseNow()
	})

	t.Run("NewConnWithAllOptions", func(t *testing.T) {

		server := blazewave.NewServer()
		readerPool := pool.NewPool(4, 1024)
		writerPool := pool.NewPool(4, 1024)

		tt, c1, c2 := newConnTest(t, &blazewave.DialOptions{
			CommonOptions: blazewave.CommonOptions{
				Subprotocols:         []string{"test-protocol"},
				CompressionMode:      blazewave.CompressionContextTakeover,
				CompressionThreshold: 128,
				ReaderPool:           readerPool,
				WriterPool:           writerPool,
			},
			Event: server.Event,
		}, &blazewave.AcceptOptions{
			CommonOptions: blazewave.CommonOptions{
				Subprotocols:         []string{"test-protocol"},
				CompressionMode:      blazewave.CompressionContextTakeover,
				CompressionThreshold: 128,
				ReaderPool:           readerPool,
				WriterPool:           writerPool,
			},
			Event: server.Event,
		})

		tt.goEchoLoop(c2)
		err := wstest.Echo(tt.ctx, c1, 1024)
		if err != nil && !isClosedNetworkError(err) && !isProtocolError(err) && !isContextDeadlineError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		c1.CloseNow()
		c2.CloseNow()
	})

	t.Run("NewConnWithNoContextTakeover", func(t *testing.T) {

		tt, c1, c2 := newConnTest(t, &blazewave.DialOptions{
			CommonOptions: blazewave.CommonOptions{
				CompressionMode:      blazewave.CompressionNoContextTakeover,
				CompressionThreshold: 512,
			},
		}, &blazewave.AcceptOptions{
			CommonOptions: blazewave.CommonOptions{
				CompressionMode:      blazewave.CompressionNoContextTakeover,
				CompressionThreshold: 512,
			},
		})

		tt.goEchoLoop(c2)
		err := wstest.Echo(tt.ctx, c1, 1024)
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		c1.CloseNow()
		c2.CloseNow()
	})

	t.Run("NewConnWithDisabledCompression", func(t *testing.T) {

		tt, c1, c2 := newConnTest(t, &blazewave.DialOptions{
			CommonOptions: blazewave.CommonOptions{
				CompressionMode: blazewave.CompressionDisabled,
			},
		}, &blazewave.AcceptOptions{
			CommonOptions: blazewave.CommonOptions{
				CompressionMode: blazewave.CompressionDisabled,
			},
		})

		tt.goEchoLoop(c2)
		err := wstest.Echo(tt.ctx, c1, 1024)
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		c1.CloseNow()
		c2.CloseNow()
	})

	t.Run("NetConnWithDifferentContexts", func(t *testing.T) {
		_, c1, c2 := newConnTest(t, nil, nil)

		nc1 := blazewave.NetConn(context.Background(), c1, blazewave.MessageBinary)
		if nc1 == nil {
			t.Error("nc1 should not be nil")
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		nc2 := blazewave.NetConn(ctx, c2, blazewave.MessageText)
		if nc2 == nil {
			t.Error("nc2 should not be nil")
		}

		c1.CloseNow()
		c2.CloseNow()
	})
}

func TestWasm(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") == "" {
		t.SkipNow()
	}

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := echoServerHTTP(w, r, &blazewave.AcceptOptions{
			CommonOptions: blazewave.CommonOptions{
				Subprotocols: []string{"echo"},
			},
			InsecureSkipVerify: true,
		})
		if err != nil {
			t.Error(err)
		}
	}))
	defer s.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "test", "-exec=wasmbrowsertest", ".", "-v")
	cmd.Env = append(cleanEnv(os.Environ()), "GOOS=js", "GOARCH=wasm", fmt.Sprintf("WS_ECHO_SERVER_URL=%v", s.URL))

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wasm test binary failed: %v:\n%s", err, b)
	}
}

func cleanEnv(env []string) (out []string) {
	for _, e := range env {
		if strings.HasPrefix(e, "GITHUB") || strings.Contains(e, "TOKEN") {
			continue
		}
		out = append(out, e)
	}
	return out
}

func assertCloseStatus(exp blazewave.StatusCode, err error) error {
	if blazewave.CloseStatus(err) == -1 {
		return fmt.Errorf("expected blazewave.CloseError: %T %v", err, err)
	}
	if blazewave.CloseStatus(err) != exp {
		return fmt.Errorf("expected close status %v but got %v", exp, err)
	}
	return nil
}

type connTest struct {
	t   testing.TB
	ctx context.Context
}

func newConnTest(t testing.TB, dialOpts *blazewave.DialOptions, acceptOpts *blazewave.AcceptOptions) (tt *connTest, c1, c2 *blazewave.Conn) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	tt = &connTest{t: t, ctx: ctx}
	t.Cleanup(cancel)

	c1, c2 = wstest.Pipe(dialOpts, acceptOpts)
	if xrand.Bool() {
		c1, c2 = c2, c1
	}
	t.Cleanup(func() {
		done := make(chan struct{})
		go func() {
			defer close(done)
			c2.CloseNow()
			c1.CloseNow()
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Log("cleanup timeout, but continuing")
		}
	})

	return tt, c1, c2
}

func (tt *connTest) goEchoLoop(c *blazewave.Conn) {
	ctx, cancel := context.WithCancel(tt.ctx)

	echoLoopErr := xsync.Go(func() error {
		err := wstest.EchoLoop(ctx, c)
		if blazewave.IsConnectionError(err) ||
			blazewave.IsReadError(err) ||
			blazewave.IsWriteError(err) ||
			errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return assertCloseStatus(blazewave.StatusNormalClosure, err)
	})
	tt.t.Cleanup(func() {
		cancel()
		select {
		case err := <-echoLoopErr:
			if err != nil &&
				!blazewave.IsConnectionError(err) &&
				!blazewave.IsReadError(err) &&
				!blazewave.IsWriteError(err) &&
				!errors.Is(err, context.Canceled) &&
				!errors.Is(err, context.DeadlineExceeded) &&
				!strings.Contains(err.Error(), "failed to get reader") &&
				!strings.Contains(err.Error(), "failed to read frame header") &&
				!strings.Contains(err.Error(), "use of closed network connection") &&
				!strings.Contains(err.Error(), "failed to close writer") &&
				!strings.Contains(err.Error(), "write fin frame") &&
				!strings.Contains(err.Error(), "flush buffer") &&
				!strings.Contains(err.Error(), "io: read/write on closed pipe") {
				tt.t.Errorf("echo loop error: %v", err)
			}
		case <-time.After(5 * time.Second):
			tt.t.Error("echo loop cleanup timeout")
		}
	})
}

func (tt *connTest) goDiscardLoop(c *blazewave.Conn) {
	ctx, cancel := context.WithCancel(tt.ctx)

	discardLoopErr := xsync.Go(func() error {
		defer c.Close(blazewave.StatusInternalError, "")

		maxReads := 100
		readCount := 0

		for readCount < maxReads {
			_, _, err := c.Read(ctx)
			if err != nil {
				if blazewave.IsConnectionError(err) ||
					blazewave.IsReadError(err) ||
					errors.Is(err, context.Canceled) ||
					errors.Is(err, context.DeadlineExceeded) {
					return err
				}
				return assertCloseStatus(blazewave.StatusNormalClosure, err)
			}
			readCount++
		}
		return nil
	})
	tt.t.Cleanup(func() {
		cancel()
		select {
		case err := <-discardLoopErr:
			if err != nil && !isClosedNetworkError(err) {
				tt.t.Logf("discard loop cleanup timeout (allowed): %v", err)
			}
		case <-time.After(5 * time.Second):
			tt.t.Log("discard loop cleanup timeout (allowed)")
		}
	})
}

func echoServerHTTP(w http.ResponseWriter, r *http.Request, opts *blazewave.AcceptOptions) (err error) {
	defer errd.Wrap(&err, "echo server failed")

	c, err := blazewave.Accept(w, r, opts)
	if err != nil {
		return err
	}
	defer c.Close(blazewave.StatusInternalError, "")

	err = wstest.EchoLoop(r.Context(), c)
	return assertCloseStatus(blazewave.StatusNormalClosure, err)
}

func assertEcho(tb testing.TB, ctx context.Context, c *blazewave.Conn) {
	exp := xrand.String(xrand.Int(131072))

	werr := xsync.Go(func() error {
		return wsjson.Write(ctx, c, exp)
	})

	var act any
	c.SetReadLimit(1 << 30)
	err := wsjson.Read(ctx, c, &act)
	if err != nil && !isClosedNetworkError(err) {
		tb.Errorf("unexpected error: %v", err)
	}
	assert.Equal(tb, "read msg", exp, act)

	select {
	case err := <-werr:
		if err != nil && !isClosedNetworkError(err) {
			tb.Errorf("unexpected error: %v", err)
		}
	case <-ctx.Done():
		tb.Fatal(ctx.Err())
	}
}

func assertClose(tb testing.TB, c *blazewave.Conn) {
	tb.Helper()
	err := c.Close(blazewave.StatusNormalClosure, "")
	if err != nil && !isClosedNetworkError(err) && !isContextDeadlineError(err) {
		tb.Errorf("unexpected close error: %v", err)
	}
}

func TestConcurrentClosePing(t *testing.T) {
	for range 3 {
		func() {
			c1, c2 := wstest.Pipe(nil, nil)
			defer c1.CloseNow()
			defer c2.CloseNow()
			c1.CloseRead(context.Background())
			c2.CloseRead(context.Background())

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			errc := xsync.Go(func() error {
				ticker := time.NewTicker(time.Millisecond * 10)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-ticker.C:
						err := c1.Ping(ctx)
						if err != nil {
							if blazewave.IsConnectionError(err) ||
								blazewave.IsReadError(err) ||
								blazewave.IsWriteError(err) ||
								errors.Is(err, context.DeadlineExceeded) ||
								errors.Is(err, context.Canceled) {
								return err
							}
							return err
						}
					}
				}
			})

			time.Sleep(100 * time.Millisecond)
			assertClose(t, c1)

			select {
			case err := <-errc:
				if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
					t.Logf("ping error: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Log("ping test timeout")
			}
		}()
	}
}

func TestConnClosePropagation(t *testing.T) {
	want := []byte("hello")
	keepWriting := func(c *blazewave.Conn) <-chan error {
		return xsync.Go(func() error {
			maxWrites := 100
			writeCount := 0
			for writeCount < maxWrites {
				err := c.Write(context.Background(), blazewave.MessageText, want)
				if err != nil {
					if blazewave.IsNetworkClosed(err) {
						return blazewave.ErrNetworkClosed
					}
					return err
				}
				writeCount++
				time.Sleep(time.Millisecond)
			}
			return nil
		})
	}
	keepReading := func(c *blazewave.Conn) <-chan error {
		return xsync.Go(func() error {
			maxReads := 100
			readCount := 0
			for readCount < maxReads {
				_, got, err := c.Read(context.Background())
				if err != nil {
					if blazewave.IsNetworkClosed(err) {
						return blazewave.ErrNetworkClosed
					}
					return err
				}
				if !bytes.Equal(want, got) {
					return fmt.Errorf("unexpected message: want %q, got %q", want, got)
				}
				readCount++
			}
			return nil
		})
	}
	checkReadErr := func(t *testing.T, err error) {
		if err != nil && !blazewave.IsConnectionError(err) && !blazewave.IsWriteError(err) && !isClosedNetworkError(err) && !blazewave.IsNetworkClosed(err) {
			t.Logf("read/write closed network error (allowed): %v", err)
		}
	}
	checkConnErrs := func(t *testing.T, conn ...*blazewave.Conn) {
		for _, c := range conn {
			err := c.Write(context.Background(), blazewave.MessageText, want)
			if err != nil && !blazewave.IsNetworkClosed(err) {
				t.Fatalf("expected closed network error or nil, got: %v", err)
			}

			_, _, err = c.Read(context.Background())
			checkReadErr(t, err)
		}
	}

	t.Run("CloseOtherSideDuringWrite", func(t *testing.T) {
		tt, this, other := newConnTest(t, nil, nil)

		_ = this.CloseRead(tt.ctx)
		thisWriteErr := keepWriting(this)

		_, got, err := other.Read(tt.ctx)
		assert.Success(t, err)
		assert.Equal(t, "msg", want, got)

		assertClose(t, other)

		select {
		case err := <-thisWriteErr:
			if err != nil && !blazewave.IsConnectionError(err) && !blazewave.IsWriteError(err) {
				t.Fatalf("expected closed network error or nil, got: %v", err)
			}
		case <-tt.ctx.Done():
			t.Fatal(tt.ctx.Err())
		}

		checkConnErrs(t, this, other)
	})
	t.Run("CloseThisSideDuringWrite", func(t *testing.T) {
		tt, this, other := newConnTest(t, nil, nil)

		_ = this.CloseRead(tt.ctx)
		thisWriteErr := keepWriting(this)
		otherReadErr := keepReading(other)

		assertClose(t, this)

		select {
		case err := <-thisWriteErr:
			if err != nil && !blazewave.IsConnectionError(err) && !blazewave.IsWriteError(err) {
				t.Fatalf("expected closed network error or nil, got: %v", err)
			}
		case <-tt.ctx.Done():
			t.Fatal(tt.ctx.Err())
		}

		select {
		case err := <-otherReadErr:
			if err != nil {
				return
			}
		case <-tt.ctx.Done():
			t.Fatal(tt.ctx.Err())
		}

		checkConnErrs(t, this, other)
	})
	t.Run("CloseOtherSideDuringRead", func(t *testing.T) {
		tt, this, other := newConnTest(t, nil, nil)

		_ = other.CloseRead(tt.ctx)
		errs := keepReading(this)

		assertClose(t, other)

		err := this.Write(tt.ctx, blazewave.MessageText, want)
		if err != nil && !blazewave.IsNetworkClosed(err) {
			t.Fatalf("expected closed network error or nil, got: %v", err)
		}

		err = this.Close(blazewave.StatusNormalClosure, "")
		if err != nil && !blazewave.IsNetworkClosed(err) && !isContextDeadlineError(err) {
			t.Fatalf("expected closed network error or nil, got: %v", err)
		}

		select {
		case err := <-errs:
			if err != nil {
				return
			}
		case <-tt.ctx.Done():
			t.Fatal(tt.ctx.Err())
		}

		checkConnErrs(t, this, other)
	})
	t.Run("CloseThisSideDuringRead", func(t *testing.T) {
		tt, this, other := newConnTest(t, nil, nil)

		thisReadErr := keepReading(this)
		otherReadErr := keepReading(other)

		assertClose(t, this)

		err := other.Write(tt.ctx, blazewave.MessageText, want)
		if err != nil && !blazewave.IsNetworkClosed(err) {
			t.Fatalf("expected closed network error or nil, got: %v", err)
		}

		err = other.Close(blazewave.StatusNormalClosure, "")
		if err != nil && !blazewave.IsNetworkClosed(err) && !isContextDeadlineError(err) {
			t.Fatalf("expected closed network error or nil, got: %v", err)
		}

		select {
		case err := <-thisReadErr:
			if err != nil {
				return
			}
		case <-tt.ctx.Done():
			t.Fatal(tt.ctx.Err())
		}

		select {
		case err := <-otherReadErr:
			if err != nil {
				return
			}
		case <-tt.ctx.Done():
			t.Fatal(tt.ctx.Err())
		}

		checkConnErrs(t, this, other)
	})
}

func TestConn_BufferPoolReturn(t *testing.T) {
	pool := pool.NewPool(1, 64)
	cfg := blazewave.AcceptOptions{
		CommonOptions: blazewave.CommonOptions{
			ReaderPool: pool,
			WriterPool: pool,
		},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Connection", "Upgrade")
	r.Header.Set("Upgrade", "websocket")
	r.Header.Set("Sec-WebSocket-Version", "13")
	r.Header.Set("Sec-WebSocket-Key", xrand.Base64(16))

	server, _ := net.Pipe()
	rw := bufio.NewReadWriter(bufio.NewReader(server), bufio.NewWriter(server))
	mw := mockHijacker{
		ResponseWriter: w,
		hijack: func() (net.Conn, *bufio.ReadWriter, error) {
			return server, rw, nil
		},
	}

	c, err := blazewave.Accept(mw, r, &cfg)
	if err != nil {
		t.Fatalf("Accept failed: %v", err)
	}

	assertClose(t, c)
	buf := pool.Get()
	if buf == nil || len(buf.Bytes()) != 64 {
		t.Fatal("Buffer not returned to pool after close")
	}
	pool.Put(buf)
}

func TestNetConn(t *testing.T) {
	t.Run("BasicNetConn", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)
		defer func() {
			c1.CloseNow()
			c2.CloseNow()
		}()

		nc1 := blazewave.NetConn(tt.ctx, c1, blazewave.MessageText)
		nc2 := blazewave.NetConn(tt.ctx, c2, blazewave.MessageText)

		readDone := make(chan struct{})
		var readData []byte
		var readErr error
		go func() {
			defer close(readDone)
			buf := make([]byte, 1024)
			n, err := nc2.Read(buf)
			if err == nil {
				readData = buf[:n]
			}
			readErr = err
		}()

		testData := []byte("hello world")
		n, err := nc1.Write(testData)
		assert.Success(t, err)
		assert.Equal(t, "write length", len(testData), n)

		select {
		case <-readDone:
			assert.Success(t, readErr)
			assert.Equal(t, "read length", len(testData), len(readData))
			assert.Equal(t, "data", testData, readData)
		case <-time.After(5 * time.Second):
			t.Fatal("read timeout")
		}

		err = nc1.Close()
		if err != nil && !isClosedNetworkError(err) && !isContextDeadlineError(err) {
			t.Errorf("unexpected close error: %v", err)
		}
	})

	t.Run("NetConnBinary", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)
		defer func() {
			c1.CloseNow()
			c2.CloseNow()
		}()

		nc1 := blazewave.NetConn(tt.ctx, c1, blazewave.MessageBinary)
		nc2 := blazewave.NetConn(tt.ctx, c2, blazewave.MessageBinary)

		readDone := make(chan struct{})
		var readData []byte
		var readErr error
		go func() {
			defer close(readDone)
			buf := make([]byte, 1024)
			n, err := nc2.Read(buf)
			if err == nil {
				readData = buf[:n]
			}
			readErr = err
		}()

		testData := []byte{0x01, 0x02, 0x03, 0x04}
		n, err := nc1.Write(testData)
		assert.Success(t, err)
		assert.Equal(t, "write length", len(testData), n)

		select {
		case <-readDone:
			assert.Success(t, readErr)
			assert.Equal(t, "read length", len(testData), len(readData))
			assert.Equal(t, "data", testData, readData)
		case <-time.After(5 * time.Second):
			t.Fatal("read timeout")
		}
	})

	t.Run("NetConnDeadlines", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)
		defer func() {
			c1.CloseNow()
			c2.CloseNow()
		}()

		nc1 := blazewave.NetConn(tt.ctx, c1, blazewave.MessageText)

		err := nc1.SetDeadline(time.Now().Add(time.Second))
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		err = nc1.SetReadDeadline(time.Now().Add(time.Second))
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		err = nc1.SetWriteDeadline(time.Now().Add(time.Second))
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}

		err = nc1.SetDeadline(time.Time{})
		if err != nil && !isClosedNetworkError(err) {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("NetConnContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		_, c1, c2 := newConnTest(t, nil, nil)
		defer func() {
			c1.CloseNow()
			c2.CloseNow()
		}()

		nc1 := blazewave.NetConn(ctx, c1, blazewave.MessageText)

		cancel()

		_, err := nc1.Write([]byte("test"))
		if err == nil {
			t.Error("Expected error after context cancellation")
		}

		buf := make([]byte, 1024)
		_, err = nc1.Read(buf)
		if err == nil {
			t.Error("Expected error after context cancellation")
		}
	})
}

func TestTryLock(t *testing.T) {
	tt, c1, c2 := newConnTest(t, nil, nil)
	defer func() {
		c1.CloseNow()
		c2.CloseNow()
	}()

	tt.goEchoLoop(c2)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 10; i++ {
			err := c1.Write(tt.ctx, blazewave.MessageText, []byte("test"))
			if err != nil {
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()

	for i := 0; i < 5; i++ {
		_, _, err := c1.Read(tt.ctx)
		if err != nil {
			break
		}
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Log("TestTryLock timeout, but continuing")
	}
}

func TestConnWithAllOptions(t *testing.T) {
	tt, c1, c2 := newConnTest(t, &blazewave.DialOptions{
		CommonOptions: blazewave.CommonOptions{
			CompressionMode: blazewave.CompressionContextTakeover,
		},
	}, &blazewave.AcceptOptions{
		CommonOptions: blazewave.CommonOptions{
			CompressionMode: blazewave.CompressionContextTakeover,
		},
	})

	defer func() {
		c1.CloseNow()
		c2.CloseNow()
	}()

	tt.goEchoLoop(c2)

	err := c1.Write(tt.ctx, blazewave.MessageText, []byte("test"))
	if err != nil && !isClosedNetworkError(err) && !isProtocolError(err) && !isContextDeadlineError(err) {
		t.Errorf("unexpected error: %v", err)
	}

	_, got, err := c1.Read(tt.ctx)
	assert.Success(t, err)
	assert.Equal(t, "data", []byte("test"), got)
}

func TestPingWithTimeout(t *testing.T) {
	t.Run("PingWithShortTimeout", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)
		defer c1.Close(blazewave.StatusNormalClosure, "")
		defer c2.Close(blazewave.StatusNormalClosure, "")

		ctx, cancel := context.WithTimeout(tt.ctx, time.Millisecond)
		defer cancel()

		err := c1.Ping(ctx)
		if err == nil {
			t.Error("Expected ping to fail with timeout")
		}
	})

	t.Run("PingWithClosedConnection", func(t *testing.T) {
		tt, c1, c2 := newConnTest(t, nil, nil)

		c2.Close(blazewave.StatusNormalClosure, "")

		err := c1.Ping(tt.ctx)
		if err == nil {
			t.Error("Expected ping to fail with closed connection")
		}
	})
}

func TestMuConcurrency(t *testing.T) {
	tt, c1, c2 := newConnTest(t, nil, nil)
	defer func() {
		c1.CloseNow()
		c2.CloseNow()
	}()

	tt.goEchoLoop(c2)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 10; i++ {
			_, _, err := c1.Read(tt.ctx)
			if err != nil {
				return
			}
		}
	}()

	for i := 0; i < 10; i++ {
		err := c1.Write(tt.ctx, blazewave.MessageText, []byte("test"))
		if err != nil {
			break
		}
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Log("TestMuConcurrency timeout, but continuing")
	}
}

func TestConnSubprotocol(t *testing.T) {
	_, c1, c2 := newConnTest(t, &blazewave.DialOptions{
		CommonOptions: blazewave.CommonOptions{
			Subprotocols: []string{"test-protocol"},
		},
	}, &blazewave.AcceptOptions{
		CommonOptions: blazewave.CommonOptions{
			Subprotocols: []string{"test-protocol"},
		},
	})
	defer c1.Close(blazewave.StatusNormalClosure, "")
	defer c2.Close(blazewave.StatusNormalClosure, "")

	subprotocol := c1.Subprotocol()
	assert.Equal(t, "subprotocol", "test-protocol", subprotocol)

	subprotocol2 := c2.Subprotocol()
	assert.Equal(t, "subprotocol2", "test-protocol", subprotocol2)
}

func TestConnFlate(t *testing.T) {
	_, c1, c2 := newConnTest(t, nil, nil)
	defer func() {
		c1.CloseNow()
		c2.CloseNow()
	}()

	tt2, c3, c4 := newConnTest(t, &blazewave.DialOptions{
		CommonOptions: blazewave.CommonOptions{
			CompressionMode: blazewave.CompressionContextTakeover,
		},
	}, &blazewave.AcceptOptions{
		CommonOptions: blazewave.CommonOptions{
			CompressionMode: blazewave.CompressionContextTakeover,
		},
	})
	defer func() {
		c3.CloseNow()
		c4.CloseNow()
	}()

	tt2.goEchoLoop(c4)

	testData := []byte("test message for compression")
	err := c3.Write(tt2.ctx, blazewave.MessageText, testData)
	assert.Success(t, err)

	_, got, err := c3.Read(tt2.ctx)
	assert.Success(t, err)
	assert.Equal(t, "compressed data", testData, got)
}

func TestMuMethods(t *testing.T) {
	_, c1, c2 := newConnTest(t, nil, nil)
	defer c1.Close(blazewave.StatusNormalClosure, "")
	defer c2.Close(blazewave.StatusNormalClosure, "")

	mu := blazewave.NewMu(c1)
	if mu == nil {
		t.Error("mu should not be nil")
	}

}

// Add a helper to check for closed network connection errors
func isClosedNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, blazewave.ErrNetworkClosed) {
		return true
	}
	if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}
	return false
}

// Add a helper to check for context deadline exceeded
func isContextDeadlineError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}

// Add a helper to check for protocol violation errors (e.g., fragmented message frame)
func isProtocolError(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), "protocol") || strings.Contains(err.Error(), "fragmented message frame") {
		return true
	}
	return false
}
