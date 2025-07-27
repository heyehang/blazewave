package pool

import (
	"math/bits"

	"github.com/heyehang/blazewave/core/timer"
)

// Option ring pool options.
type option struct {
	Reader ringPool
	Writer ringPool
	Timers ringTimerPool
}

type ringPool struct {
	poolSize int
	bufNum   int
	bufSize  int
}

type ringTimerPool struct {
	poolSize int
	capacity int
}

// RingPool ring pool.
type RingPool struct {
	opt     *option
	readers []Pool
	writers []Pool
	timers  []timer.Timer
}

// newDefaultRingPool new a default ring pool.
func newDefaultRingPool() *option {
	return &option{
		Reader: ringPool{
			poolSize: 32,
			bufNum:   1024,
			bufSize:  8192,
		},
		Writer: ringPool{
			poolSize: 32,
			bufNum:   1024,
			bufSize:  8192,
		},
		Timers: ringTimerPool{
			poolSize: 32,
			capacity: 2048,
		},
	}
}

// ringPoolOption ring pool option.
type Option func(*option)

func WithRingPool(reader, writer ringPool) Option {
	return func(opt *option) {
		opt.Reader = reader
		opt.Writer = writer
	}
}

func WithTimerPool(timers ringTimerPool) Option {
	return func(opt *option) {
		opt.Timers = timers
	}
}

// NewRingPool new a ring pool.
func NewRingPool(options ...Option) (r *RingPool) {
	var i int

	opt := newDefaultRingPool()
	for _, option := range options {
		option(opt)
	}

	r = &RingPool{
		opt: opt,
	}

	opt.Reader.poolSize = nextPowerOfTwo(opt.Reader.poolSize)
	opt.Writer.poolSize = nextPowerOfTwo(opt.Writer.poolSize)
	opt.Timers.poolSize = nextPowerOfTwo(opt.Timers.poolSize)

	r.readers = make([]Pool, opt.Reader.poolSize)
	for i = 0; i < opt.Reader.poolSize; i++ {
		r.readers[i].Init(opt.Reader.bufNum, opt.Reader.bufSize)
	}

	r.writers = make([]Pool, opt.Writer.poolSize)
	for i = 0; i < opt.Writer.poolSize; i++ {
		r.writers[i].Init(opt.Writer.bufNum, opt.Writer.bufSize)
	}

	r.timers = make([]timer.Timer, opt.Timers.poolSize)
	for i = 0; i < opt.Timers.poolSize; i++ {
		r.timers[i].Init(opt.Timers.capacity)
	}

	return
}

// Reader get a reader memory buffer.
func (r *RingPool) Reader(rn int) *Pool {
	return &(r.readers[rn&(r.opt.Reader.poolSize-1)])
}

// Writer get a writer memory buffer pool.
func (r *RingPool) Writer(rn int) *Pool {
	return &(r.writers[rn&(r.opt.Writer.poolSize-1)])
}

// Timer get a timer.
func (r *RingPool) Timer(tn int) *timer.Timer {
	return &(r.timers[tn&(r.opt.Timers.poolSize-1)])
}

// nextPowerOfTwo get the next power of two.
func nextPowerOfTwo(n int) int {
	if n <= 0 {
		return 1
	}
	return 1 << uint(bits.Len(uint(n)))
}
