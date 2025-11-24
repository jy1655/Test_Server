package auth

import (
	"errors"
	"regexp"
	"time"
)

// User represents a user in the system
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Never expose password hash
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// CreateUserRequest represents user creation request
type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginRequest represents login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents login response
type LoginResponse struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

var (
	ErrInvalidUsername      = errors.New("invalid username: must be 3-20 characters, alphanumeric and underscore only")
	ErrInvalidPassword      = errors.New("invalid password: must be at least 8 characters")
	ErrUsernameTaken        = errors.New("username already taken")
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUnauthorized         = errors.New("unauthorized")
)

// Username validation regex: 3-20 characters, alphanumeric and underscore
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)

// ValidateUsername checks if username meets requirements
func ValidateUsername(username string) error {
	if !usernameRegex.MatchString(username) {
		return ErrInvalidUsername
	}
	return nil
}

// ValidatePassword checks if password meets requirements
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrInvalidPassword
	}
	return nil
}

// Validate validates user creation request
func (r *CreateUserRequest) Validate() error {
	if err := ValidateUsername(r.Username); err != nil {
		return err
	}
	if err := ValidatePassword(r.Password); err != nil {
		return err
	}
	return nil
}

// Validate validates login request
func (r *LoginRequest) Validate() error {
	if r.Username == "" || r.Password == "" {
		return ErrInvalidCredentials
	}
	return nil
}
