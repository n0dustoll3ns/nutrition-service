package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiterConfig holds configuration for rate limiter
type RateLimiterConfig struct {
	Enabled           bool
	RequestsPerMinute int
	Burst             int
	SkipPaths         []string
}

// rateLimiter maintains rate limiting state
type rateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	config   RateLimiterConfig
}

// visitor represents a single visitor with rate limiter
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// newRateLimiter creates a new rate limiter
func newRateLimiter(config RateLimiterConfig) *rateLimiter {
	return &rateLimiter{
		visitors: make(map[string]*visitor),
		config:   config,
	}
}

// getVisitor returns or creates a visitor for the given IP
func (rl *rateLimiter) getVisitor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(
			rate.Limit(rl.config.RequestsPerMinute)/60,
			rl.config.Burst,
		)
		rl.visitors[ip] = &visitor{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// cleanupVisitors removes old visitors to prevent memory leak
func (rl *rateLimiter) cleanupVisitors() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for ip, v := range rl.visitors {
		if time.Since(v.lastSeen) > 5*time.Minute {
			delete(rl.visitors, ip)
		}
	}
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(config RateLimiterConfig) gin.HandlerFunc {
	if !config.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	limiter := newRateLimiter(config)

	// Start cleanup goroutine
	go func() {
		for {
			time.Sleep(time.Minute)
			limiter.cleanupVisitors()
		}
	}()

	return func(c *gin.Context) {
		// Skip rate limiting for specified paths
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Get client IP
		ip := getClientIP(c)

		// Get or create rate limiter for this IP
		lim := limiter.getVisitor(ip)

		// Check if request is allowed
		if !lim.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too Many Requests",
				"message": "Rate limit exceeded. Please try again later.",
				"retry_after": "60 seconds",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// NewRateLimitMiddleware creates a new rate limit middleware with default configuration
func NewRateLimitMiddleware(cfg RateLimiterConfig) gin.HandlerFunc {
	defaultConfig := RateLimiterConfig{
		Enabled:           cfg.Enabled,
		RequestsPerMinute: cfg.RequestsPerMinute,
		Burst:             cfg.Burst,
		SkipPaths: []string{
			"/health",
			"/api/v1/auth/login",    // Allow more attempts for login
			"/api/v1/auth/register", // Allow registration attempts
		},
	}

	return RateLimitMiddleware(defaultConfig)
}

// getClientIP extracts client IP address from request
func getClientIP(c *gin.Context) string {
	// Try to get IP from X-Forwarded-For header first
	ip := c.Request.Header.Get("X-Forwarded-For")
	if ip == "" {
		// Fall back to remote address
		ip = c.Request.RemoteAddr
		// Remove port if present
		if idx := lastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}
	}
	
	return ip
}

// lastIndex returns the index of the last instance of sep in s
func lastIndex(s, sep string) int {
	// Simple implementation for finding last index
	for i := len(s) - len(sep); i >= 0; i-- {
		if s[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}