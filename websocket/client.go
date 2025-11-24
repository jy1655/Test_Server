package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10
)

// ClientType represents the type of WebSocket client
type ClientType string

const (
	ClientTypeWeb       ClientType = "web"       // Web browser client
	ClientTypeVideo     ClientType = "video"     // Video streaming client (Raspberry Pi)
	ClientTypeControl   ClientType = "control"   // Control client (Raspberry Pi)
	ClientTypeTelemetry ClientType = "telemetry" // Telemetry client (GPS/sensors)
	ClientTypePending   ClientType = "pending"   // Not yet identified
)

// Client represents a WebSocket client connection
type Client struct {
	// Hub that manages this client
	hub *Hub

	// WebSocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte

	// Client type (web, video, control, telemetry)
	clientType ClientType

	// User ID (if authenticated)
	userID int64

	// Username (if authenticated)
	username string

	// Connection ID for handshake validation
	connectionID string

	// Maximum message size allowed from peer
	maxMessageSize int64

	// Handshake completion flag (protected by handshakeMu)
	handshakeComplete bool
	handshakeMu       sync.RWMutex
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, clientType ClientType, userID int64, username string, maxMessageSize int64) *Client {
	return &Client{
		hub:            hub,
		conn:           conn,
		send:           make(chan []byte, 256),
		clientType:     clientType,
		userID:         userID,
		username:       username,
		maxMessageSize: maxMessageSize,
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetReadLimit(c.maxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Route message through hub
		c.hub.RouteMessage(c, message)
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendJSON sends a JSON message to the client
func (c *Client) SendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	select {
	case c.send <- data:
		return nil
	default:
		return websocket.ErrCloseSent
	}
}

// Run starts the client's read and write pumps
func (c *Client) Run() {
	go c.writePump()
	go c.readPump()
}

// SetConnectionID sets the connection ID for handshake validation
func (c *Client) SetConnectionID(id string) {
	c.connectionID = id
}

// GetConnectionID returns the connection ID
func (c *Client) GetConnectionID() string {
	return c.connectionID
}

// MarkHandshakeComplete marks the handshake as complete
func (c *Client) MarkHandshakeComplete() {
	c.handshakeMu.Lock()
	defer c.handshakeMu.Unlock()
	c.handshakeComplete = true
}

// IsHandshakeComplete returns whether handshake is complete
func (c *Client) IsHandshakeComplete() bool {
	c.handshakeMu.RLock()
	defer c.handshakeMu.RUnlock()
	return c.handshakeComplete
}
