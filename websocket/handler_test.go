package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// mockAuthValidator is a mock authentication validator
type mockAuthValidator struct {
	shouldFail bool
}

func (m *mockAuthValidator) ValidateToken(token string) (int64, string, error) {
	if m.shouldFail || token == "invalid" {
		return 0, "", &mockError{"invalid token"}
	}
	return 1, "testuser", nil
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

// TestNewHandler tests handler creation
func TestNewHandler(t *testing.T) {
	hub := NewHub()
	auth := &mockAuthValidator{}

	tests := []struct {
		name            string
		allowedNetworks []string
		enableWhitelist bool
		expectNetworks  int
	}{
		{
			name:            "Whitelist disabled",
			allowedNetworks: []string{"192.168.1.0/24"},
			enableWhitelist: false,
			expectNetworks:  0,
		},
		{
			name:            "Whitelist enabled with valid CIDR",
			allowedNetworks: []string{"192.168.1.0/24", "10.0.0.0/8"},
			enableWhitelist: true,
			expectNetworks:  2,
		},
		{
			name:            "Whitelist with invalid CIDR",
			allowedNetworks: []string{"192.168.1.0/24", "invalid"},
			enableWhitelist: true,
			expectNetworks:  1, // Only valid CIDR should be parsed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(hub, auth, tt.allowedNetworks, tt.enableWhitelist,
				10*time.Second, 65536)

			if handler == nil {
				t.Fatal("NewHandler() returned nil")
			}

			if handler.hub != hub {
				t.Error("Handler hub not set correctly")
			}

			if handler.auth != auth {
				t.Error("Handler auth not set correctly")
			}

			if handler.enableWhitelist != tt.enableWhitelist {
				t.Errorf("Expected enableWhitelist=%v, got %v", tt.enableWhitelist, handler.enableWhitelist)
			}

			if tt.enableWhitelist && len(handler.allowedNetworks) != tt.expectNetworks {
				t.Errorf("Expected %d networks, got %d", tt.expectNetworks, len(handler.allowedNetworks))
			}
		})
	}
}

// TestIsIPAllowed tests IP whitelist validation
func TestIsIPAllowed(t *testing.T) {
	hub := NewHub()
	auth := &mockAuthValidator{}

	tests := []struct {
		name            string
		allowedNetworks []string
		enableWhitelist bool
		remoteAddr      string
		expectAllowed   bool
	}{
		{
			name:            "Whitelist disabled - all IPs allowed",
			allowedNetworks: []string{"192.168.1.0/24"},
			enableWhitelist: false,
			remoteAddr:      "1.2.3.4:5678",
			expectAllowed:   true,
		},
		{
			name:            "IP in allowed network",
			allowedNetworks: []string{"192.168.1.0/24"},
			enableWhitelist: true,
			remoteAddr:      "192.168.1.100:5678",
			expectAllowed:   true,
		},
		{
			name:            "IP not in allowed network",
			allowedNetworks: []string{"192.168.1.0/24"},
			enableWhitelist: true,
			remoteAddr:      "10.0.0.1:5678",
			expectAllowed:   false,
		},
		{
			name:            "IP in multiple networks",
			allowedNetworks: []string{"192.168.1.0/24", "10.0.0.0/8"},
			enableWhitelist: true,
			remoteAddr:      "10.1.2.3:5678",
			expectAllowed:   true,
		},
		{
			name:            "Localhost allowed",
			allowedNetworks: []string{"127.0.0.0/8"},
			enableWhitelist: true,
			remoteAddr:      "127.0.0.1:5678",
			expectAllowed:   true,
		},
		{
			name:            "IP without port",
			allowedNetworks: []string{"192.168.1.0/24"},
			enableWhitelist: true,
			remoteAddr:      "192.168.1.50",
			expectAllowed:   true,
		},
		{
			name:            "Allow all network (0.0.0.0/0)",
			allowedNetworks: []string{"0.0.0.0/0"},
			enableWhitelist: true,
			remoteAddr:      "123.45.67.89:5678",
			expectAllowed:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(hub, auth, tt.allowedNetworks, tt.enableWhitelist,
				10*time.Second, 65536)
			allowed := handler.isIPAllowed(tt.remoteAddr)

			if allowed != tt.expectAllowed {
				t.Errorf("Expected isIPAllowed=%v, got %v for IP %s", tt.expectAllowed, allowed, tt.remoteAddr)
			}
		})
	}
}

// TestServeHTTPAuth tests authentication in ServeHTTP
func TestServeHTTPAuth(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	tests := []struct {
		name           string
		token          string
		authShouldFail bool
		expectStatus   int
	}{
		{
			name:           "Valid token in query",
			token:          "valid-token",
			authShouldFail: false,
			expectStatus:   http.StatusSwitchingProtocols, // WebSocket upgrade
		},
		{
			name:           "Missing token",
			token:          "",
			authShouldFail: false,
			expectStatus:   http.StatusUnauthorized,
		},
		{
			name:           "Invalid token",
			token:          "invalid",
			authShouldFail: false,
			expectStatus:   http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &mockAuthValidator{shouldFail: tt.authShouldFail}
			handler := NewHandler(hub, auth, []string{"0.0.0.0/0"}, false,
				10*time.Second, 65536)

			// Create test request
			req := httptest.NewRequest("GET", "/ws", nil)
			if tt.token != "" {
				q := req.URL.Query()
				q.Add("token", tt.token)
				req.URL.RawQuery = q.Encode()
			}

			// Record response
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			// For WebSocket upgrade, we expect connection upgrade or rejection
			// Since we're not using a real WebSocket client, we check auth rejection
			if tt.expectStatus == http.StatusUnauthorized {
				if rec.Code != http.StatusUnauthorized {
					t.Errorf("Expected status %d, got %d", tt.expectStatus, rec.Code)
				}
			}
		})
	}
}

// TestServeHTTPIPWhitelist tests IP whitelist in ServeHTTP
func TestServeHTTPIPWhitelist(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	auth := &mockAuthValidator{}

	tests := []struct {
		name            string
		allowedNetworks []string
		enableWhitelist bool
		remoteAddr      string
		expectStatus    int
	}{
		{
			name:            "Allowed IP",
			allowedNetworks: []string{"192.168.1.0/24"},
			enableWhitelist: true,
			remoteAddr:      "192.168.1.100:5678",
			expectStatus:    http.StatusSwitchingProtocols, // Would proceed to auth
		},
		{
			name:            "Blocked IP",
			allowedNetworks: []string{"192.168.1.0/24"},
			enableWhitelist: true,
			remoteAddr:      "10.0.0.1:5678",
			expectStatus:    http.StatusForbidden,
		},
		{
			name:            "Whitelist disabled - all allowed",
			allowedNetworks: []string{"192.168.1.0/24"},
			enableWhitelist: false,
			remoteAddr:      "10.0.0.1:5678",
			expectStatus:    http.StatusSwitchingProtocols, // Would proceed to auth
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(hub, auth, tt.allowedNetworks, tt.enableWhitelist,
				10*time.Second, 65536)

			req := httptest.NewRequest("GET", "/ws?token=valid", nil)
			req.RemoteAddr = tt.remoteAddr

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if tt.expectStatus == http.StatusForbidden {
				if rec.Code != http.StatusForbidden {
					t.Errorf("Expected status %d, got %d", tt.expectStatus, rec.Code)
				}
			}
		})
	}
}

// TestServeHTTPXForwardedFor tests X-Forwarded-For header handling
func TestServeHTTPXForwardedFor(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	auth := &mockAuthValidator{}

	handler := NewHandler(hub, auth, []string{"192.168.1.0/24"}, true,
		10*time.Second, 65536)

	tests := []struct {
		name           string
		remoteAddr     string
		xForwardedFor  string
		expectBlocked  bool
	}{
		{
			name:           "Use X-Forwarded-For - allowed",
			remoteAddr:     "10.0.0.1:5678",
			xForwardedFor:  "192.168.1.100",
			expectBlocked:  false,
		},
		{
			name:           "Use X-Forwarded-For - blocked",
			remoteAddr:     "192.168.1.100:5678",
			xForwardedFor:  "10.0.0.1",
			expectBlocked:  true,
		},
		{
			name:           "Multiple IPs in X-Forwarded-For - use first",
			remoteAddr:     "10.0.0.1:5678",
			xForwardedFor:  "192.168.1.100, 10.0.0.2, 10.0.0.3",
			expectBlocked:  false,
		},
		{
			name:           "No X-Forwarded-For - use RemoteAddr",
			remoteAddr:     "192.168.1.100:5678",
			xForwardedFor:  "",
			expectBlocked:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws?token=valid", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if tt.expectBlocked {
				if rec.Code != http.StatusForbidden {
					t.Errorf("Expected request to be blocked (403), got %d", rec.Code)
				}
			} else {
				if rec.Code == http.StatusForbidden {
					t.Error("Expected request to be allowed, but it was blocked")
				}
			}
		})
	}
}

// TestGenerateConnectionID tests connection ID generation
func TestGenerateConnectionID(t *testing.T) {
	remoteAddr := "192.168.1.100:5678"

	id1 := generateConnectionID(remoteAddr)
	time.Sleep(2 * time.Millisecond)
	id2 := generateConnectionID(remoteAddr)

	// IDs should be different (timestamp-based)
	if id1 == id2 {
		t.Error("Connection IDs should be unique")
	}

	// IDs should contain remote address
	if !strings.Contains(id1, "192.168.1.100:5678") {
		t.Errorf("Connection ID should contain remote address, got %s", id1)
	}

	// IDs should be in expected format (address_timestamp)
	parts := strings.Split(id1, "_")
	if len(parts) != 2 {
		t.Errorf("Connection ID should have format 'address_timestamp', got %s", id1)
	}
}

// TestAuthValidatorInterface tests auth validator interface compliance
func TestAuthValidatorInterface(t *testing.T) {
	var _ AuthValidator = (*mockAuthValidator)(nil)
}
