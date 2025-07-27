# 🔥 BlazeWave

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE.txt)
[![Test Coverage](https://img.shields.io/badge/Coverage-65.3%25-brightgreen.svg)](https://github.com/heyehang/blazewave)

> **高性能、事件驱动的 Go WebSocket 库** ⚡

[English](README.md) | [中文](README_zh.md)

BlazeWave 是一个受 [nhooyr/websocket](https://github.com/nhooyr/websocket) 启发并改造的现代、生产就绪的 WebSocket 库。它提供极速性能，支持高性能操作、智能缓冲区池和全面的事件处理。

## ✨ 特性

- **🚀 行业领先性能**: 3.8× 吞吐量提升，2.7× 延迟优化
- **⚡ 高性能架构**: 业界靠前的高性能WebSocket库
- **🎯 事件驱动**: 全面的事件系统，支持中间件
- **🛡️ 生产就绪**: RFC 6455 合规、压缩支持、健壮的错误处理
- **🔧 开发者友好**: 简单 API、类型安全、广泛测试（65.3% 覆盖率）
- **🌐 跨平台**: 支持 Go 1.21+ 在所有主要平台上，包括 WASM

## 📊 性能基准测试

> **测试环境**: Apple M1 Max (10核), 32GB RAM, Go 1.24.5 darwin/arm64

### ⚡ 读写0拷贝和读写0GC指标

**📊 性能指标**

| 指标 | 数值 | 单位 |
|------|------|------|
| **吞吐量** | 82,650 | ops/sec (35.23 MB/s) |
| **延迟** | 13,887 | ns/op (13.9 μs) |
| **内存** | 327 | B/op (16 allocs/op) |
| **效率** | 99.2% | (对比基准) |

### 🚀 自身性能模式对比矩阵

| 指标 | 标准模式 | 高性能模式 | 提升幅度 | 倍数 |
|------|----------|------------|----------|------|
| **吞吐量** | 37,077 ops/sec | 82,650 ops/sec | **+123%** ⚡ | **2.23×** |
| **延迟** | 32,381 ns | 13,887 ns | **-57%** ⚡ | **2.33×** |
| **内存** | 742 B/op | 327 B/op | **-56%** ⚡ | **2.27×** |
| **分配次数** | 32 allocs/op | 16 allocs/op | **-50%** ⚡ | **2.00×** |

### 🏆 高性能模式与其他WebSocket库对比

#### 📊 技术特性对比表

| 库 | 读写0拷贝和读写0GC | 缓冲区池 | 事件驱动 |
|----|---------------------|----------|----------|
| **BlazeWave** | ✅ 原生支持 | ✅ 智能池化 | ✅ 全面事件系统 |
| nhooyr/websocket | ❌ 无 | ❌ 无 | ❌ 无 |
| gorilla/websocket | ❌ 无 | ❌ 无 | ✅ 基础事件 |
| fasthttp/websocket | ❌ 无 | ❌ 无 | ✅ 基础事件 |

#### 🎯 核心优势

**🚀 BlazeWave 核心优势**

| 特性 | 描述 | 优势 |
|------|------|------|
| ⚡ **读写0拷贝和读写0GC** | 零内存拷贝，零GC压力 | 极致性能优化 |
| 🚀 **智能缓冲区池** | 跨连接内存复用，减少GC压力 | 内存效率提升 |
| 🎯 **全面事件系统** | 支持中间件和自定义事件处理，超越基础事件 | 开发体验优化 |
| 🛡️ **生产就绪** | RFC 6455 合规，健壮的错误处理 | 企业级稳定性 |
| 🔧 **开发者友好** | 简单API，类型安全，广泛测试 | 快速上手 |


## 📦 安装

```bash
go get github.com/heyehang/blazewave
```

## ⚡ 性能优势

### 高性能架构
BlazeWave 的高性能设计消除了不必要的内存分配：
- **直接缓冲区访问**: 高频读写操作绕过中间缓冲区
- **内存池复用**: 智能缓冲区池减少 GC 压力

### 内存效率
- **缓冲区池**: 跨连接的可重用缓冲区
- **定时器池**: 跨连接的可重用的定时器
- **GC 友好**: 减少垃圾回收开销

### 高吞吐量
- **高性能处理**: 优化的消息处理流程
- **持续读写**: 高效的网络I/O处理
- **并发优化**: 缓冲区池和定时器优化



## 🚀 快速开始

> **注意**: 以下示例展示标准模式用法。高性能模式的高阶用法请参考 [BlazeWave Pulse](https://github.com/heyehang/blazewave-pulse) 或本地查看案例或本地单元测试。

> **💡 提示**: 高性能模式通过使用优化的缓冲区池和共享 timer 池来实现，适用于生产环境。

### 标准模式

#### 服务器

```go
package main

import (
    "context"
    "log"
    "net/http"
    "github.com/heyehang/blazewave"
    // "github.com/heyehang/blazewave/core/pool"  // 高性能模式需要
    // "github.com/heyehang/blazewave/core/timer"  // 共享timer池需要
)

func main() {
    // 标准模式：使用默认配置
    server := blazewave.NewServer()
    
    // 高性能模式：使用自定义缓冲区池和共享timer池
    // rPool := pool.NewPool(64, 4*1024)
    // wPool := pool.NewPool(64, 4*1024)
    // sharedTimer := timer.NewTimer(100)  // 共享timer池，容量100
    // server := blazewave.NewServer(
    //     blazewave.WithServerReaderPool(rPool),
    //     blazewave.WithServerWriterPool(wPool),
    //     blazewave.WithServerHeartbeatTimer(sharedTimer),
    // )
    
    server.OnTextMessage(func(ctx context.Context, conn *blazewave.Conn, payload []byte) error {
        log.Printf("收到: %s", string(payload))
        return conn.Write(ctx, blazewave.MessageText, payload)
    })
    
    http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        conn, err := server.Accept(w, r)
        if err != nil {
            log.Printf("接受连接失败: %v", err)
            return
        }
        defer conn.Close(blazewave.StatusNormalClosure, "")
    })
    
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

#### 客户端

```go
package main

import (
    "context"
    "log"
    "time"
    "github.com/heyehang/blazewave"
    // "github.com/heyehang/blazewave/core/pool"  // 高性能模式需要
    // "github.com/heyehang/blazewave/core/timer"  // 共享timer池需要
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    // 标准模式：使用默认配置
    client := blazewave.NewClient()
    
    // 高性能模式：使用自定义缓冲区池和共享timer池
    // rPool := pool.NewPool(64, 4*1024)
    // wPool := pool.NewPool(64, 4*1024)
    // sharedTimer := timer.NewTimer(100)  // 共享timer池，容量100
    // client := blazewave.NewClient(
    //     blazewave.WithClientReaderPool(rPool),
    //     blazewave.WithClientWriterPool(wPool),
    //     blazewave.WithClientHeartbeatTimer(sharedTimer),
    // )
    
    // 注册事件处理器
    client.OnConnect(func(ctx context.Context, conn *blazewave.Conn) error {
        log.Println("已连接到服务器!")
        return nil
    })
    
    client.OnTextMessage(func(ctx context.Context, conn *blazewave.Conn, payload []byte) error {
        log.Printf("收到消息: %s", string(payload))
        return nil
    })
    
    client.OnDisconnect(func(ctx context.Context, conn *blazewave.Conn) error {
        log.Println("与服务器断开连接")
        return nil
    })
    
    // 连接到服务器
    conn, _, err := client.Dial(ctx, "ws://localhost:8080/ws", nil)
    if err != nil {
        log.Fatal("连接失败:", err)
    }
    defer conn.Close(blazewave.StatusNormalClosure, "")
    
    // 发送消息
    err = conn.Write(ctx, blazewave.MessageText, []byte("你好，BlazeWave!"))
    if err != nil {
        log.Fatal("写入失败:", err)
    }
    
    // 保持连接一段时间
    time.Sleep(5 * time.Second)
}
```



## 🏗️ 架构

**🏗️ BlazeWave 架构设计**

### 核心组件

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

**性能优化说明**：
- **缓冲区池复用**: 跨连接共享缓冲区，减少高频的内存分配和GC回收
- **Timer 池复用**: 使用 `WithServerHeartbeatTimer()` 和 `WithClientHeartbeatTimer()` 共享 timer 池
- **高性能优化**: 使用优化的缓冲区池和共享 timer 池

## 🎯 最佳实践 & 参考项目

### 核心思想：事件驱动实时应用

BlazeWave 围绕**事件驱动实时应用**的核心思想设计。要查看最佳实现示例和生产就绪模式，请参考我们的参考项目：

#### **[BlazeWave Pulse](https://github.com/heyehang/blazewave-pulse)** ⚡

> **展示 BlazeWave 核心概念的权威参考实现**

- **实时协作**: 多用户交互式应用
- **事件驱动架构**: 完整的事件系统实现
- **生产模式**: 可扩展、可维护的代码结构
- **性能优化**: 高性能、缓冲区池

该项目展示了使用 BlazeWave 构建高性能实时应用的最佳实践。

## 📚 可用 Demo

查看我们的交互式示例：

- **[聊天 Demo](./examples/chat/)**: 带用户管理的实时聊天应用
- ![Chat Demo](/examples/chat/static/chat_example.gif)
- **[共享吉祥物](./examples/shared-mascot/)**: 协作吉祥物移动演示
- ![Mascot Demo Animation](/examples/shared-mascot/static/mascot.gif)
- **[更多示例](./examples/README.md)**: 完整示例列表

## 🙏 致谢

- 受 [nhooyr/websocket](https://github.com/nhooyr/websocket) 启发和改造
- 基于 Go 优秀的标准库构建

## ⭐ Star 历史

[![Star History Chart](https://api.star-history.com/svg?repos=heyehang/blazewave&type=Date)](https://star-history.com/#heyehang/blazewave&Date)

---