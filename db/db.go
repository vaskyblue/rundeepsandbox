package db

import (
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go-deepsandbox/config"
	"go-deepsandbox/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// InitDB initializes the database connection
func InitDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.DatabaseURL
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	return db, nil
}

// MigrateDB runs database migrations
func MigrateDB(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Dataset{},
		&models.CodeExecution{},
	)
}

// GetDB returns a database connection
func GetDB(db *gorm.DB) (*gorm.DB, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	
	// Ensure connection is alive
	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}
	
	return db, nil
}

// CloseDB closes the database connection
func CloseDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	
	return sqlDB.Close()
}

// InitRedis initializes the Redis connection
func InitRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       0,
	})
	
	return client, nil
}

// CurrentTimestamp returns the current timestamp in seconds
func CurrentTimestamp() float64 {
	return float64(time.Now().UnixNano()) / 1e9
}

// TaskQueue handles task queue operations
type TaskQueue struct {
	Redis *redis.Client
}

// NewTaskQueue creates a new task queue
func NewTaskQueue(redisClient *redis.Client) *TaskQueue {
	return &TaskQueue{
		Redis: redisClient,
	}
}

// GetTaskQueue returns a task queue instance
func GetTaskQueue(redisClient *redis.Client) *TaskQueue {
	return NewTaskQueue(redisClient)
}

// SubmitCodeExecution submits a new code execution task to the queue
func (tq *TaskQueue) SubmitCodeExecution(datasetID, code, userID string, timeout int, priority string) (string, error) {
	// In a real implementation, this would interact with a task queue system
	// For this example, we'll just create a task ID
	taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
	
	// Here you would actually submit the task to a queue system
	// For example, using Redis list operations or a dedicated task queue
	
	return taskID, nil
}

// GetTaskStatus gets the status of a task
func (tq *TaskQueue) GetTaskStatus(taskID string) (*models.TaskStatus, error) {
	// In a real implementation, this would fetch the task status from Redis or the database
	return &models.TaskStatus{
		TaskID:   taskID,
		Status:   "pending",
		Progress: 0,
	}, nil
}

// CancelTask cancels a task
func (tq *TaskQueue) CancelTask(taskID, userID string) (bool, error) {
	// In a real implementation, this would cancel the task in the queue
	return true, nil
} 