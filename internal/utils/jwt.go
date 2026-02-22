package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/yourusername/auth-service/internal/config"
	"github.com/yourusername/auth-service/internal/model"
)

// JWTManager manages JWT token operations
type JWTManager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(cfg *config.Config) *JWTManager {
	return &JWTManager{
		accessSecret:  []byte(cfg.JWT.AccessTokenSecret),
		refreshSecret: []byte(cfg.JWT.RefreshTokenSecret),
		accessExpiry:  cfg.JWT.AccessTokenExpiry,
		refreshExpiry: cfg.JWT.RefreshTokenExpiry,
	}
}

// GenerateTokenPair generates a new access and refresh token pair
func (m *JWTManager) GenerateTokenPair(userID uuid.UUID, email string) (*model.TokenPair, error) {
	// Generate token IDs
	accessTokenID, err := generateTokenID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token ID: %w", err)
	}

	refreshTokenID, err := generateTokenID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token ID: %w", err)
	}

	// Create access token
	accessToken, err := m.createToken(
		accessTokenID,
		userID,
		email,
		model.TokenTypeAccess,
		m.accessSecret,
		m.accessExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create access token: %w", err)
	}

	// Create refresh token
	refreshToken, err := m.createToken(
		refreshTokenID,
		userID,
		email,
		model.TokenTypeRefresh,
		m.refreshSecret,
		m.refreshExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	return &model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(m.accessExpiry.Seconds()),
	}, nil
}

// ValidateAccessToken validates an access token
func (m *JWTManager) ValidateAccessToken(tokenString string) (*model.TokenClaims, error) {
	return m.validateToken(tokenString, m.accessSecret, model.TokenTypeAccess)
}

// ValidateRefreshToken validates a refresh token
func (m *JWTManager) ValidateRefreshToken(tokenString string) (*model.TokenClaims, error) {
	return m.validateToken(tokenString, m.refreshSecret, model.TokenTypeRefresh)
}

// ParseToken parses a token without validation (for debugging/inspection)
func (m *JWTManager) ParseToken(tokenString string) (*jwt.Token, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	
	// Try with access secret first
	token, err := parser.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return m.accessSecret, nil
	})
	
	if err == nil && token.Valid {
		return token, nil
	}
	
	// Try with refresh secret
	token, err = parser.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return m.refreshSecret, nil
	})
	
	return token, err
}

// createToken creates a JWT token
func (m *JWTManager) createToken(
	tokenID string,
	userID uuid.UUID,
	email string,
	tokenType model.TokenType,
	secret []byte,
	expiry time.Duration,
) (string, error) {
	now := time.Now()
	expiresAt := now.Add(expiry)

	claims := &model.TokenClaims{
		TokenID: tokenID,
		UserID:  userID.String(),
		Email:   email,
		Exp:     expiresAt.Unix(),
		Iat:     now.Unix(),
		Type:    tokenType,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// validateToken validates a JWT token
func (m *JWTManager) validateToken(
	tokenString string,
	secret []byte,
	expectedType model.TokenType,
) (*model.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &model.TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*model.TokenClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Validate token type
	if claims.Type != expectedType {
		return nil, fmt.Errorf("invalid token type: expected %s, got %s", expectedType, claims.Type)
	}

	// Validate expiration
	if time.Unix(claims.Exp, 0).Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}

	return claims, nil
}

// generateTokenID generates a random token ID
func generateTokenID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}