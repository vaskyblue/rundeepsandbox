package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"go-deepsandbox/config"
	"go-deepsandbox/controllers"
	"go-deepsandbox/middleware"
)

// RegisterAuthRoutes registers authentication routes
func RegisterAuthRoutes(router *gin.Engine, db *gorm.DB, cfg *config.Config) {
	auth := middleware.NewAuth(db, cfg)
	authController := controllers.NewAuthController(db, cfg)

	authGroup := router.Group("/api/v1/auth")
	{
		// Public routes
		authGroup.POST("/token", authController.Login)
		authGroup.POST("/register", authController.Register)

		// Protected routes
		userGroup := authGroup.Group("")
		userGroup.Use(auth.AuthMiddleware())
		{
			userGroup.GET("/users/me", authController.GetCurrentUser)
			userGroup.PUT("/users/me", authController.UpdateUser)
		}

		// Admin routes
		adminGroup := authGroup.Group("")
		adminGroup.Use(auth.AdminMiddleware())
		{
			adminGroup.GET("/admin/users", authController.ListUsers)
		}
	}
}

// RegisterDatasetRoutes registers dataset routes
func RegisterDatasetRoutes(router *gin.Engine, db *gorm.DB, cfg *config.Config) {
	auth := middleware.NewAuth(db, cfg)
	datasetController := controllers.NewDatasetController(db, cfg)

	// All dataset routes require authentication
	datasetGroup := router.Group("/api/v1")
	datasetGroup.Use(auth.AuthMiddleware())
	{
		datasetGroup.POST("/datasets/upload", datasetController.UploadDataset)
		datasetGroup.GET("/datasets", datasetController.ListDatasets)
		datasetGroup.GET("/datasets/:dataset_id", datasetController.GetDataset)
		datasetGroup.DELETE("/datasets/:dataset_id", datasetController.DeleteDataset)
	}
}

// RegisterExecutionRoutes registers code execution routes
func RegisterExecutionRoutes(router *gin.Engine, db *gorm.DB, redisClient *redis.Client, cfg *config.Config) {
	auth := middleware.NewAuth(db, cfg)
	executionController := controllers.NewExecutionController(db, redisClient, cfg)

	// All execution routes require authentication
	executionGroup := router.Group("/api/v1")
	executionGroup.Use(auth.AuthMiddleware())
	{
		// Execution routes with quota middleware
		execQuotaGroup := executionGroup.Group("")
		execQuotaGroup.Use(auth.ExecutionQuotaMiddleware(redisClient))
		{
			execQuotaGroup.POST("/execute", executionController.ExecuteCode)
		}

		// Task management routes
		executionGroup.GET("/tasks/:task_id", executionController.GetTaskStatus)
		executionGroup.DELETE("/tasks/:task_id", executionController.CancelTask)

		// Admin routes
		adminGroup := executionGroup.Group("")
		adminGroup.Use(auth.AdminMiddleware())
		{
			adminGroup.GET("/admin/queue-status", executionController.GetQueueStatus)
		}
	}
} 