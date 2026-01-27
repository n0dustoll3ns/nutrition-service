package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware is a placeholder authentication middleware
// In a real application, this would validate JWT tokens or session cookies
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for health check endpoint
		if c.Request.URL.Path == "/health" {
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

		// TODO: In a real application, validate the JWT token here
		// For now, we'll just accept any non-empty token
		// This is a temporary implementation until auth service is complete

		// Set user ID in context (placeholder)
		// In real implementation, extract user ID from token
		// For now, use a fixed UUID for testing
		c.Set("user_id", "123e4567-e89b-12d3-a456-426614174000")
		c.Set("token", token)

		c.Next()
	}
}