package auth

import (
	"golang.org/x/crypto/bcrypt"
)

const (
	// Cost for bcrypt hashing (higher = more secure but slower)
	// 12 is a good balance between security and performance
	bcryptCost = 12
)

// HashPassword generates bcrypt hash from plain text password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword compares plain text password with hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
