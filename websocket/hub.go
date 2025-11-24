package websocket

import (
	"log"
	"sync"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients by type
	clients map[ClientType]map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe access to clients map
	mu sync.RWMutex
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[ClientType]map[*Client]bool),
		register:   make(chan *Client, 10),   // Buffered channel to prevent blocking
		unregister: make(chan *Client, 10),   // Buffered channel to prevent blocking
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🚨 Hub.Run() panic recovered: %v", r)
		}
	}()

	for {
		select {
		case client := <-h.register:
			log.Printf("📥 Processing register for %s (type=%s)", client.username, client.clientType)
			h.mu.Lock()
			if h.clients[client.clientType] == nil {
				h.clients[client.clientType] = make(map[*Client]bool)
			}
			h.clients[client.clientType][client] = true
			// Calculate count without calling GetClientCount() to avoid potential issues
			count := 0
			for _, clients := range h.clients {
				count += len(clients)
			}
			h.mu.Unlock()

			log.Printf("Client registered: type=%s, user=%s (total: %d)",
				client.clientType, client.username, count)

		case client := <-h.unregister:
			log.Printf("📤 Processing unregister for %s (type=%s)", client.username, client.clientType)
			log.Printf("🔒 Attempting to lock mutex for unregister...")
			h.mu.Lock()
			log.Printf("✅ Mutex locked for unregister")
			if clients, ok := h.clients[client.clientType]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					log.Printf("🗑️  Deleted client from map, about to close send channel...")

					// Safely close channel with panic recovery
					func() {
						defer func() {
							if r := recover(); r != nil {
								log.Printf("🚨 Panic while closing send channel: %v", r)
							}
						}()
						close(client.send)
						log.Printf("✅ Send channel closed successfully")
					}()

					// Calculate count without calling GetClientCount() to avoid deadlock
					count := 0
					for _, clients := range h.clients {
						count += len(clients)
					}
					log.Printf("Client unregistered: type=%s, user=%s (total: %d)",
						client.clientType, client.username, count)
				} else {
					log.Printf("⚠️  Client not found in map for unregister: %s", client.username)
				}
			} else {
				log.Printf("⚠️  Client type map not found for unregister: %s", client.clientType)
			}
			log.Printf("🔓 About to unlock mutex...")
			h.mu.Unlock()
			log.Printf("✅ Mutex unlocked")
		}
	}
}

// RegisterClient registers a new client
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient unregisters a client
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// BroadcastToType sends a message to all clients of a specific type
func (h *Hub) BroadcastToType(clientType ClientType, message []byte) {
	h.mu.RLock()
	clients := h.clients[clientType]
	h.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- message:
		default:
			// Client's send buffer is full, unregister it
			go h.UnregisterClient(client)
		}
	}
}

// BroadcastToAll sends a message to all clients
func (h *Hub) BroadcastToAll(message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, clients := range h.clients {
		for client := range clients {
			select {
			case client.send <- message:
			default:
				go h.UnregisterClient(client)
			}
		}
	}
}

// GetClientCount returns the total number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, clients := range h.clients {
		count += len(clients)
	}
	return count
}

// GetClientCountByType returns the number of clients of a specific type
func (h *Hub) GetClientCountByType(clientType ClientType) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[clientType]; ok {
		return len(clients)
	}
	return 0
}

// GetStats returns statistics about connected clients
func (h *Hub) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total"] = h.GetClientCount()
	stats["web"] = len(h.clients[ClientTypeWeb])
	stats["video"] = len(h.clients[ClientTypeVideo])
	stats["control"] = len(h.clients[ClientTypeControl])
	stats["telemetry"] = len(h.clients[ClientTypeTelemetry])
	stats["pending"] = len(h.clients[ClientTypePending])

	return stats
}
