package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds application configuration
type Config struct {
	Server ServerConfig
	Auth   AuthConfig
	DB     DBConfig
	TURN   TURNConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host                 string
	Port                 string
	AllowedOrigins       []string
	AllowedNetworks      []string // IP whitelist (CIDR format)
	RateLimit            int
	HandshakeTimeout     time.Duration
	EnableIPWhitelist    bool
	MaxMessageSize       int64
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret string
	JWTExpiry time.Duration
}

// DBConfig holds database configuration
type DBConfig struct {
	Path string
}

// TURNConfig holds TURN server configuration
type TURNConfig struct {
	Server   string
	Username string
	Password string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file (ignore error if it doesn't exist)
	_ = godotenv.Load()

	return &Config{
		Server: ServerConfig{
			Host:              getEnv("SERVER_HOST", "0.0.0.0"),
			Port:              getEnv("SERVER_PORT", "8080"),
			AllowedOrigins:    getEnvSlice("ALLOWED_ORIGINS", ",", []string{"*"}),
			AllowedNetworks:   getEnvSlice("ALLOWED_NETWORKS", ",", []string{"0.0.0.0/0"}), // Allow all by default
			RateLimit:         getEnvInt("RATE_LIMIT", 100),
			HandshakeTimeout:  getEnvDuration("HANDSHAKE_TIMEOUT", "10s"),
			EnableIPWhitelist: getEnvBool("ENABLE_IP_WHITELIST", false),
			MaxMessageSize:    int64(getEnvInt("MAX_MESSAGE_SIZE", 65536)), // 64KB
		},
		Auth: AuthConfig{
			JWTSecret: getEnv("JWT_SECRET", "change-this-secret-key-in-production"),
			JWTExpiry: getEnvDuration("JWT_EXPIRY", "24h"),
		},
		DB: DBConfig{
			Path: getEnv("DB_PATH", "./users.db"),
		},
		TURN: TURNConfig{
			Server:   getEnv("TURN_SERVER", ""),
			Username: getEnv("TURN_USERNAME", ""),
			Password: getEnv("TURN_PASSWORD", ""),
		},
	}, nil
}

// getEnv gets environment variable or returns default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvInt gets environment variable as int or returns default value
func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intVal, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intVal
}

// getEnvSlice gets environment variable as slice or returns default value
func getEnvSlice(key, separator string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return strings.Split(value, separator)
}

// getEnvDuration gets environment variable as duration or returns default value
func getEnvDuration(key, defaultValue string) time.Duration {
	value := getEnv(key, defaultValue)
	duration, err := time.ParseDuration(value)
	if err != nil {
		duration, _ = time.ParseDuration(defaultValue)
	}
	return duration
}

// getEnvBool gets environment variable as bool or returns default value
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	boolVal, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolVal
}
