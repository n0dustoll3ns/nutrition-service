package model

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenType represents the type of token
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
	TokenTypeReset   TokenType = "reset"
)

// TokenPair represents a pair of access and refresh tokens
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"` // "Bearer"
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// RefreshToken represents a refresh token in the database
type RefreshToken struct {
	ID         uuid.UUID  `db:"id"`
	UserID     uuid.UUID  `db:"user_id"`
	TokenHash  string     `db:"token_hash"`
	DeviceInfo *string    `db:"device_info"`
	IPAddress  *string    `db:"ip_address"`
	ExpiresAt  time.Time  `db:"expires_at"`
	RevokedAt  *time.Time `db:"revoked_at"`
	CreatedAt  time.Time  `db:"created_at"`
}

// PasswordResetToken represents a password reset token
type PasswordResetToken struct {
	ID        uuid.UUID  `db:"id"`
	UserID    uuid.UUID  `db:"user_id"`
	TokenHash string     `db:"token_hash"`
	ExpiresAt time.Time  `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
	CreatedAt time.Time  `db:"created_at"`
}

// RevokedAccessToken represents a revoked access token
type RevokedAccessToken struct {
	ID        uuid.UUID  `db:"id"`
	TokenID   string     `db:"token_id"`
	UserID    uuid.UUID  `db:"user_id"`
	ExpiresAt time.Time  `db:"expires_at"`
	RevokedAt time.Time  `db:"revoked_at"`
	Reason    *string    `db:"reason"`
}

// TokenClaims represents JWT claims
type TokenClaims struct {
	TokenID string    `json:"jti"`
	UserID  string    `json:"sub"`
	Email   string    `json:"email"`
	Exp     int64     `json:"exp"`
	Iat     int64     `json:"iat"`
	Type    TokenType `json:"type"`
}

// Valid implements jwt.Claims interface
func (c *TokenClaims) Valid() error {
	// Check if token is expired
	if c.Exp < time.Now().Unix() {
		return fmt.Errorf("token is expired")
	}
	return nil
}

// GetAudience implements jwt.Claims interface
func (c *TokenClaims) GetAudience() (jwt.ClaimStrings, error) {
	return jwt.ClaimStrings{}, nil
}

// GetExpirationTime implements jwt.Claims interface
func (c *TokenClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(time.Unix(c.Exp, 0)), nil
}

// GetIssuedAt implements jwt.Claims interface
func (c *TokenClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(time.Unix(c.Iat, 0)), nil
}

// GetIssuer implements jwt.Claims interface
func (c *TokenClaims) GetIssuer() (string, error) {
	return "", nil
}

// GetNotBefore implements jwt.Claims interface
func (c *TokenClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(time.Unix(c.Iat, 0)), nil
}

// GetSubject implements jwt.Claims interface
func (c *TokenClaims) GetSubject() (string, error) {
	return c.UserID, nil
}

// PasswordResetRequest represents a password reset request
type PasswordResetRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// PasswordResetConfirm represents password reset confirmation
type PasswordResetConfirm struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}
