package websocket

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin checking based on config
		return true
	},
}

// Handler handles WebSocket upgrade requests
type Handler struct {
	hub  *Hub
	auth AuthValidator
}

// AuthValidator validates authentication tokens
type AuthValidator interface {
	ValidateToken(token string) (userID int64, username string, err error)
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, auth AuthValidator) *Handler {
	return &Handler{
		hub:  hub,
		auth: auth,
	}
}

// ServeHTTP upgrades HTTP connection to WebSocket
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get token from query parameter or header
	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.Header.Get("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
	}

	// Validate token
	if token == "" {
		http.Error(w, "Missing authentication token", http.StatusUnauthorized)
		return
	}

	userID, username, err := h.auth.ValidateToken(token)
	if err != nil {
		http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
		return
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create client with pending type (will be determined during handshake)
	client := NewClient(h.hub, conn, ClientTypePending, userID, username)

	// Register client
	h.hub.RegisterClient(client)

	// Send handshake request
	handshakeReq := map[string]interface{}{
		"type":      "handshake_request",
		"timestamp": time.Now().Unix(),
	}
	if err := client.SendJSON(handshakeReq); err != nil {
		log.Printf("Failed to send handshake request: %v", err)
		h.hub.UnregisterClient(client)
		return
	}

	// Start client's read/write pumps
	client.Run()
}
