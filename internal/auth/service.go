package auth

import (
	"database/sql"
	"errors"
	"strings"

	"mangahub/pkg/models"

	"golang.org/x/crypto/bcrypt"
)

// Service contains core authentication and user-management logic.
type Service struct {
	DB *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{DB: db}
}

// RegisterUser handles UC-001: create a new user account with validation and hashing.
func (s *Service) RegisterUser(username, email, password string) error {
	if username == "" || email == "" || password == "" {
		return errors.New("missing required fields")
	}
	if !isStrongPassword(password) {
		return errors.New("weak_password")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("hash_error")
	}

	_, err = s.DB.Exec(
		`INSERT INTO users (id, username, email, password_hash) VALUES (?, ?, ?, ?)`,
		"user_"+username, username, email, string(hash),
	)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "UNIQUE constraint failed: users.username") {
			return errors.New("username_exists")
		}
		if strings.Contains(msg, "UNIQUE constraint failed: users.email") {
			return errors.New("email_exists")
		}
		return err
	}
	return nil
}

// AuthenticateUser handles UC-002: validate credentials and return the user.
// The identifier can be either a username or an email.
func (s *Service) AuthenticateUser(identifier, password string, byEmail bool) (*models.User, error) {
	if identifier == "" || password == "" {
		return nil, errors.New("missing_credentials")
	}

	query := `SELECT id, username, email, password_hash, created_at FROM users WHERE `
	if byEmail {
		query += `email = ?`
	} else {
		query += `username = ?`
	}

	var u models.User
	err := s.DB.QueryRow(query, identifier).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("account_not_found")
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid_credentials")
	}

	return &u, nil
}

// isStrongPassword enforces a simple strength rule set:
// - at least 8 characters
// - must contain at least one letter and one digit.
func isStrongPassword(pw string) bool {
	if len(pw) < 8 {
		return false
	}
	hasLetter := false
	hasDigit := false
	for _, ch := range pw {
		switch {
		case ch >= '0' && ch <= '9':
			hasDigit = true
		case (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z'):
			hasLetter = true
		}
	}
	return hasLetter && hasDigit
}


