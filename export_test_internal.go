//go:build !js

package blazewave

import (
	"io"

	"github.com/heyehang/blazewave/core/pool"
	"github.com/heyehang/blazewave/internal/bufio"
)

// This file exports private functions and types for testing purposes.
// It uses package blazewave to access private members.

// Export private functions for testing
var (
	NewConn                       = newConn
	ParseClosePayload             = parseClosePayload
	ValidWireCloseCode            = validWireCloseCode
	NewMsgReader                  = newMsgReader
	NewMsgWriter                  = newMsgWriter
	NewLimitReader                = newLimitReader
	InitDialReaderWriter          = initDialReaderWriter
	InitAcceptReaderWriter        = initAcceptReaderWriter
	HandshakeRequest              = handshakeRequest
	SecWebSocketKeyFunc           = secWebSocketKey
	VerifySubprotocol             = verifySubprotocol
	VerifyServerExtensions        = verifyServerExtensions
	VerifyClientRequest           = verifyClientRequest
	AuthenticateOrigin            = authenticateOrigin
	Match                         = match
	SelectSubprotocol             = selectSubprotocol
	SelectDeflate                 = selectDeflate
	AcceptDeflate                 = acceptDeflate
	HeaderContainsTokenIgnoreCase = headerContainsTokenIgnoreCase
	WebsocketExtensions           = websocketExtensions
	HeaderTokens                  = headerTokens
	ReadFrameHeader               = readFrameHeader
	WriteFrameHeader              = writeFrameHeader
	Mask                          = mask
	MaskGo                        = maskGo
	GetFlateReader                = getFlateReader
	PutFlateReader                = putFlateReader
	GetFlateWriter                = getFlateWriter
	PutFlateWriter                = putFlateWriter
	SlidingWindowPool             = slidingWindowPool
	ExtractBufioWriterBuf         = extractBufioWriterBuf
	NewMu                         = newMu
	Hijacker                      = hijacker
)

// Export types for testing
type (
	ConnConfig         = connConfig
	MsgReader          = msgReader
	MsgWriter          = msgWriter
	LimitReader        = limitReader
	Mu                 = mu
	WebsocketExtension = websocketExtension
	CompressionOptions = compressionOptions
	RwUnwrapper        = rwUnwrapper
	Header             = header
	SlidingWindow      = slidingWindow
)

// Export constants for testing
const (
	MaxControlPayload = maxControlPayload
	MaxCloseReason    = maxCloseReason
)

// Export compression options for testing
func GetCompressionOptions(mode CompressionMode) *compressionOptions {
	return mode.opts()
}

// Create compression options for testing
func NewCompressionOptions(clientNoContextTakeover, serverNoContextTakeover bool) *compressionOptions {
	return &compressionOptions{
		clientNoContextTakeover: clientNoContextTakeover,
		serverNoContextTakeover: serverNoContextTakeover,
	}
}

// Accessor methods for Conn private fields
func (c *Conn) GetBrBuf() *pool.Buffer {
	return c.brBuf
}

func (c *Conn) GetBwBuf() *pool.Buffer {
	return c.bwBuf
}

// Create conn config for testing
func NewConnConfig(rwc io.ReadWriteCloser, client bool, brPool, bwPool pool.BufferPool, br *bufio.Reader, brBuf *pool.Buffer, bw *bufio.Writer, bwBuf *pool.Buffer) connConfig {
	return connConfig{
		rwc:    rwc,
		client: client,
		brPool: brPool,
		br:     br,
		brBuf:  brBuf,
		bwPool: bwPool,
		bw:     bw,
		bwBuf:  bwBuf,
	}
}
