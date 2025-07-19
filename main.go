package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/emagen-ai/cagen-quota/internal/auth"
	"github.com/emagen-ai/cagen-quota/internal/config"
	"github.com/emagen-ai/cagen-quota/internal/database"
	"github.com/emagen-ai/cagen-quota/internal/handlers"
	"github.com/emagen-ai/cagen-quota/internal/middleware"
	"github.com/emagen-ai/cagen-quota/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger := setupLogger(cfg.LogLevel, cfg.LogFormat)
	logger.Info("Starting Cagen Quota Service v1.0")

	// Initialize database
	db, err := database.NewConnection(cfg.DatabaseURL, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	// Initialize database schema
	if err := db.InitSchema(); err != nil {
		logger.WithError(err).Fatal("Failed to initialize database schema")
	}
	logger.Info("Database schema initialized successfully")

	// Initialize auth client
	authClient, err := setupAuthClient(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to setup auth client")
	}

	// Initialize services
	quotaService := services.NewQuotaService(db, authClient, logger)

	// Initialize handlers
	quotaHandler := handlers.NewQuotaHandler(quotaService, authClient, logger)

	// Set gin mode
	if cfg.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router
	router := setupRouter(quotaHandler, logger, cfg)

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Infof("Quota Service starting on port %s", cfg.Port)
		logger.Infof("Environment: %s", cfg.Environment)
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down quota service...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	}

	logger.Info("Quota service stopped")
}

func setupLogger(logLevel, logFormat string) *logrus.Logger {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set log format
	if logFormat == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	return logger
}

func setupAuthClient(cfg *config.Config, logger *logrus.Logger) (*auth.AuthClient, error) {
	// Get or generate shared key
	var sharedKey []byte
	var err error

	if cfg.QuotaServiceSecretKey != "" {
		// Decode existing key
		sharedKey, err = base64.StdEncoding.DecodeString(cfg.QuotaServiceSecretKey)
		if err != nil {
			return nil, fmt.Errorf("invalid service secret key format: %w", err)
		}
		if len(sharedKey) != 32 {
			return nil, fmt.Errorf("service secret key must be 32 bytes, got %d", len(sharedKey))
		}
		logger.Info("Using configured service secret key")
	} else if cfg.Environment == "development" {
		// Generate a temporary key for development
		sharedKey = make([]byte, 32)
		copy(sharedKey, []byte("dev-key-for-testing-only-32bytes"))
		logger.Warn("Using development key - not suitable for production")
	} else {
		return nil, fmt.Errorf("CAGEN_QUOTA_SERVICE_SECRET_KEY is required")
	}

	// Create auth client
	authClient := auth.NewAuthClient(cfg.QuotaServiceID, cfg.AuthServiceURL, sharedKey, logger)

	// Configure service key if needed (development mode)
	if cfg.Environment == "development" {
		logger.Info("Configuring service key with auth service...")
		if err := authClient.ConfigureServiceKey(); err != nil {
			logger.WithError(err).Warn("Failed to configure service key with auth service")
		}
	}

	return authClient, nil
}

func setupRouter(quotaHandler *handlers.QuotaHandler, logger *logrus.Logger, cfg *config.Config) *gin.Engine {
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(ginLogger(logger))

	// Add request ID middleware
	router.Use(func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	})

	// CORS middleware with proper configuration
	corsConfig := middleware.DefaultCORSConfig()
	
	// Parse allowed origins from config
	if cfg.AllowedOrigins != "" {
		origins := strings.Split(cfg.AllowedOrigins, ",")
		corsConfig.AllowOrigins = make([]string, 0, len(origins))
		for _, origin := range origins {
			trimmed := strings.TrimSpace(origin)
			if trimmed != "" {
				corsConfig.AllowOrigins = append(corsConfig.AllowOrigins, trimmed)
			}
		}
		logger.Infof("CORS allowed origins: %v", corsConfig.AllowOrigins)
	}
	
	router.Use(middleware.CORS(corsConfig, logger))

	// Public routes
	router.GET("/health", quotaHandler.HealthCheck)
	
	// Quota API (v1)
	v1 := router.Group("/api/v1")
	{
		// Core quota operations
		v1.POST("/quotas/create", quotaHandler.CreateQuota)
		v1.POST("/quotas/:id/allocate", quotaHandler.AllocateQuota)
		v1.POST("/quotas/:id/release", quotaHandler.ReleaseQuota)
		v1.GET("/quotas/:id", quotaHandler.GetQuota)
		v1.GET("/quotas", quotaHandler.ListQuotas)
		
		// Permission management
		v1.POST("/quotas/:id/permissions/grant", quotaHandler.GrantPermission)
		
		// Usage management
		v1.POST("/quotas/:id/usage/allocate", quotaHandler.AllocateUsage)
		v1.POST("/quotas/:id/usage/deallocate", quotaHandler.DeallocateUsage)
		v1.GET("/runtime-usage", quotaHandler.ListRuntimeUsage)
	}

	// Development endpoints (only in development mode)
	if cfg.Environment == "development" {
		dev := router.Group("/dev")
		{
			dev.GET("/info", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"service":     "cagen-quota",
					"version":     "1.0.0",
					"environment": cfg.Environment,
					"database":    "connected",
					"auth_url":    cfg.AuthServiceURL,
				})
			})

			dev.POST("/test-auth", func(c *gin.Context) {
				var request struct {
					ServiceID     string `json:"service_id"`
					EncryptedData string `json:"encrypted_data"`
				}
				
				if err := c.ShouldBindJSON(&request); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"message": "Auth test endpoint - encrypted data received",
					"data": gin.H{
						"service_id":      request.ServiceID,
						"encrypted_length": len(request.EncryptedData),
					},
				})
			})
		}
	}

	return router
}

func ginLogger(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log only if not a health check
		if path != "/health" {
			// Fill the params
			param := gin.LogFormatterParams{
				Request:    c.Request,
				TimeStamp:  time.Now(),
				Latency:    time.Since(start),
				ClientIP:   c.ClientIP(),
				Method:     c.Request.Method,
				StatusCode: c.Writer.Status(),
				ErrorMessage: c.Errors.ByType(gin.ErrorTypePrivate).String(),
				BodySize:   c.Writer.Size(),
				Keys:       c.Keys,
			}

			if raw != "" {
				param.Path = path + "?" + raw
			} else {
				param.Path = path
			}

			logger.WithFields(logrus.Fields{
				"method":      param.Method,
				"path":        param.Path,
				"status":      param.StatusCode,
				"latency":     param.Latency,
				"client_ip":   param.ClientIP,
				"body_size":   param.BodySize,
				"request_id":  c.GetString("request_id"),
			}).Info("HTTP Request")
		}
	}
}