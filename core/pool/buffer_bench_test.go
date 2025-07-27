package pool

import (
	"fmt"
	"testing"
)

func BenchmarkBufferPoolBasic(b *testing.B) {
	sizes := []int{64, 1024} // Only test two sizes

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			pool := NewPool(8, size)

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

func BenchmarkBufferPoolReuse(b *testing.B) {
	poolSizes := []int{4, 16}  // Only test two pool sizes
	bufSizes := []int{64, 256} // Only test two buffer sizes

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

func BenchmarkBufferPoolConcurrent(b *testing.B) {
	concurrencyLevels := []int{2, 8} // Only test two concurrency levels
	poolSize := 16
	bufSize := 256

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("concurrency_%d", concurrency), func(b *testing.B) {
			pool := NewPool(poolSize, bufSize)

			b.ReportAllocs()
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					buf := pool.Get()
					_ = buf.Bytes()
					pool.Put(buf)
				}
			})
		})
	}
}

func BenchmarkBufferPoolStress(b *testing.B) {
	pool := NewPool(4, 64)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for j := 0; j < 5; j++ { // Reduced from 20 to 5
			buf := pool.Get()
			pool.Put(buf)
		}
	}
}

func BenchmarkBufferPoolExhaustion(b *testing.B) {
	pool := NewPool(2, 64)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf1 := pool.Get()
		buf2 := pool.Get()
		buf3 := pool.Get() // This will trigger expansion

		pool.Put(buf1)
		pool.Put(buf2)
		pool.Put(buf3)
	}
}

func BenchmarkBufferPoolMixed(b *testing.B) {
	pool := NewPool(8, 256)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bufs := make([]*Buffer, 0, 4)

		for j := 0; j < 4; j++ {
			buf := pool.Get()
			bufs = append(bufs, buf)
		}

		for _, buf := range bufs {
			pool.Put(buf)
		}
	}
}

func BenchmarkBufferPoolWrite(b *testing.B) {
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

func BenchmarkBufferPoolRead(b *testing.B) {
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

func BenchmarkBufferPoolOverhead(b *testing.B) {
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

func BenchmarkBufferPoolGrowth(b *testing.B) {
	pool := NewPool(1, 64) // Small initial pool

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bufs := make([]*Buffer, 0, 3) // Reduced from 5 to 3
		for j := 0; j < 3; j++ {
			buf := pool.Get()
			bufs = append(bufs, buf)
		}

		for _, buf := range bufs {
			pool.Put(buf)
		}
	}
}

func BenchmarkBufferPoolNilSafety(b *testing.B) {
	pool := NewPool(4, 64)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pool.Put(nil)

		buf := pool.Get()
		pool.Put(buf)
	}
}

func BenchmarkBufferPoolMemoryEfficiency(b *testing.B) {
	poolSizes := []int{4, 8}   // Only test two pool sizes
	bufSizes := []int{64, 256} // Only test two buffer sizes

	for _, poolSize := range poolSizes {
		for _, bufSize := range bufSizes {
			b.Run(fmt.Sprintf("pool_%d_buf_%d", poolSize, bufSize), func(b *testing.B) {
				pool := NewPool(poolSize, bufSize)

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					bufs := make([]*Buffer, 0, poolSize)
					for j := 0; j < poolSize; j++ {
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
			})
		}
	}
}

func BenchmarkBufferPoolConcurrentStress(b *testing.B) {
	pool := NewPool(8, 256)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for j := 0; j < 3; j++ { // Reduced from 10 to 3
				buf := pool.Get()
				_ = buf.Bytes()
				pool.Put(buf)
			}
		}
	})
}

func BenchmarkBufferPoolMixedWorkload(b *testing.B) {
	pool := NewPool(16, 512)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			switch counter % 4 {
			case 0:
				buf := pool.Get()
				pool.Put(buf)
			case 1:
				buf := pool.Get()
				_ = buf.Bytes()
				pool.Put(buf)
			case 2:
				bufs := make([]*Buffer, 0, 3)
				for j := 0; j < 3; j++ {
					buf := pool.Get()
					bufs = append(bufs, buf)
				}
				for _, buf := range bufs {
					pool.Put(buf)
				}
			case 3:
				buf := pool.Get()
				data := make([]byte, 100)
				copy(buf.Bytes()[:len(data)], data)
				_ = buf.Bytes()[:len(data)]
				pool.Put(buf)
			}
			counter++
		}
	})
}
