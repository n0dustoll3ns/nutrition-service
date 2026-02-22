package utils

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"

	"github.com/yourusername/auth-service/internal/config"
	"golang.org/x/crypto/bcrypt"
)

// PasswordValidator validates password strength
type PasswordValidator struct {
	minLength       int
	requireUpper    bool
	requireLower    bool
	requireNumbers  bool
	requireSpecial  bool
}

// NewPasswordValidator creates a new password validator
func NewPasswordValidator(cfg *config.Config) *PasswordValidator {
	return &PasswordValidator{
		minLength:       cfg.Security.PasswordMinLength,
		requireUpper:    cfg.Security.PasswordRequireUppercase,
		requireLower:    cfg.Security.PasswordRequireLowercase,
		requireNumbers:  cfg.Security.PasswordRequireNumbers,
		requireSpecial:  cfg.Security.PasswordRequireSpecial,
	}
}

// Validate validates a password against security requirements
func (v *PasswordValidator) Validate(password string) error {
	if len(password) < v.minLength {
		return fmt.Errorf("password must be at least %d characters long", v.minLength)
	}

	if v.requireUpper {
		if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
			return fmt.Errorf("password must contain at least one uppercase letter")
		}
	}

	if v.requireLower {
		if !regexp.MustCompile(`[a-z]`).MatchString(password) {
			return fmt.Errorf("password must contain at least one lowercase letter")
		}
	}

	if v.requireNumbers {
		if !regexp.MustCompile(`[0-9]`).MatchString(password) {
			return fmt.Errorf("password must contain at least one number")
		}
	}

	if v.requireSpecial {
		if !regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?~]`).MatchString(password) {
			return fmt.Errorf("password must contain at least one special character")
		}
	}

	return nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string, cost int) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

// CheckPasswordHash compares a password with its hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateRandomPassword generates a random password
func GenerateRandomPassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{}|;:,.<>?"
	
	bytes := make([]byte, length)
	for i := range bytes {
		b, err := randInt(0, len(charset))
		if err != nil {
			return "", fmt.Errorf("failed to generate random byte: %w", err)
		}
		bytes[i] = charset[b]
	}
	
	return string(bytes), nil
}

// MaskPassword masks a password for logging
func MaskPassword(password string) string {
	if len(password) == 0 {
		return ""
	}
	
	// Show first and last character, mask the rest
	if len(password) <= 2 {
		return strings.Repeat("*", len(password))
	}
	
	return string(password[0]) + strings.Repeat("*", len(password)-2) + string(password[len(password)-1])
}

// randInt generates a random integer in range [min, max)
func randInt(min, max int) (int, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	
	// Convert bytes to uint64
	var n uint64
	for i := 0; i < 8; i++ {
		n = n<<8 | uint64(b[i])
	}
	
	return min + int(n%uint64(max-min)), nil
}