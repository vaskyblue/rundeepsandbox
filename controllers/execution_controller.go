package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"go-deepsandbox/config"
	"go-deepsandbox/db"
	"go-deepsandbox/middleware"
	"go-deepsandbox/models"
)

// ExecutionController handles code execution related endpoints
type ExecutionController struct {
	DB          *gorm.DB
	Config      *config.Config
	RedisClient *redis.Client
	TaskQueue   *db.TaskQueue
}

// NewExecutionController creates a new execution controller
func NewExecutionController(database *gorm.DB, redisClient *redis.Client, cfg *config.Config) *ExecutionController {
	return &ExecutionController{
		DB:          database,
		Config:      cfg,
		RedisClient: redisClient,
		TaskQueue:   db.GetTaskQueue(redisClient),
	}
}

// ExecuteCode submits code for asynchronous execution
func (ec *ExecutionController) ExecuteCode(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found in context"})
		return
	}
	user := userInterface.(models.User)

	// Parse request
	var request models.CodeExecutionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify the dataset exists and user has access
	var dataset models.Dataset
	if err := ec.DB.Where("id = ?", request.DatasetID).First(&dataset).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dataset not found"})
		return
	}

	// Check user has access to this dataset
	isAdmin := false
	for _, role := range user.Roles {
		if role == "admin" {
			isAdmin = true
			break
		}
	}

	if dataset.UserID != user.ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have access to this dataset"})
		return
	}

	// Get user's max execution time
	maxExecutionTime := ec.Config.ContainerTimeout
	
	// Parse quota from JSON if it exists
	if len(user.Quota) > 0 {
		var quotaMap map[string]int
		if err := json.Unmarshal(user.Quota, &quotaMap); err == nil {
			if quota, ok := quotaMap["max_execution_time"]; ok && quota > 0 {
				maxExecutionTime = quota
			}
		}
	}

	// Determine timeout
	timeout := maxExecutionTime
	if request.Timeout != nil && *request.Timeout > 0 && *request.Timeout < maxExecutionTime {
		timeout = *request.Timeout
	}

	// Record execution in database
	execution := models.CodeExecution{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		DatasetID: request.DatasetID,
		Code:      request.Code,
		Status:    "queued",
		StartTime: 0,
		EndTime:   0,
		Results:   "",
		Error:     "",
	}

	if err := ec.DB.Create(&execution).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record execution"})
		return
	}

	// Submit to queue
	taskID, err := ec.TaskQueue.SubmitCodeExecution(
		request.DatasetID,
		request.Code,
		user.Username,
		timeout,
		"normal",
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit task to queue"})
		return
	}

	// Update execution record with task ID
	execution.ID = taskID
	ec.DB.Save(&execution)

	// Track execution for quota
	middleware.TrackExecution(ec.RedisClient, user.ID)

	// Start background processing (in a real implementation this would be a goroutine)
	go ec.processExecution(taskID, request.DatasetID, user.ID)

	c.JSON(http.StatusAccepted, gin.H{
		"task_id": taskID,
		"status":  "queued",
		"message": "Code submitted for execution",
	})
}

// GetTaskStatus checks the status of a task
func (ec *ExecutionController) GetTaskStatus(c *gin.Context) {
	// Get task ID from URL
	taskID := c.Param("task_id")

	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found in context"})
		return
	}
	user := userInterface.(models.User)

	// Get execution from database
	var execution models.CodeExecution
	if err := ec.DB.Where("id = ?", taskID).First(&execution).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Check permissions
	isAdmin := false
	for _, role := range user.Roles {
		if role == "admin" {
			isAdmin = true
			break
		}
	}

	if execution.UserID != user.ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this task"})
		return
	}

	// Get status from task queue
	status, err := ec.TaskQueue.GetTaskStatus(taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get task status"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// CancelTask cancels a task
func (ec *ExecutionController) CancelTask(c *gin.Context) {
	// Get task ID from URL
	taskID := c.Param("task_id")

	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found in context"})
		return
	}
	user := userInterface.(models.User)

	// Get execution from database
	var execution models.CodeExecution
	if err := ec.DB.Where("id = ?", taskID).First(&execution).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Check permissions
	isAdmin := false
	for _, role := range user.Roles {
		if role == "admin" {
			isAdmin = true
			break
		}
	}

	if execution.UserID != user.ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to cancel this task"})
		return
	}

	// Try to cancel the task
	cancelled, err := ec.TaskQueue.CancelTask(taskID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel task"})
		return
	}

	if !cancelled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to cancel task: task may have completed or doesn't exist"})
		return
	}

	// Update status in database
	execution.Status = "cancelled"
	execution.EndTime = float64(time.Now().Unix())
	ec.DB.Save(&execution)

	c.JSON(http.StatusOK, gin.H{
		"status":  "cancelled",
		"message": "Task has been cancelled",
	})
}

// GetUserExecutions returns all executions for a user
func (ec *ExecutionController) GetUserExecutions(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found in context"})
		return
	}
	user := userInterface.(models.User)

	// Query executions
	var executions []models.CodeExecution
	if err := ec.DB.Where("user_id = ?", user.ID).Order("created_at DESC").Find(&executions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch executions"})
		return
	}

	c.JSON(http.StatusOK, executions)
}

// GetQueueStatus returns queue statistics
func (ec *ExecutionController) GetQueueStatus(c *gin.Context) {
	// In a real implementation, this would query the task queue system
	c.JSON(http.StatusOK, gin.H{
		"queued":   0,
		"running":  0,
		"completed": 0,
		"failed":   0,
	})
}

// processExecution simulates processing an execution task
func (ec *ExecutionController) processExecution(taskID, datasetID, userID string) {
	// In a real implementation, this would be handled by a worker
	// For this example, we'll just update the status in the database
	
	// Get execution from database
	var execution models.CodeExecution
	if err := ec.DB.Where("id = ?", taskID).First(&execution).Error; err != nil {
		return
	}
	
	// Update status to running
	execution.Status = "running"
	execution.StartTime = float64(time.Now().Unix())
	ec.DB.Save(&execution)
	
	// Simulate processing time
	time.Sleep(2 * time.Second)
	
	// Update status to completed
	execution.Status = "completed"
	execution.EndTime = float64(time.Now().Unix())
	execution.Results = `{"result": "Execution completed successfully."}`
	ec.DB.Save(&execution)
} 