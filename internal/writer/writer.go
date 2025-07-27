package bytes

// Writer is a dynamically growing byte buffer for writing data.
type Writer struct {
	pos int    // current write position
	buf []byte // underlying byte slice
}

// NewWriterSize creates a new Writer with the specified initial size.
func NewWriterSize(size int) *Writer {
	if size <= 0 {
		size = 64 // default size
	}
	return &Writer{buf: make([]byte, size)}
}

// Len returns the number of bytes written so far.
func (w *Writer) Len() int {
	return w.pos
}

// Size returns the total capacity of the buffer.
func (w *Writer) Size() int {
	return len(w.buf)
}

// Reset resets the writer to be reused.
func (w *Writer) Reset() {
	w.pos = 0
}

// Buffer returns the written portion of the buffer.
func (w *Writer) Buffer() []byte {
	return w.buf[:w.pos]
}

// Peek reserves n bytes and returns a slice for direct writing.
func (w *Writer) Peek(n int) []byte {
	if n <= 0 {
		return nil
	}
	w.grow(n)
	buf := w.buf[w.pos : w.pos+n]
	w.pos += n
	return buf
}

// Write appends the given bytes to the buffer.
func (w *Writer) Write(p []byte) {
	if len(p) == 0 {
		return
	}
	w.grow(len(p))
	w.pos += copy(w.buf[w.pos:], p)
}

// grow expands the buffer if needed to accommodate n more bytes.
func (w *Writer) grow(n int) {
	if w.pos+n <= len(w.buf) {
		return
	}
	newBuf := make([]byte, 2*len(w.buf)+n)
	copy(newBuf, w.buf[:w.pos])
	w.buf = newBuf
}
