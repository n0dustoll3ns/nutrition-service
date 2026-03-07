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
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/yourusername/auth-service/internal/config"
	"github.com/yourusername/auth-service/internal/handler"
	"github.com/yourusername/auth-service/internal/importer"
	"github.com/yourusername/auth-service/internal/middleware"
	"github.com/yourusername/auth-service/internal/model"
	"github.com/yourusername/auth-service/internal/repository"
	"github.com/yourusername/auth-service/internal/utils"
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
	recipeRepo := repository.NewRecipeRepository(db)
	diaryRepo := repository.NewDiaryRepository(db)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	// DEBUG: Create super user for testing (REMOVE BEFORE DEPLOYMENT)
	createSuperUserForDebug(userRepo, cfg)

	// Initialize handlers
	foodHandler := handler.NewFoodHandler(foodRepo, recipeRepo)
	recipeHandler := handler.NewRecipeHandler(recipeRepo, foodRepo)
	diaryHandler := handler.NewDiaryHandler(diaryRepo, foodRepo, db)
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
	
	// CORS middleware (must be before other middleware)
	router.Use(middleware.NewCORSMiddleware())
	
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

		// User profile route (protected)
		protected := apiV1.Group("/me")
		protected.Use(authMiddleware)
		{
			protected.GET("", authHandler.GetCurrentUser)
		}

		// Food routes (protected)
		foods := apiV1.Group("/foods")
		foods.Use(authMiddleware)
		{
			foods.GET("/search", foodHandler.SearchFoods)
			foods.GET("/:id", foodHandler.GetFoodByID)
		}

		// Recipe routes (protected)
		recipes := apiV1.Group("/recipes")
		recipes.Use(authMiddleware)
		{
			recipes.POST("", recipeHandler.CreateRecipe)
			recipes.GET("/my", recipeHandler.GetMyRecipes)
			recipes.GET("/public", recipeHandler.GetPublicRecipes)
			recipes.GET("/search", recipeHandler.SearchRecipes)
			recipes.GET("/my-book", recipeHandler.GetMyRecipeBook)
			recipes.GET("/:id", recipeHandler.GetRecipe)
			recipes.PUT("/:id", recipeHandler.UpdateRecipe)
			recipes.DELETE("/:id", recipeHandler.DeleteRecipe)
			recipes.POST("/:id/add-to-my-book", recipeHandler.AddRecipeToMyBook)
		}

		// Combined search route (protected)
		search := apiV1.Group("/search")
		search.Use(authMiddleware)
		{
			search.GET("/combined", foodHandler.SearchCombined)
		}

		// Diary routes (protected)
		diary := apiV1.Group("/diary")
		diary.Use(authMiddleware)
		{
			diary.GET("/entries", diaryHandler.GetDiaryEntries)
			diary.POST("/entries", diaryHandler.CreateFoodEntry)
			diary.PUT("/entries/:id", diaryHandler.UpdateFoodEntry)
			diary.DELETE("/entries/:id", diaryHandler.DeleteFoodEntry)
			diary.GET("/summary", diaryHandler.GetDiarySummary)
			diary.POST("/copy", diaryHandler.CopyDiaryEntries)
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

// createSuperUserForDebug creates a super user for testing/debugging purposes
// WARNING: This is a temporary hack for development. REMOVE BEFORE DEPLOYMENT!
func createSuperUserForDebug(userRepo repository.UserRepository, cfg *config.Config) {
	ctx := context.Background()
	
	// Check if super user already exists
	user, err := userRepo.GetByEmail(ctx, "test@test.com")
	if err == nil {
		// User already exists, update password to "test"
		log.Println("DEBUG: Super user 'test@test.com' already exists, updating password...")
		
		// Hash password
		passwordHash, err := utils.HashPassword("test", cfg.Security.BcryptCost)
		if err != nil {
			log.Printf("DEBUG: Failed to hash password for super user: %v", err)
			return
		}
		
		// Update password
		if err := userRepo.UpdatePassword(ctx, user.ID, passwordHash); err != nil {
			log.Printf("DEBUG: Failed to update password for super user: %v", err)
			return
		}
		
		// Ensure user is active and verified
		update := &model.UserUpdate{
			IsActive: boolPtr(true),
		}
		if err := userRepo.Update(ctx, user.ID, update); err != nil {
			log.Printf("DEBUG: Failed to update user status: %v", err)
		}
		
		log.Println("DEBUG: Super user password updated to 'test'")
		return
	}
	
	// User doesn't exist, create it
	log.Println("DEBUG: Creating super user for testing: test@test.com / test")
	
	// Hash password
	passwordHash, err := utils.HashPassword("test", cfg.Security.BcryptCost)
	if err != nil {
		log.Printf("DEBUG: Failed to hash password for super user: %v", err)
		return
	}
	
	// Create user
	now := time.Now()
	user = &model.User{
		ID:           uuid.New(),
		Email:        "test@test.com",
		PasswordHash: passwordHash,
		FirstName:    stringPtr("Super"),
		LastName:     stringPtr("User"),
		IsActive:     true,
		IsVerified:   true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	
	if err := userRepo.Create(ctx, user); err != nil {
		log.Printf("DEBUG: Failed to create super user: %v", err)
		return
	}
	
	log.Println("DEBUG: Super user created successfully")
}

// stringPtr returns a pointer to a string (helper function)
func stringPtr(s string) *string {
	return &s
}

// boolPtr returns a pointer to a bool (helper function)
func boolPtr(b bool) *bool {
	return &b
}
