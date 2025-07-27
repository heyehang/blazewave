package pool

import (
	"sync"
	"testing"
)

func TestBufferBasic(t *testing.T) {
	p := NewPool(2, 10)
	b1 := p.Get()
	if b1 == nil || b1.Bytes() == nil || len(b1.Bytes()) != 10 {
		t.Fatal("Buffer allocation failed")
	}
	b2 := p.Get()
	if b2 == nil || b2.Bytes() == nil || len(b2.Bytes()) != 10 {
		t.Fatal("Second buffer allocation failed")
	}
	b3 := p.Get()
	if b3 == nil || b3.Bytes() == nil || len(b3.Bytes()) != 10 {
		t.Fatal("Pool did not grow as expected")
	}
	p.Put(b1)
	b4 := p.Get()
	if b4 == nil || b4.Bytes() == nil || len(b4.Bytes()) != 10 {
		t.Fatal("Buffer reuse failed")
	}
}

func TestPoolInitInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid pool parameters")
		}
	}()
	_ = NewPool(0, 0)
}

func TestPoolPutNil(t *testing.T) {
	p := NewPool(1, 8)
	p.Put(nil) // should not panic
}

func TestPoolConcurrency(t *testing.T) {
	p := NewPool(8, 16)
	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			b := p.Get()
			if b == nil || len(b.Bytes()) != 16 {
				t.Error("Concurrent buffer allocation failed")
			}
			p.Put(b)
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestBufferBytes(t *testing.T) {
	p := NewPool(1, 4)
	b := p.Get()
	copy(b.buf, []byte{1, 2, 3, 4})
	if got := b.Bytes(); len(got) != 4 || got[0] != 1 || got[3] != 4 {
		t.Error("Bytes method returned wrong data")
	}
}
