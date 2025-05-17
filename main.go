package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"go-deepsandbox/config"
	"go-deepsandbox/db"
	"go-deepsandbox/middleware"
	"go-deepsandbox/routes"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Initialize configuration
	cfg := config.NewConfig()

	// Create datasets directory if it doesn't exist
	err = os.MkdirAll(cfg.DatasetsDir, 0755)
	if err != nil {
		log.Fatalf("Failed to create datasets directory: %v", err)
	}

	// Initialize database connection
	database, err := db.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate database schemas
	err = db.MigrateDB(database)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Initialize Redis connection for rate limiting and task queue
	redisClient, err := db.InitRedis(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Create Gin router
	router := gin.Default()

	// Apply middleware
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.RateLimitMiddleware(redisClient, cfg))

	// Register routes
	routes.RegisterAuthRoutes(router, database, cfg)
	routes.RegisterDatasetRoutes(router, database, cfg)
	routes.RegisterExecutionRoutes(router, database, redisClient, cfg)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"version":   cfg.APIVersion,
			"timestamp": db.CurrentTimestamp(),
		})
	})

	// Root endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"name":        cfg.APITitle,
			"description": cfg.APIDescription,
			"version":     cfg.APIVersion,
			"docs_url":    "/docs",
		})
	})

	// Start server
	serverAddr := fmt.Sprintf(":%d", cfg.ServerPort)
	log.Printf("Starting DeepSandbox API server on %s\n", serverAddr)
	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
} 