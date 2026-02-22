package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/auth-service/internal/config"
	"github.com/yourusername/auth-service/internal/repository"
	"github.com/yourusername/auth-service/internal/utils"
)

// AuthMiddlewareConfig holds configuration for auth middleware
type AuthMiddlewareConfig struct {
	JWTManager  *utils.JWTManager
	TokenRepo   repository.TokenRepository
	SkipPaths   []string
}

// AuthMiddleware creates an authentication middleware that validates JWT tokens
func AuthMiddleware(cfg *AuthMiddlewareConfig) gin.HandlerFunc {
	skipPaths := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// Skip authentication for specified paths
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Check if it's a Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Invalid authorization format. Expected: Bearer <token>",
			})
			c.Abort()
			return
		}

		token := parts[1]
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Token is empty",
			})
			c.Abort()
			return
		}

		// Validate access token
		claims, err := cfg.JWTManager.ValidateAccessToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Invalid or expired token: " + err.Error(),
			})
			c.Abort()
			return
		}

		// Check if token is revoked
		ctx := c.Request.Context()
		isRevoked, err := cfg.TokenRepo.IsAccessTokenRevoked(ctx, claims.TokenID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal server error",
				"message": "Failed to check token status",
			})
			c.Abort()
			return
		}

		if isRevoked {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Token has been revoked",
			})
			c.Abort()
			return
		}

		// Parse user ID from claims
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Invalid user ID in token",
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", userID.String())
		c.Set("user_email", claims.Email)
		c.Set("token_id", claims.TokenID)
		c.Set("token", token)

		c.Next()
	}
}

// NewAuthMiddleware creates a new auth middleware with default configuration
func NewAuthMiddleware(cfg *config.Config, tokenRepo repository.TokenRepository) gin.HandlerFunc {
	jwtManager := utils.NewJWTManager(cfg)
	
	config := &AuthMiddlewareConfig{
		JWTManager: jwtManager,
		TokenRepo:  tokenRepo,
		SkipPaths:  []string{"/health", "/api/v1/auth/register", "/api/v1/auth/login", "/api/v1/auth/password-reset-request", "/api/v1/auth/password-reset-confirm"},
	}

	return AuthMiddleware(config)
}
