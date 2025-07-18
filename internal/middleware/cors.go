package middleware

import (
	"fmt"
	
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// CORSConfig contains the configuration for CORS middleware
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns the default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Content-Length",
			"Accept",
			"Accept-Encoding",
			"Authorization",
			"X-CSRF-Token",
			"X-Request-ID",
			"X-Requested-With",
		},
		ExposeHeaders:    []string{"Content-Length", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}
}

// CORS returns a gin middleware for CORS support
func CORS(config CORSConfig, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Check if origin is allowed
		if isOriginAllowed(origin, config.AllowOrigins) {
			// Set CORS headers
			if origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
			} else if len(config.AllowOrigins) == 1 && config.AllowOrigins[0] == "*" {
				c.Header("Access-Control-Allow-Origin", "*")
			}
			
			// Set other CORS headers
			c.Header("Access-Control-Allow-Methods", joinStrings(config.AllowMethods))
			c.Header("Access-Control-Allow-Headers", joinStrings(config.AllowHeaders))
			c.Header("Access-Control-Expose-Headers", joinStrings(config.ExposeHeaders))
			
			if config.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
			
			// Set max age for preflight requests
			if c.Request.Method == "OPTIONS" {
				c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
			}
		}
		
		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			logger.WithFields(logrus.Fields{
				"origin": origin,
				"path":   c.Request.URL.Path,
			}).Debug("CORS preflight request")
			
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	}
}

// isOriginAllowed checks if the origin is allowed
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if len(allowedOrigins) == 0 {
		return false
	}
	
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if origin == allowed {
			return true
		}
	}
	
	return false
}

// joinStrings joins strings with comma
func joinStrings(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += ", " + strs[i]
	}
	
	return result
}