package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Config struct {
	// Database
	DatabaseURL string

	// Server
	Port        string
	GinMode     string
	Environment string

	// Auth Service Integration
	AuthServiceURL         string
	QuotaServiceSecretKey  string
	QuotaServiceID         string

	// Logging
	LogLevel  string
	LogFormat string

	// Railway
	RailwayProjectID     string
	RailwayEnvironmentID string
	RailwayServiceID     string
}

func Load() *Config {
	// Load .env file if it exists (for local development)
	if err := godotenv.Load(); err != nil {
		logrus.Debug("No .env file found, using environment variables")
	}

	config := &Config{
		DatabaseURL:            getEnv("DATABASE_URL", "postgresql://localhost:5432/cagen_quota?sslmode=disable"),
		Port:                   getEnv("PORT", "8080"),
		GinMode:                getEnv("GIN_MODE", "debug"),
		Environment:            getEnv("ENVIRONMENT", "development"),
		AuthServiceURL:         getEnv("AUTH_SERVICE_URL", "https://cagen-auth-service-production.up.railway.app"),
		QuotaServiceSecretKey:  getEnv("CAGEN_QUOTA_SERVICE_SECRET_KEY", ""),
		QuotaServiceID:         getEnv("QUOTA_SERVICE_ID", "svc_cagen_quota"),
		LogLevel:               getEnv("LOG_LEVEL", "info"),
		LogFormat:              getEnv("LOG_FORMAT", "text"),
		RailwayProjectID:       getEnv("RAILWAY_PROJECT_ID", ""),
		RailwayEnvironmentID:   getEnv("RAILWAY_ENVIRONMENT_ID", ""),
		RailwayServiceID:       getEnv("RAILWAY_SERVICE_ID", ""),
	}

	// Validate required configs
	if config.QuotaServiceSecretKey == "" && config.Environment == "production" {
		logrus.Fatal("CAGEN_QUOTA_SERVICE_SECRET_KEY is required in production")
	}

	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}