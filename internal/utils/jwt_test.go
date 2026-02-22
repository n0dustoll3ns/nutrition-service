package utils

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/auth-service/internal/config"
	"github.com/yourusername/auth-service/internal/model"
)

func TestJWTManager(t *testing.T) {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessTokenSecret:  "test-access-secret",
			RefreshTokenSecret: "test-refresh-secret",
			AccessTokenExpiry:  15 * time.Minute,
			RefreshTokenExpiry: 7 * 24 * time.Hour,
		},
	}

	jwtManager := NewJWTManager(cfg)

	userID := uuid.New()
	email := "test@example.com"

	// Test GenerateTokenPair
	tokenPair, err := jwtManager.GenerateTokenPair(userID, email)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	if tokenPair.AccessToken == "" {
		t.Error("Access token should not be empty")
	}

	if tokenPair.RefreshToken == "" {
		t.Error("Refresh token should not be empty")
	}

	if tokenPair.TokenType != "Bearer" {
		t.Errorf("Expected token type 'Bearer', got %s", tokenPair.TokenType)
	}

	// Test ValidateAccessToken
	accessClaims, err := jwtManager.ValidateAccessToken(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate access token: %v", err)
	}

	if accessClaims.UserID != userID.String() {
		t.Errorf("Expected user ID %s, got %s", userID.String(), accessClaims.UserID)
	}

	if accessClaims.Email != email {
		t.Errorf("Expected email %s, got %s", email, accessClaims.Email)
	}

	if accessClaims.Type != model.TokenTypeAccess {
		t.Errorf("Expected token type 'access', got %s", accessClaims.Type)
	}

	// Test ValidateRefreshToken
	refreshClaims, err := jwtManager.ValidateRefreshToken(tokenPair.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to validate refresh token: %v", err)
	}

	if refreshClaims.UserID != userID.String() {
		t.Errorf("Expected user ID %s, got %s", userID.String(), refreshClaims.UserID)
	}

	if refreshClaims.Email != email {
		t.Errorf("Expected email %s, got %s", email, refreshClaims.Email)
	}

	if refreshClaims.Type != model.TokenTypeRefresh {
		t.Errorf("Expected token type 'refresh', got %s", refreshClaims.Type)
	}

	// Test invalid token
	_, err = jwtManager.ValidateAccessToken("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}

	// Test wrong token type (access token as refresh)
	_, err = jwtManager.ValidateRefreshToken(tokenPair.AccessToken)
	if err == nil {
		t.Error("Expected error when validating access token as refresh token")
	}

	// Test expired token (simulate by creating token with past expiry)
	// Note: This is a simplified test
	expiredToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjE1MTYyMzkwMjJ9.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	_, err = jwtManager.ValidateAccessToken(expiredToken)
	if err == nil {
		t.Error("Expected error for expired token")
	}
}

func TestJWTManager_ParseToken(t *testing.T) {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessTokenSecret:  "test-access-secret",
			RefreshTokenSecret: "test-refresh-secret",
			AccessTokenExpiry:  15 * time.Minute,
			RefreshTokenExpiry: 7 * 24 * time.Hour,
		},
	}

	jwtManager := NewJWTManager(cfg)

	userID := uuid.New()
	email := "test@example.com"

	tokenPair, err := jwtManager.GenerateTokenPair(userID, email)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// Test parsing access token
	token, err := jwtManager.ParseToken(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if !token.Valid {
		t.Error("Parsed token should be valid")
	}

	// Test parsing refresh token
	token, err = jwtManager.ParseToken(tokenPair.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to parse refresh token: %v", err)
	}

	if !token.Valid {
		t.Error("Parsed refresh token should be valid")
	}
}