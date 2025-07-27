//go:build !js

package blazewave

import (
	"github.com/heyehang/blazewave/internal/util"
)

func (c *Conn) RecordBytesWritten() *int {
	var bytesWritten int
	c.bw.Reset(util.WriterFunc(func(p []byte) (int, error) {
		bytesWritten += len(p)
		return c.rwc.Write(p)
	}))
	return &bytesWritten
}

func (c *Conn) RecordBytesRead() *int {
	var bytesRead int
	c.br.Reset(util.ReaderFunc(func(p []byte) (int, error) {
		n, err := c.rwc.Read(p)
		bytesRead += n
		return n, err
	}))
	return &bytesRead
}

var ErrClosed = ErrNetworkClosed

var (
	ExportedDial         = dial
	SecWebSocketAccept   = secWebSocketAccept
	SecWebSocketKey      = secWebSocketKey
	VerifyServerResponse = verifyServerResponse
)

var CompressionModeOpts = CompressionMode.opts
