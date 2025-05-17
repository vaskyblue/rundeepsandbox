package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config represents the application configuration
type Config struct {
	// API Settings
	APITitle       string
	APIDescription string
	APIVersion     string
	ServerPort     int

	// Security Settings
	SecretKey               string
	JWTAlgorithm            string
	AccessTokenExpireMinutes int

	// Rate Limiting
	RateLimitWindow       int // seconds
	MaxRequestsPerWindow  int // per user
	MaxExecutionsPerDay   int // code executions per day

	// Redis Configuration
	RedisHost string
	RedisPort int
	RedisPassword string

	// Celery/Task Configuration
	CeleryBrokerURL    string
	CeleryResultBackend string

	// Container Settings
	ContainerImage       string
	ContainerMemoryLimit string
	ContainerCPULimit    string
	ContainerNetwork     string
	ContainerTimeout     int
	ExecutionPoolSize    int

	// Data Paths
	DatasetsDir string

	// PostgreSQL Database
	PostgresHost     string
	PostgresPort     int
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	DatabaseURL      string
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsInt gets an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// NewConfig creates a new configuration with values from environment variables
func NewConfig() *Config {
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnvAsInt("REDIS_PORT", 6379)
	
	postgresHost := getEnv("POSTGRES_HOST", "localhost")
	postgresPort := getEnvAsInt("POSTGRES_PORT", 5432)
	postgresUser := getEnv("POSTGRES_USER", "deepsandbox")
	postgresPassword := getEnv("POSTGRES_PASSWORD", "deepsandbox")
	postgresDB := getEnv("POSTGRES_DB", "deepsandbox")
	
	databaseURL := getEnv("DATABASE_URL", 
		fmt.Sprintf("postgresql://%s:%s@%s:%d/%s", 
			postgresUser, 
			postgresPassword, 
			postgresHost, 
			postgresPort, 
			postgresDB,
		),
	)
	
	celeryBrokerURL := getEnv("CELERY_BROKER_URL", fmt.Sprintf("redis://%s:%d/1", redisHost, redisPort))
	celeryResultBackend := getEnv("CELERY_RESULT_BACKEND", fmt.Sprintf("redis://%s:%d/2", redisHost, redisPort))
	
	return &Config{
		// API Settings
		APITitle:       getEnv("API_TITLE", "DeepSandbox API"),
		APIDescription: getEnv("API_DESCRIPTION", "Secure and scalable API for executing Python data analysis on client datasets"),
		APIVersion:     getEnv("API_VERSION", "2.0.0"),
		ServerPort:     getEnvAsInt("SERVER_PORT", 8000),
		
		// Security Settings
		SecretKey:               getEnv("SECRET_KEY", "09d25e094faa6ca2556c818166b7a9563b93f7099f6f0f4caa6cf63b88e8d3e7"),
		JWTAlgorithm:            "HS256",
		AccessTokenExpireMinutes: getEnvAsInt("ACCESS_TOKEN_EXPIRE_MINUTES", 30),
		
		// Rate Limiting
		RateLimitWindow:      getEnvAsInt("RATE_LIMIT_WINDOW", 60),
		MaxRequestsPerWindow: getEnvAsInt("MAX_REQUESTS_PER_WINDOW", 100),
		MaxExecutionsPerDay:  getEnvAsInt("MAX_EXECUTIONS_PER_DAY", 1000),
		
		// Redis Configuration
		RedisHost:     redisHost,
		RedisPort:     redisPort,
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		
		// Celery/Task Configuration
		CeleryBrokerURL:    celeryBrokerURL,
		CeleryResultBackend: celeryResultBackend,
		
		// Container Settings
		ContainerImage:       getEnv("CONTAINER_IMAGE", "python:3.10-slim"),
		ContainerMemoryLimit: getEnv("CONTAINER_MEMORY_LIMIT", "2g"),
		ContainerCPULimit:    getEnv("CONTAINER_CPU_LIMIT", "1"),
		ContainerNetwork:     getEnv("CONTAINER_NETWORK", "none"),
		ContainerTimeout:     getEnvAsInt("CONTAINER_TIMEOUT", 300),
		ExecutionPoolSize:    getEnvAsInt("EXECUTION_POOL_SIZE", 10),
		
		// Data Paths
		DatasetsDir: getEnv("DATASETS_DIR", "datasets"),
		
		// PostgreSQL Database
		PostgresHost:     postgresHost,
		PostgresPort:     postgresPort,
		PostgresUser:     postgresUser,
		PostgresPassword: postgresPassword,
		PostgresDB:       postgresDB,
		DatabaseURL:      databaseURL,
	}
}

// JWTExpiration returns the JWT token expiration duration
func (c *Config) JWTExpiration() time.Duration {
	return time.Duration(c.AccessTokenExpireMinutes) * time.Minute
} 