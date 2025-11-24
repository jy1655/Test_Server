package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Service handles authentication logic
type Service struct {
	db        *DB
	jwtSecret []byte
	jwtExpiry time.Duration
}

// Claims represents JWT claims
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// NewService creates a new auth service
func NewService(db *DB, jwtSecret string, jwtExpiry time.Duration) *Service {
	return &Service{
		db:        db,
		jwtSecret: []byte(jwtSecret),
		jwtExpiry: jwtExpiry,
	}
}

// Register creates a new user
func (s *Service) Register(req *CreateUserRequest) (*User, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	user, err := s.db.CreateUser(req.Username, req.Password)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Login authenticates a user and returns JWT token
func (s *Service) Login(req *LoginRequest) (*LoginResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Get user by username
	user, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		if err == ErrUserNotFound {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Check password
	if !CheckPassword(req.Password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// Update last login
	if err := s.db.UpdateLastLogin(user.ID); err != nil {
		// Log error but don't fail login
		fmt.Printf("Failed to update last login for user %d: %v\n", user.ID, err)
	}

	// Generate JWT token
	token, err := s.GenerateToken(user)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token: token,
		User:  user,
	}, nil
}

// GenerateToken generates a JWT token for a user
func (s *Service) GenerateToken(user *User) (string, error) {
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// ValidateToken validates a JWT token and returns claims
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrUnauthorized
}

// GetUserFromToken validates token and retrieves user
func (s *Service) GetUserFromToken(tokenString string) (*User, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		return nil, err
	}

	return user, nil
}
