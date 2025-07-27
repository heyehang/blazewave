package pool

import (
	"fmt"
	"testing"
)

func BenchmarkBufferPoolReuseEfficiency(b *testing.B) {
	poolSizes := []int{4, 16}   // Only test two pool sizes
	bufSizes := []int{64, 1024} // Only test two buffer sizes

	for _, poolSize := range poolSizes {
		for _, bufSize := range bufSizes {
			b.Run(fmt.Sprintf("pool_%d_buf_%d", poolSize, bufSize), func(b *testing.B) {
				pool := NewPool(poolSize, bufSize)

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					buf := pool.Get()
					_ = buf.Bytes()
					pool.Put(buf)
				}
			})
		}
	}
}

func BenchmarkBufferPoolConcurrentReuse(b *testing.B) {
	pool := NewPool(16, 256)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get()
			_ = buf.Bytes()
			pool.Put(buf)
		}
	})
}

func BenchmarkBufferPoolExhaustionAndReuse(b *testing.B) {
	pool := NewPool(2, 64) // Small pool, easy to exhaust

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf1 := pool.Get()
		buf2 := pool.Get()
		buf3 := pool.Get() // Trigger expansion

		pool.Put(buf1)
		pool.Put(buf2)
		pool.Put(buf3)
	}
}

func BenchmarkBufferPoolMemoryReuse(b *testing.B) {
	pool := NewPool(8, 256)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bufs := make([]*Buffer, 0, 8)
		for j := 0; j < 8; j++ {
			buf := pool.Get()
			bufs = append(bufs, buf)
		}

		for _, buf := range bufs {
			_ = buf.Bytes()
		}

		for _, buf := range bufs {
			pool.Put(buf)
		}
	}
}

func BenchmarkBufferPoolOverheadComparison(b *testing.B) {
	b.Run("with_pool", func(b *testing.B) {
		pool := NewPool(8, 256)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := pool.Get()
			_ = buf.Bytes()
			pool.Put(buf)
		}
	})

	b.Run("without_pool", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := make([]byte, 256)
			_ = buf
		}
	})
}

func BenchmarkBufferPoolWriteReuse(b *testing.B) {
	pool := NewPool(4, 1024)
	data := make([]byte, 512)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := pool.Get()
		copy(buf.Bytes()[:len(data)], data)
		pool.Put(buf)
	}
}

func BenchmarkBufferPoolReadReuse(b *testing.B) {
	pool := NewPool(4, 1024)
	data := make([]byte, 512)

	buf := pool.Get()
	copy(buf.Bytes()[:len(data)], data)
	pool.Put(buf)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := pool.Get()
		_ = buf.Bytes()[:len(data)]
		pool.Put(buf)
	}
}
