package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/yourusername/auth-service/internal/config"
	"github.com/yourusername/auth-service/internal/model"
	"github.com/yourusername/auth-service/internal/repository"
	"github.com/yourusername/auth-service/internal/utils"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	userRepo        repository.UserRepository
	tokenRepo       repository.TokenRepository
	jwtManager      *utils.JWTManager
	passwordValidator *utils.PasswordValidator
	cfg             *config.Config
	validate        *validator.Validate
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	cfg *config.Config,
) *AuthHandler {
	return &AuthHandler{
		userRepo:        userRepo,
		tokenRepo:       tokenRepo,
		jwtManager:      utils.NewJWTManager(cfg),
		passwordValidator: utils.NewPasswordValidator(cfg),
		cfg:             cfg,
		validate:        validator.New(),
	}
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req model.UserCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Validate request
	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": err.Error(),
		})
		return
	}

	// Validate password strength
	if err := h.passwordValidator.Validate(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Weak password",
			"message": err.Error(),
		})
		return
	}

	// Check if user already exists
	ctx := c.Request.Context()
	exists, err := h.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Failed to check user existence",
		})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "User already exists",
			"message": "A user with this email already exists",
		})
		return
	}

	// Hash password
	passwordHash, err := utils.HashPassword(req.Password, h.cfg.Security.BcryptCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Failed to hash password",
		})
		return
	}

	// Create user
	now := time.Now()
	user := &model.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: passwordHash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		IsActive:     true,
		IsVerified:   false, // Email verification required
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.userRepo.Create(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Failed to create user",
		})
		return
	}

	// Generate tokens
	tokenPair, err := h.jwtManager.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Failed to generate tokens",
		})
		return
	}

	// Create refresh token record
	refreshTokenHash := hashToken(tokenPair.RefreshToken)
	refreshToken := &model.RefreshToken{
		ID:         uuid.New(),
		UserID:     user.ID,
		TokenHash:  refreshTokenHash,
		DeviceInfo: getDeviceInfo(c),
		IPAddress:  getIPAddress(c),
		ExpiresAt:  time.Now().Add(h.cfg.JWT.RefreshTokenExpiry),
		CreatedAt:  time.Now(),
	}

	if err := h.tokenRepo.CreateRefreshToken(ctx, refreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Failed to save refresh token",
		})
		return
	}

	// Send email verification if enabled
	if h.cfg.Email.Enabled {
		// TODO: Implement email verification
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user":    user.ToResponse(),
		"tokens":  tokenPair,
	})
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.UserLogin
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Validate request
	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": err.Error(),
		})
		return
	}

	// Get user by email
	ctx := c.Request.Context()
	user, err := h.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		// Return generic error to avoid user enumeration
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Invalid credentials",
			"message": "Invalid email or password",
		})
		return
	}

	// Check if user is active
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Account disabled",
			"message": "Your account has been disabled",
		})
		return
	}

	// Verify password
	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Invalid credentials",
			"message": "Invalid email or password",
		})
		return
	}

	// Update last login
	now := time.Now()
	if err := h.userRepo.UpdateLastLogin(ctx, user.ID, now); err != nil {
		// Log error but continue
		fmt.Printf("Failed to update last login: %v\n", err)
	}

	// Generate tokens
	tokenPair, err := h.jwtManager.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Failed to generate tokens",
		})
		return
	}

	// Create or update refresh token
	refreshTokenHash := hashToken(tokenPair.RefreshToken)
	deviceInfo := getDeviceInfo(c)
	
	// Check if refresh token exists for this device
	existingToken, err := h.tokenRepo.GetRefreshTokenByUserAndDevice(ctx, user.ID, deviceInfo)
	if err == nil {
		// Revoke old token
		h.tokenRepo.RevokeRefreshToken(ctx, existingToken.ID)
	}

	// Create new refresh token
	refreshToken := &model.RefreshToken{
		ID:         uuid.New(),
		UserID:     user.ID,
		TokenHash:  refreshTokenHash,
		DeviceInfo: deviceInfo,
		IPAddress:  getIPAddress(c),
		ExpiresAt:  time.Now().Add(h.cfg.JWT.RefreshTokenExpiry),
		CreatedAt:  time.Now(),
	}

	if err := h.tokenRepo.CreateRefreshToken(ctx, refreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Failed to save refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user":    user.ToResponse(),
		"tokens":  tokenPair,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get refresh token from request
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": "Refresh token is required",
		})
		return
	}

	// Validate refresh token
	claims, err := h.jwtManager.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid token",
			"message": err.Error(),
		})
		return
	}

	// Get user ID from claims
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid token",
			"message": "Invalid user ID in token",
		})
		return
	}

	// Hash the refresh token to find it in database
	tokenHash := hashToken(req.RefreshToken)
	
	// Get and revoke the refresh token
	ctx := c.Request.Context()
	token, err := h.tokenRepo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		// Token might already be revoked or not found
		c.JSON(http.StatusOK, gin.H{
			"message": "Logout successful",
		})
		return
	}

	// Verify token belongs to the user
	if token.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Forbidden",
			"message": "Token does not belong to user",
		})
		return
	}

	// Revoke the token
	if err := h.tokenRepo.RevokeRefreshToken(ctx, token.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Failed to revoke token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful",
	})
}

// Refresh handles token refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": "Refresh token is required",
		})
		return
	}

	// Validate refresh token
	claims, err := h.jwtManager.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Invalid token",
			"message": err.Error(),
		})
		return
	}

	// Get user ID from claims
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid token",
			"message": "Invalid user ID in token",
		})
		return
	}

	// Hash the refresh token to find it in database
	tokenHash := hashToken(req.RefreshToken)
	
	// Check if refresh token exists and is not revoked
	ctx := c.Request.Context()
	token, err := h.tokenRepo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Invalid token",
			"message": "Refresh token not found",
		})
		return
	}

	// Check if token is revoked
	if token.RevokedAt != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Token revoked",
			"message": "Refresh token has been revoked",
		})
		return
	}

	// Check if token is expired
	if token.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Token expired",
			"message": "Refresh token has expired",
		})
		return
	}

	// Verify token belongs to the user
	if token.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Forbidden",
			"message": "Token does not belong to user",
		})
		return
	}

	// Get user
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Failed to get user",
		})
		return
	}

	// Check if user is active
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Account disabled",
			"message": "Your account has been disabled",
		})
		return
	}

	// Generate new token pair
	tokenPair, err := h.jwtManager.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Failed to generate tokens",
		})
		return
	}

	// Create new refresh token record
	newRefreshTokenHash := hashToken(tokenPair.RefreshToken)
	newRefreshToken := &model.RefreshToken{
		ID:         uuid.New(),
		UserID:     user.ID,
		TokenHash:  newRefreshTokenHash,
		DeviceInfo: token.DeviceInfo,
		IPAddress:  token.IPAddress,
		ExpiresAt:  time.Now().Add(h.cfg.JWT.RefreshTokenExpiry),
		CreatedAt:  time.Now(),
	}

	// Revoke old token
	if err := h.tokenRepo.RevokeRefreshToken(ctx, token.ID); err != nil {
		// Log error but continue
		fmt.Printf("Failed to revoke old refresh token: %v\n", err)
	}

	// Create new token
	if err := h.tokenRepo.CreateRefreshToken(ctx, newRefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Failed to save refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token refreshed successfully",
		"tokens":  tokenPair,
	})
}

// PasswordResetRequest handles password reset request
func (h *AuthHandler) PasswordResetRequest(c *gin.Context) {
	var req model.PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Validate request
	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": err.Error(),
		})
		return
	}

	// Check if user exists
	ctx := c.Request.Context()
	user, err := h.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		// Return success even if user doesn't exist to avoid enumeration
		c.JSON(http.StatusOK, gin.H{
			"message": "If the email exists, a password reset link has been sent",
		})
		return
	}

	// Check if user is active
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Account disabled",
			"message": "Your account has been disabled",
		})
		return
	}

	// Generate reset token
	resetToken := uuid.New().String()
	tokenHash := hashToken(resetToken)
	
	// Create password reset token record
	resetTokenRecord := &model.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hours expiry
		CreatedAt: time.Now(),
	}

	if err := h.tokenRepo.CreatePasswordResetToken(ctx, resetTokenRecord); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Failed to create reset token",
		})
		return
	}

	// Send email if enabled
	if h.cfg.Email.Enabled {
		// TODO: Implement email sending with reset token
		// For now, return token in response (in production, this should be sent via email)
		c.JSON(http.StatusOK, gin.H{
			"message": "Password reset token generated",
			"token":   resetToken, // Remove this in production
		})
		return
	}

	// If email is disabled, return token in response (for testing)
	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset token generated",
		"token":   resetToken,
	})
}

// PasswordResetConfirm handles password reset confirmation
func (h *AuthHandler) PasswordResetConfirm(c *gin.Context) {
	var req model.PasswordResetConfirm
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Validate request
	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": err.Error(),
		})
		return
	}

	// Validate password strength
	if err := h.passwordValidator.Validate(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Weak password",
			"message": err.Error(),
		})
		return
	}

	// Hash the token to find it in database
	tokenHash := hashToken(req.Token)
	
	// Get password reset token
	ctx := c.Request.Context()
	resetToken, err := h.tokenRepo.GetPasswordResetTokenByHash(ctx, tokenHash)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid token",
			"message": "Password reset token is invalid or expired",
		})
		return
	}

	// Check if token is already used
	if resetToken.UsedAt != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Token already used",
			"message": "This password reset token has already been used",
		})
		return
	}

	// Check if token is expired
	if resetToken.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Token expired",
			"message": "Password reset token has expired",
		})
		return
	}

	// Get user
	user, err := h.userRepo.GetByID(ctx, resetToken.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Failed to get user",
		})
		return
	}

	// Check if user is active
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Account disabled",
			"message": "Your account has been disabled",
		})
		return
	}

	// Hash new password
	passwordHash, err := utils.HashPassword(req.Password, h.cfg.Security.BcryptCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Failed to hash password",
		})
		return
	}

	// Update user password using the new UpdatePassword method
	if err := h.userRepo.UpdatePassword(ctx, user.ID, passwordHash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Failed to update password",
		})
		return
	}

	// Mark token as used
	if err := h.tokenRepo.MarkPasswordResetTokenUsed(ctx, resetToken.ID); err != nil {
		// Log error but continue
		fmt.Printf("Failed to mark password reset token as used: %v\n", err)
	}

	// Revoke all refresh tokens for this user (security measure)
	if err := h.tokenRepo.RevokeAllRefreshTokensForUser(ctx, user.ID); err != nil {
		// Log error but continue
		fmt.Printf("Failed to revoke refresh tokens: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset successful",
	})
}

// GetCurrentUser returns the current authenticated user
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	// Get user ID from context (set by middleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User ID not found in context",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid user ID",
			"message": "Invalid user ID format",
		})
		return
	}

	// Get user
	ctx := c.Request.Context()
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "User not found",
			"message": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user.ToResponse(),
	})
}

// hashToken creates a SHA256 hash of a token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// getDeviceInfo extracts device information from request
func getDeviceInfo(c *gin.Context) *string {
	userAgent := c.Request.UserAgent()
	if userAgent == "" {
		return nil
	}
	return &userAgent
}

// getIPAddress extracts IP address from request
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
