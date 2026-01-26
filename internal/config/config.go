package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Security SecurityConfig `mapstructure:"security"`
	Email    EmailConfig    `mapstructure:"email"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Importer ImporterConfig `mapstructure:"importer"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	Host         string        `mapstructure:"host"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	Schema          string        `mapstructure:"schema"`
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	AccessTokenSecret  string        `mapstructure:"access_token_secret"`
	RefreshTokenSecret string        `mapstructure:"refresh_token_secret"`
	AccessTokenExpiry  time.Duration `mapstructure:"access_token_expiry"`
	RefreshTokenExpiry time.Duration `mapstructure:"refresh_token_expiry"`
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	BcryptCost              int  `mapstructure:"bcrypt_cost"`
	PasswordMinLength       int  `mapstructure:"password_min_length"`
	PasswordRequireUppercase bool `mapstructure:"password_require_uppercase"`
	PasswordRequireLowercase bool `mapstructure:"password_require_lowercase"`
	PasswordRequireNumbers  bool `mapstructure:"password_require_numbers"`
	PasswordRequireSpecial  bool `mapstructure:"password_require_special"`
}

// EmailConfig holds email configuration
type EmailConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	SMTPHost     string `mapstructure:"smtp_host"`
	SMTPPort     int    `mapstructure:"smtp_port"`
	SMTPUsername string `mapstructure:"smtp_username"`
	SMTPPassword string `mapstructure:"smtp_password"`
	FromAddress  string `mapstructure:"from_address"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	RequestsPerMinute int  `mapstructure:"requests_per_minute"`
	Burst             int  `mapstructure:"burst"`
}

// ImporterConfig holds USDA food importer configuration
type ImporterConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	JSONPath       string `mapstructure:"json_path"`
	Schema         string `mapstructure:"schema"`
	ImportOnStartup bool   `mapstructure:"import_on_startup"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Set config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
	}

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Bind environment variables
	v.AutomaticEnv()

	// Unmarshal config
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.idle_timeout", "60s")

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "postgres")
	v.SetDefault("database.dbname", "auth_service")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "5m")
	v.SetDefault("database.schema", "auth")

	// JWT defaults
	v.SetDefault("jwt.access_token_secret", "change-me-in-production")
	v.SetDefault("jwt.refresh_token_secret", "change-me-in-production-too")
	v.SetDefault("jwt.access_token_expiry", "15m")
	v.SetDefault("jwt.refresh_token_expiry", "7d")

	// Security defaults
	v.SetDefault("security.bcrypt_cost", 12)
	v.SetDefault("security.password_min_length", 8)
	v.SetDefault("security.password_require_uppercase", true)
	v.SetDefault("security.password_require_lowercase", true)
	v.SetDefault("security.password_require_numbers", true)
	v.SetDefault("security.password_require_special", true)

	// Email defaults
	v.SetDefault("email.enabled", false)
	v.SetDefault("email.smtp_host", "smtp.gmail.com")
	v.SetDefault("email.smtp_port", 587)
	v.SetDefault("email.from_address", "noreply@example.com")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output", "stdout")

	// Rate limit defaults
	v.SetDefault("rate_limit.enabled", true)
	v.SetDefault("rate_limit.requests_per_minute", 60)
	v.SetDefault("rate_limit.burst", 10)

	// Importer defaults
	v.SetDefault("importer.enabled", true)
	v.SetDefault("importer.json_path", "/app/data/foods.json")
	v.SetDefault("importer.schema", "nutrition")
	v.SetDefault("importer.import_on_startup", true)
}
