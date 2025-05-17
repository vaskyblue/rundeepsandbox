package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"go-deepsandbox/config"
	"go-deepsandbox/models"
)

// AuthController handles authentication related endpoints
type AuthController struct {
	DB     *gorm.DB
	Config *config.Config
}

// NewAuthController creates a new auth controller
func NewAuthController(db *gorm.DB, cfg *config.Config) *AuthController {
	return &AuthController{
		DB:     db,
		Config: cfg,
	}
}

// Login handles user login
func (ac *AuthController) Login(c *gin.Context) {
	var loginRequest models.LoginRequest
	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user in database
	var user models.User
	if err := ac.DB.Where("username = ?", loginRequest.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// Check password
	if err := user.CheckPassword(loginRequest.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// Check if user is disabled
	if user.Disabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "User account is disabled"})
		return
	}

	// Generate token
	token, expiresIn, err := ac.generateToken(user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, models.TokenResponse{
		AccessToken: token,
		TokenType:   "bearer",
		ExpiresIn:   expiresIn,
	})
}

// Register handles user registration
func (ac *AuthController) Register(c *gin.Context) {
	var userCreate models.UserCreate
	if err := c.ShouldBindJSON(&userCreate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if username already exists
	var existingUser models.User
	if err := ac.DB.Where("username = ?", userCreate.Username).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already registered"})
		return
	}

	// Check if email already exists
	if userCreate.Email != "" {
		if err := ac.DB.Where("email = ?", userCreate.Email).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email already registered"})
			return
		}
	}

	// Default quota
	quotaMap := map[string]int{
		"max_datasets":           10,
		"max_dataset_size_mb":    2000,
		"max_executions_per_day": 1000,
		"max_execution_time":     300,
	}
	
	quotaJSON, err := json.Marshal(quotaMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create quota"})
		return
	}

	// Create new user
	user := models.User{
		ID:       uuid.New().String(),
		Username: userCreate.Username,
		Email:    userCreate.Email,
		FullName: userCreate.FullName,
		Disabled: false,
		Roles:    pq.StringArray{"user"},
		Quota:    quotaJSON,
	}

	// Set password
	if err := user.SetPassword(userCreate.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Save user to database
	if err := ac.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, user.ToUserResponse())
}

// GetCurrentUser returns information about the current user
func (ac *AuthController) GetCurrentUser(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found in context"})
		return
	}

	user := userInterface.(models.User)
	c.JSON(http.StatusOK, user.ToUserResponse())
}

// UpdateUser updates user information
func (ac *AuthController) UpdateUser(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found in context"})
		return
	}
	currentUser := userInterface.(models.User)

	// Parse update data
	var userUpdate models.UserUpdate
	if err := c.ShouldBindJSON(&userUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user from database
	var user models.User
	if err := ac.DB.Where("id = ?", currentUser.ID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update user fields
	if userUpdate.Email != "" {
		// Check if email already exists
		var existingUser models.User
		if err := ac.DB.Where("email = ? AND id != ?", userUpdate.Email, user.ID).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email already registered"})
			return
		}
		user.Email = userUpdate.Email
	}

	if userUpdate.FullName != "" {
		user.FullName = userUpdate.FullName
	}

	if userUpdate.Password != "" {
		if err := user.SetPassword(userUpdate.Password); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
	}

	// Only admin can change these fields
	isAdmin := false
	for _, role := range currentUser.Roles {
		if role == "admin" {
			isAdmin = true
			break
		}
	}

	if isAdmin {
		if userUpdate.Disabled != nil {
			user.Disabled = *userUpdate.Disabled
		}

		if userUpdate.Roles != nil {
			user.Roles = userUpdate.Roles
		}

		if len(userUpdate.Quota) > 0 {
			user.Quota = userUpdate.Quota
		}
	}

	// Save changes
	if err := ac.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, user.ToUserResponse())
}

// ListUsers lists all users (admin only)
func (ac *AuthController) ListUsers(c *gin.Context) {
	var users []models.User
	
	// Get query parameters
	skip := c.DefaultQuery("skip", "0")
	limit := c.DefaultQuery("limit", "100")

	// Convert string parameters to integers
	skipInt, err := strconv.Atoi(skip)
	if err != nil {
		skipInt = 0
	}

	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		limitInt = 100
	}

	// Query users
	if err := ac.DB.Offset(skipInt).Limit(limitInt).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	// Convert to response format
	response := make([]models.UserResponse, len(users))
	for i, user := range users {
		response[i] = user.ToUserResponse()
	}

	c.JSON(http.StatusOK, response)
}

// generateToken generates a new JWT token for a user
func (ac *AuthController) generateToken(username string) (string, int, error) {
	expirationTime := time.Now().Add(ac.Config.JWTExpiration())
	expiresIn := int(ac.Config.JWTExpiration().Seconds())

	claims := jwt.MapClaims{
		"sub": username,
		"exp": expirationTime.Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(ac.Config.SecretKey))
	if err != nil {
		return "", 0, err
	}

	return tokenString, expiresIn, nil
} 