package main

import (
	"fmt"
	"log"
	"net/http"
	"oculo-pilot-server/api"
	"oculo-pilot-server/auth"
	"oculo-pilot-server/config"
	"oculo-pilot-server/middleware"
	"oculo-pilot-server/websocket"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
)

const version = "1.0.0"

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := auth.NewDB(cfg.DB.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("✅ Database initialized")

	// Create default admin user if no users exist
	if err := createDefaultUser(db); err != nil {
		log.Printf("Warning: %v", err)
	}

	// Initialize auth service
	authService := auth.NewService(db, cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry)

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	log.Println("✅ WebSocket hub started")

	// Create router
	router := mux.NewRouter()

	// Apply middleware
	router.Use(middleware.Logging)
	router.Use(middleware.CORS(cfg.Server.AllowedOrigins))

	// Health check (no auth required)
	router.Handle("/health", api.NewHealthHandler(version)).Methods("GET")

	// Auth endpoints (no auth required)
	router.Handle("/api/login", api.NewLoginHandler(authService)).Methods("POST", "OPTIONS")
	router.Handle("/api/register", api.NewRegisterHandler(authService)).Methods("POST", "OPTIONS")

	// WebSocket endpoint (requires auth)
	wsHandler := websocket.NewHandler(hub, &authValidator{authService})
	router.Handle("/ws", wsHandler)

	// Static files
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))

	// Start server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("🚀 Server starting on %s", addr)
	log.Printf("🔐 JWT expiry: %v", cfg.Auth.JWTExpiry)
	log.Printf("🌐 Allowed origins: %v", cfg.Server.AllowedOrigins)

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	log.Println("✅ Server is running")
	log.Println("📝 Endpoints:")
	log.Println("   GET  /health          - Health check")
	log.Println("   POST /api/login       - User login")
	log.Println("   POST /api/register    - User registration")
	log.Println("   WS   /ws?token=<jwt>  - WebSocket connection")

	<-stop
	log.Println("🛑 Shutting down server...")
}

// authValidator adapts auth.Service to websocket.AuthValidator interface
type authValidator struct {
	service *auth.Service
}

func (av *authValidator) ValidateToken(token string) (int64, string, error) {
	claims, err := av.service.ValidateToken(token)
	if err != nil {
		return 0, "", err
	}
	return claims.UserID, claims.Username, nil
}

// createDefaultUser creates a default admin user if no users exist
func createDefaultUser(db *auth.DB) error {
	users, err := db.ListUsers()
	if err != nil {
		return fmt.Errorf("failed to list users: %v", err)
	}

	if len(users) == 0 {
		// Create default admin user
		username := "admin"
		password := "admin123" // Default password (should be changed immediately)

		_, err := db.CreateUser(username, password)
		if err != nil {
			return fmt.Errorf("failed to create default user: %v", err)
		}

		log.Println("⚠️  Default admin user created:")
		log.Println("   Username: admin")
		log.Println("   Password: admin123")
		log.Println("   ⚠️  CHANGE THIS PASSWORD IMMEDIATELY!")
	}

	return nil
}
