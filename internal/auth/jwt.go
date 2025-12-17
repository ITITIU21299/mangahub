package auth

import (
	"errors"
	"time"

	"mangahub/pkg/models"

	"github.com/golang-jwt/jwt/v4"
)

// GenerateJWT creates a signed JWT for an authenticated user.
func GenerateJWT(secret []byte, u *models.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":   u.ID,
		"usr":   u.Username,
		"email": u.Email,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseUserIDFromToken validates a JWT and extracts the user ID ("sub" claim).
func ParseUserIDFromToken(secret []byte, raw string) (string, error) {
	tok, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	})
	if err != nil || !tok.Valid {
		return "", errors.New("invalid_token")
	}

	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid_claims")
	}
	id, _ := claims["sub"].(string)
	if id == "" {
		return "", errors.New("missing_sub")
	}
	return id, nil
}


