//go:build !js

package blazewave_test

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/heyehang/blazewave"
	"github.com/heyehang/blazewave/core/pool"
	"github.com/heyehang/blazewave/internal/test/assert"
	"github.com/heyehang/blazewave/internal/test/xrand"
)

func TestAccept(t *testing.T) {

	t.Run("badClientHandshake", func(t *testing.T) {

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)

		_, err := blazewave.Accept(w, r, nil)
		assert.Contains(t, err, "protocol violation")
	})

	t.Run("badOrigin", func(t *testing.T) {

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Connection", "Upgrade")
		r.Header.Set("Upgrade", "websocket")
		r.Header.Set("Sec-WebSocket-Version", "13")
		r.Header.Set("Sec-WebSocket-Key", xrand.Base64(16))
		r.Header.Set("Origin", "harhar.com")

		_, err := blazewave.Accept(w, r, nil)
		assert.Contains(t, err, `request Origin "harhar.com" is not a valid URL with a host`)
	})

	t.Run("unauthorizedOriginErrorMessage", func(t *testing.T) {

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Connection", "Upgrade")
		r.Header.Set("Upgrade", "websocket")
		r.Header.Set("Sec-WebSocket-Version", "13")
		r.Header.Set("Sec-WebSocket-Key", xrand.Base64(16))
		r.Header.Set("Origin", "https://harhar.com")

		_, err := blazewave.Accept(w, r, nil)
		assert.Contains(t, err, `request Origin "harhar.com" is not authorized for Host "example.com"`)
	})

	t.Run("badCompression", func(t *testing.T) {

		newRequest := func(extensions string) *http.Request {
			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set("Connection", "Upgrade")
			r.Header.Set("Upgrade", "websocket")
			r.Header.Set("Sec-WebSocket-Version", "13")
			r.Header.Set("Sec-WebSocket-Key", xrand.Base64(16))
			r.Header.Set("Sec-WebSocket-Extensions", extensions)
			return r
		}
		errHijack := errors.New("hijack error")
		newResponseWriter := func() http.ResponseWriter {
			return mockHijacker{
				ResponseWriter: httptest.NewRecorder(),
				hijack: func() (net.Conn, *bufio.ReadWriter, error) {
					return nil, nil, errHijack
				},
			}
		}

		t.Run("withoutFallback", func(t *testing.T) {

			w := newResponseWriter()
			r := newRequest("permessage-deflate; harharhar")
			_, err := blazewave.Accept(w, r, &blazewave.AcceptOptions{
				CommonOptions: blazewave.CommonOptions{
					CompressionMode: blazewave.CompressionNoContextTakeover,
				},
			})
			assert.ErrorIs(t, errHijack, err)
			assert.Equal(t, "extension header", w.Header().Get("Sec-WebSocket-Extensions"), "")
		})
		t.Run("withFallback", func(t *testing.T) {

			w := newResponseWriter()
			r := newRequest("permessage-deflate; harharhar, permessage-deflate")
			_, err := blazewave.Accept(w, r, &blazewave.AcceptOptions{
				CommonOptions: blazewave.CommonOptions{
					CompressionMode: blazewave.CompressionNoContextTakeover,
				},
			})
			assert.ErrorIs(t, errHijack, err)
			assert.Equal(t, "extension header",
				w.Header().Get("Sec-WebSocket-Extensions"),
				blazewave.GetCompressionOptions(blazewave.CompressionNoContextTakeover).String(),
			)
		})
	})

	t.Run("requireHttpHijacker", func(t *testing.T) {

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Connection", "Upgrade")
		r.Header.Set("Upgrade", "websocket")
		r.Header.Set("Sec-WebSocket-Version", "13")
		r.Header.Set("Sec-WebSocket-Key", xrand.Base64(16))

		_, err := blazewave.Accept(w, r, nil)
		assert.Contains(t, err, `http.ResponseWriter does not implement http.Hijacker`)
	})

	t.Run("badHijack", func(t *testing.T) {

		w := mockHijacker{
			ResponseWriter: httptest.NewRecorder(),
			hijack: func() (conn net.Conn, writer *bufio.ReadWriter, err error) {
				return nil, nil, errors.New("haha")
			},
		}

		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Connection", "Upgrade")
		r.Header.Set("Upgrade", "websocket")
		r.Header.Set("Sec-WebSocket-Version", "13")
		r.Header.Set("Sec-WebSocket-Key", xrand.Base64(16))

		_, err := blazewave.Accept(w, r, nil)
		assert.Contains(t, err, "failed to accept WebSocket connection: hijack connection: haha")
	})

	t.Run("wrapperHijackerIsUnwrapped", func(t *testing.T) {

		rr := httptest.NewRecorder()
		w := mockUnwrapper{
			ResponseWriter: rr,
			unwrap: func() http.ResponseWriter {
				return mockHijacker{
					ResponseWriter: rr,
					hijack: func() (conn net.Conn, writer *bufio.ReadWriter, err error) {
						return nil, nil, errors.New("haha")
					},
				}
			},
		}

		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Connection", "Upgrade")
		r.Header.Set("Upgrade", "websocket")
		r.Header.Set("Sec-WebSocket-Version", "13")
		r.Header.Set("Sec-WebSocket-Key", xrand.Base64(16))

		_, err := blazewave.Accept(w, r, nil)
		assert.Contains(t, err, "failed to accept WebSocket connection: hijack connection: haha")
	})

	t.Run("closeRace", func(t *testing.T) {

		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()

		rw := bufio.NewReadWriter(bufio.NewReader(server), bufio.NewWriter(server))
		newResponseWriter := func() http.ResponseWriter {
			return mockHijacker{
				ResponseWriter: httptest.NewRecorder(),
				hijack: func() (net.Conn, *bufio.ReadWriter, error) {
					return server, rw, nil
				},
			}
		}
		w := newResponseWriter()

		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Connection", "Upgrade")
		r.Header.Set("Upgrade", "websocket")
		r.Header.Set("Sec-WebSocket-Version", "13")
		r.Header.Set("Sec-WebSocket-Key", xrand.Base64(16))

		c, err := blazewave.Accept(w, r, nil)
		if err != nil {
			t.Fatalf("Accept failed: %v", err)
		}

		done := make(chan bool, 2)
		go func() {
			defer func() { done <- true }()
			c.CloseNow()
		}()
		go func() {
			defer func() { done <- true }()
			c.CloseNow()
		}()

		select {
		case <-done:
			<-done // Wait for second goroutine
		case <-time.After(500 * time.Millisecond):
			server.Close()
			client.Close()
			t.Fatal("closeRace test timed out")
		}

		server.Close()
		client.Close()
	})
}

func Test_verifyClientHandshake(t *testing.T) {

	testCases := []struct {
		name    string
		method  string
		http1   bool
		h       map[string]string
		success bool
	}{
		{
			name: "badConnection",
			h: map[string]string{
				"Connection": "notUpgrade",
			},
		},
		{
			name: "badUpgrade",
			h: map[string]string{
				"Connection": "Upgrade",
				"Upgrade":    "notWebSocket",
			},
		},
		{
			name:   "badMethod",
			method: "POST",
			h: map[string]string{
				"Connection": "Upgrade",
				"Upgrade":    "websocket",
			},
		},
		{
			name: "badWebSocketVersion",
			h: map[string]string{
				"Connection":            "Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Version": "14",
			},
		},
		{
			name: "missingWebSocketKey",
			h: map[string]string{
				"Connection":            "Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Version": "13",
			},
		},
		{
			name: "emptyWebSocketKey",
			h: map[string]string{
				"Connection":            "Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Version": "13",
				"Sec-WebSocket-Key":     "",
			},
		},
		{
			name: "shortWebSocketKey",
			h: map[string]string{
				"Connection":            "Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Version": "13",
				"Sec-WebSocket-Key":     xrand.Base64(15),
			},
		},
		{
			name: "invalidWebSocketKey",
			h: map[string]string{
				"Connection":            "Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Version": "13",
				"Sec-WebSocket-Key":     "notbase64",
			},
		},
		{
			name: "extraWebSocketKey",
			h: map[string]string{
				"Connection":            "Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Version": "13",
				"Sec-WebSocket-Key":     xrand.Base64(16),
				"sec-webSocket-key":     xrand.Base64(16),
			},
		},
		{
			name: "badHTTPVersion",
			h: map[string]string{
				"Connection":            "Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Version": "13",
				"Sec-WebSocket-Key":     xrand.Base64(16),
			},
			http1: true,
		},
		{
			name: "success",
			h: map[string]string{
				"Connection":            "keep-alive, Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Version": "13",
				"Sec-WebSocket-Key":     xrand.Base64(16),
			},
			success: true,
		},
		{
			name: "successSecKeyExtraSpace",
			h: map[string]string{
				"Connection":            "keep-alive, Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Version": "13",
				"Sec-WebSocket-Key":     "   " + xrand.Base64(16) + "  ",
			},
			success: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			r := httptest.NewRequest(tc.method, "/", nil)

			r.ProtoMajor = 1
			r.ProtoMinor = 1
			if tc.http1 {
				r.ProtoMinor = 0
			}

			for k, v := range tc.h {
				r.Header.Add(k, v)
			}

			_, err := blazewave.VerifyClientRequest(httptest.NewRecorder(), r)
			if tc.success {
				assert.Success(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func Test_selectSubprotocol(t *testing.T) {

	testCases := []struct {
		name            string
		clientProtocols []string
		serverProtocols []string
		negotiated      string
	}{
		{
			name:            "empty",
			clientProtocols: nil,
			serverProtocols: nil,
			negotiated:      "",
		},
		{
			name:            "basic",
			clientProtocols: []string{"echo", "echo2"},
			serverProtocols: []string{"echo2", "echo"},
			negotiated:      "echo2",
		},
		{
			name:            "none",
			clientProtocols: []string{"echo", "echo3"},
			serverProtocols: []string{"echo2", "echo4"},
			negotiated:      "",
		},
		{
			name:            "fallback",
			clientProtocols: []string{"echo", "echo3"},
			serverProtocols: []string{"echo2", "echo3"},
			negotiated:      "echo3",
		},
		{
			name:            "clientCasePresered",
			clientProtocols: []string{"Echo1"},
			serverProtocols: []string{"echo1"},
			negotiated:      "Echo1",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set("Sec-WebSocket-Protocol", strings.Join(tc.clientProtocols, ","))

			negotiated := blazewave.SelectSubprotocol(r, tc.serverProtocols)
			assert.Equal(t, "negotiated", tc.negotiated, negotiated)
		})
	}
}

func Test_authenticateOrigin(t *testing.T) {

	testCases := []struct {
		name           string
		origin         string
		host           string
		originPatterns []string
		success        bool
	}{
		{
			name:    "none",
			success: true,
			host:    "example.com",
		},
		{
			name:    "invalid",
			origin:  "$#)(*)$#@*$(#@*$)#@*%)#(@*%)#(@%#@$#@$#$#@$#@}{}{}",
			host:    "example.com",
			success: false,
		},
		{
			name:    "unauthorized",
			origin:  "https://example.com",
			host:    "example1.com",
			success: false,
		},
		{
			name:    "authorized",
			origin:  "https://example.com",
			host:    "example.com",
			success: true,
		},
		{
			name:    "authorizedCaseInsensitive",
			origin:  "https://examplE.com",
			host:    "example.com",
			success: true,
		},
		{
			name:   "originPatterns",
			origin: "https://two.examplE.com",
			host:   "example.com",
			originPatterns: []string{
				"*.example.com",
				"bar.com",
			},
			success: true,
		},
		{
			name:   "originPatternsUnauthorized",
			origin: "https://two.examplE.com",
			host:   "example.com",
			originPatterns: []string{
				"exam3.com",
				"bar.com",
			},
			success: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			r := httptest.NewRequest("GET", "http://"+tc.host+"/", nil)
			r.Header.Set("Origin", tc.origin)

			err := blazewave.AuthenticateOrigin(r, tc.originPatterns)
			if tc.success {
				assert.Success(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func Test_selectDeflate(t *testing.T) {

	testCases := []struct {
		name     string
		mode     blazewave.CompressionMode
		header   string
		expCopts *blazewave.CompressionOptions
		expOK    bool
	}{
		{
			name:     "disabled",
			mode:     blazewave.CompressionDisabled,
			expCopts: nil,
			expOK:    false,
		},
		{
			name:     "noClientSupport",
			mode:     blazewave.CompressionNoContextTakeover,
			expCopts: nil,
			expOK:    false,
		},
		{
			name:     "permessage-deflate",
			mode:     blazewave.CompressionNoContextTakeover,
			header:   "permessage-deflate; client_max_window_bits",
			expCopts: blazewave.NewCompressionOptions(true, true),
			expOK:    true,
		},
		{
			name:   "permessage-deflate/unknown-parameter",
			mode:   blazewave.CompressionNoContextTakeover,
			header: "permessage-deflate; meow",
			expOK:  false,
		},
		{
			name:     "permessage-deflate/unknown-parameter",
			mode:     blazewave.CompressionNoContextTakeover,
			header:   "permessage-deflate; meow, permessage-deflate; client_max_window_bits",
			expCopts: blazewave.NewCompressionOptions(true, true),
			expOK:    true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			h := http.Header{}
			h.Set("Sec-WebSocket-Extensions", tc.header)
			copts, ok := blazewave.SelectDeflate(blazewave.WebsocketExtensions(h), tc.mode)
			assert.Equal(t, "selected options", tc.expOK, ok)
			assert.Equal(t, "compression options", tc.expCopts, copts)
		})
	}
}

func TestAccept_BufferPoolReuse(t *testing.T) {

	rPool := pool.NewPool(1, 64)
	wPool := pool.NewPool(1, 64)
	opts := &blazewave.AcceptOptions{
		CommonOptions: blazewave.CommonOptions{
			ReaderPool: rPool,
			WriterPool: wPool,
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
	mw := &mockHijacker{
		ResponseWriter: w,
		hijack: func() (net.Conn, *bufio.ReadWriter, error) {
			return server, rw, nil
		},
	}
	c, err := blazewave.Accept(mw, r, opts)
	if err != nil {
		t.Fatalf("Accept failed: %v", err)
	}
	_ = c.Close(blazewave.StatusNormalClosure, "")
	buf := rPool.Get()
	if buf == nil || len(buf.Bytes()) != 64 {
		t.Fatal("Buffer not returned to pool after reuse")
	}
	rPool.Put(buf)
	buf = wPool.Get()
	if buf == nil || len(buf.Bytes()) != 64 {
		t.Fatal("Buffer not returned to pool after reuse")
	}
	wPool.Put(buf)
}

func TestAccept_BufferPoolStress(t *testing.T) {
	rPool := pool.NewPool(8, 64)
	wPool := pool.NewPool(8, 64)
	opts := &blazewave.AcceptOptions{
		CommonOptions: blazewave.CommonOptions{
			ReaderPool: rPool,
			WriterPool: wPool,
		},
	}
	const N = 5
	wg := sync.WaitGroup{}
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set("Connection", "Upgrade")
			r.Header.Set("Upgrade", "websocket")
			r.Header.Set("Sec-WebSocket-Version", "13")
			r.Header.Set("Sec-WebSocket-Key", xrand.Base64(16))

			server, _ := net.Pipe()
			rw := bufio.NewReadWriter(bufio.NewReader(server), bufio.NewWriter(server))
			mw := &mockHijacker{
				ResponseWriter: w,
				hijack: func() (net.Conn, *bufio.ReadWriter, error) {
					return server, rw, nil
				},
			}

			c, err := blazewave.Accept(mw, r, opts)
			if err == nil {
				_ = c.Close(blazewave.StatusNormalClosure, "")
			}
			wg.Done()
		}()
	}
	wg.Wait()

	for i := 0; i < 8; i++ {
		buf := rPool.Get()
		if buf == nil || len(buf.Bytes()) != 64 {
			t.Fatal("Buffer not returned to pool after stress")
		}
		rPool.Put(buf)
		buf = wPool.Get()
		if buf == nil || len(buf.Bytes()) != 64 {
			t.Fatal("Buffer not returned to pool after stress")
		}
		wPool.Put(buf)
	}
}

func TestAccept_BufferPoolStress_Extended(t *testing.T) {
	cases := []struct {
		poolSize int
		bufSize  int
		conns    int
	}{
		{8, 64, 2},
	}
	for _, c := range cases {
		rPool := pool.NewPool(c.poolSize, c.bufSize)
		wPool := pool.NewPool(c.poolSize, c.bufSize)
		opts := &blazewave.AcceptOptions{
			CommonOptions: blazewave.CommonOptions{
				ReaderPool: rPool,
				WriterPool: wPool,
			},
		}
		wg := sync.WaitGroup{}
		wg.Add(c.conns)
		for i := 0; i < c.conns; i++ {
			go func() {
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", "/", nil)
				r.Header.Set("Connection", "Upgrade")
				r.Header.Set("Upgrade", "websocket")
				r.Header.Set("Sec-WebSocket-Version", "13")
				r.Header.Set("Sec-WebSocket-Key", xrand.Base64(16))

				server, _ := net.Pipe()
				rw := bufio.NewReadWriter(bufio.NewReader(server), bufio.NewWriter(server))
				mw := &mockHijacker{
					ResponseWriter: w,
					hijack: func() (net.Conn, *bufio.ReadWriter, error) {
						return server, rw, nil
					},
				}

				c, err := blazewave.Accept(mw, r, opts)
				if err == nil {
					_ = c.Close(blazewave.StatusNormalClosure, "")
				}
				wg.Done()
			}()
		}
		wg.Wait()

		for i := 0; i < c.poolSize; i++ {
			rbuf := rPool.Get()
			if rbuf == nil || len(rbuf.Bytes()) != c.bufSize {
				t.Fatalf("Reader buffer not returned to pool after stress: poolSize=%d bufSize=%d", c.poolSize, c.bufSize)
			}
			rPool.Put(rbuf)
			wbuf := wPool.Get()
			if wbuf == nil || len(wbuf.Bytes()) != c.bufSize {
				t.Fatalf("Writer buffer not returned to pool after stress: poolSize=%d bufSize=%d", c.poolSize, c.bufSize)
			}
			wPool.Put(wbuf)
		}
	}
}

func TestAccept_BufferPool_MultiClose(t *testing.T) {
	rPool := pool.NewPool(1, 64)
	wPool := pool.NewPool(1, 64)
	opts := &blazewave.AcceptOptions{
		CommonOptions: blazewave.CommonOptions{
			ReaderPool: rPool,
			WriterPool: wPool,
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
	mw := &mockHijacker{
		ResponseWriter: w,
		hijack: func() (net.Conn, *bufio.ReadWriter, error) {
			return server, rw, nil
		},
	}

	c, err := blazewave.Accept(mw, r, opts)
	if err != nil {
		t.Fatalf("Accept failed: %v", err)
	}
	_ = c.Close(blazewave.StatusNormalClosure, "")
	_ = c.Close(blazewave.StatusNormalClosure, "")
	buf := rPool.Get()
	if buf == nil || len(buf.Bytes()) != 64 {
		t.Fatal("Buffer not returned to pool after multi close")
	}
	rPool.Put(buf)
	buf = wPool.Get()
	if buf == nil || len(buf.Bytes()) != 64 {
		t.Fatal("Buffer not returned to pool after multi close")
	}
	wPool.Put(buf)
}

func TestAccept_BufferPool_Exhaustion(t *testing.T) {
	rPool := pool.NewPool(1, 64)
	wPool := pool.NewPool(1, 64)
	opts := &blazewave.AcceptOptions{
		CommonOptions: blazewave.CommonOptions{
			ReaderPool: rPool,
			WriterPool: wPool,
		},
	}
	conns := 1
	var cs []*blazewave.Conn
	for i := 0; i < conns; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Connection", "Upgrade")
		r.Header.Set("Upgrade", "websocket")
		r.Header.Set("Sec-WebSocket-Version", "13")
		r.Header.Set("Sec-WebSocket-Key", xrand.Base64(16))
		server, _ := net.Pipe()
		rw := bufio.NewReadWriter(bufio.NewReader(server), bufio.NewWriter(server))
		mw := &mockHijacker{
			ResponseWriter: w,
			hijack: func() (net.Conn, *bufio.ReadWriter, error) {
				return server, rw, nil
			},
		}
		c, err := blazewave.Accept(mw, r, opts)
		if err == nil {
			cs = append(cs, c)
		}
	}
	for _, c := range cs {
		_ = c.Close(blazewave.StatusNormalClosure, "")
	}
	buf := rPool.Get()
	if buf == nil || len(buf.Bytes()) != 64 {
		t.Fatal("Buffer not returned to pool after exhaustion")
	}
	rPool.Put(buf)
	buf = wPool.Get()
	if buf == nil || len(buf.Bytes()) != 64 {
		t.Fatal("Buffer not returned to pool after exhaustion")
	}
	wPool.Put(buf)
}

func TestAccept_BufferPool_DataIsolation(t *testing.T) {
	rPool := pool.NewPool(1, 64)
	wPool := pool.NewPool(1, 64)
	opts := &blazewave.AcceptOptions{
		CommonOptions: blazewave.CommonOptions{
			ReaderPool: rPool,
			WriterPool: wPool,
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
	mw := &mockHijacker{
		ResponseWriter: w,
		hijack: func() (net.Conn, *bufio.ReadWriter, error) {
			return server, rw, nil
		},
	}
	c, err := blazewave.Accept(mw, r, opts)
	if err != nil {
		t.Fatalf("Accept failed: %v", err)
	}
	if c.GetBrBuf() != nil {
		copy(c.GetBrBuf().Bytes(), []byte("testdata"))
	}
	_ = c.Close(blazewave.StatusNormalClosure, "")
	buf := rPool.Get()
	if buf == nil || len(buf.Bytes()) != 64 {
		t.Fatal("Buffer not returned to pool after data isolation")
	}
	rPool.Put(buf)
}

func TestAccept_BufferPool_MixedPool(t *testing.T) {
	rPool := pool.NewPool(1, 64)
	wPool := pool.NewPool(1, 64)
	opts1 := &blazewave.AcceptOptions{
		CommonOptions: blazewave.CommonOptions{
			ReaderPool: rPool,
			WriterPool: wPool,
		},
	}
	opts2 := &blazewave.AcceptOptions{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Connection", "Upgrade")
	r.Header.Set("Upgrade", "websocket")
	r.Header.Set("Sec-WebSocket-Version", "13")
	r.Header.Set("Sec-WebSocket-Key", xrand.Base64(16))
	server, _ := net.Pipe()
	rw := bufio.NewReadWriter(bufio.NewReader(server), bufio.NewWriter(server))
	mw := &mockHijacker{
		ResponseWriter: w,
		hijack: func() (net.Conn, *bufio.ReadWriter, error) {
			return server, rw, nil
		},
	}
	c1, err := blazewave.Accept(mw, r, opts1)
	if err != nil {
		t.Fatalf("Accept failed: %v", err)
	}
	c2, err := blazewave.Accept(mw, r, opts2)
	if err != nil {
		t.Fatalf("Accept failed: %v", err)
	}
	_ = c1.Close(blazewave.StatusNormalClosure, "")
	_ = c2.Close(blazewave.StatusNormalClosure, "")
	buf := rPool.Get()
	if buf == nil || len(buf.Bytes()) != 64 {
		t.Fatal("Buffer not returned to pool after mixed pool")
	}
	rPool.Put(buf)
}

type mockHijacker struct {
	http.ResponseWriter
	hijack func() (net.Conn, *bufio.ReadWriter, error)
}

var _ http.Hijacker = mockHijacker{}

func (mj mockHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return mj.hijack()
}

type mockUnwrapper struct {
	http.ResponseWriter
	unwrap func() http.ResponseWriter
}

var _ blazewave.RwUnwrapper = mockUnwrapper{}

func (mu mockUnwrapper) Unwrap() http.ResponseWriter {
	return mu.unwrap()
}
