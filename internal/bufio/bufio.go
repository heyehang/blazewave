package bufio

import (
	"bytes"
	"errors"
	"io"
	"unsafe"
)

const (
	defaultBufSize = 4096
)

var (
	// ErrInvalidUnreadByte invalid use of UnreadByete
	ErrInvalidUnreadByte = errors.New("bufio: invalid use of UnreadByte")
	// ErrInvalidUnreadRune invalid use of UnreadRune
	ErrInvalidUnreadRune = errors.New("bufio: invalid use of UnreadRune")
	// ErrBufferFull buffer full
	ErrBufferFull = errors.New("bufio: buffer full")
	// ErrNegativeCount negative count
	ErrNegativeCount = errors.New("bufio: negative count")
)

// Reader implements buffering for an io.Reader object.
type Reader struct {
	buffer   []byte    // underlying buffer
	reader   io.Reader // source reader
	readPos  int       // read pointer
	writePos int       // write pointer
	err      error     // last error
}

const minReadBufferSize = 16
const maxConsecutiveEmptyReads = 100

// NewReaderSize returns a new Reader whose buffer has at least the specified
// size. If the argument io.Reader is already a Reader with large enough
// size, it returns the underlying Reader.
func NewReaderSize(rd io.Reader, size int) *Reader {
	// Is it already a Reader?
	b, ok := rd.(*Reader)
	if ok && len(b.buffer) >= size {
		return b
	}
	if size < minReadBufferSize {
		size = minReadBufferSize
	}
	r := new(Reader)
	r.resetReader(make([]byte, size), rd)
	return r
}

// NewReader returns a new Reader whose buffer has the default size.
func NewReader(rd io.Reader) *Reader {
	return NewReaderSize(rd, defaultBufSize)
}

// Reset discards any buffered data, resets all state, and switches
// the buffered reader to read from r.
func (r *Reader) Reset(rd io.Reader) {
	r.resetReader(r.buffer, rd)
}

// ResetBuffer discards any buffered data, resets all state, and switches
// the buffered reader to read from r, using the provided buffer.
func (r *Reader) ResetBuffer(rd io.Reader, buf []byte) {
	r.resetReader(buf, rd)
}

func (r *Reader) resetReader(buf []byte, rd io.Reader) {
	*r = Reader{
		buffer: buf,
		reader: rd,
	}
}

var errNegativeRead = errors.New("bufio: reader returned negative count from Read")

// fillBuffer reads a new chunk into the buffer.
func (r *Reader) fillBuffer() {
	// Slide existing data to beginning.
	if r.readPos > 0 {
		copy(r.buffer, r.buffer[r.readPos:r.writePos])
		r.writePos -= r.readPos
		r.readPos = 0
	}

	if r.writePos >= len(r.buffer) {
		panic("bufio: tried to fill full buffer")
	}

	// Read new data: try a limited number of times.
	for i := maxConsecutiveEmptyReads; i > 0; i-- {
		n, err := r.reader.Read(r.buffer[r.writePos:])
		if n < 0 {
			panic(errNegativeRead)
		}
		r.writePos += n
		if err != nil {
			r.err = err
			return
		}
		if n > 0 {
			return
		}
	}
	r.err = io.ErrNoProgress
}

// consumeError returns and clears the last error.
func (r *Reader) consumeError() error {
	err := r.err
	r.err = nil
	return err
}

// Peek returns the next n bytes without advancing the reader. The bytes stop
// being valid at the next read call. If Peek returns fewer than n bytes, it
// also returns an error explaining why the read is short. The error is
// ErrBufferFull if n is larger than r's buffer size.
func (r *Reader) Peek(n int) ([]byte, error) {
	if n < 0 {
		return nil, ErrNegativeCount
	}
	if n > len(r.buffer) {
		return nil, ErrBufferFull
	}
	// 0 <= n <= len(buffer)
	for r.writePos-r.readPos < n && r.err == nil {
		r.fillBuffer() // r.writePos-r.readPos < len(buffer) => buffer is not full
	}

	var err error
	if avail := r.writePos - r.readPos; avail < n {
		// not enough data in buffer
		n = avail
		err = r.consumeError()
		if err == nil {
			err = ErrBufferFull
		}
	}
	return r.buffer[r.readPos : r.readPos+n], err
}

// Pop returns the next n bytes and advances the reader. The bytes stop
// being valid at the next read call. If Pop returns fewer than n bytes, it
// also returns an error explaining why the read is short. The error is
// ErrBufferFull if n is larger than r's buffer size.
func (r *Reader) Pop(n int) ([]byte, error) {
	d, err := r.Peek(n)
	if err == nil {
		r.readPos += n
		return d, err
	}
	return nil, err
}

// Discard skips the next n bytes, returning the number of bytes discarded.
//
// If Discard skips fewer than n bytes, it also returns an error.
// If 0 <= n <= r.Buffered(), Discard is guaranteed to succeed without
// reading from the underlying io.Reader.
func (r *Reader) Discard(n int) (discarded int, err error) {
	if n < 0 {
		return 0, ErrNegativeCount
	}
	if n == 0 {
		return
	}
	remain := n
	for {
		skip := r.Buffered()
		if skip == 0 {
			r.fillBuffer()
			skip = r.Buffered()
		}
		if skip > remain {
			skip = remain
		}
		r.readPos += skip
		remain -= skip
		if remain == 0 {
			return n, nil
		}
		if r.err != nil {
			return n - remain, r.consumeError()
		}
	}
}

// Read reads data into p.
// It returns the number of bytes read into p.
// It calls Read at most once on the underlying Reader,
// hence n may be less than len(p).
// At EOF, the count will be zero and err will be io.EOF.
func (r *Reader) Read(p []byte) (n int, err error) {
	n = len(p)
	if n == 0 {
		return 0, r.consumeError()
	}
	if r.readPos == r.writePos {
		if r.err != nil {
			return 0, r.consumeError()
		}
		if len(p) >= len(r.buffer) {
			// Large read, empty buffer.
			// Read directly into p to avoid copy.
			n, r.err = r.reader.Read(p)
			if n < 0 {
				panic(errNegativeRead)
			}
			return n, r.consumeError()
		}
		r.fillBuffer() // buffer is empty
		if r.readPos == r.writePos {
			return 0, r.consumeError()
		}
	}

	// copy as much as we can
	n = copy(p, r.buffer[r.readPos:r.writePos])
	r.readPos += n
	return n, nil
}

// ReadByte reads and returns a single byte.
// If no byte is available, returns an error.
func (r *Reader) ReadByte() (c byte, err error) {
	//r.lastRuneSize = -1
	for r.readPos == r.writePos {
		if r.err != nil {
			return 0, r.consumeError()
		}
		r.fillBuffer() // buffer is empty
	}
	c = r.buffer[r.readPos]
	r.readPos++
	//b.lastByte = int(c)
	return c, nil
}

// ReadSlice reads until the first occurrence of delim in the input,
// returning a slice pointing at the bytes in the buffer.
// The bytes stop being valid at the next read.
// If ReadSlice encounters an error before finding a delimiter,
// it returns all the data in the buffer and the error itself (often io.EOF).
// ReadSlice fails with error ErrBufferFull if the buffer fills without a delim.
// Because the data returned from ReadSlice will be overwritten
// by the next I/O operation, most clients should use
// ReadBytes or ReadString instead.
// ReadSlice returns err != nil if and only if line does not end in delim.
func (r *Reader) ReadSlice(delim byte) (line []byte, err error) {
	for {
		// Search buffer.
		if i := bytes.IndexByte(r.buffer[r.readPos:r.writePos], delim); i >= 0 {
			line = r.buffer[r.readPos : r.readPos+i+1]
			r.readPos += i + 1
			break
		}

		// Pending error?
		if r.err != nil {
			line = r.buffer[r.readPos:r.writePos]
			r.readPos = r.writePos
			err = r.consumeError()
			break
		}

		// Buffer full?
		if r.Buffered() >= len(r.buffer) {
			r.readPos = r.writePos
			line = r.buffer
			err = ErrBufferFull
			break
		}

		r.fillBuffer() // buffer is not full
	}
	return
}

// ReadLine is a low-level line-reading primitive. Most callers should use
// ReadBytes('\n') or ReadString('\n') instead or use a Scanner.
//
// ReadLine tries to return a single line, not including the end-of-line bytes.
// If the line was too long for the buffer then isPrefix is set and the
// beginning of the line is returned. The rest of the line will be returned
// from future calls. isPrefix will be false when returning the last fragment
// of the line. The returned buffer is only valid until the next call to
// ReadLine. ReadLine either returns a non-nil line or it returns an error,
// never both.
//
// The text returned from ReadLine does not include the line end ("\r\n" or "\n").
// No indication or error is given if the input ends without a final line end.
// Calling UnreadByte after ReadLine will always unread the last byte read
// (possibly a character belonging to the line end) even if that byte is not
// part of the line returned by ReadLine.
func (r *Reader) ReadLine() (line []byte, isPrefix bool, err error) {
	line, err = r.ReadSlice('\n')
	if err == ErrBufferFull {
		// Handle the case where "\r\n" straddles the buffer.
		if len(line) > 0 && line[len(line)-1] == '\r' {
			// Put the '\r' back on buf and drop it from line.
			// Let the next call to ReadLine check for "\r\n".
			if r.readPos == 0 {
				// should be unreachable
				panic("bufio: tried to rewind past start of buffer")
			}
			r.readPos--
			line = line[:len(line)-1]
		}
		return line, true, nil
	}

	if len(line) == 0 {
		if err != nil {
			line = nil
		}
		return
	}
	err = nil

	if line[len(line)-1] == '\n' {
		drop := 1
		if len(line) > 1 && line[len(line)-2] == '\r' {
			drop = 2
		}
		line = line[:len(line)-drop]
	}
	return
}

// Buffered returns the number of bytes that can be read from the current buffer.
func (r *Reader) Buffered() int { return r.writePos - r.readPos }

// buffered output

// Writer implements buffering for an io.Writer object.
type Writer struct {
	err         error     // last error
	buffer      []byte    // underlying buffer
	bufferedLen int       // number of bytes buffered
	writer      io.Writer // destination writer
}

// NewWriterSize returns a new Writer whose buffer has at least the specified size.
func NewWriterSize(w io.Writer, size int) *Writer {
	// Idempotent: if w is already a Writer with large enough buffer, return it directly.
	if bw, ok := w.(*Writer); ok && len(bw.buffer) >= size {
		return bw
	}
	if size <= 0 {
		size = defaultBufSize
	}
	return &Writer{
		buffer: make([]byte, size),
		writer: w,
	}
}

// NewWriter returns a new Writer whose buffer has the default size.
func NewWriter(w io.Writer) *Writer {
	return NewWriterSize(w, defaultBufSize)
}

// Reset discards any buffered data, resets all state, and switches the buffered writer to write to w.
func (w *Writer) Reset(writer io.Writer) {
	w.err = nil
	w.bufferedLen = 0
	w.writer = writer
}

// ResetBuffer discards any buffered data, resets all state, and switches the buffered writer to write to w, using the provided buffer.
func (w *Writer) ResetBuffer(writer io.Writer, buf []byte) {
	w.err = nil
	w.buffer = buf
	w.bufferedLen = 0
	w.writer = writer
}

// Flush writes any buffered data to the underlying io.Writer.
func (w *Writer) Flush() error {
	if w.err != nil {
		err := w.err
		w.err = nil
		return err
	}
	return w.flushBuffer()
}

// flushBuffer writes the buffer to the underlying writer.
func (w *Writer) flushBuffer() error {
	if w.bufferedLen == 0 {
		return w.err
	}
	n, err := w.writer.Write(w.buffer[:w.bufferedLen])
	if n < w.bufferedLen && err == nil {
		err = io.ErrShortWrite
	}
	if err != nil {
		if n > 0 && n < w.bufferedLen {
			copy(w.buffer, w.buffer[n:w.bufferedLen])
		}
		w.bufferedLen -= n
		w.err = err
		return err
	}
	w.bufferedLen = 0
	return nil
}

// Available returns how many bytes are unused in the buffer.
func (w *Writer) Available() int { return len(w.buffer) - w.bufferedLen }

// Buffered returns the number of bytes that have been written into the current buffer.
func (w *Writer) Buffered() int { return w.bufferedLen }

// Write writes the contents of p into the buffer. It returns the number of bytes written.
func (w *Writer) Write(p []byte) (nn int, err error) {
	for len(p) > w.Available() && w.err == nil {
		var n int
		if w.bufferedLen == 0 {
			// Large write, empty buffer.
			n, w.err = w.writer.Write(p)
			if n < 0 {
				panic(errNegativeRead)
			}
			nn += n
			p = p[n:]
			if w.err != nil {
				return nn, w.err
			}
			continue
		}
		copy(w.buffer[w.bufferedLen:], p)
		n = len(w.buffer) - w.bufferedLen
		w.bufferedLen += n
		p = p[n:]
		nn += n
		if err := w.flushBuffer(); err != nil {
			return nn, err
		}
	}
	if w.err != nil {
		return nn, w.err
	}
	copy(w.buffer[w.bufferedLen:], p)
	w.bufferedLen += len(p)
	return nn + len(p), nil
}

// WriteRaw writes p directly to the underlying writer, bypassing the buffer.
func (w *Writer) WriteRaw(p []byte) (nn int, err error) {
	return w.writer.Write(p)
}

func (w *Writer) WriteByte(b byte) error {
	_, err := w.Write([]byte{b})
	return err
}

// Peek returns a slice of the next n bytes in the buffer, growing the buffer if necessary.
func (w *Writer) Peek(n int) ([]byte, error) {
	if n < 0 {
		return nil, ErrNegativeCount
	}
	if n > len(w.buffer)-w.bufferedLen {
		return nil, ErrBufferFull
	}
	return w.buffer[w.bufferedLen : w.bufferedLen+n], nil
}

// WriteString writes the contents of s into the buffer.
func (w *Writer) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// WriteHackString writes the contents of s into the buffer.
func (w *Writer) WriteHackString(s string) (int, error) {
	return w.Write(unsafe.Slice(unsafe.StringData(s), len(s)))
}
