package utils

import (
	"testing"

	"github.com/yourusername/auth-service/internal/config"
)

func TestPasswordValidator(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			PasswordMinLength:       8,
			PasswordRequireUppercase: true,
			PasswordRequireLowercase: true,
			PasswordRequireNumbers:   true,
			PasswordRequireSpecial:   true,
		},
	}

	validator := NewPasswordValidator(cfg)

	// Test valid password
	validPassword := "Test123!@#"
	err := validator.Validate(validPassword)
	if err != nil {
		t.Errorf("Valid password should pass validation: %v", err)
	}

	// Test too short password
	shortPassword := "Test1!"
	err = validator.Validate(shortPassword)
	if err == nil {
		t.Error("Too short password should fail validation")
	}

	// Test missing uppercase
	noUpperPassword := "test123!@#"
	err = validator.Validate(noUpperPassword)
	if err == nil {
		t.Error("Password without uppercase should fail validation")
	}

	// Test missing lowercase
	noLowerPassword := "TEST123!@#"
	err = validator.Validate(noLowerPassword)
	if err == nil {
		t.Error("Password without lowercase should fail validation")
	}

	// Test missing numbers
	noNumberPassword := "TestPassword!@#"
	err = validator.Validate(noNumberPassword)
	if err == nil {
		t.Error("Password without numbers should fail validation")
	}

	// Test missing special characters
	noSpecialPassword := "TestPassword123"
	err = validator.Validate(noSpecialPassword)
	if err == nil {
		t.Error("Password without special characters should fail validation")
	}
}

func TestHashPassword(t *testing.T) {
	password := "TestPassword123!@#"

	// Test hash password
	hash, err := HashPassword(password, 10) // Use lower cost for faster tests
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	if hash == password {
		t.Error("Hash should not be the same as password")
	}

	// Test check password hash
	if !CheckPasswordHash(password, hash) {
		t.Error("Password should match hash")
	}

	// Test wrong password
	if CheckPasswordHash("WrongPassword", hash) {
		t.Error("Wrong password should not match hash")
	}

	// Test empty password
	hash2, err := HashPassword("", 10)
	if err != nil {
		t.Fatalf("Failed to hash empty password: %v", err)
	}

	if !CheckPasswordHash("", hash2) {
		t.Error("Empty password should match its hash")
	}
}

func TestGenerateRandomPassword(t *testing.T) {
	// Test generate random password
	password, err := GenerateRandomPassword(16)
	if err != nil {
		t.Fatalf("Failed to generate random password: %v", err)
	}

	if len(password) != 16 {
		t.Errorf("Expected password length 16, got %d", len(password))
	}

	// Test generate password with different lengths
	for _, length := range []int{8, 12, 20, 32} {
		password, err := GenerateRandomPassword(length)
		if err != nil {
			t.Fatalf("Failed to generate random password of length %d: %v", length, err)
		}

		if len(password) != length {
			t.Errorf("Expected password length %d, got %d", length, len(password))
		}
	}

	// Test invalid length
	_, err = GenerateRandomPassword(0)
	if err == nil {
		t.Error("Expected error for zero length password")
	}

	_, err = GenerateRandomPassword(-1)
	if err == nil {
		t.Error("Expected error for negative length password")
	}
}

func TestMaskPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected string
	}{
		{"Empty password", "", ""},
		{"Single character", "a", "*"},
		{"Two characters", "ab", "**"},
		{"Three characters", "abc", "a*c"},
		{"Long password", "password123", "p********3"},
		{"Very long password", "superSecretPassword123!@#", "s************************#"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskPassword(tt.password)
			if result != tt.expected {
				t.Errorf("MaskPassword(%q) = %q, expected %q", tt.password, result, tt.expected)
			}
		})
	}
}