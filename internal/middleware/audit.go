package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/auth-service/internal/repository"
)

// AuditMiddlewareConfig holds configuration for audit middleware
type AuditMiddlewareConfig struct {
	AuditRepo repository.AuditRepository
	SkipPaths []string
}

// AuditMiddleware creates an audit logging middleware
func AuditMiddleware(cfg *AuditMiddlewareConfig) gin.HandlerFunc {
	skipPaths := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// Skip audit for specified paths
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Get user ID from context (if available)
		var userID *uuid.UUID
		if userIDStr, exists := c.Get("user_id"); exists {
			if id, err := uuid.Parse(userIDStr.(string)); err == nil {
				userID = &id
			}
		}

		// Get IP address
		ipAddress := getIPAddress(c)

		// Get user agent
		userAgent := c.Request.UserAgent()
		if userAgent == "" {
			userAgent = "Unknown"
		}

		// Determine action based on HTTP method and path
		action := determineAction(c.Request.Method, c.Request.URL.Path)

		// Create audit log entry
		auditLog := &repository.AuditLogCreate{
			UserID:    userID,
			Action:    action,
			IPAddress: ipAddress,
			UserAgent: &userAgent,
			Metadata: map[string]interface{}{
				"method":     c.Request.Method,
				"path":       c.Request.URL.Path,
				"user_agent": userAgent,
				"ip":         ipAddress,
			},
		}

		// Extract resource type and ID from path if possible
		if resourceType, resourceID := extractResourceInfo(c.Request.URL.Path); resourceType != "" {
			auditLog.ResourceType = &resourceType
			if resourceID != "" {
				if id, err := uuid.Parse(resourceID); err == nil {
					auditLog.ResourceID = &id
				}
			}
		}

		// Store the audit log to be saved after request completes
		c.Set("audit_log", auditLog)

		// Process request
		c.Next()

		// Get response status
		status := c.Writer.Status()
		
		// Update metadata with response info
		if metadata, ok := auditLog.Metadata.(map[string]interface{}); ok {
			metadata["status_code"] = status
			metadata["response_time"] = time.Since(time.Now()).String()
		}

		// Save audit log in background (non-blocking)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := cfg.AuditRepo.Create(ctx, auditLog); err != nil {
				// Log error but don't fail the request
				// In production, use proper logging
				println("Failed to save audit log:", err.Error())
			}
		}()
	}
}

// NewAuditMiddleware creates a new audit middleware with default configuration
func NewAuditMiddleware(auditRepo repository.AuditRepository) gin.HandlerFunc {
	config := &AuditMiddlewareConfig{
		AuditRepo: auditRepo,
		SkipPaths: []string{"/health"},
	}

	return AuditMiddleware(config)
}

// determineAction determines the audit action based on HTTP method and path
func determineAction(method, path string) string {
	// Check for auth-related paths
	if strings.Contains(path, "/auth/") {
		if strings.Contains(path, "/login") {
			return string(repository.AuditActionLogin)
		}
		if strings.Contains(path, "/logout") {
			return string(repository.AuditActionLogout)
		}
		if strings.Contains(path, "/register") {
			return string(repository.AuditActionRegister)
		}
		if strings.Contains(path, "/password-reset") {
			return string(repository.AuditActionPasswordReset)
		}
	}

	// Determine action based on HTTP method
	switch method {
	case "GET":
		if strings.Contains(path, "/search") {
			return string(repository.AuditActionSearch)
		}
		return string(repository.AuditActionView)
	case "POST":
		return string(repository.AuditActionCreate)
	case "PUT", "PATCH":
		return string(repository.AuditActionUpdate)
	case "DELETE":
		return string(repository.AuditActionDelete)
	default:
		return "UNKNOWN"
	}
}

// extractResourceInfo extracts resource type and ID from URL path
func extractResourceInfo(path string) (string, string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	
	if len(parts) < 2 {
		return "", ""
	}

	// Look for resource patterns like /api/v1/resource/{id}
	for i, part := range parts {
		if part == "foods" || part == "diary" || part == "users" {
			resourceType := part
			var resourceID string
			
			// Check if next part is a UUID
			if i+1 < len(parts) {
				nextPart := parts[i+1]
				if _, err := uuid.Parse(nextPart); err == nil {
					resourceID = nextPart
				}
			}
			
			return resourceType, resourceID
		}
	}

	return "", ""
}

// getIPAddress extracts IP address from request (same as in auth_handler)
func getIPAddress(c *gin.Context) *string {
	// Try to get IP from X-Forwarded-For header first
	ip := c.Request.Header.Get("X-Forwarded-For")
	if ip == "" {
		// Fall back to remote address
		ip = c.Request.RemoteAddr
		// Remove port if present
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}
	}
	
	if ip == "" {
		return nil
	}
	return &ip
}