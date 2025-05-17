package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"

	"go-deepsandbox/config"
	"go-deepsandbox/models"
)

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RateLimitMiddleware limits the number of requests per user/IP within a time window
func RateLimitMiddleware(redisClient *redis.Client, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user ID from token or use IP as identifier
		userID := "anonymous"
		authHeader := c.GetHeader("Authorization")
		
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims := jwt.MapClaims{}
			
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(cfg.SecretKey), nil
			})
			
			if err == nil && token.Valid {
				if sub, ok := claims["sub"].(string); ok {
					userID = sub
				}
			}
		}
		
		// If no valid token, use IP as identifier
		if userID == "anonymous" {
			userID = "ip:" + c.ClientIP()
		}
		
		// Rate limit key
		key := fmt.Sprintf("ratelimit:%s", userID)
		
		// Check current count
		ctx := context.Background()
		count, err := redisClient.Get(ctx, key).Int()
		
		// If key doesn't exist, create it
		if errors.Is(err, redis.Nil) {
			redisClient.Set(ctx, key, 1, time.Duration(cfg.RateLimitWindow)*time.Second)
		} else if err == nil {
			// Check if limit exceeded
			if count >= cfg.MaxRequestsPerWindow {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": "Too many requests. Please try again later.",
				})
				c.Abort()
				return
			}
			
			// Increment counter
			redisClient.Incr(ctx, key)
		}
		
		c.Next()
	}
}

// Auth is a helper struct for authentication middleware
type Auth struct {
	DB     *gorm.DB
	Config *config.Config
}

// NewAuth creates a new Auth instance
func NewAuth(db *gorm.DB, cfg *config.Config) *Auth {
	return &Auth{
		DB:     db,
		Config: cfg,
	}
}

// AuthMiddleware checks if the request has a valid JWT token
func (a *Auth) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}
		
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}
		
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := jwt.MapClaims{}
		
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(a.Config.SecretKey), nil
		})
		
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}
		
		// Extract username from token
		username, ok := claims["sub"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}
		
		// Find user in database
		var user models.User
		if err := a.DB.Where("username = ?", username).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}
		
		// Check if user is disabled
		if user.Disabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "User account is disabled"})
			c.Abort()
			return
		}
		
		// Set user in context
		c.Set("user", user)
		c.Next()
	}
}

// AdminMiddleware checks if the user has admin role
func (a *Auth) AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First apply the auth middleware
		a.AuthMiddleware()(c)
		
		// If aborted, return
		if c.IsAborted() {
			return
		}
		
		// Get user from context
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found in context"})
			c.Abort()
			return
		}
		
		// Check if user has admin role
		userObj := user.(models.User)
		isAdmin := false
		for _, role := range userObj.Roles {
			if role == "admin" {
				isAdmin = true
				break
			}
		}
		
		if !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// ExecutionQuotaMiddleware checks if the user has exceeded their execution quota
func (a *Auth) ExecutionQuotaMiddleware(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context (assuming AuthMiddleware has been applied)
		userInterface, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found in context"})
			c.Abort()
			return
		}
		
		user := userInterface.(models.User)
		
		// Check execution quota
		ctx := context.Background()
		key := fmt.Sprintf("execution_quota:%s:%s", user.ID, time.Now().Format("2006-01-02"))
		
		count, err := redisClient.Get(ctx, key).Int()
		if errors.Is(err, redis.Nil) {
			// First execution of the day
			redisClient.Set(ctx, key, 1, 24*time.Hour)
		} else if err == nil {
			// Get user's max executions per day from quota
			maxExecutions := a.Config.MaxExecutionsPerDay
			
			// Parse quota from JSON if it exists
			if len(user.Quota) > 0 {
				var quotaMap map[string]int
				if err := json.Unmarshal(user.Quota, &quotaMap); err == nil {
					if quota, ok := quotaMap["max_executions_per_day"]; ok && quota > 0 {
						maxExecutions = quota
					}
				}
			}
			
			if count >= maxExecutions {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "You have exceeded your daily execution quota"})
				c.Abort()
				return
			}
			
			// Increment counter
			redisClient.Incr(ctx, key)
		}
		
		c.Next()
	}
}

// TrackExecution increments the execution counter for a user
func TrackExecution(redisClient *redis.Client, userID string) error {
	ctx := context.Background()
	key := fmt.Sprintf("execution_quota:%s:%s", userID, time.Now().Format("2006-01-02"))
	
	_, err := redisClient.Incr(ctx, key).Result()
	if err != nil {
		return err
	}
	
	// Set expiration if it doesn't exist
	redisClient.Expire(ctx, key, 24*time.Hour)
	return nil
} 