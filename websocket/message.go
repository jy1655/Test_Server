package websocket

import (
	"encoding/json"
	"log"
	"time"
)

// Message represents a WebSocket message
type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// HandshakeResponse represents handshake response from client
type HandshakeResponse struct {
	Type         string     `json:"type"`
	ConnectionID string     `json:"connection_id"`
	ClientType   ClientType `json:"client_type"`
	AuthToken    string     `json:"auth_token,omitempty"`
}

// RouteMessage routes a message from sender to appropriate recipients
func (h *Hub) RouteMessage(sender *Client, rawMessage []byte) {
	var msg Message
	if err := json.Unmarshal(rawMessage, &msg); err != nil {
		log.Printf("Invalid message format from %s: %v", sender.clientType, err)
		return
	}

	log.Printf("Message received: type=%s from client_type=%s user=%s",
		msg.Type, sender.clientType, sender.username)

	switch msg.Type {
	case "handshake_response":
		h.handleHandshake(sender, rawMessage)

	case "ping":
		h.handlePing(sender, rawMessage)

	case "pong":
		// Just log pong messages
		log.Printf("Pong received from %s", sender.clientType)

	case "control_command":
		// Control commands from web clients go to control clients
		if sender.clientType == ClientTypeWeb {
			h.BroadcastToType(ClientTypeControl, rawMessage)
			log.Printf("Routed control command to %d control clients",
				h.GetClientCountByType(ClientTypeControl))
		}

	case "control_response":
		// Control responses from control clients go back to web clients
		if sender.clientType == ClientTypeControl {
			h.BroadcastToType(ClientTypeWeb, rawMessage)
			log.Printf("Routed control response to %d web clients",
				h.GetClientCountByType(ClientTypeWeb))
		}

	case "offer", "answer", "ice-candidate":
		// WebRTC signaling
		h.handleWebRTCSignaling(sender, msg.Type, rawMessage)

	case "video_client_ready":
		// Video client is ready, notify web clients
		h.BroadcastToType(ClientTypeWeb, rawMessage)
		log.Printf("Notified %d web clients that video is ready",
			h.GetClientCountByType(ClientTypeWeb))

	case "emergency_stop":
		// Emergency stop broadcasts to all control clients
		h.BroadcastToType(ClientTypeControl, rawMessage)
		log.Printf("🚨 Emergency stop broadcast to %d control clients",
			h.GetClientCountByType(ClientTypeControl))

	case "route_update", "location_update":
		// Telemetry updates go to web clients
		h.BroadcastToType(ClientTypeWeb, rawMessage)
		log.Printf("Forwarded %s to %d web clients",
			msg.Type, h.GetClientCountByType(ClientTypeWeb))

	case "control_client_connect":
		// Legacy Python client type identification (before handshake)
		log.Printf("Legacy control client identification from %s", sender.username)
		// Modern clients should use handshake protocol instead

	case "video_client_connect":
		// Legacy Python client type identification (before handshake)
		log.Printf("Legacy video client identification from %s", sender.username)
		// Modern clients should use handshake protocol instead

	case "emergency_stop_reset":
		// Reset emergency stop state - broadcast to control clients
		h.BroadcastToType(ClientTypeControl, rawMessage)
		log.Printf("🔄 Emergency stop reset broadcast to %d control clients",
			h.GetClientCountByType(ClientTypeControl))

	case "get_status":
		// Return server status to requester
		h.handleGetStatus(sender)

	case "webrtc_connected":
		// WebRTC connection established notification
		h.BroadcastToType(ClientTypeWeb, rawMessage)
		log.Printf("📡 WebRTC connection status forwarded to web clients")

	default:
		// Unknown message type - broadcast to all except sender
		log.Printf("Unknown message type: %s, broadcasting to all", msg.Type)
		h.broadcastExceptSender(sender, rawMessage)
	}
}

// handleGetStatus returns server statistics to client
func (h *Hub) handleGetStatus(client *Client) {
	stats := h.GetStats()
	response := map[string]interface{}{
		"type":  "status_response",
		"stats": stats,
		"timestamp": time.Now().Unix(),
	}

	if err := client.SendJSON(response); err != nil {
		log.Printf("Failed to send status response: %v", err)
	}
}

// handleHandshake processes handshake response from client
func (h *Hub) handleHandshake(client *Client, rawMessage []byte) {
	var handshake HandshakeResponse
	if err := json.Unmarshal(rawMessage, &handshake); err != nil {
		log.Printf("❌ Invalid handshake response JSON: %v", err)
		return
	}

	log.Printf("🔍 Handshake validation: conn_id=%s, client_id=%s, type=%s",
		handshake.ConnectionID, client.GetConnectionID(), handshake.ClientType)

	// Validate connection ID
	if handshake.ConnectionID != client.GetConnectionID() {
		log.Printf("❌ Invalid connection ID in handshake: expected=%s, got=%s",
			client.GetConnectionID(), handshake.ConnectionID)
		return
	}

	// Validate client type
	validTypes := map[ClientType]bool{
		ClientTypeWeb:       true,
		ClientTypeVideo:     true,
		ClientTypeControl:   true,
		ClientTypeTelemetry: true,
	}
	if !validTypes[handshake.ClientType] {
		log.Printf("❌ Invalid client type in handshake: %s", handshake.ClientType)
		return
	}

	log.Printf("✅ Handshake validation passed")

	// Mark handshake as complete
	client.MarkHandshakeComplete()

	// Update client type - just change the field, hub.Run() will handle map updates
	log.Printf("🔍 Current client type: %s (checking if pending)", client.clientType)
	if client.clientType == ClientTypePending {
		log.Printf("✅ Client is pending, updating type to %s", handshake.ClientType)

		// Update client type field (this will be picked up by hub.Run() when it processes register)
		oldType := client.clientType
		client.clientType = handshake.ClientType

		// If client is already registered in hub, we need to move it to the correct map
		log.Printf("🔒 handleHandshake: Attempting to lock mutex...")
		h.mu.Lock()
		log.Printf("✅ handleHandshake: Mutex locked")
		if clients, ok := h.clients[oldType]; ok {
			if _, exists := clients[client]; exists {
				// Client is already in hub, move it to new type
				delete(clients, client)
				if h.clients[client.clientType] == nil {
					h.clients[client.clientType] = make(map[*Client]bool)
				}
				h.clients[client.clientType][client] = true
				log.Printf("🔄 Moved client from %s to %s", oldType, client.clientType)
			}
		}
		log.Printf("🔓 handleHandshake: About to unlock mutex...")
		h.mu.Unlock()
		log.Printf("✅ handleHandshake: Mutex unlocked")

		log.Printf("✅ Client handshake completed: type=%s, user=%s",
			client.clientType, client.username)

		// Check if video clients are available
		videoAvailable := h.GetClientCountByType(ClientTypeVideo) > 0

		// Send Python-compatible confirmation
		response := map[string]interface{}{
			"type":                    "connection_established",
			"client_type":             client.clientType,
			"status":                  "connected",
			"video_clients_available": videoAvailable,
			"timestamp":               time.Now().Unix(),
		}
		if err := client.SendJSON(response); err != nil {
			log.Printf("❌ Failed to send connection_established to %s: %v", client.username, err)
			return
		}
		log.Printf("📨 Sent connection_established to %s", client.username)

		// If video client connected, notify web clients
		if handshake.ClientType == ClientTypeVideo {
			h.notifyWebClientsVideoReady()
		}
	}
}

// notifyWebClientsVideoReady notifies web clients that video is available
func (h *Hub) notifyWebClientsVideoReady() {
	notification := map[string]interface{}{
		"type":      "video_client_ready",
		"status":    "ready",
		"timestamp": time.Now().Unix(),
	}

	data, err := json.Marshal(notification)
	if err != nil {
		log.Printf("Failed to marshal video ready notification: %v", err)
		return
	}

	h.BroadcastToType(ClientTypeWeb, data)
	log.Printf("📹 Notified %d web clients that video is ready",
		h.GetClientCountByType(ClientTypeWeb))
}

// handlePing responds to ping messages with pong
func (h *Hub) handlePing(client *Client, rawMessage []byte) {
	var pingMsg map[string]interface{}
	if err := json.Unmarshal(rawMessage, &pingMsg); err != nil {
		return
	}

	pongMsg := map[string]interface{}{
		"type":      "pong",
		"timestamp": pingMsg["timestamp"],
	}

	client.SendJSON(pongMsg)
}

// handleWebRTCSignaling routes WebRTC signaling messages
func (h *Hub) handleWebRTCSignaling(sender *Client, msgType string, rawMessage []byte) {
	switch sender.clientType {
	case ClientTypeWeb:
		// Web client's offer/ice-candidate goes to video client
		h.BroadcastToType(ClientTypeVideo, rawMessage)
		log.Printf("Routed %s from web to %d video clients",
			msgType, h.GetClientCountByType(ClientTypeVideo))

	case ClientTypeVideo:
		// Video client's answer/ice-candidate goes to web clients
		h.BroadcastToType(ClientTypeWeb, rawMessage)
		log.Printf("Routed %s from video to %d web clients",
			msgType, h.GetClientCountByType(ClientTypeWeb))

	default:
		log.Printf("Unexpected WebRTC signaling from %s", sender.clientType)
	}
}

// broadcastExceptSender sends message to all clients except the sender
func (h *Hub) broadcastExceptSender(sender *Client, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, clients := range h.clients {
		for client := range clients {
			if client != sender {
				select {
				case client.send <- message:
				default:
					go h.UnregisterClient(client)
				}
			}
		}
	}
}
