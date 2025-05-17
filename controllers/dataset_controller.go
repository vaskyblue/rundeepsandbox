package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"go-deepsandbox/config"
	"go-deepsandbox/models"
)

// DatasetController handles dataset related endpoints
type DatasetController struct {
	DB     *gorm.DB
	Config *config.Config
}

// NewDatasetController creates a new dataset controller
func NewDatasetController(db *gorm.DB, cfg *config.Config) *DatasetController {
	return &DatasetController{
		DB:     db,
		Config: cfg,
	}
}

// UploadDataset handles dataset uploads
func (dc *DatasetController) UploadDataset(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found in context"})
		return
	}
	user := userInterface.(models.User)

	// Get file from request
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}
	defer file.Close()

	// Validate file format
	filename := header.Filename
	if !strings.HasSuffix(filename, ".csv") && !strings.HasSuffix(filename, ".parquet") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported file format. Use CSV or Parquet."})
		return
	}

	// Generate unique ID for dataset
	datasetID := uuid.New().String()

	// Create directory for dataset if it doesn't exist
	datasetDir := filepath.Join(dc.Config.DatasetsDir, user.ID)
	err = os.MkdirAll(datasetDir, 0755)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory for dataset"})
		return
	}

	// Create file path
	filePath := filepath.Join(datasetDir, datasetID+filepath.Ext(filename))

	// Save file
	out, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	defer out.Close()

	// Copy file data
	size, err := io.Copy(out, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to copy file data"})
		return
	}

	// Get file size in MB
	sizeMB := float64(size) / (1024 * 1024)

	// Check user quota
	maxDatasetSizeMB := 2000
	
	// Parse quota from JSON if it exists
	if len(user.Quota) > 0 {
		var quotaMap map[string]int
		if err := json.Unmarshal(user.Quota, &quotaMap); err == nil {
			if quota, ok := quotaMap["max_dataset_size_mb"]; ok && quota > 0 {
				maxDatasetSizeMB = quota
			}
		}
	}

	if sizeMB > float64(maxDatasetSizeMB) {
		// Remove file if it exceeds quota
		os.Remove(filePath)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("File size exceeds the allowed limit of %d MB", maxDatasetSizeMB),
		})
		return
	}

	// In a real implementation, you would analyze the file here to extract row count, columns, etc.
	// For this example, we'll use placeholder values
	rowCount := 1000
	columns := []string{"column1", "column2", "column3"}
	schema := "{}"

	// Create dataset record
	dataset := models.Dataset{
		ID:          datasetID,
		UserID:      user.ID,
		Filename:    filename,
		ContentType: header.Header.Get("Content-Type"),
		Size:        size,
		SizeMB:      math.Round(sizeMB*100) / 100, // Round to 2 decimal places
		RowCount:    rowCount,
		Columns:     columns,
		Schema:      schema,
	}

	// Save to database
	if err := dc.DB.Create(&dataset).Error; err != nil {
		// Remove file if database operation fails
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save dataset metadata"})
		return
	}

	c.JSON(http.StatusCreated, dataset.ToDatasetMetadata())
}

// ListDatasets lists datasets for the current user
func (dc *DatasetController) ListDatasets(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found in context"})
		return
	}
	user := userInterface.(models.User)

	// Check if user is admin
	isAdmin := false
	for _, role := range user.Roles {
		if role == "admin" {
			isAdmin = true
			break
		}
	}

	var datasets []models.Dataset
	query := dc.DB

	// For regular users, only show their own datasets
	if !isAdmin {
		query = query.Where("user_id = ?", user.ID)
	}

	// Execute query
	if err := query.Find(&datasets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch datasets"})
		return
	}

	// Convert to response format
	response := make([]models.DatasetMetadata, len(datasets))
	for i, dataset := range datasets {
		response[i] = dataset.ToDatasetMetadata()
	}

	c.JSON(http.StatusOK, response)
}

// GetDataset gets information and preview of a dataset
func (dc *DatasetController) GetDataset(c *gin.Context) {
	// Get dataset ID from URL
	datasetID := c.Param("dataset_id")

	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found in context"})
		return
	}
	user := userInterface.(models.User)

	// Get limit from query parameter
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 1000 {
		limit = 100
	}

	// Get dataset from database
	var dataset models.Dataset
	if err := dc.DB.Where("id = ?", datasetID).First(&dataset).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dataset not found"})
		return
	}

	// Check if user has access to this dataset
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

	// In a real implementation, you would read and parse the dataset file here
	// For this example, we'll use placeholder data
	schemaMap := map[string]interface{}{}
	if dataset.Schema != "" {
		json.Unmarshal([]byte(dataset.Schema), &schemaMap)
	}

	// Create sample data
	dataSample := make([]map[string]interface{}, 0, limit)
	for i := 0; i < limit && i < dataset.RowCount; i++ {
		row := map[string]interface{}{}
		for _, col := range dataset.Columns {
			row[col] = fmt.Sprintf("Sample data for %s row %d", col, i)
		}
		dataSample = append(dataSample, row)
	}

	response := models.DatasetResponse{
		DatasetMetadata: dataset.ToDatasetMetadata(),
		Schema:          schemaMap,
		DataSample:      dataSample,
	}

	c.JSON(http.StatusOK, response)
}

// DeleteDataset deletes a dataset
func (dc *DatasetController) DeleteDataset(c *gin.Context) {
	// Get dataset ID from URL
	datasetID := c.Param("dataset_id")

	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found in context"})
		return
	}
	user := userInterface.(models.User)

	// Get dataset from database
	var dataset models.Dataset
	if err := dc.DB.Where("id = ?", datasetID).First(&dataset).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dataset not found"})
		return
	}

	// Check if user has access to this dataset
	isAdmin := false
	for _, role := range user.Roles {
		if role == "admin" {
			isAdmin = true
			break
		}
	}

	if dataset.UserID != user.ID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this dataset"})
		return
	}

	// Delete the file
	filePath := filepath.Join(dc.Config.DatasetsDir, dataset.UserID, datasetID+filepath.Ext(dataset.Filename))
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete dataset file"})
		return
	}

	// Delete from database
	if err := dc.DB.Delete(&dataset).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete dataset metadata"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Dataset %s deleted successfully", datasetID)})
} 