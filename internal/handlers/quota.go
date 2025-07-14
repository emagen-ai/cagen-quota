package handlers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/emagen-ai/cagen-quota/internal/auth"
	"github.com/emagen-ai/cagen-quota/internal/models"
	"github.com/emagen-ai/cagen-quota/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// QuotaHandler handles quota-related HTTP requests
type QuotaHandler struct {
	quotaService *services.QuotaService
	authClient   *auth.AuthClient
	logger       *logrus.Logger
}

// NewQuotaHandler creates a new quota handler
func NewQuotaHandler(quotaService *services.QuotaService, authClient *auth.AuthClient, logger *logrus.Logger) *QuotaHandler {
	return &QuotaHandler{
		quotaService: quotaService,
		authClient:   authClient,
		logger:       logger,
	}
}

// CreateQuota handles quota creation requests
func (qh *QuotaHandler) CreateQuota(c *gin.Context) {
	var request models.QuotaCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		qh.respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Decrypt user info
	userInfo, err := qh.decryptUserInfo(request.ServiceID, request.EncryptedData)
	if err != nil {
		qh.respondError(c, http.StatusUnauthorized, "Failed to decrypt user credentials", err)
		return
	}

	// Create quota
	quota, err := qh.quotaService.CreateQuota(userInfo, &request)
	if err != nil {
		qh.logger.WithError(err).WithFields(logrus.Fields{
			"user_id": userInfo.UserID,
			"name":    request.Name,
			"type":    request.Type,
		}).Error("Failed to create quota")
		qh.respondError(c, http.StatusInternalServerError, "Failed to create quota", err)
		return
	}

	qh.respondSuccess(c, http.StatusCreated, "Quota created successfully", quota)
}

// AllocateQuota handles quota allocation requests
func (qh *QuotaHandler) AllocateQuota(c *gin.Context) {
	parentQuotaID := c.Param("id")
	if parentQuotaID == "" {
		qh.respondError(c, http.StatusBadRequest, "Parent quota ID is required", nil)
		return
	}

	var request models.QuotaAllocateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		qh.respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Decrypt user info
	userInfo, err := qh.decryptUserInfo(request.ServiceID, request.EncryptedData)
	if err != nil {
		qh.respondError(c, http.StatusUnauthorized, "Failed to decrypt user credentials", err)
		return
	}

	// Allocate quota
	childQuota, err := qh.quotaService.AllocateQuota(userInfo, parentQuotaID, &request)
	if err != nil {
		qh.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":          userInfo.UserID,
			"parent_quota_id":  parentQuotaID,
			"allocate_mb":      request.AllocateMB,
		}).Error("Failed to allocate quota")
		qh.respondError(c, http.StatusInternalServerError, "Failed to allocate quota", err)
		return
	}

	qh.respondSuccess(c, http.StatusCreated, "Quota allocated successfully", childQuota)
}

// ReleaseQuota handles quota release requests
func (qh *QuotaHandler) ReleaseQuota(c *gin.Context) {
	quotaID := c.Param("id")
	if quotaID == "" {
		qh.respondError(c, http.StatusBadRequest, "Quota ID is required", nil)
		return
	}

	var request struct {
		ServiceID     string `json:"service_id" binding:"required"`
		EncryptedData string `json:"encrypted_data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		qh.respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Decrypt user info
	userInfo, err := qh.decryptUserInfo(request.ServiceID, request.EncryptedData)
	if err != nil {
		qh.respondError(c, http.StatusUnauthorized, "Failed to decrypt user credentials", err)
		return
	}

	// Release quota
	err = qh.quotaService.ReleaseQuota(userInfo, quotaID)
	if err != nil {
		qh.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":  userInfo.UserID,
			"quota_id": quotaID,
		}).Error("Failed to release quota")
		qh.respondError(c, http.StatusInternalServerError, "Failed to release quota", err)
		return
	}

	qh.respondSuccess(c, http.StatusOK, "Quota released successfully", nil)
}

// GetQuota handles quota retrieval requests
func (qh *QuotaHandler) GetQuota(c *gin.Context) {
	quotaID := c.Param("id")
	if quotaID == "" {
		qh.respondError(c, http.StatusBadRequest, "Quota ID is required", nil)
		return
	}

	// Get encrypted data from query params or headers
	serviceID := c.Query("service_id")
	encryptedData := c.Query("encrypted_data")
	
	if serviceID == "" || encryptedData == "" {
		qh.respondError(c, http.StatusBadRequest, "service_id and encrypted_data are required", nil)
		return
	}

	// Decrypt user info
	userInfo, err := qh.decryptUserInfo(serviceID, encryptedData)
	if err != nil {
		qh.respondError(c, http.StatusUnauthorized, "Failed to decrypt user credentials", err)
		return
	}

	// Get quota
	quota, err := qh.quotaService.GetQuota(userInfo, quotaID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			qh.respondError(c, http.StatusNotFound, "Quota not found", err)
		} else if strings.Contains(err.Error(), "insufficient permissions") {
			qh.respondError(c, http.StatusForbidden, "Insufficient permissions", err)
		} else {
			qh.logger.WithError(err).WithFields(logrus.Fields{
				"user_id":  userInfo.UserID,
				"quota_id": quotaID,
			}).Error("Failed to get quota")
			qh.respondError(c, http.StatusInternalServerError, "Failed to get quota", err)
		}
		return
	}

	qh.respondSuccess(c, http.StatusOK, "Quota retrieved successfully", quota)
}

// GrantPermission handles quota permission grant requests
func (qh *QuotaHandler) GrantPermission(c *gin.Context) {
	quotaID := c.Param("id")
	if quotaID == "" {
		qh.respondError(c, http.StatusBadRequest, "Quota ID is required", nil)
		return
	}

	var request models.QuotaGrantPermissionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		qh.respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Decrypt user info
	userInfo, err := qh.decryptUserInfo(request.ServiceID, request.EncryptedData)
	if err != nil {
		qh.respondError(c, http.StatusUnauthorized, "Failed to decrypt user credentials", err)
		return
	}

	// Grant permission through auth service
	err = qh.authClient.GrantPermission(userInfo, request.TargetUserID, quotaID, request.Permissions)
	if err != nil {
		qh.logger.WithError(err).WithFields(logrus.Fields{
			"admin_user_id":  userInfo.UserID,
			"target_user_id": request.TargetUserID,
			"quota_id":       quotaID,
			"permissions":    request.Permissions,
		}).Error("Failed to grant quota permission")
		qh.respondError(c, http.StatusInternalServerError, "Failed to grant permission", err)
		return
	}

	qh.respondSuccess(c, http.StatusOK, "Permission granted successfully", nil)
}

// AllocateUsage handles usage allocation requests
func (qh *QuotaHandler) AllocateUsage(c *gin.Context) {
	quotaID := c.Param("id")
	if quotaID == "" {
		qh.respondError(c, http.StatusBadRequest, "Quota ID is required", nil)
		return
	}

	var request models.QuotaUsageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		qh.respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Decrypt user info
	userInfo, err := qh.decryptUserInfo(request.ServiceID, request.EncryptedData)
	if err != nil {
		qh.respondError(c, http.StatusUnauthorized, "Failed to decrypt user credentials", err)
		return
	}

	// Allocate usage
	err = qh.quotaService.AllocateUsage(userInfo, quotaID, &request)
	if err != nil {
		qh.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":     userInfo.UserID,
			"quota_id":    quotaID,
			"usage_mb":    request.UsageMB,
			"resource_id": request.ResourceID,
		}).Error("Failed to allocate usage")
		qh.respondError(c, http.StatusInternalServerError, "Failed to allocate usage", err)
		return
	}

	qh.respondSuccess(c, http.StatusOK, "Usage allocated successfully", nil)
}

// DeallocateUsage handles usage deallocation requests
func (qh *QuotaHandler) DeallocateUsage(c *gin.Context) {
	quotaID := c.Param("id")
	if quotaID == "" {
		qh.respondError(c, http.StatusBadRequest, "Quota ID is required", nil)
		return
	}

	var request models.QuotaUsageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		qh.respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Decrypt user info
	userInfo, err := qh.decryptUserInfo(request.ServiceID, request.EncryptedData)
	if err != nil {
		qh.respondError(c, http.StatusUnauthorized, "Failed to decrypt user credentials", err)
		return
	}

	// Deallocate usage
	err = qh.quotaService.DeallocateUsage(userInfo, quotaID, &request)
	if err != nil {
		qh.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":     userInfo.UserID,
			"quota_id":    quotaID,
			"usage_mb":    request.UsageMB,
			"resource_id": request.ResourceID,
		}).Error("Failed to deallocate usage")
		qh.respondError(c, http.StatusInternalServerError, "Failed to deallocate usage", err)
		return
	}

	qh.respondSuccess(c, http.StatusOK, "Usage deallocated successfully", nil)
}

// ListQuotas handles quota listing requests (placeholder for future implementation)
func (qh *QuotaHandler) ListQuotas(c *gin.Context) {
	// Get pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	quotaType := c.Query("type")

	// Get encrypted data from query params
	serviceID := c.Query("service_id")
	encryptedData := c.Query("encrypted_data")
	
	if serviceID == "" || encryptedData == "" {
		qh.respondError(c, http.StatusBadRequest, "service_id and encrypted_data are required", nil)
		return
	}

	// Decrypt user info
	userInfo, err := qh.decryptUserInfo(serviceID, encryptedData)
	if err != nil {
		qh.respondError(c, http.StatusUnauthorized, "Failed to decrypt user credentials", err)
		return
	}

	// For now, return empty list - this can be implemented later
	response := &models.QuotaListResponse{
		Quotas:     []models.Quota{},
		TotalCount: 0,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: 0,
	}

	qh.logger.WithFields(logrus.Fields{
		"user_id":    userInfo.UserID,
		"page":       page,
		"page_size":  pageSize,
		"quota_type": quotaType,
	}).Info("List quotas requested (not implemented)")

	qh.respondSuccess(c, http.StatusOK, "Quotas listed successfully", response)
}

// HealthCheck handles health check requests
func (qh *QuotaHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "cagen-quota",
		"version":   "1.0.0",
		"timestamp": "2025-01-12T00:00:00Z",
	})
}

// Helper methods

func (qh *QuotaHandler) decryptUserInfo(serviceID, encryptedData string) (*auth.UserInfo, error) {
	if serviceID != qh.authClient.ServiceID() {
		return nil, fmt.Errorf("invalid service ID")
	}

	// For the quota service, we'll use a simple approach
	// In a real implementation, you would decrypt the data using the same method as auth service
	// For now, we'll create a mock user info based on the encrypted data
	
	// Decode base64 to simulate decryption
	decoded, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("invalid encrypted data format")
	}

	// This is a simplified mock - in reality, you'd implement proper AES-GCM decryption
	userInfo := &auth.UserInfo{
		UserID:         "user_mock", // This should come from actual decryption
		SessionID:      "session_mock",
		OrganizationID: "org_default",
		TeamIDs:        []string{"team_default"},
		Timestamp:      1641945600000, // Mock timestamp
		Nonce:          "mock-nonce",
	}

	// Log that we're using mock data
	qh.logger.Debug("Using mock user info for development", len(decoded))

	return userInfo, nil
}

func (qh *QuotaHandler) respondSuccess(c *gin.Context, status int, message string, data interface{}) {
	response := gin.H{
		"success": true,
		"message": message,
	}
	if data != nil {
		response["data"] = data
	}
	c.JSON(status, response)
}

func (qh *QuotaHandler) respondError(c *gin.Context, status int, message string, err error) {
	response := gin.H{
		"success": false,
		"error":   message,
	}

	// Log the error
	logFields := logrus.Fields{
		"status":  status,
		"message": message,
		"path":    c.Request.URL.Path,
		"method":  c.Request.Method,
	}

	if err != nil {
		logFields["error_detail"] = err.Error()
	}

	if status >= 500 {
		qh.logger.WithFields(logFields).Error("Internal server error")
	} else {
		qh.logger.WithFields(logFields).Warn("Request error")
	}

	c.JSON(status, response)
}

// ServiceID returns the service ID (helper for auth client)
func (qh *QuotaHandler) ServiceID() string {
	return "svc_cagen_quota" // This should match the configured service ID
}