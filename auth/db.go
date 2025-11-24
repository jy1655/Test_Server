package auth

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps database operations for user management
type DB struct {
	conn *sql.DB
}

// NewDB creates a new database connection and initializes schema
func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Initialize schema
	if err := initSchema(conn); err != nil {
		conn.Close()
		return nil, err
	}

	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// initSchema creates the users table if it doesn't exist
func initSchema(conn *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		last_login_at DATETIME
	);

	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	`

	_, err := conn.Exec(schema)
	return err
}

// CreateUser creates a new user with hashed password
func (db *DB) CreateUser(username, password string) (*User, error) {
	// Validate input
	req := CreateUserRequest{Username: username, Password: password}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Check if username already exists
	exists, err := db.UsernameExists(username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameTaken
	}

	// Hash password
	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Insert user
	now := time.Now()
	result, err := db.conn.Exec(
		"INSERT INTO users (username, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?)",
		username, passwordHash, now, now,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// GetUserByUsername retrieves a user by username
func (db *DB) GetUserByUsername(username string) (*User, error) {
	user := &User{}
	err := db.conn.QueryRow(
		"SELECT id, username, password_hash, created_at, updated_at, last_login_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (db *DB) GetUserByID(id int64) (*User, error) {
	user := &User{}
	err := db.conn.QueryRow(
		"SELECT id, username, password_hash, created_at, updated_at, last_login_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// UsernameExists checks if a username is already taken
func (db *DB) UsernameExists(username string) (bool, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdateLastLogin updates the last login timestamp for a user
func (db *DB) UpdateLastLogin(userID int64) error {
	now := time.Now()
	_, err := db.conn.Exec(
		"UPDATE users SET last_login_at = ?, updated_at = ? WHERE id = ?",
		now, now, userID,
	)
	return err
}

// ListUsers returns all users (for admin purposes)
func (db *DB) ListUsers() ([]*User, error) {
	rows, err := db.conn.Query(
		"SELECT id, username, password_hash, created_at, updated_at, last_login_at FROM users ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		if err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// DeleteUser deletes a user by ID
func (db *DB) DeleteUser(userID int64) error {
	result, err := db.conn.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}
