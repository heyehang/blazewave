package pool

import (
	"sync"
)

// Buffer represents a reusable byte buffer.
type Buffer struct {
	buf  []byte  // underlying byte slice
	next *Buffer // next free buffer in the pool
}

// Bytes returns the underlying byte slice.
func (b *Buffer) Bytes() []byte {
	return b.buf
}

var _ BufferPool = (*Pool)(nil)

type BufferPool interface {
	Get() *Buffer
	Put(b *Buffer)
}

// Pool manages a set of pre-allocated Buffers for reuse.
type Pool struct {
	mu       sync.Mutex // protects freeList
	freeList *Buffer    // head of the free buffer linked list
	maxSize  int        // total size of all buffers
	bufNum   int        // number of buffers
	bufSize  int        // size of each buffer
}

// NewPool creates a new buffer pool with the given number and size of buffers.
func NewPool(bufNum, bufSize int) *Pool {
	p := &Pool{}
	p.init(bufNum, bufSize)
	return p
}

// Init initializes the buffer pool with the given number and size of buffers.
func (p *Pool) Init(bufNum, bufSize int) {
	p.init(bufNum, bufSize)
}

// init allocates and links the buffers in the pool.
func (p *Pool) init(bufNum, bufSize int) {
	if bufNum <= 0 || bufSize <= 0 {
		panic("invalid buffer pool parameters")
	}
	p.bufNum = bufNum
	p.bufSize = bufSize
	p.maxSize = bufNum * bufSize
	p.grow()
}

// grow allocates a new batch of buffers and adds them to the free list.
func (p *Pool) grow() {
	buffers := make([]Buffer, p.bufNum)
	data := make([]byte, p.maxSize)
	for i := 0; i < p.bufNum; i++ {
		buffers[i].buf = data[i*p.bufSize : (i+1)*p.bufSize]
		if i < p.bufNum-1 {
			buffers[i].next = &buffers[i+1]
		}
	}
	p.freeList = &buffers[0]
}

// Get retrieves a free buffer from the pool, growing the pool if necessary.
func (p *Pool) Get() *Buffer {
	p.mu.Lock()
	if p.freeList == nil {
		p.grow()
	}
	b := p.freeList
	if b != nil {
		p.freeList = b.next
		b.next = nil // clear next pointer
	}
	p.mu.Unlock()
	return b
}

// Put returns a buffer to the pool for reuse.
func (p *Pool) Put(b *Buffer) {
	if b == nil {
		return
	}
	p.mu.Lock()
	b.next = p.freeList
	p.freeList = b
	p.mu.Unlock()
}
