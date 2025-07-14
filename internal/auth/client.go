package auth

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// AuthClient handles communication with the auth service
type AuthClient struct {
	serviceID   string
	sharedKey   []byte
	authBaseURL string
	httpClient  *http.Client
	logger      *logrus.Logger
}

// UserInfo represents user information to be encrypted
type UserInfo struct {
	UserID         string   `json:"user_id"`
	SessionID      string   `json:"session_id"`
	OrganizationID string   `json:"organization_id"`
	TeamIDs        []string `json:"team_ids"`
	Timestamp      int64    `json:"timestamp"`
	Nonce          string   `json:"nonce"`
}

// PermissionCheckRequest represents a permission check request
type PermissionCheckRequest struct {
	ServiceID            string   `json:"service_id"`
	EncryptedData        string   `json:"encrypted_data"`
	ResourceID           string   `json:"resource_id"`
	RequestedPermissions []string `json:"requested_permissions"`
}

// PermissionCheckResponse represents the response from permission check
type PermissionCheckResponse struct {
	Success bool              `json:"success"`
	Data    *PermissionResult `json:"data,omitempty"`
	Error   string            `json:"error,omitempty"`
}

// PermissionResult contains the permission check results
type PermissionResult struct {
	UserID             string   `json:"user_id"`
	ResourceID         string   `json:"resource_id"`
	GrantedPermissions []string `json:"granted_permissions"`
	DeniedPermissions  []string `json:"denied_permissions"`
	ResourceExists     bool     `json:"resource_exists"`
	CacheTTL           int      `json:"cache_ttl"`
}

// PermissionGrantRequest represents a permission grant request
type PermissionGrantRequest struct {
	ServiceID     string   `json:"service_id"`
	EncryptedData string   `json:"encrypted_data"`
	TargetUserID  string   `json:"target_user_id"`
	ResourceID    string   `json:"resource_id"`
	Permissions   []string `json:"permissions"`
	ExpiresAt     *int64   `json:"expires_at,omitempty"`
}

// ResourceCreateRequest represents a resource creation request
type ResourceCreateRequest struct {
	ServiceID     string `json:"service_id"`
	EncryptedData string `json:"encrypted_data"`
	ResourceID    string `json:"resource_id"`
	ResourceType  string `json:"resource_type"`
	DisplayName   string `json:"display_name"`
	Description   string `json:"description"`
	Metadata      string `json:"metadata"`
}

// NewAuthClient creates a new auth service client
func NewAuthClient(serviceID, authBaseURL string, sharedKey []byte, logger *logrus.Logger) *AuthClient {
	return &AuthClient{
		serviceID:   serviceID,
		sharedKey:   sharedKey,
		authBaseURL: authBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// EncryptUserInfo encrypts user information for sending to auth service
func (ac *AuthClient) EncryptUserInfo(userInfo *UserInfo) (string, error) {
	// Set timestamp and nonce if not already set
	if userInfo.Timestamp == 0 {
		userInfo.Timestamp = time.Now().UnixMilli()
	}
	if userInfo.Nonce == "" {
		userInfo.Nonce = uuid.New().String()
	}

	// Serialize to JSON
	plaintext, err := json.Marshal(userInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal user info: %w", err)
	}

	// Create cipher
	block, err := aes.NewCipher(ac.sharedKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	// Combine nonce and ciphertext
	encrypted := append(nonce, ciphertext...)

	// Encode to base64
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// CheckPermission checks if a user has specific permissions on a resource
func (ac *AuthClient) CheckPermission(userInfo *UserInfo, resourceID string, permissions []string) (bool, error) {
	// Encrypt user info
	encryptedData, err := ac.EncryptUserInfo(userInfo)
	if err != nil {
		return false, fmt.Errorf("failed to encrypt user info: %w", err)
	}

	// Prepare request
	request := PermissionCheckRequest{
		ServiceID:            ac.serviceID,
		EncryptedData:        encryptedData,
		ResourceID:           resourceID,
		RequestedPermissions: permissions,
	}

	// Send request
	var response PermissionCheckResponse
	err = ac.sendRequest("POST", "/api/v1/permission/check", request, &response)
	if err != nil {
		return false, fmt.Errorf("permission check request failed: %w", err)
	}

	if !response.Success {
		ac.logger.WithFields(logrus.Fields{
			"user_id":     userInfo.UserID,
			"resource_id": resourceID,
			"permissions": permissions,
			"error":       response.Error,
		}).Debug("Permission check failed")
		return false, nil
	}

	// Check if all requested permissions are granted
	if response.Data == nil {
		return false, nil
	}

	grantedSet := make(map[string]bool)
	for _, perm := range response.Data.GrantedPermissions {
		grantedSet[perm] = true
	}

	for _, requested := range permissions {
		if !grantedSet[requested] {
			return false, nil
		}
	}

	return true, nil
}

// GrantPermission grants permissions to a user
func (ac *AuthClient) GrantPermission(adminUserInfo *UserInfo, targetUserID, resourceID string, permissions []string) error {
	// Encrypt admin user info
	encryptedData, err := ac.EncryptUserInfo(adminUserInfo)
	if err != nil {
		return fmt.Errorf("failed to encrypt admin user info: %w", err)
	}

	// Prepare request
	request := PermissionGrantRequest{
		ServiceID:     ac.serviceID,
		EncryptedData: encryptedData,
		TargetUserID:  targetUserID,
		ResourceID:    resourceID,
		Permissions:   permissions,
	}

	// Send request
	var response map[string]interface{}
	err = ac.sendRequest("POST", "/api/v1/permission/grant", request, &response)
	if err != nil {
		return fmt.Errorf("permission grant request failed: %w", err)
	}

	if success, ok := response["success"]; !ok || !success.(bool) {
		errorMsg := "unknown error"
		if msg, exists := response["error"]; exists {
			errorMsg = msg.(string)
		}
		return fmt.Errorf("permission grant failed: %s", errorMsg)
	}

	return nil
}

// CreateResource creates a new resource in the auth service
func (ac *AuthClient) CreateResource(userInfo *UserInfo, resourceID, resourceType, displayName, description string) error {
	// Encrypt user info
	encryptedData, err := ac.EncryptUserInfo(userInfo)
	if err != nil {
		return fmt.Errorf("failed to encrypt user info: %w", err)
	}

	// Prepare request
	request := ResourceCreateRequest{
		ServiceID:     ac.serviceID,
		EncryptedData: encryptedData,
		ResourceID:    resourceID,
		ResourceType:  resourceType,
		DisplayName:   displayName,
		Description:   description,
		Metadata:      "{}",
	}

	// Send request
	var response map[string]interface{}
	err = ac.sendRequest("POST", "/api/v1/resources/create", request, &response)
	if err != nil {
		return fmt.Errorf("resource creation request failed: %w", err)
	}

	if success, ok := response["success"]; !ok || !success.(bool) {
		errorMsg := "unknown error"
		if msg, exists := response["error"]; exists {
			errorMsg = msg.(string)
		}
		return fmt.Errorf("resource creation failed: %s", errorMsg)
	}

	return nil
}

// sendRequest sends an HTTP request to the auth service
func (ac *AuthClient) sendRequest(method, endpoint string, body interface{}, response interface{}) error {
	// Marshal request body
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	// Create request
	url := ac.authBaseURL + endpoint
	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "cagen-quota-service/1.0")

	// Send request
	resp, err := ac.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	if response != nil {
		err = json.Unmarshal(respBody, response)
		if err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// ConfigureServiceKey configures the service key with auth service (for initial setup)
func (ac *AuthClient) ConfigureServiceKey() error {
	if len(ac.sharedKey) == 0 {
		return fmt.Errorf("shared key not configured")
	}

	keyBase64 := base64.StdEncoding.EncodeToString(ac.sharedKey)
	request := map[string]string{
		"shared_key": keyBase64,
	}

	var response map[string]interface{}
	endpoint := fmt.Sprintf("/api/v1/services/%s/configure-key", ac.serviceID)
	err := ac.sendRequest("POST", endpoint, request, &response)
	if err != nil {
		return fmt.Errorf("service key configuration failed: %w", err)
	}

	if success, ok := response["success"]; !ok || !success.(bool) {
		errorMsg := "unknown error"
		if msg, exists := response["error"]; exists {
			errorMsg = msg.(string)
		}
		return fmt.Errorf("service key configuration failed: %s", errorMsg)
	}

	ac.logger.Info("Service key configured successfully")
	return nil
}

// ServiceID returns the service ID
func (ac *AuthClient) ServiceID() string {
	return ac.serviceID
}

// Quota permission constants
const (
	QuotaPermissionRead  = "read"
	QuotaPermissionAdmin = "admin"
	QuotaPermissionOwner = "owner"
)