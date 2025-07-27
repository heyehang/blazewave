package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/heyehang/blazewave"
	"github.com/heyehang/blazewave/core/pool"
	"github.com/heyehang/blazewave/core/timer"
)

// ImagePosition represents the position of the shared image
type ImagePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// ClientMessage represents messages from clients
type ClientMessage struct {
	Type     string        `json:"type"`
	Position ImagePosition `json:"position,omitempty"`
	UserID   string        `json:"userId,omitempty"`
	Time     string        `json:"time"`
}

// ServerMessage represents messages from server
type ServerMessage struct {
	Type     string        `json:"type"`
	Position ImagePosition `json:"position,omitempty"`
	UserID   string        `json:"userId,omitempty"`
	Message  string        `json:"message,omitempty"`
	Time     string        `json:"time"`
	Users    []string      `json:"users,omitempty"`
}

// SharedImageServer manages the shared image application
type SharedImageServer struct {
	blazeWaveServer *blazewave.Server
	connections     map[*blazewave.Conn]string // conn -> userID
	mu              sync.RWMutex               // Protect shared state

	// Shared image state
	currentPosition ImagePosition
	lastUpdateTime  time.Time
}

// NewSharedImageServer creates a new shared image server
func NewSharedImageServer() *SharedImageServer {
	rPool := pool.NewPool(64, 4*1024)
	wPool := pool.NewPool(64, 4*1024)
	sharedTimer := timer.NewTimer(100)
	sis := &SharedImageServer{
		blazeWaveServer: blazewave.NewServer(
			blazewave.WithServerReaderPool(rPool),
			blazewave.WithServerWriterPool(wPool),
			blazewave.WithServerHeartbeatTimer(sharedTimer),
		),
		connections:     make(map[*blazewave.Conn]string),
		currentPosition: ImagePosition{X: 100, Y: 100}, // Default position
		lastUpdateTime:  time.Now(),
	}

	sis.setupEventHandlers()
	return sis
}

// createServerMessage creates a ServerMessage with current timestamp
func (sis *SharedImageServer) createServerMessage(msgType, userID, message string, position *ImagePosition) ServerMessage {
	msg := ServerMessage{
		Type: msgType,
		Time: time.Now().Format("15:04:05"),
	}

	if userID != "" {
		msg.UserID = userID
	}
	if message != "" {
		msg.Message = message
	}
	if position != nil {
		msg.Position = *position
	}

	return msg
}

// handleDisconnection safely handles a connection disconnection
func (sis *SharedImageServer) handleDisconnection(conn *blazewave.Conn, reason string) {
	sis.mu.Lock()
	defer sis.mu.Unlock()

	userID := sis.connections[conn]
	delete(sis.connections, conn)

	if userID != "" {
		log.Printf("User %s disconnected: %s", userID, reason)

		// Notify other users
		disconnectMsg := sis.createServerMessage("user_disconnected", userID, fmt.Sprintf("User %s has left", userID), nil)
		go sis.broadcastMessage(disconnectMsg, conn)
		go sis.broadcastUsersList()
	} else {
		log.Printf("Unregistered user disconnected: %s", reason)
	}
}

// setupEventHandlers configures all event handlers
func (sis *SharedImageServer) setupEventHandlers() {
	// Handle new connections
	sis.blazeWaveServer.OnConnect(func(ctx context.Context, conn *blazewave.Conn) error {
		log.Printf("New connection established")

		sis.mu.Lock()
		sis.connections[conn] = ""
		sis.mu.Unlock()

		// Send current image position to new user
		positionMsg := sis.createServerMessage("image_position", "", "", &sis.currentPosition)
		return sis.sendMessage(conn, positionMsg)
	})

	// Handle disconnections
	sis.blazeWaveServer.OnDisconnect(func(ctx context.Context, conn *blazewave.Conn) error {
		sis.handleDisconnection(conn, "EventManager disconnect")
		return nil
	})

	// Handle text messages
	sis.blazeWaveServer.OnTextMessage(func(ctx context.Context, conn *blazewave.Conn, payload []byte) error {
		var clientMsg ClientMessage
		if err := json.Unmarshal(payload, &clientMsg); err != nil {
			log.Printf("Failed to parse JSON message: %v", err)
			return nil
		}

		log.Printf("Received message: %s", clientMsg.Type)

		switch clientMsg.Type {
		case "register":
			sis.mu.Lock()
			sis.connections[conn] = clientMsg.UserID
			sis.mu.Unlock()

			log.Printf("User %s registered", clientMsg.UserID)

			// Send confirmation
			confirmMsg := sis.createServerMessage("registered", clientMsg.UserID, "Successfully registered", nil)
			sis.sendMessage(conn, confirmMsg)

			// Send full user list to the new user
			usersListMsg := ServerMessage{
				Type:  "users_list",
				Users: sis.getAllUserIDs(),
				Time:  time.Now().Format("15:04:05"),
			}
			sis.sendMessage(conn, usersListMsg)

			// Notify others and broadcast updated user list
			joinMsg := sis.createServerMessage("user_joined", clientMsg.UserID, fmt.Sprintf("User %s has joined", clientMsg.UserID), nil)
			sis.broadcastMessage(joinMsg, conn)
			sis.broadcastUsersList()

		case "move_image":
			sis.mu.Lock()
			sis.currentPosition = clientMsg.Position
			sis.lastUpdateTime = time.Now()
			sis.mu.Unlock()

			log.Printf("Image moved to (%f, %f) by user %s", clientMsg.Position.X, clientMsg.Position.Y, clientMsg.UserID)

			// Broadcast new position to all users except the sender
			positionMsg := sis.createServerMessage("image_position", clientMsg.UserID, "", &clientMsg.Position)
			sis.broadcastMessage(positionMsg, conn)

		case "get_position":
			sis.mu.RLock()
			position := sis.currentPosition
			sis.mu.RUnlock()

			positionMsg := sis.createServerMessage("image_position", "", "", &position)
			return sis.sendMessage(conn, positionMsg)

		default:
			log.Printf("Unknown message type: %s", clientMsg.Type)
		}

		return nil
	})

	// Handle errors
	sis.blazeWaveServer.OnError(func(ctx context.Context, conn *blazewave.Conn, err error) error {
		// Check if this is a connection close error
		if errors.Is(err, blazewave.ErrConnectionClosed) ||
			errors.Is(err, blazewave.ErrNetworkClosed) ||
			errors.Is(err, blazewave.ErrBrokenPipe) ||
			errors.Is(err, blazewave.ErrWriteBrokenPipe) {

			sis.handleDisconnection(conn, err.Error())
			return nil
		}

		log.Printf("Error on connection: %v", err)
		return nil
	})

	// Handle close events
	sis.blazeWaveServer.OnClose(func(ctx context.Context, conn *blazewave.Conn, code blazewave.StatusCode, reason string) error {
		sis.handleDisconnection(conn, fmt.Sprintf("Close event: code=%d, reason=%s", code, reason))
		return nil
	})
}

// sendMessage sends a message to a specific connection
func (sis *SharedImageServer) sendMessage(conn *blazewave.Conn, msg ServerMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.Write(ctx, blazewave.MessageText, payload); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// broadcastMessage sends a message to all connections except the sender
func (sis *SharedImageServer) broadcastMessage(msg ServerMessage, sender *blazewave.Conn) {
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal broadcast message: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Thread-safe access to connections
	sis.mu.RLock()
	connections := make([]*blazewave.Conn, 0, len(sis.connections))
	for conn := range sis.connections {
		if conn != sender {
			connections = append(connections, conn)
		}
	}
	sis.mu.RUnlock()

	log.Printf("Broadcasting message '%s' to %d connections", msg.Type, len(connections))

	// Send to all connections
	successCount := 0
	for _, conn := range connections {
		if err := conn.Write(ctx, blazewave.MessageText, payload); err != nil {
			log.Printf("Failed to send message: %v", err)
		} else {
			successCount++
		}
	}

	log.Printf("Broadcast completed: %d/%d messages sent successfully", successCount, len(connections))
}

func (sis *SharedImageServer) getAllUserIDs() []string {
	sis.mu.RLock()
	defer sis.mu.RUnlock()
	users := make([]string, 0, len(sis.connections))
	for _, user := range sis.connections {
		if user != "" {
			users = append(users, user)
		}
	}
	return users
}

func (sis *SharedImageServer) broadcastUsersList() {
	users := sis.getAllUserIDs()
	msg := ServerMessage{
		Type:  "users_list",
		Users: users,
		Time:  time.Now().Format("15:04:05"),
	}
	sis.mu.RLock()
	for conn := range sis.connections {
		_ = sis.sendMessage(conn, msg)
	}
	sis.mu.RUnlock()
}

// ServeHTTP implements http.Handler
func (sis *SharedImageServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, err := sis.blazeWaveServer.Accept(w, r)
	if err != nil {
		log.Printf("Failed to accept WebSocket connection: %v", err)
		return
	}
	log.Printf("Event-driven WebSocket connection established")
}

func main() {
	server := NewSharedImageServer()

	// Serve static files
	log.Println("Serving static files from: static/")

	// Handle root path to serve index.html
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "./shared-mascot/static/index.html")
		} else {
			http.FileServer(http.Dir("./shared-mascot/static")).ServeHTTP(w, r)
		}
	})
	http.Handle("/ws", server)

	port := ":8888"
	log.Printf("Shared Image server starting on %s", port)
	log.Printf("Open http://localhost%s in your browser", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
