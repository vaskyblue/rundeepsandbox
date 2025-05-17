package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"github.com/lib/pq"
)

// User represents a user in the system
type User struct {
	ID             string          `json:"id" gorm:"primaryKey"`
	Username       string          `json:"username" gorm:"uniqueIndex"`
	Email          string          `json:"email" gorm:"uniqueIndex"`
	FullName       string          `json:"full_name"`
	HashedPassword string          `json:"-" gorm:"column:hashed_password"`
	Disabled       bool            `json:"disabled" gorm:"default:false"`
	Roles          pq.StringArray  `json:"roles" gorm:"type:text[]"`
	Quota          json.RawMessage `json:"quota" gorm:"type:jsonb"`
	CreatedAt      time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}

// Dataset represents a dataset uploaded by a user
type Dataset struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	UserID      string    `json:"user_id" gorm:"index"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	SizeMB      float64   `json:"size_mb"`
	RowCount    int       `json:"row_count"`
	Columns     []string  `json:"columns" gorm:"type:text[]"`
	Schema      string    `json:"schema" gorm:"type:jsonb"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// CodeExecution represents a code execution request
type CodeExecution struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	UserID    string    `json:"user_id" gorm:"index"`
	DatasetID string    `json:"dataset_id" gorm:"index"`
	Code      string    `json:"code" gorm:"type:text"`
	Status    string    `json:"status" gorm:"index"`
	Results   string    `json:"results" gorm:"type:jsonb"`
	StartTime float64   `json:"start_time"`
	EndTime   float64   `json:"end_time"`
	Error     string    `json:"error" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// BeforeCreate will generate a UUID for entities before creation
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return
}

// BeforeCreate will generate a UUID for datasets before creation
func (d *Dataset) BeforeCreate(tx *gorm.DB) (err error) {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return
}

// SetPassword sets the hashed password field from a plain-text password
func (u *User) SetPassword(password string) error {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.HashedPassword = string(hashedBytes)
	return nil
}

// CheckPassword checks if the provided password matches the hashed password
func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.HashedPassword), []byte(password))
}

// DTO models for API requests and responses

// UserCreate is the DTO for creating a new user
type UserCreate struct {
	Username  string `json:"username" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	FullName  string `json:"full_name"`
	Password  string `json:"password" binding:"required,min=8"`
}

// UserUpdate is the DTO for updating user information
type UserUpdate struct {
	Email     string         `json:"email" binding:"omitempty,email"`
	FullName  string         `json:"full_name"`
	Password  string         `json:"password" binding:"omitempty,min=8"`
	Disabled  *bool          `json:"disabled,omitempty"`
	Roles     pq.StringArray `json:"roles,omitempty"`
	Quota     json.RawMessage `json:"quota,omitempty"`
}

// UserResponse is the DTO for returning user information
type UserResponse struct {
	ID        string          `json:"id"`
	Username  string          `json:"username"`
	Email     string          `json:"email"`
	FullName  string          `json:"full_name"`
	Disabled  bool            `json:"disabled"`
	Roles     pq.StringArray  `json:"roles"`
	Quota     map[string]int  `json:"quota"`
	CreatedAt time.Time       `json:"created_at"`
}

// ToUserResponse converts a User model to a UserResponse DTO
func (u *User) ToUserResponse() UserResponse {
	var quotaMap map[string]int
	if len(u.Quota) > 0 {
		if err := json.Unmarshal(u.Quota, &quotaMap); err != nil {
			quotaMap = make(map[string]int)
		}
	} else {
		quotaMap = make(map[string]int)
	}

	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		FullName:  u.FullName,
		Disabled:  u.Disabled,
		Roles:     u.Roles,
		Quota:     quotaMap,
		CreatedAt: u.CreatedAt,
	}
}

// LoginRequest is the DTO for user login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// TokenResponse is the DTO for authentication token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// DatasetMetadata is the DTO for dataset metadata
type DatasetMetadata struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	SizeMB      float64   `json:"size_mb"`
	RowCount    int       `json:"row_count"`
	Columns     []string  `json:"columns"`
	CreatedAt   time.Time `json:"created_at"`
}

// ToDatasetMetadata converts a Dataset model to a DatasetMetadata DTO
func (d *Dataset) ToDatasetMetadata() DatasetMetadata {
	return DatasetMetadata{
		ID:          d.ID,
		Filename:    d.Filename,
		ContentType: d.ContentType,
		Size:        d.Size,
		SizeMB:      d.SizeMB,
		RowCount:    d.RowCount,
		Columns:     d.Columns,
		CreatedAt:   d.CreatedAt,
	}
}

// DatasetResponse is the DTO for dataset response with data preview
type DatasetResponse struct {
	DatasetMetadata
	Schema     map[string]interface{}   `json:"schema"`
	DataSample []map[string]interface{} `json:"data_sample"`
}

// CodeExecutionRequest is the DTO for code execution requests
type CodeExecutionRequest struct {
	DatasetID string `json:"dataset_id" binding:"required"`
	Code      string `json:"code" binding:"required"`
	Timeout   *int   `json:"timeout,omitempty"`
}

// TaskStatus is the DTO for task status information
type TaskStatus struct {
	TaskID    string                 `json:"task_id"`
	Status    string                 `json:"status"`
	Progress  float64                `json:"progress"`
	StartTime float64                `json:"start_time,omitempty"`
	EndTime   float64                `json:"end_time,omitempty"`
	Results   map[string]interface{} `json:"results,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// ToTaskStatus converts a CodeExecution model to a TaskStatus DTO
func (c *CodeExecution) ToTaskStatus() TaskStatus {
	var results map[string]interface{}
	if c.Results != "" {
		// In a real implementation, you'd parse JSON from c.Results
		results = map[string]interface{}{}
	}

	return TaskStatus{
		TaskID:    c.ID,
		Status:    c.Status,
		Progress:  100.0,
		StartTime: c.StartTime,
		EndTime:   c.EndTime,
		Results:   results,
		Error:     c.Error,
	}
}