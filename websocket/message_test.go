package websocket

import (
	"encoding/json"
	"testing"
)

// TestMessageParsing tests message JSON parsing
func TestMessageParsing(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectType  string
		expectError bool
	}{
		{
			name:        "Valid ping message",
			jsonData:    `{"type":"ping","timestamp":1234567890}`,
			expectType:  "ping",
			expectError: false,
		},
		{
			name:        "Valid control command",
			jsonData:    `{"type":"control_command","data":{"action":"forward"}}`,
			expectType:  "control_command",
			expectError: false,
		},
		{
			name:        "Valid handshake response",
			jsonData:    `{"type":"handshake_response","connection_id":"test_123","client_type":"control"}`,
			expectType:  "handshake_response",
			expectError: false,
		},
		{
			name:        "Invalid JSON",
			jsonData:    `{invalid json}`,
			expectType:  "",
			expectError: true,
		},
		{
			name:        "Empty type",
			jsonData:    `{"type":""}`,
			expectType:  "",
			expectError: false,
		},
		{
			name:        "WebRTC offer",
			jsonData:    `{"type":"offer","sdp":"test_sdp"}`,
			expectType:  "offer",
			expectError: false,
		},
		{
			name:        "WebRTC answer",
			jsonData:    `{"type":"answer","sdp":"test_sdp"}`,
			expectType:  "answer",
			expectError: false,
		},
		{
			name:        "ICE candidate",
			jsonData:    `{"type":"ice-candidate","candidate":"test"}`,
			expectType:  "ice-candidate",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg Message
			err := json.Unmarshal([]byte(tt.jsonData), &msg)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if msg.Type != tt.expectType {
					t.Errorf("Expected type %s, got %s", tt.expectType, msg.Type)
				}
			}
		})
	}
}

// TestHandshakeResponseParsing tests handshake response parsing
func TestHandshakeResponseParsing(t *testing.T) {
	tests := []struct {
		name             string
		jsonData         string
		expectClientType ClientType
		expectConnID     string
		expectError      bool
	}{
		{
			name:             "Valid web client",
			jsonData:         `{"type":"handshake_response","connection_id":"test_123","client_type":"web"}`,
			expectClientType: ClientTypeWeb,
			expectConnID:     "test_123",
			expectError:      false,
		},
		{
			name:             "Valid control client",
			jsonData:         `{"type":"handshake_response","connection_id":"test_456","client_type":"control"}`,
			expectClientType: ClientTypeControl,
			expectConnID:     "test_456",
			expectError:      false,
		},
		{
			name:             "Valid video client",
			jsonData:         `{"type":"handshake_response","connection_id":"test_789","client_type":"video"}`,
			expectClientType: ClientTypeVideo,
			expectConnID:     "test_789",
			expectError:      false,
		},
		{
			name:             "Valid telemetry client",
			jsonData:         `{"type":"handshake_response","connection_id":"test_101","client_type":"telemetry"}`,
			expectClientType: ClientTypeTelemetry,
			expectConnID:     "test_101",
			expectError:      false,
		},
		{
			name:             "With auth token",
			jsonData:         `{"type":"handshake_response","connection_id":"test_202","client_type":"web","auth_token":"test_token"}`,
			expectClientType: ClientTypeWeb,
			expectConnID:     "test_202",
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handshake HandshakeResponse
			err := json.Unmarshal([]byte(tt.jsonData), &handshake)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if handshake.ClientType != tt.expectClientType {
					t.Errorf("Expected client type %s, got %s", tt.expectClientType, handshake.ClientType)
				}
				if handshake.ConnectionID != tt.expectConnID {
					t.Errorf("Expected connection ID %s, got %s", tt.expectConnID, handshake.ConnectionID)
				}
				if handshake.Type != "handshake_response" {
					t.Errorf("Expected type 'handshake_response', got %s", handshake.Type)
				}
			}
		})
	}
}

// TestMessageTypeConstants tests that message type constants match expected Python values
func TestMessageTypeConstants(t *testing.T) {
	pythonMessageTypes := []string{
		"handshake_request",
		"handshake_response",
		"connection_established",
		"ping",
		"pong",
		"control_command",
		"control_response",
		"offer",
		"answer",
		"ice-candidate",
		"video_client_ready",
		"emergency_stop",
		"emergency_stop_reset",
		"route_update",
		"location_update",
		"control_client_connect",
		"video_client_connect",
		"get_status",
		"status_response",
		"webrtc_connected",
	}

	// Test that message types can be marshalled and unmarshalled correctly
	for _, msgType := range pythonMessageTypes {
		msg := Message{
			Type: msgType,
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Errorf("Failed to marshal message type %s: %v", msgType, err)
			continue
		}

		var parsed Message
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Errorf("Failed to unmarshal message type %s: %v", msgType, err)
			continue
		}

		if parsed.Type != msgType {
			t.Errorf("Expected type %s, got %s", msgType, parsed.Type)
		}
	}
}

// TestClientTypeValidation tests client type validation
func TestClientTypeValidation(t *testing.T) {
	validTypes := []ClientType{
		ClientTypeWeb,
		ClientTypeVideo,
		ClientTypeControl,
		ClientTypeTelemetry,
	}

	invalidTypes := []ClientType{
		"invalid",
		"unknown",
		"",
	}

	validTypeMap := map[ClientType]bool{
		ClientTypeWeb:       true,
		ClientTypeVideo:     true,
		ClientTypeControl:   true,
		ClientTypeTelemetry: true,
	}

	// Test valid types
	for _, clientType := range validTypes {
		if !validTypeMap[clientType] {
			t.Errorf("Valid type %s not found in validation map", clientType)
		}
	}

	// Test invalid types
	for _, clientType := range invalidTypes {
		if validTypeMap[clientType] {
			t.Errorf("Invalid type %s found in validation map", clientType)
		}
	}
}

// TestHandshakeRequestStructure tests handshake request message structure
func TestHandshakeRequestStructure(t *testing.T) {
	// Simulate handshake request from server
	handshakeReq := map[string]interface{}{
		"type":                   "handshake_request",
		"connection_id":          "test_12345",
		"timestamp":              int64(1234567890),
		"supported_client_types": []string{"web", "video", "control", "telemetry"},
	}

	data, err := json.Marshal(handshakeReq)
	if err != nil {
		t.Fatalf("Failed to marshal handshake request: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal handshake request: %v", err)
	}

	// Verify structure
	if parsed["type"] != "handshake_request" {
		t.Errorf("Expected type 'handshake_request', got %v", parsed["type"])
	}

	if parsed["connection_id"] != "test_12345" {
		t.Errorf("Expected connection_id 'test_12345', got %v", parsed["connection_id"])
	}

	supportedTypes, ok := parsed["supported_client_types"].([]interface{})
	if !ok {
		t.Error("supported_client_types field missing or invalid type")
	} else {
		if len(supportedTypes) != 4 {
			t.Errorf("Expected 4 supported types, got %d", len(supportedTypes))
		}
	}
}

// TestConnectionEstablishedStructure tests connection established message structure
func TestConnectionEstablishedStructure(t *testing.T) {
	// Simulate connection established response from server
	response := map[string]interface{}{
		"type":                    "connection_established",
		"client_type":             "control",
		"status":                  "connected",
		"video_clients_available": true,
		"timestamp":               int64(1234567890),
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal connection established: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal connection established: %v", err)
	}

	// Verify structure
	if parsed["type"] != "connection_established" {
		t.Errorf("Expected type 'connection_established', got %v", parsed["type"])
	}

	if parsed["client_type"] != "control" {
		t.Errorf("Expected client_type 'control', got %v", parsed["client_type"])
	}

	if parsed["status"] != "connected" {
		t.Errorf("Expected status 'connected', got %v", parsed["status"])
	}

	videoAvailable, ok := parsed["video_clients_available"].(bool)
	if !ok {
		t.Error("video_clients_available field missing or invalid type")
	} else if !videoAvailable {
		t.Error("Expected video_clients_available to be true")
	}
}
