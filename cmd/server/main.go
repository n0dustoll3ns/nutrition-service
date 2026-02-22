package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/yourusername/auth-service/internal/config"
	"github.com/yourusername/auth-service/internal/handler"
	"github.com/yourusername/auth-service/internal/importer"
	"github.com/yourusername/auth-service/internal/middleware"
	"github.com/yourusername/auth-service/internal/repository"
)

func main() {
	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Start USDA food import in background
	go runFoodImport(cfg)

	// Connect to database
	db, err := connectToDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	foodRepo := repository.NewFoodRepository(db)
	diaryRepo := repository.NewDiaryRepository(db)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	// Initialize handlers
	foodHandler := handler.NewFoodHandler(foodRepo)
	diaryHandler := handler.NewDiaryHandler(diaryRepo, foodRepo)
	authHandler := handler.NewAuthHandler(userRepo, tokenRepo, cfg)

	// Set Gin mode
	if gin.Mode() == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	
	// Rate limiting middleware
	rateLimitConfig := middleware.RateLimiterConfig{
		Enabled:           cfg.RateLimit.Enabled,
		RequestsPerMinute: cfg.RateLimit.RequestsPerMinute,
		Burst:             cfg.RateLimit.Burst,
	}
	router.Use(middleware.NewRateLimitMiddleware(rateLimitConfig))
	
	// Audit logging middleware
	router.Use(middleware.NewAuditMiddleware(auditRepo))
	
	// Auth middleware (will be applied to protected routes)
	authMiddleware := middleware.NewAuthMiddleware(cfg, tokenRepo)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().UTC(),
		})
	})

	// API v1 routes
	apiV1 := router.Group("/api/v1")
	{
		// Auth routes (public)
		auth := apiV1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/password-reset-request", authHandler.PasswordResetRequest)
			auth.POST("/password-reset-confirm", authHandler.PasswordResetConfirm)
		}

		// Protected routes (require authentication)
		protected := apiV1.Group("/protected")
		protected.Use(authMiddleware)
		{
			protected.GET("/me", authHandler.GetCurrentUser)

			// Food routes (protected)
			foods := protected.Group("/foods")
			{
				foods.GET("/search", foodHandler.SearchFoods)
				foods.GET("/:id", foodHandler.GetFoodByID)
			}

			// Diary routes (protected)
			diary := protected.Group("/diary")
			{
				diary.GET("/entries", diaryHandler.GetDiaryEntries)
				diary.POST("/entries", diaryHandler.CreateFoodEntry)
				diary.PUT("/entries/:id", diaryHandler.UpdateFoodEntry)
				diary.DELETE("/entries/:id", diaryHandler.DeleteFoodEntry)
				diary.GET("/summary", diaryHandler.GetDiarySummary)
				diary.POST("/copy", diaryHandler.CopyDiaryEntries)
			}
		}
	}

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting server on %s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}

// connectToDatabase establishes a connection to PostgreSQL database
func connectToDatabase(cfg *config.Config) (*sql.DB, error) {
	// Build database URL from config
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	// Open database connection
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Connected to database at %s:%d/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	return db, nil
}

// runFoodImport runs the USDA food import process
func runFoodImport(cfg *config.Config) {
	// Check if importer is enabled
	if !cfg.Importer.Enabled {
		log.Println("USDA food importer is disabled in configuration")
		return
	}

	if !cfg.Importer.ImportOnStartup {
		log.Println("USDA food import on startup is disabled in configuration")
		return
	}

	log.Println("Starting USDA food import process...")
	
	// Build database URL from config
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	// Configure importer
	importerConfig := importer.Config{
		DatabaseURL: dbURL,
		JSONPath:    cfg.Importer.JSONPath,
		Schema:      cfg.Importer.Schema,
	}

	// Create and run importer
	imp := importer.New(importerConfig)
	
	// Run import with error handling
	if err := imp.Run(); err != nil {
		log.Printf("USDA food import failed: %v", err)
		log.Println("Server will continue running despite import failure")
	} else {
		log.Println("USDA food import completed successfully")
	}
}
