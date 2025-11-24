package websocket

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
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
	hub              *Hub
	auth             AuthValidator
	allowedNetworks  []*net.IPNet
	enableWhitelist  bool
	handshakeTimeout time.Duration
	maxMessageSize   int64
}

// AuthValidator validates authentication tokens
type AuthValidator interface {
	ValidateToken(token string) (userID int64, username string, err error)
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, auth AuthValidator, allowedNetworks []string, enableWhitelist bool, handshakeTimeout time.Duration, maxMessageSize int64) *Handler {
	// Parse CIDR networks
	var networks []*net.IPNet
	if enableWhitelist {
		for _, cidr := range allowedNetworks {
			_, network, err := net.ParseCIDR(cidr)
			if err != nil {
				log.Printf("⚠️  Invalid CIDR notation '%s': %v", cidr, err)
				continue
			}
			networks = append(networks, network)
		}
		log.Printf("🔒 IP whitelist enabled with %d networks", len(networks))
	} else {
		log.Printf("ℹ️  IP whitelist disabled - accepting all connections")
	}

	return &Handler{
		hub:              hub,
		auth:             auth,
		allowedNetworks:  networks,
		enableWhitelist:  enableWhitelist,
		handshakeTimeout: handshakeTimeout,
		maxMessageSize:   maxMessageSize,
	}
}

// isIPAllowed checks if the client IP is in the allowed networks
func (h *Handler) isIPAllowed(remoteAddr string) bool {
	if !h.enableWhitelist {
		return true
	}

	// Extract IP from address (remove port)
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// If no port, use the address as-is
		host = remoteAddr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		log.Printf("⚠️  Failed to parse IP address: %s", host)
		return false
	}

	// Check against allowed networks
	for _, network := range h.allowedNetworks {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// ServeHTTP upgrades HTTP connection to WebSocket
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remoteAddr := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// Use first IP from X-Forwarded-For header
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			remoteAddr = strings.TrimSpace(ips[0])
		}
	}

	log.Printf("🔌 Connection attempt from %s", remoteAddr)

	// Check IP whitelist
	if !h.isIPAllowed(remoteAddr) {
		log.Printf("🚫 IP blocked by whitelist: %s", remoteAddr)
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

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
		log.Printf("❌ Missing auth token from %s", remoteAddr)
		http.Error(w, "Missing authentication token", http.StatusUnauthorized)
		return
	}

	userID, username, err := h.auth.ValidateToken(token)
	if err != nil {
		log.Printf("❌ Invalid auth token from %s: %v", remoteAddr, err)
		http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
		return
	}

	log.Printf("✅ Authentication successful: user=%s (id=%d) from %s", username, userID, remoteAddr)

	// Upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed for %s: %v", username, err)
		return
	}

	log.Printf("🔄 WebSocket upgraded for %s, waiting for handshake...", username)

	// Create client with pending type (will be determined during handshake)
	client := NewClient(h.hub, conn, ClientTypePending, userID, username, h.maxMessageSize)

	// Generate unique connection ID for this handshake
	connectionID := generateConnectionID(r.RemoteAddr)
	client.SetConnectionID(connectionID)

	// Register client
	h.hub.RegisterClient(client)

	// Start client's read/write pumps BEFORE sending handshake
	client.Run()

	// Give goroutines a moment to start
	time.Sleep(10 * time.Millisecond)

	// Send handshake request (Python-compatible) after pumps are running
	handshakeReq := map[string]interface{}{
		"type":                   "handshake_request",
		"connection_id":          connectionID,
		"timestamp":              time.Now().Unix(),
		"supported_client_types": []string{"web", "video", "control", "telemetry"},
	}
	if err := client.SendJSON(handshakeReq); err != nil {
		log.Printf("❌ Failed to send handshake request to %s: %v", username, err)
		h.hub.UnregisterClient(client)
		return
	}

	log.Printf("📤 Handshake request sent to %s (connection_id=%s)", username, connectionID)

	// Start handshake timeout monitoring
	go h.monitorHandshakeTimeout(client, connectionID, username)
}

// generateConnectionID creates a unique connection ID for handshake
func generateConnectionID(remoteAddr string) string {
	return fmt.Sprintf("%s_%d", remoteAddr, time.Now().UnixNano()/1000000)
}

// monitorHandshakeTimeout monitors handshake completion and closes connection if timeout occurs
func (h *Handler) monitorHandshakeTimeout(client *Client, connectionID, username string) {
	// Wait for handshake timeout
	time.Sleep(h.handshakeTimeout)

	// Check if handshake is complete
	if !client.IsHandshakeComplete() {
		log.Printf("⏱️ Handshake timeout for %s (connection_id=%s) after %v",
			username, connectionID, h.handshakeTimeout)
		// Unregister client - this will close the connection
		h.hub.UnregisterClient(client)
	} else {
		log.Printf("✅ Handshake completed within timeout for %s", username)
	}
}
