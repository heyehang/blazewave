//go:build !js

package blazewave_test

import (
	"bufio"
	"fmt"
	"net"
	"net/http/httptest"
	"testing"

	"github.com/heyehang/blazewave"
	"github.com/heyehang/blazewave/core/pool"
	"github.com/heyehang/blazewave/internal/test/xrand"
)

func BenchmarkAccept(b *testing.B) {
	poolSize := 8
	bufSize := 256

	b.Run(fmt.Sprintf("pool_%d_buf_%d", poolSize, bufSize), func(b *testing.B) {
		rPool := pool.NewPool(poolSize, bufSize)
		wPool := pool.NewPool(poolSize, bufSize)

		acceptOpts := &blazewave.AcceptOptions{
			CommonOptions: blazewave.CommonOptions{
				ReaderPool: rPool,
				WriterPool: wPool,
			},
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set("Connection", "Upgrade")
			r.Header.Set("Upgrade", "websocket")
			r.Header.Set("Sec-WebSocket-Version", "13")
			r.Header.Set("Sec-WebSocket-Key", xrand.Base64(16))

			hijacker := mockHijacker{
				ResponseWriter: w,
				hijack: func() (net.Conn, *bufio.ReadWriter, error) {
					conn1, conn2 := net.Pipe()
					rw := bufio.NewReadWriter(bufio.NewReader(conn2), bufio.NewWriter(conn2))
					return conn1, rw, nil
				},
			}

			conn, err := blazewave.Accept(hijacker, r, acceptOpts)
			if err != nil {
				b.Fatal(err)
			}

			conn.Close(blazewave.StatusNormalClosure, "")
		}
	})

	b.Run("without_pool", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set("Connection", "Upgrade")
			r.Header.Set("Upgrade", "websocket")
			r.Header.Set("Sec-WebSocket-Version", "13")
			r.Header.Set("Sec-WebSocket-Key", xrand.Base64(16))

			hijacker := mockHijacker{
				ResponseWriter: w,
				hijack: func() (net.Conn, *bufio.ReadWriter, error) {
					conn1, conn2 := net.Pipe()
					rw := bufio.NewReadWriter(bufio.NewReader(conn2), bufio.NewWriter(conn2))
					return conn1, rw, nil
				},
			}

			conn, err := blazewave.Accept(hijacker, r, nil)
			if err != nil {
				b.Fatal(err)
			}

			conn.Close(blazewave.StatusNormalClosure, "")
		}
	})
}

func BenchmarkAcceptConcurrent(b *testing.B) {
	poolSize := 8

	b.Run(fmt.Sprintf("pool_%d", poolSize), func(b *testing.B) {
		rPool := pool.NewPool(poolSize, 256)
		wPool := pool.NewPool(poolSize, 256)

		acceptOpts := &blazewave.AcceptOptions{
			CommonOptions: blazewave.CommonOptions{
				ReaderPool: rPool,
				WriterPool: wPool,
			},
		}

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", "/", nil)
				r.Header.Set("Connection", "Upgrade")
				r.Header.Set("Upgrade", "websocket")
				r.Header.Set("Sec-WebSocket-Version", "13")
				r.Header.Set("Sec-WebSocket-Key", xrand.Base64(16))

				hijacker := mockHijacker{
					ResponseWriter: w,
					hijack: func() (net.Conn, *bufio.ReadWriter, error) {
						conn1, conn2 := net.Pipe()
						rw := bufio.NewReadWriter(bufio.NewReader(conn2), bufio.NewWriter(conn2))
						return conn1, rw, nil
					},
				}

				conn, err := blazewave.Accept(hijacker, r, acceptOpts)
				if err != nil {
					b.Fatal(err)
				}

				conn.Close(blazewave.StatusNormalClosure, "")
			}
		})
	})

	b.Run("without_pool", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", "/", nil)
				r.Header.Set("Connection", "Upgrade")
				r.Header.Set("Upgrade", "websocket")
				r.Header.Set("Sec-WebSocket-Version", "13")
				r.Header.Set("Sec-WebSocket-Key", xrand.Base64(16))

				hijacker := mockHijacker{
					ResponseWriter: w,
					hijack: func() (net.Conn, *bufio.ReadWriter, error) {
						conn1, conn2 := net.Pipe()
						rw := bufio.NewReadWriter(bufio.NewReader(conn2), bufio.NewWriter(conn2))
						return conn1, rw, nil
					},
				}

				conn, err := blazewave.Accept(hijacker, r, nil)
				if err != nil {
					b.Fatal(err)
				}

				conn.Close(blazewave.StatusNormalClosure, "")
			}
		})
	})
}

func BenchmarkAcceptOverhead(b *testing.B) {
	b.Run("with_pool", func(b *testing.B) {
		rPool := pool.NewPool(8, 256)
		wPool := pool.NewPool(8, 256)

		acceptOpts := &blazewave.AcceptOptions{
			CommonOptions: blazewave.CommonOptions{
				ReaderPool: rPool,
				WriterPool: wPool,
			},
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set("Connection", "Upgrade")
			r.Header.Set("Upgrade", "websocket")
			r.Header.Set("Sec-WebSocket-Version", "13")
			r.Header.Set("Sec-WebSocket-Key", xrand.Base64(16))

			hijacker := mockHijacker{
				ResponseWriter: w,
				hijack: func() (net.Conn, *bufio.ReadWriter, error) {
					conn1, conn2 := net.Pipe()
					rw := bufio.NewReadWriter(bufio.NewReader(conn2), bufio.NewWriter(conn2))
					return conn1, rw, nil
				},
			}

			conn, err := blazewave.Accept(hijacker, r, acceptOpts)
			if err != nil {
				b.Fatal(err)
			}

			conn.Close(blazewave.StatusNormalClosure, "")
		}
	})

	b.Run("without_pool", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set("Connection", "Upgrade")
			r.Header.Set("Upgrade", "websocket")
			r.Header.Set("Sec-WebSocket-Version", "13")
			r.Header.Set("Sec-WebSocket-Key", xrand.Base64(16))

			hijacker := mockHijacker{
				ResponseWriter: w,
				hijack: func() (net.Conn, *bufio.ReadWriter, error) {
					conn1, conn2 := net.Pipe()
					rw := bufio.NewReadWriter(bufio.NewReader(conn2), bufio.NewWriter(conn2))
					return conn1, rw, nil
				},
			}

			conn, err := blazewave.Accept(hijacker, r, nil)
			if err != nil {
				b.Fatal(err)
			}

			conn.Close(blazewave.StatusNormalClosure, "")
		}
	})
}
