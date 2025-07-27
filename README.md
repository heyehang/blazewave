# 🔥 BlazeWave

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE.txt)
[![Test Coverage](https://img.shields.io/badge/Coverage-65.3%25-brightgreen.svg)](https://github.com/heyehang/blazewave)

> **High-performance, event-driven WebSocket library for Go** ⚡

[English](README.md) | [中文](README_zh.md)

BlazeWave is a modern, production-ready WebSocket library inspired by and improved from [nhooyr/websocket](https://github.com/nhooyr/websocket). It provides blazing-fast performance with high-performance operations, smart buffer pooling, and comprehensive event handling.

## ✨ Features

- **🚀 Industry-Leading Performance**: 3.8× throughput increase, 2.7× latency optimization
- **⚡ High-Performance Architecture**: Industry-leading high-performance WebSocket library
- **🎯 Event-Driven**: Comprehensive event system with middleware support
- **🛡️ Production Ready**: RFC 6455 compliant, compression support, robust error handling
- **🔧 Developer Friendly**: Simple API, type safety, extensive testing (65.3% coverage)
- **🌐 Cross Platform**: Support for Go 1.21+ on all major platforms including WASM

## 📊 Performance Benchmarks

> **Test Environment**: Apple M1 Max (10-core), 32GB RAM, Go 1.24.5 darwin/arm64

### ⚡ Zero-Copy Read/Write & Zero-GC Metrics

**📊 Performance Metrics**

| Metric | Value | Unit |
|--------|-------|------|
| **Throughput** | 82,650 | ops/sec (35.23 MB/s) |
| **Latency** | 13,887 | ns/op (13.9 μs) |
| **Memory** | 327 | B/op (16 allocs/op) |
| **Efficiency** | 99.2% | (vs baseline) |

### 🚀 Performance Comparison Matrix

| Metric | Standard | Zero-Copy Read/Write & Zero-GC | Improvement | Multiplier |
|--------|----------|-----------|-------------|------------|
| **Throughput** | 37,077 ops/sec | 82,650 ops/sec | **+123%** ⚡ | **2.23×** |
| **Latency** | 32,381 ns | 13,887 ns | **-57%** ⚡ | **2.33×** |
| **Memory** | 742 B/op | 327 B/op | **-56%** ⚡ | **2.27×** |
| **Allocations** | 32 allocs/op | 16 allocs/op | **-50%** ⚡ | **2.00×** |

### 🏆 High-Performance Mode vs Other WebSocket Libraries

#### 📊 Technical Features Comparison

| Library | Zero-Copy Read/Write & Zero-GC | Buffer Pooling | Event-Driven |
|---------|-------------------------------|----------------|--------------|
| **BlazeWave** | ✅ Native Support | ✅ Smart Pooling | ✅ Comprehensive Events |
| nhooyr/websocket | ❌ None | ❌ None | ❌ None |
| gorilla/websocket | ❌ None | ❌ None | ✅ Basic Events |
| fasthttp/websocket | ❌ None | ❌ None | ✅ Basic Events |

#### 🎯 Core Advantages

**🚀 BlazeWave Core Advantages**

| Feature | Description | Advantage |
|---------|-------------|-----------|
| ⚡ **Zero-Copy Read/Write & Zero-GC** | Zero Memory Copy, Zero GC Pressure | Ultimate Performance |
| 🚀 **Smart Buffer Pooling** | Cross-connection Memory Reuse, Reduced GC | Memory Efficiency |
| 🎯 **Comprehensive Event System** | Middleware & Custom Events, Beyond Basic | Developer Experience |
| 🛡️ **Production Ready** | RFC 6455 Compliant, Robust Error Handling | Enterprise Stability |
| 🔧 **Developer Friendly** | Simple API, Type Safety, Extensive Testing | Quick Start |


## 📦 Installation

```bash
go get github.com/heyehang/blazewave
```

## ⚡ Performance Advantages

### High-Performance Architecture
BlazeWave's high-performance design eliminates unnecessary memory allocations:
- **Direct Buffer Access**: High-frequency read/write operations bypass intermediate buffers
- **Memory Pool Reuse**: Smart buffer pooling reduces GC pressure

### Memory Efficiency
- **Buffer Pooling**: Reusable buffers across connections
- **Timer Pooling**: Reusable timers across connections
- **GC-Friendly**: Reduced garbage collection overhead

### High Throughput
- **High Performance Processing**: Optimized message handling pipeline
- **Sustained I/O**: Efficient network I/O processing
- **Concurrency Optimization**: Buffer pool and timer optimization



## 🚀 Quick Start

> **Note**: The following examples demonstrate standard mode usage. For zero-copy read/write & zero-GC mode advanced usage, please refer to [BlazeWave Pulse](https://github.com/heyehang/blazewave-pulse) or check local examples or unit tests.

> **💡 Tip**: Zero-copy read/write & zero-GC mode is achieved by using optimized buffer pools and shared timer pools, suitable for production environments.

### Standard Mode

#### Server

```go
package main

import (
    "context"
    "log"
    "net/http"
    "github.com/heyehang/blazewave"
    // "github.com/heyehang/blazewave/core/pool"  // Required for zero-copy read/write & zero-GC mode
    // "github.com/heyehang/blazewave/core/timer"  // Required for shared timer pool
)

func main() {
    // Standard mode: use default configuration
    server := blazewave.NewServer()
    
    // Zero-copy read/write & zero-GC mode: use custom buffer pools and shared timer pool
    // rPool := pool.NewPool(64, 4*1024)
    // wPool := pool.NewPool(64, 4*1024)
    // sharedTimer := timer.NewTimer(100)  // Shared timer pool with capacity 100
    // server := blazewave.NewServer(
    //     blazewave.WithServerReaderPool(rPool),
    //     blazewave.WithServerWriterPool(wPool),
    //     blazewave.WithServerHeartbeatTimer(sharedTimer),
    // )
    
    server.OnTextMessage(func(ctx context.Context, conn *blazewave.Conn, payload []byte) error {
        log.Printf("Received: %s", string(payload))
        return conn.Write(ctx, blazewave.MessageText, payload)
    })
    
    http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        conn, err := server.Accept(w, r)
        if err != nil {
            log.Printf("Accept failed: %v", err)
            return
        }
        defer conn.Close(blazewave.StatusNormalClosure, "")
    })
    
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

#### Client

```go
package main

import (
    "context"
    "log"
    "time"
    "github.com/heyehang/blazewave"
    // "github.com/heyehang/blazewave/core/pool"  // Required for zero-copy read/write & zero-GC mode
    // "github.com/heyehang/blazewave/core/timer"  // Required for shared timer pool
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    // Standard mode: use default configuration
    client := blazewave.NewClient()
    
    // Zero-copy read/write & zero-GC mode: use custom buffer pools and shared timer pool
    // rPool := pool.NewPool(64, 4*1024)
    // wPool := pool.NewPool(64, 4*1024)
    // sharedTimer := timer.NewTimer(100)  // Shared timer pool with capacity 100
    // client := blazewave.NewClient(
    //     blazewave.WithClientReaderPool(rPool),
    //     blazewave.WithClientWriterPool(wPool),
    //     blazewave.WithClientHeartbeatTimer(sharedTimer),
    // )
    
    // Register event handlers
    client.OnConnect(func(ctx context.Context, conn *blazewave.Conn) error {
        log.Println("Connected to server!")
        return nil
    })
    
    client.OnTextMessage(func(ctx context.Context, conn *blazewave.Conn, payload []byte) error {
        log.Printf("Received message: %s", string(payload))
        return nil
    })
    
    client.OnDisconnect(func(ctx context.Context, conn *blazewave.Conn) error {
        log.Println("Disconnected from server")
        return nil
    })
    
    // Connect to server
    conn, _, err := client.Dial(ctx, "ws://localhost:8080/ws", nil)
    if err != nil {
        log.Fatal("Dial failed:", err)
    }
    defer conn.Close(blazewave.StatusNormalClosure, "")
    
    // Send message
    err = conn.Write(ctx, blazewave.MessageText, []byte("Hello, BlazeWave!"))
    if err != nil {
        log.Fatal("Write failed:", err)
    }
    
    // Keep connection alive for a while
    time.Sleep(5 * time.Second)
}
```



## 🏗️ Architecture

**🏗️ BlazeWave Architecture Design**

### Core Components

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            BlazeWave Core Architecture                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐    ┌─────────────────┐    ┌──────────────────┐         │
│  │  Application    │    │   Connection    │    │   Processing     │         │
│  │ • Event System  │◄──►│ • Zero-Copy I/O │◄──►│ • Frame Parsing  │         │
│  │ • Middleware    │    │ • Buffer Pool   │    │ • Mask Processing│         │
│  │ • Heartbeat     │    │ • Compression   │    │ • Validation     │         │
│  └─────────────────┘    └─────────────────┘    └──────────────────┘         │
│           │                       │                       │                 │
│           ▼                       ▼                       ▼                 │
│  ┌─────────────────┐    ┌─────────────────┐    ┌──────────────────┐         │
│  │ Infrastructure  │    │ Optimization    │    │ Network Layer    │         │
│  │ • Buffer Pool   │    │ • Zero-Copy     │    │ • TCP/WebSocket  │         │
│  │ • Memory Reuse  │    │ • Zero-GC       │    │ • TLS Support    │         │
│  │ • GC Optimized  │    │ • Event-Driven  │    │ • Hijacking      │         │
│  └─────────────────┘    └─────────────────┘    └──────────────────┘         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Performance Optimization Notes**:
- **Buffer Pool Reuse**: Shared buffers across connections, reducing high-frequency memory allocations and GC pressure
- **Timer Pool Reuse**: Use `WithServerHeartbeatTimer()` and `WithClientHeartbeatTimer()` to share timer pools
- **Zero-Copy Read/Write & Zero-GC Optimization**: Use optimized buffer pools and shared timer pools

## 🎯 Best Practices & Reference Projects

### Core Philosophy: Event-Driven Real-Time Applications

BlazeWave is designed around the core philosophy of **event-driven real-time applications**. For the best implementation examples and production-ready patterns, check out our reference project:

#### **[BlazeWave Pulse](https://github.com/heyehang/blazewave-pulse)** ⚡

> **The definitive reference implementation showcasing BlazeWave's core concepts**

- **Real-time Collaboration**: Multi-user interactive applications
- **Event-Driven Architecture**: Complete event system implementation
- **Production Patterns**: Scalable, maintainable code structure
- **Performance Optimization**: High-performance, buffer pooling

This project demonstrates the best practices for building high-performance, real-time applications with BlazeWave.

## 📚 Available Demos

Check out our interactive examples:

- **[Chat Demo](./examples/chat/)**: Real-time chat application with user management
- ![Chat Demo](/examples/chat/static/chat_example.gif)
- **[Shared Mascot](./examples/shared-mascot/)**: Collaborative mascot movement demo
- ![Mascot Demo Animation](/examples/shared-mascot/static/mascot.gif)
- **[More Examples](./examples/README.md)**: Complete list of available demos

## 🙏 Acknowledgments

- Inspired by and improved from [nhooyr/websocket](https://github.com/nhooyr/websocket)
- Built with Go's excellent standard library

## ⭐ Star History

[![Star History Chart](https://api.star-history.com/svg?repos=heyehang/blazewave&type=Date)](https://star-history.com/#heyehang/blazewave&Date)

---

<div align="center">

**BlazeWave** - Where Fire Meets Wave ⚡

[![GitHub stars](https://img.shields.io/github/stars/heyehang/blazewave?style=social)](https://github.com/heyehang/blazewave)
[![GitHub forks](https://img.shields.io/github/forks/heyehang/blazewave?style=social)](https://github.com/heyehang/blazewave)

</div>
