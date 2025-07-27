//go:build !js

package wstest

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"net/http/httptest"

	"github.com/heyehang/blazewave"
)

// Pipe is used to create an in memory connection
// between two blazewaves analogous to net.Pipe.
func Pipe(dialOpts *blazewave.DialOptions, acceptOpts *blazewave.AcceptOptions) (clientConn, serverConn *blazewave.Conn) {
	tt := fakeTransport{
		h: func(w http.ResponseWriter, r *http.Request) {
			serverConn, _ = blazewave.Accept(w, r, acceptOpts)
		},
	}

	if dialOpts == nil {
		dialOpts = &blazewave.DialOptions{}
	}
	_dialOpts := *dialOpts
	dialOpts = &_dialOpts
	dialOpts.HTTPClient = &http.Client{
		Transport: tt,
	}

	clientConn, _, _ = blazewave.Dial(context.Background(), "ws://example.com", dialOpts)
	return clientConn, serverConn
}

type fakeTransport struct {
	h http.HandlerFunc
}

func (t fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	clientConn, serverConn := net.Pipe()

	hj := testHijacker{
		ResponseRecorder: httptest.NewRecorder(),
		serverConn:       serverConn,
	}

	t.h.ServeHTTP(hj, r)

	resp := hj.ResponseRecorder.Result()
	if resp.StatusCode == http.StatusSwitchingProtocols {
		resp.Body = clientConn
	}
	return resp, nil
}

type testHijacker struct {
	*httptest.ResponseRecorder
	serverConn net.Conn
}

var _ http.Hijacker = testHijacker{}

func (hj testHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return hj.serverConn, bufio.NewReadWriter(bufio.NewReader(hj.serverConn), bufio.NewWriter(hj.serverConn)), nil
}
