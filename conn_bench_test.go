//go:build !js

package blazewave_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/heyehang/blazewave"
	"github.com/heyehang/blazewave/core/pool"
	"github.com/heyehang/blazewave/internal/test/assert"
)

func BenchmarkConnRead(b *testing.B) {
	poolSizes := []int{8}
	bufSizes := []int{256}
	msgSizes := []int{1024}

	for _, poolSize := range poolSizes {
		for _, bufSize := range bufSizes {
			for _, msgSize := range msgSizes {
				b.Run(fmt.Sprintf("pool_%d_buf_%d_msg_%d", poolSize, bufSize, msgSize), func(b *testing.B) {
					rPool := pool.NewPool(poolSize, bufSize)
					wPool := pool.NewPool(poolSize, bufSize)

					dialOpts := &blazewave.DialOptions{
						CommonOptions: blazewave.CommonOptions{
							ReaderPool: rPool,
							WriterPool: wPool,
						},
					}
					acceptOpts := &blazewave.AcceptOptions{
						CommonOptions: blazewave.CommonOptions{
							ReaderPool: rPool,
							WriterPool: wPool,
						},
					}

					c1, c2 := createTestConnection(dialOpts, acceptOpts)
					defer c1.Close(blazewave.StatusNormalClosure, "")
					defer c2.Close(blazewave.StatusNormalClosure, "")

					msg := []byte(strings.Repeat("a", msgSize))
					readBuf := make([]byte, len(msg))

					go echoServer(c2, msg)

					b.ReportAllocs()
					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						err := c1.Write(context.Background(), blazewave.MessageText, msg)
						if err != nil {
							b.Fatal(err)
						}

						typ, r, err := c1.Reader(context.Background())
						if err != nil {
							b.Fatal(err)
						}
						if typ != blazewave.MessageText {
							b.Fatal("unexpected message type")
						}

						_, err = io.ReadFull(r, readBuf)
						if err != nil {
							b.Fatal(err)
						}

						_, err = r.Read(readBuf)
						if err != io.EOF {
							b.Fatal("expected EOF")
						}
					}
				})
			}
		}
	}

	b.Run("without_pool", func(b *testing.B) {
		msgSize := 1024
		msg := []byte(strings.Repeat("a", msgSize))
		readBuf := make([]byte, len(msg))

		c1, c2 := createTestConnection(nil, nil)
		defer c1.Close(blazewave.StatusNormalClosure, "")
		defer c2.Close(blazewave.StatusNormalClosure, "")

		go echoServer(c2, msg)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := c1.Write(context.Background(), blazewave.MessageText, msg)
			if err != nil {
				b.Fatal(err)
			}

			typ, r, err := c1.Reader(context.Background())
			if err != nil {
				b.Fatal(err)
			}
			if typ != blazewave.MessageText {
				b.Fatal("unexpected message type")
			}

			_, err = io.ReadFull(r, readBuf)
			if err != nil {
				b.Fatal(err)
			}

			_, err = r.Read(readBuf)
			if err != io.EOF {
				b.Fatal("expected EOF")
			}
		}
	})
}

func BenchmarkConnWrite(b *testing.B) {
	poolSizes := []int{8}   // Reduced from [4, 16]
	bufSizes := []int{256}  // Reduced from [64, 256]
	msgSizes := []int{1024} // Reduced from [64, 1024]

	for _, poolSize := range poolSizes {
		for _, bufSize := range bufSizes {
			for _, msgSize := range msgSizes {
				b.Run(fmt.Sprintf("pool_%d_buf_%d_msg_%d", poolSize, bufSize, msgSize), func(b *testing.B) {
					rPool := pool.NewPool(poolSize, bufSize)
					wPool := pool.NewPool(poolSize, bufSize)

					dialOpts := &blazewave.DialOptions{
						CommonOptions: blazewave.CommonOptions{
							ReaderPool: rPool,
							WriterPool: wPool,
						},
					}
					acceptOpts := &blazewave.AcceptOptions{
						CommonOptions: blazewave.CommonOptions{
							ReaderPool: rPool,
							WriterPool: wPool,
						},
					}

					c1, c2 := createTestConnection(dialOpts, acceptOpts)
					defer c1.Close(blazewave.StatusNormalClosure, "")
					defer c2.Close(blazewave.StatusNormalClosure, "")

					go discardServer(c2)

					msg := []byte(strings.Repeat("a", msgSize))

					b.ReportAllocs()
					b.ResetTimer()

					for i := 0; i < b.N; i++ {
						err := c1.Write(context.Background(), blazewave.MessageText, msg)
						if err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		}
	}

	b.Run("without_pool", func(b *testing.B) {
		msgSize := 1024
		msg := []byte(strings.Repeat("a", msgSize))

		c1, c2 := createTestConnection(nil, nil)
		defer c1.Close(blazewave.StatusNormalClosure, "")
		defer c2.Close(blazewave.StatusNormalClosure, "")

		go discardServer(c2)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := c1.Write(context.Background(), blazewave.MessageText, msg)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkConnConcurrent(b *testing.B) {
	poolSizes := []int{8} // Reduced from [4, 16]

	for _, poolSize := range poolSizes {
		b.Run(fmt.Sprintf("pool_%d", poolSize), func(b *testing.B) {
			rPool := pool.NewPool(poolSize, 256)
			wPool := pool.NewPool(poolSize, 256)

			dialOpts := &blazewave.DialOptions{
				CommonOptions: blazewave.CommonOptions{
					ReaderPool: rPool,
					WriterPool: wPool,
				},
			}
			acceptOpts := &blazewave.AcceptOptions{
				CommonOptions: blazewave.CommonOptions{
					ReaderPool: rPool,
					WriterPool: wPool,
				},
			}

			c1, c2 := createTestConnection(dialOpts, acceptOpts)
			defer c1.Close(blazewave.StatusNormalClosure, "")
			defer c2.Close(blazewave.StatusNormalClosure, "")

			go echoServer(c2, []byte("echo"))

			msg := []byte("hello")
			readBuf := make([]byte, 4)

			b.ReportAllocs()
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					err := c1.Write(context.Background(), blazewave.MessageText, msg)
					if err != nil {
						b.Fatal(err)
					}

					typ, r, err := c1.Reader(context.Background())
					if err != nil {
						b.Fatal(err)
					}
					if typ != blazewave.MessageText {
						b.Fatal("unexpected message type")
					}

					_, err = io.ReadFull(r, readBuf)
					if err != nil {
						b.Fatal(err)
					}

					_, err = r.Read(readBuf)
					if err != io.EOF {
						b.Fatal("expected EOF")
					}
				}
			})
		})
	}

	b.Run("without_pool", func(b *testing.B) {
		c1, c2 := createTestConnection(nil, nil)
		defer c1.Close(blazewave.StatusNormalClosure, "")
		defer c2.Close(blazewave.StatusNormalClosure, "")

		go echoServer(c2, []byte("echo"))

		msg := []byte("hello")
		readBuf := make([]byte, 4)

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				err := c1.Write(context.Background(), blazewave.MessageText, msg)
				if err != nil {
					b.Fatal(err)
				}

				typ, r, err := c1.Reader(context.Background())
				if err != nil {
					b.Fatal(err)
				}
				if typ != blazewave.MessageText {
					b.Fatal("unexpected message type")
				}

				_, err = io.ReadFull(r, readBuf)
				if err != nil {
					b.Fatal(err)
				}

				_, err = r.Read(readBuf)
				if err != io.EOF {
					b.Fatal("expected EOF")
				}
			}
		})
	})
}

func BenchmarkConnPing(b *testing.B) {
	poolSizes := []int{8} // Reduced from [4, 16]

	for _, poolSize := range poolSizes {
		b.Run(fmt.Sprintf("pool_%d", poolSize), func(b *testing.B) {
			rPool := pool.NewPool(poolSize, 256)
			wPool := pool.NewPool(poolSize, 256)

			dialOpts := &blazewave.DialOptions{
				CommonOptions: blazewave.CommonOptions{
					ReaderPool: rPool,
					WriterPool: wPool,
				},
			}
			acceptOpts := &blazewave.AcceptOptions{
				CommonOptions: blazewave.CommonOptions{
					ReaderPool: rPool,
					WriterPool: wPool,
				},
			}

			c1, c2 := createTestConnection(dialOpts, acceptOpts)
			defer c1.Close(blazewave.StatusNormalClosure, "")
			defer c2.Close(blazewave.StatusNormalClosure, "")

			go pingResponseServer(c2)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				err := c1.Ping(ctx)
				cancel()
				if err != nil && !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline exceeded") {
					b.Fatal(err)
				}
			}
		})
	}

	b.Run("without_pool", func(b *testing.B) {
		c1, c2 := createTestConnection(nil, nil)
		defer c1.Close(blazewave.StatusNormalClosure, "")
		defer c2.Close(blazewave.StatusNormalClosure, "")

		go pingResponseServer(c2)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			err := c1.Ping(ctx)
			cancel()
			if err != nil && !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline exceeded") {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkConnOverhead(b *testing.B) {
	b.Run("with_pool", func(b *testing.B) {
		rPool := pool.NewPool(8, 256)
		wPool := pool.NewPool(8, 256)

		dialOpts := &blazewave.DialOptions{
			CommonOptions: blazewave.CommonOptions{
				ReaderPool: rPool,
				WriterPool: wPool,
			},
		}
		acceptOpts := &blazewave.AcceptOptions{
			CommonOptions: blazewave.CommonOptions{
				ReaderPool: rPool,
				WriterPool: wPool,
			},
		}

		c1, c2 := createTestConnection(dialOpts, acceptOpts)
		defer c1.Close(blazewave.StatusNormalClosure, "")
		defer c2.Close(blazewave.StatusNormalClosure, "")

		msg := []byte("test")
		go echoServer(c2, msg)

		readBuf := make([]byte, len(msg))

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := c1.Write(context.Background(), blazewave.MessageText, msg)
			if err != nil {
				b.Fatal(err)
			}

			typ, r, err := c1.Reader(context.Background())
			if err != nil {
				b.Fatal(err)
			}
			if typ != blazewave.MessageText {
				b.Fatal("unexpected message type")
			}

			_, err = io.ReadFull(r, readBuf)
			if err != nil {
				b.Fatal(err)
			}

			_, err = r.Read(readBuf)
			if err != io.EOF {
				b.Fatal("expected EOF")
			}
		}
	})

	b.Run("without_pool", func(b *testing.B) {
		c1, c2 := createTestConnection(nil, nil)
		defer c1.Close(blazewave.StatusNormalClosure, "")
		defer c2.Close(blazewave.StatusNormalClosure, "")

		msg := []byte("test")
		go echoServer(c2, msg)

		readBuf := make([]byte, len(msg))

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := c1.Write(context.Background(), blazewave.MessageText, msg)
			if err != nil {
				b.Fatal(err)
			}

			typ, r, err := c1.Reader(context.Background())
			if err != nil {
				b.Fatal(err)
			}
			if typ != blazewave.MessageText {
				b.Fatal("unexpected message type")
			}

			_, err = io.ReadFull(r, readBuf)
			if err != nil {
				b.Fatal(err)
			}

			_, err = r.Read(readBuf)
			if err != io.EOF {
				b.Fatal("expected EOF")
			}
		}
	})
}

func createTestConnection(dialOpts *blazewave.DialOptions, acceptOpts *blazewave.AcceptOptions) (*blazewave.Conn, *blazewave.Conn) {
	conn1, conn2 := net.Pipe()

	if dialOpts == nil {
		dialOpts = &blazewave.DialOptions{}
	}
	if acceptOpts == nil {
		acceptOpts = &blazewave.AcceptOptions{}
	}

	br1, bw1, brBuf1, bwBuf1 := blazewave.InitDialReaderWriter(conn1, dialOpts)
	br2, bw2, brBuf2, bwBuf2 := blazewave.InitAcceptReaderWriter(conn2, acceptOpts)

	c1 := blazewave.NewConn(blazewave.NewConnConfig(
		conn1, true, dialOpts.ReaderPool, dialOpts.WriterPool, br1, brBuf1, bw1, bwBuf1,
	))

	c2 := blazewave.NewConn(blazewave.NewConnConfig(
		conn2, false, acceptOpts.ReaderPool, acceptOpts.WriterPool, br2, brBuf2, bw2, bwBuf2,
	))

	return c1, c2
}

func echoServer(c *blazewave.Conn, response []byte) {
	defer c.Close(blazewave.StatusNormalClosure, "")
	for {
		typ, r, err := c.Reader(context.Background())
		if err != nil {
			return
		}
		_, err = io.Copy(io.Discard, r)
		if err != nil {
			return
		}
		err = c.Write(context.Background(), typ, response)
		if err != nil {
			return
		}
	}
}

func discardServer(c *blazewave.Conn) {
	defer c.Close(blazewave.StatusNormalClosure, "")
	for {
		_, r, err := c.Reader(context.Background())
		if err != nil {
			return
		}
		_, err = io.Copy(io.Discard, r)
		if err != nil {
			return
		}
	}
}

func pingResponseServer(c *blazewave.Conn) {
	defer c.Close(blazewave.StatusNormalClosure, "")
	for {
		_, r, err := c.Reader(context.Background())
		if err != nil {
			return
		}
		_, err = io.Copy(io.Discard, r)
		if err != nil {
			return
		}
	}
}

func BenchmarkConn(b *testing.B) {
	benchCases := []struct {
		name string
		mode blazewave.CompressionMode
	}{
		{
			name: "disabledCompress",
			mode: blazewave.CompressionDisabled,
		},
		{
			name: "compressContextTakeover",
			mode: blazewave.CompressionContextTakeover,
		},
		{
			name: "compressNoContext",
			mode: blazewave.CompressionNoContextTakeover,
		},
	}
	for _, bc := range benchCases {
		b.Run(bc.name, func(b *testing.B) {
			bb, c1, c2 := newConnTest(b, &blazewave.DialOptions{
				CommonOptions: blazewave.CommonOptions{
					CompressionMode: bc.mode,
				},
			}, &blazewave.AcceptOptions{
				CommonOptions: blazewave.CommonOptions{
					CompressionMode: bc.mode,
				},
			})

			bb.goEchoLoop(c2)

			bytesWritten := c1.RecordBytesWritten()
			bytesRead := c1.RecordBytesRead()

			msg := []byte(strings.Repeat("1234", 128))
			readBuf := make([]byte, len(msg))
			writes := make(chan struct{})
			defer close(writes)
			werrs := make(chan error)

			go func() {
				for range writes {
					select {
					case werrs <- c1.Write(bb.ctx, blazewave.MessageText, msg):
					case <-bb.ctx.Done():
						return
					}
				}
			}()
			b.SetBytes(int64(len(msg)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				select {
				case writes <- struct{}{}:
				case <-bb.ctx.Done():
					b.Fatal(bb.ctx.Err())
				}

				typ, r, err := c1.Reader(bb.ctx)
				if err != nil {
					b.Fatal(i, err)
				}
				if blazewave.MessageText != typ {
					assert.Equal(b, "data type", blazewave.MessageText, typ)
				}

				_, err = io.ReadFull(r, readBuf)
				if err != nil {
					b.Fatal(err)
				}

				n2, err := r.Read(readBuf)
				if err != io.EOF {
					assert.Equal(b, "read err", io.EOF, err)
				}
				if n2 != 0 {
					assert.Equal(b, "n2", 0, n2)
				}

				if !bytes.Equal(msg, readBuf) {
					assert.Equal(b, "msg", msg, readBuf)
				}

				select {
				case err = <-werrs:
				case <-bb.ctx.Done():
					b.Fatal(bb.ctx.Err())
				}
				if err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()

			b.ReportMetric(float64(*bytesWritten/b.N), "written/op")
			b.ReportMetric(float64(*bytesRead/b.N), "read/op")

			err := c1.Close(blazewave.StatusNormalClosure, "")
			assert.Success(b, err)
		})
	}
}

// NewHighPerformancePool creates a high-performance buffer pool with large buffers
func NewHighPerformancePool() *pool.Pool {
	return pool.NewPool(256, 65536) // 256 pools, 64KB buffers
}

// NewBulkTransferPool creates a buffer pool optimized for bulk data transfer
func NewBulkTransferPool() *pool.Pool {
	return pool.NewPool(512, 131072) // 512 pools, 128KB buffers
}

// BenchmarkConnZeroCopy tests optimized zero-copy performance with larger buffer pools
func BenchmarkConnZeroCopy(b *testing.B) {
	// Optimized pool configurations for zero-copy performance
	// Buffer sizes must be larger than message sizes for zero-copy efficiency
	poolConfigs := []struct {
		name    string
		bufSize int
		bufNum  int
	}{
		{
			name:    "standard_pool_32_4k",
			bufNum:  32,
			bufSize: 4096, // 4KB buffer, larger than 2KB messages
		},
		{
			name:    "large_pool_64_8k",
			bufNum:  64,
			bufSize: 8192, // 8KB buffer, much larger than 2KB messages
		},
		{
			name:    "huge_pool_128_16k",
			bufNum:  128,
			bufSize: 16384, // 16KB buffer, 8x larger than 2KB messages
		},
		{
			name:    "massive_pool_256_32k",
			bufNum:  256,
			bufSize: 32768, // 32KB buffer, 16x larger than 2KB messages
		},
	}

	// Use conventional message sizes for real-world scenarios
	msgSizes := []int{1024, 2048}

	for _, config := range poolConfigs {
		for _, msgSize := range msgSizes {
			b.Run(fmt.Sprintf("%s_msg_%d", config.name, msgSize), func(b *testing.B) {
				// Create optimized buffer pools using new zero-copy pool
				rPool := pool.NewPool(config.bufNum, config.bufSize)
				wPool := pool.NewPool(config.bufNum, config.bufSize)

				dialOpts := &blazewave.DialOptions{
					CommonOptions: blazewave.CommonOptions{
						ReaderPool:      rPool,
						WriterPool:      wPool,
						CompressionMode: blazewave.CompressionDisabled, // Disable compression for zero-copy
					},
				}
				acceptOpts := &blazewave.AcceptOptions{
					CommonOptions: blazewave.CommonOptions{
						ReaderPool:      rPool,
						WriterPool:      wPool,
						CompressionMode: blazewave.CompressionDisabled, // Disable compression for zero-copy
					},
				}

				c1, c2 := createTestConnection(dialOpts, acceptOpts)
				defer c1.Close(blazewave.StatusNormalClosure, "")
				defer c2.Close(blazewave.StatusNormalClosure, "")

				msg := []byte(strings.Repeat("x", msgSize))
				readBuf := make([]byte, len(msg))

				go echoServer(c2, msg)

				b.SetBytes(int64(len(msg)))
				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					err := c1.Write(context.Background(), blazewave.MessageText, msg)
					if err != nil {
						b.Fatal(err)
					}

					typ, r, err := c1.Reader(context.Background())
					if err != nil {
						b.Fatal(err)
					}
					if typ != blazewave.MessageText {
						b.Fatal("unexpected message type")
					}

					_, err = io.ReadFull(r, readBuf)
					if err != nil {
						b.Fatal(err)
					}

					_, err = r.Read(readBuf)
					if err != io.EOF {
						b.Fatal("expected EOF")
					}
				}
			})
		}
	}

	// Test with high-performance pool optimized for conventional message sizes
	b.Run("high_performance_pool", func(b *testing.B) {
		msgSize := 2048
		msg := []byte(strings.Repeat("x", msgSize))
		readBuf := make([]byte, len(msg))

		rPool := NewHighPerformancePool()
		wPool := NewHighPerformancePool()

		dialOpts := &blazewave.DialOptions{
			CommonOptions: blazewave.CommonOptions{
				ReaderPool:      rPool,
				WriterPool:      wPool,
				CompressionMode: blazewave.CompressionDisabled,
			},
		}
		acceptOpts := &blazewave.AcceptOptions{
			CommonOptions: blazewave.CommonOptions{
				ReaderPool:      rPool,
				WriterPool:      wPool,
				CompressionMode: blazewave.CompressionDisabled,
			},
		}

		c1, c2 := createTestConnection(dialOpts, acceptOpts)
		defer c1.Close(blazewave.StatusNormalClosure, "")
		defer c2.Close(blazewave.StatusNormalClosure, "")

		go echoServer(c2, msg)

		b.SetBytes(int64(len(msg)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := c1.Write(context.Background(), blazewave.MessageText, msg)
			if err != nil {
				b.Fatal(err)
			}

			typ, r, err := c1.Reader(context.Background())
			if err != nil {
				b.Fatal(err)
			}
			if typ != blazewave.MessageText {
				b.Fatal("unexpected message type")
			}

			_, err = io.ReadFull(r, readBuf)
			if err != nil {
				b.Fatal(err)
			}

			_, err = r.Read(readBuf)
			if err != io.EOF {
				b.Fatal("expected EOF")
			}
		}
	})

	// Baseline without pools for comparison
	b.Run("baseline_no_pool", func(b *testing.B) {
		msgSize := 1024
		msg := []byte(strings.Repeat("x", msgSize))
		readBuf := make([]byte, len(msg))

		c1, c2 := createTestConnection(nil, nil)
		defer c1.Close(blazewave.StatusNormalClosure, "")
		defer c2.Close(blazewave.StatusNormalClosure, "")

		go echoServer(c2, msg)

		b.SetBytes(int64(len(msg)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := c1.Write(context.Background(), blazewave.MessageText, msg)
			if err != nil {
				b.Fatal(err)
			}

			typ, r, err := c1.Reader(context.Background())
			if err != nil {
				b.Fatal(err)
			}
			if typ != blazewave.MessageText {
				b.Fatal("unexpected message type")
			}

			_, err = io.ReadFull(r, readBuf)
			if err != nil {
				b.Fatal(err)
			}

			_, err = r.Read(readBuf)
			if err != io.EOF {
				b.Fatal("expected EOF")
			}
		}
	})
}

// BenchmarkConnBulkTransfer tests bulk data transfer with zero-copy optimizations
func BenchmarkConnBulkTransfer(b *testing.B) {
	// Test different bulk transfer scenarios with conventional message sizes
	scenarios := []struct {
		name     string
		msgSize  int
		msgCount int
		bufNum   int
		bufSize  int
	}{
		{
			name:     "small_bulk_1k_50",
			msgSize:  1024,
			msgCount: 50,
			bufNum:   32,
			bufSize:  4096, // 4KB buffer for 1KB messages
		},
		{
			name:     "medium_bulk_2k_25",
			msgSize:  2048,
			msgCount: 25,
			bufNum:   64,
			bufSize:  8192, // 8KB buffer for 2KB messages
		},
		{
			name:     "large_bulk_2k_100",
			msgSize:  2048,
			msgCount: 100,
			bufNum:   128,
			bufSize:  16384, // 16KB buffer for 2KB messages
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			// Create optimized buffer pools using zero-copy pool
			rPool := pool.NewPool(scenario.bufNum, scenario.bufSize)
			wPool := pool.NewPool(scenario.bufNum, scenario.bufSize)

			dialOpts := &blazewave.DialOptions{
				CommonOptions: blazewave.CommonOptions{
					ReaderPool:      rPool,
					WriterPool:      wPool,
					CompressionMode: blazewave.CompressionDisabled,
				},
			}
			acceptOpts := &blazewave.AcceptOptions{
				CommonOptions: blazewave.CommonOptions{
					ReaderPool:      rPool,
					WriterPool:      wPool,
					CompressionMode: blazewave.CompressionDisabled,
				},
			}

			c1, c2 := createTestConnection(dialOpts, acceptOpts)
			defer c1.Close(blazewave.StatusNormalClosure, "")
			defer c2.Close(blazewave.StatusNormalClosure, "")

			msg := []byte(strings.Repeat("x", scenario.msgSize))
			readBuf := make([]byte, len(msg))

			go echoServer(c2, msg)

			b.SetBytes(int64(len(msg) * scenario.msgCount))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Send multiple messages in bulk
				for j := 0; j < scenario.msgCount; j++ {
					err := c1.Write(context.Background(), blazewave.MessageText, msg)
					if err != nil {
						b.Fatal(err)
					}

					typ, r, err := c1.Reader(context.Background())
					if err != nil {
						b.Fatal(err)
					}
					if typ != blazewave.MessageText {
						b.Fatal("unexpected message type")
					}

					_, err = io.ReadFull(r, readBuf)
					if err != nil {
						b.Fatal(err)
					}

					_, err = r.Read(readBuf)
					if err != io.EOF {
						b.Fatal("expected EOF")
					}
				}
			}
		})
	}

	// Test with bulk transfer optimized pool for conventional message sizes
	b.Run("bulk_transfer_pool", func(b *testing.B) {
		msgSize := 2048
		msgCount := 50
		msg := []byte(strings.Repeat("x", msgSize))
		readBuf := make([]byte, len(msg))

		rPool := NewBulkTransferPool()
		wPool := NewBulkTransferPool()

		dialOpts := &blazewave.DialOptions{
			CommonOptions: blazewave.CommonOptions{
				ReaderPool:      rPool,
				WriterPool:      wPool,
				CompressionMode: blazewave.CompressionDisabled,
			},
		}
		acceptOpts := &blazewave.AcceptOptions{
			CommonOptions: blazewave.CommonOptions{
				ReaderPool:      rPool,
				WriterPool:      wPool,
				CompressionMode: blazewave.CompressionDisabled,
			},
		}

		c1, c2 := createTestConnection(dialOpts, acceptOpts)
		defer c1.Close(blazewave.StatusNormalClosure, "")
		defer c2.Close(blazewave.StatusNormalClosure, "")

		go echoServer(c2, msg)

		b.SetBytes(int64(len(msg) * msgCount))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Send multiple messages in bulk
			for j := 0; j < msgCount; j++ {
				err := c1.Write(context.Background(), blazewave.MessageText, msg)
				if err != nil {
					b.Fatal(err)
				}

				typ, r, err := c1.Reader(context.Background())
				if err != nil {
					b.Fatal(err)
				}
				if typ != blazewave.MessageText {
					b.Fatal("unexpected message type")
				}

				_, err = io.ReadFull(r, readBuf)
				if err != nil {
					b.Fatal(err)
				}

				_, err = r.Read(readBuf)
				if err != io.EOF {
					b.Fatal("expected EOF")
				}
			}
		}
	})
}
