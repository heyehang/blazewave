//go:build !js

package blazewave

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/heyehang/blazewave/core/pool"
	"github.com/heyehang/blazewave/internal/bufio"
	"github.com/heyehang/blazewave/internal/errd"
)

// DialOptions represents Dial's options.
type DialOptions struct {
	CommonOptions

	// HTTPClient is used for the connection.
	// Its Transport must return writable bodies for WebSocket handshakes.
	// http.Transport does beginning with Go 1.12.
	HTTPClient *http.Client

	// HTTPHeader specifies the HTTP headers included in the handshake request.
	HTTPHeader http.Header

	// Host optionally overrides the Host HTTP header to send. If empty, the value
	// of URL.Host will be used.
	Host string

	// Event provides event-driven capabilities for WebSocket connections.
	// When set, it will handle all events (connect, message, ping, pong, close, error) automatically.
	// This takes precedence over individual callback functions like OnPingReceived, OnPongReceived, etc.
	Event *Event
}

func (opts *DialOptions) cloneWithDefaults(ctx context.Context) (context.Context, context.CancelFunc, *DialOptions) {
	var cancel context.CancelFunc

	var o DialOptions
	if opts != nil {
		o = *opts
	}
	if o.HTTPClient == nil {
		o.HTTPClient = http.DefaultClient
	}
	if o.HTTPClient.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, o.HTTPClient.Timeout)

		newClient := *o.HTTPClient
		newClient.Timeout = 0
		o.HTTPClient = &newClient
	}
	if o.HTTPHeader == nil {
		o.HTTPHeader = http.Header{}
	}
	newClient := *o.HTTPClient
	oldCheckRedirect := o.HTTPClient.CheckRedirect
	newClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		switch req.URL.Scheme {
		case "ws":
			req.URL.Scheme = "http"
		case "wss":
			req.URL.Scheme = "https"
		}
		if oldCheckRedirect != nil {
			return oldCheckRedirect(req, via)
		}
		return nil
	}
	o.HTTPClient = &newClient

	return ctx, cancel, &o
}

// Dial performs a WebSocket handshake on url.
//
// The response is the WebSocket handshake response from the server.
// You never need to close resp.Body yourself.
//
// If an error occurs, the returned response may be non nil.
// However, you can only read the first 1024 bytes of the body.
//
// This function requires at least Go 1.12 as it uses a new feature
// in net/http to perform WebSocket handshakes.
// See docs on the HTTPClient option and https://github.com/golang/go/issues/26937#issuecomment-415855861
//
// URLs with http/https schemes will work and are interpreted as ws/wss.
func Dial(ctx context.Context, u string, opts *DialOptions) (*Conn, *http.Response, error) {
	return dial(ctx, u, opts, nil)
}

func dial(ctx context.Context, urls string, opts *DialOptions, rand io.Reader) (_ *Conn, _ *http.Response, err error) {
	defer errd.Wrap(&err, "failed to WebSocket dial")

	var cancel context.CancelFunc
	ctx, cancel, opts = opts.cloneWithDefaults(ctx)
	if cancel != nil {
		defer cancel()
	}

	secWebSocketKey, err := secWebSocketKey(rand)
	if err != nil {
		return nil, nil, WrapConnectionError(err, "generate Sec-WebSocket-Key")
	}

	var copts *compressionOptions
	if opts.CompressionMode != CompressionDisabled {
		copts = opts.CompressionMode.opts()
	}

	resp, err := handshakeRequest(ctx, urls, opts, copts, secWebSocketKey)
	if err != nil {
		return nil, resp, err
	}
	respBody := resp.Body
	resp.Body = nil
	defer func() {
		if err != nil {
			// We read a bit of the body for easier debugging.
			r := io.LimitReader(respBody, 1024)

			timer := time.AfterFunc(time.Second*3, func() {
				respBody.Close()
			})
			defer timer.Stop()

			b, _ := io.ReadAll(r)
			respBody.Close()
			resp.Body = io.NopCloser(bytes.NewReader(b))
		}
	}()

	copts, err = verifyServerResponse(opts, copts, secWebSocketKey, resp)
	if err != nil {
		return nil, resp, err
	}

	rwc, ok := respBody.(io.ReadWriteCloser)
	if !ok {
		return nil, resp, WrapProtocolViolation("response body is not a io.ReadWriteCloser: %T", respBody)
	}

	var (
		br, bw, brBuf, bwBuf = initDialReaderWriter(rwc, opts)
	)

	c := newConn(connConfig{
		subprotocol:       resp.Header.Get("Sec-WebSocket-Protocol"),
		rwc:               rwc,
		client:            true,
		copts:             copts,
		flateThreshold:    opts.CompressionThreshold,
		event:             opts.Event,
		brPool:            opts.ReaderPool,
		br:                br,
		brBuf:             brBuf,
		bwPool:            opts.WriterPool,
		bw:                bw,
		bwBuf:             bwBuf,
		heartbeatInterval: opts.HeartbeatInterval,
		heartbeatTimeout:  opts.HeartbeatTimeout,
		heartbeatTimer:    opts.HeartbeatTimer,
	})

	return c, resp, nil
}

func handshakeRequest(ctx context.Context, urls string, opts *DialOptions, copts *compressionOptions, secWebSocketKey string) (*http.Response, error) {
	u, err := url.Parse(urls)
	if err != nil {
		return nil, WrapConnectionError(err, "parse url")
	}

	switch u.Scheme {
	case "ws":
		u.Scheme = "http"
	case "wss":
		u.Scheme = "https"
	case "http", "https":
	default:
		return nil, WrapProtocolViolation("unexpected url scheme: %q", u.Scheme)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, WrapConnectionError(err, "create new http request")
	}
	if len(opts.Host) > 0 {
		req.Host = opts.Host
	}
	req.Header = opts.HTTPHeader.Clone()
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", secWebSocketKey)
	if len(opts.Subprotocols) > 0 {
		req.Header.Set("Sec-WebSocket-Protocol", strings.Join(opts.Subprotocols, ","))
	}
	if copts != nil {
		req.Header.Set("Sec-WebSocket-Extensions", copts.String())
	}

	resp, err := opts.HTTPClient.Do(req)
	if err != nil {
		return nil, WrapConnectionError(err, "send handshake request")
	}
	return resp, nil
}

func secWebSocketKey(rr io.Reader) (string, error) {
	if rr == nil {
		rr = rand.Reader
	}
	b := make([]byte, 16)
	_, err := io.ReadFull(rr, b)
	if err != nil {
		return "", WrapConnectionError(err, "read random data from rand.Reader")
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func verifyServerResponse(opts *DialOptions, copts *compressionOptions, secWebSocketKey string, resp *http.Response) (*compressionOptions, error) {
	if resp.StatusCode != http.StatusSwitchingProtocols {
		return nil, WrapProtocolViolation("expected handshake response status code %v but got %v", http.StatusSwitchingProtocols, resp.StatusCode)
	}

	if !headerContainsTokenIgnoreCase(resp.Header, "Connection", "Upgrade") {
		return nil, WrapProtocolViolation("Connection header %q does not contain Upgrade", resp.Header.Get("Connection"))
	}

	if !headerContainsTokenIgnoreCase(resp.Header, "Upgrade", "WebSocket") {
		return nil, WrapProtocolViolation("Upgrade header %q does not contain websocket", resp.Header.Get("Upgrade"))
	}

	if resp.Header.Get("Sec-WebSocket-Accept") != secWebSocketAccept(secWebSocketKey) {
		return nil, WrapProtocolViolation("invalid Sec-WebSocket-Accept %q, key %q",
			resp.Header.Get("Sec-WebSocket-Accept"),
			secWebSocketKey,
		)
	}

	err := verifySubprotocol(opts.Subprotocols, resp)
	if err != nil {
		return nil, err
	}

	return verifyServerExtensions(copts, resp.Header)
}

func verifySubprotocol(subprotos []string, resp *http.Response) error {
	proto := resp.Header.Get("Sec-WebSocket-Protocol")
	if proto == "" {
		return nil
	}

	for _, sp2 := range subprotos {
		if strings.EqualFold(sp2, proto) {
			return nil
		}
	}

	return WrapProtocolViolation("unexpected Sec-WebSocket-Protocol from server: %q", proto)
}

func verifyServerExtensions(copts *compressionOptions, h http.Header) (*compressionOptions, error) {
	exts := websocketExtensions(h)
	if len(exts) == 0 {
		return nil, nil
	}

	ext := exts[0]
	if ext.name != "permessage-deflate" || len(exts) > 1 || copts == nil {
		return nil, WrapProtocolViolation("unsupported extensions from server: %+v", exts[1:])
	}

	_copts := *copts
	copts = &_copts

	for _, p := range ext.params {
		switch p {
		case "client_no_context_takeover":
			copts.clientNoContextTakeover = true
			continue
		case "server_no_context_takeover":
			copts.serverNoContextTakeover = true
			continue
		}
		if strings.HasPrefix(p, "server_max_window_bits=") {
			// We can't adjust the deflate window, but decoding with a larger window is acceptable.
			continue
		}

		return nil, WrapProtocolViolation("unsupported permessage-deflate parameter: %q", p)
	}

	return copts, nil
}

func initDialReaderWriter(netConn io.ReadWriter, opts *DialOptions) (*bufio.Reader, *bufio.Writer, *pool.Buffer, *pool.Buffer) {
	var (
		br           *bufio.Reader
		bw           *bufio.Writer
		brBuf, bwBuf *pool.Buffer
	)

	if opts.ReaderPool != nil {
		brBuf = opts.ReaderPool.Get()
		br = bufio.NewReaderSize(netConn, len(brBuf.Bytes()))
		br.ResetBuffer(netConn, brBuf.Bytes())
	} else {
		br = bufio.NewReader(netConn)
	}

	if opts.WriterPool != nil {
		bwBuf = opts.WriterPool.Get()
		bw = bufio.NewWriterSize(netConn, len(bwBuf.Bytes()))
		bw.ResetBuffer(netConn, bwBuf.Bytes())
	} else {
		bw = bufio.NewWriter(netConn)
	}

	return br, bw, brBuf, bwBuf
}
