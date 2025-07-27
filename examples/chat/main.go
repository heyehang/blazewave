package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/heyehang/blazewave"
	"github.com/heyehang/blazewave/core/pool"
	"github.com/heyehang/blazewave/core/timer"
)

type Message struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	User    string `json:"user,omitempty"`
	Time    string `json:"time"`
}

type ChatServer struct {
	server      *blazewave.Server
	connections map[*blazewave.Conn]string // conn -> username
	mu          sync.RWMutex
}

func NewChatServer() *ChatServer {
	rPool := pool.NewPool(64, 4*1024)
	wPool := pool.NewPool(64, 4*1024)
	sharedTimer := timer.NewTimer(100)
	cs := &ChatServer{
		server: blazewave.NewServer(
			blazewave.WithServerReaderPool(rPool),
			blazewave.WithServerWriterPool(wPool),
			blazewave.WithServerHeartbeatTimer(sharedTimer),
			blazewave.WithServerSubprotocols([]string{"chat"})),
		connections: make(map[*blazewave.Conn]string),
	}
	cs.registerEvents()
	return cs
}

func (cs *ChatServer) registerEvents() {
	cs.server.OnConnect(func(ctx context.Context, conn *blazewave.Conn) error {
		cs.mu.Lock()
		cs.connections[conn] = ""
		cs.mu.Unlock()
		return cs.send(conn, Message{
			Type:    "system",
			Content: "Welcome! Please send your username.",
			Time:    now(),
		})
	})

	cs.server.OnDisconnect(func(ctx context.Context, conn *blazewave.Conn) error {
		cs.removeConn(conn, "disconnected")
		return nil
	})

	cs.server.OnTextMessage(func(ctx context.Context, conn *blazewave.Conn, payload []byte) error {
		var msg Message
		if err := json.Unmarshal(payload, &msg); err != nil || msg.Type == "" {
			username := string(payload)
			cs.mu.Lock()
			if cs.connections[conn] == "" {
				cs.connections[conn] = username
				cs.mu.Unlock()
				cs.broadcast(Message{
					Type:    "system",
					Content: fmt.Sprintf("%s joined the chat", username),
					Time:    now(),
				}, nil)
				return cs.send(conn, Message{
					Type:    "system",
					Content: "Username registered!",
					Time:    now(),
				})
			}
			cs.mu.Unlock()
			return nil
		}

		cs.mu.RLock()
		user := cs.connections[conn]
		cs.mu.RUnlock()
		if user == "" {
			return cs.send(conn, Message{
				Type:    "error",
				Content: "Please send your username first.",
				Time:    now(),
			})
		}

		switch msg.Type {
		case "message":
			cs.broadcast(Message{
				Type:    "message",
				Content: msg.Content,
				User:    user,
				Time:    now(),
			}, nil)
		case "ping":
			cs.send(conn, Message{Type: "pong", Content: "pong", Time: now()})
		default:
			cs.send(conn, Message{Type: "error", Content: "Unknown message type", Time: now()})
		}
		return nil
	})

	cs.server.OnBinaryMessage(func(ctx context.Context, conn *blazewave.Conn, payload []byte) error {
		return conn.Write(ctx, blazewave.MessageBinary, payload)
	})

	cs.server.OnClose(func(ctx context.Context, conn *blazewave.Conn, _ blazewave.StatusCode, reason string) error {
		cs.removeConn(conn, "closed: "+reason)
		return nil
	})
}

func (cs *ChatServer) send(conn *blazewave.Conn, msg Message) error {
	payload, _ := json.Marshal(msg)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return conn.Write(ctx, blazewave.MessageText, payload)
}

func (cs *ChatServer) broadcast(msg Message, except *blazewave.Conn) {
	payload, _ := json.Marshal(msg)
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	for conn := range cs.connections {
		if conn == except {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = conn.Write(ctx, blazewave.MessageText, payload)
		cancel()
	}
}

func (cs *ChatServer) removeConn(conn *blazewave.Conn, reason string) {
	cs.mu.Lock()
	username := cs.connections[conn]
	delete(cs.connections, conn)
	cs.mu.Unlock()
	if username != "" {
		cs.broadcast(Message{
			Type:    "system",
			Content: fmt.Sprintf("%s left the chat (%s)", username, reason),
			Time:    now(),
		}, conn)
	}
}

func now() string { return time.Now().Format("15:04:05") }

func (cs *ChatServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, _ = cs.server.Accept(w, r)
}

func main() {
	server := NewChatServer()
	http.Handle("/", http.FileServer(http.Dir("./chat/static/")))
	http.Handle("/ws", server)
	log.Println("Chat server running at http://localhost:8888")
	log.Fatal(http.ListenAndServe(":8888", nil))
}
