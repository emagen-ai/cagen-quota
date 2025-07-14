package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// Quota represents a quota entity
type Quota struct {
	ID             string     `json:"id" db:"id"`
	Name           string     `json:"name" db:"name"`
	Description    string     `json:"description" db:"description"`
	Type           string     `json:"type" db:"type"` // organization | team
	
	// Capacity in MB
	TotalMB       int64 `json:"total_mb" db:"total_mb"`
	UsedMB        int64 `json:"used_mb" db:"used_mb"`
	AllocatedMB   int64 `json:"allocated_mb" db:"allocated_mb"`
	AvailableMB   int64 `json:"available_mb" db:"available_mb"` // computed: total - used - allocated
	
	// Hierarchy
	ParentQuotaID *string `json:"parent_quota_id" db:"parent_quota_id"`
	Level         int     `json:"level" db:"level"`
	Path          string  `json:"path" db:"path"`
	
	// Ownership
	OwnerID        string  `json:"owner_id" db:"owner_id"`
	OrganizationID string  `json:"organization_id" db:"organization_id"`
	TeamID         *string `json:"team_id" db:"team_id"`
	
	// Status
	Status    string     `json:"status" db:"status"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" db:"deleted_at"`
}

// QuotaUsage represents quota usage records
type QuotaUsage struct {
	ID         string    `json:"id" db:"id"`
	QuotaID    string    `json:"quota_id" db:"quota_id"`
	UserID     string    `json:"user_id" db:"user_id"`
	ResourceID string    `json:"resource_id" db:"resource_id"`
	UsageMB    int64     `json:"usage_mb" db:"usage_mb"`
	Operation  string    `json:"operation" db:"operation"` // allocate | deallocate
	Reason     string    `json:"reason" db:"reason"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// QuotaAuditLog represents audit logs for quota operations
type QuotaAuditLog struct {
	ID           string    `json:"id" db:"id"`
	QuotaID      string    `json:"quota_id" db:"quota_id"`
	ActionType   string    `json:"action_type" db:"action_type"`
	ActorUserID  string    `json:"actor_user_id" db:"actor_user_id"`
	TargetUserID *string   `json:"target_user_id" db:"target_user_id"`
	Details      JSONMap   `json:"details" db:"details"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// JSONMap is a custom type for handling JSONB in PostgreSQL
type JSONMap map[string]interface{}

// Value implements the driver.Valuer interface for JSONMap
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for JSONMap
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, j)
	case string:
		return json.Unmarshal([]byte(v), j)
	default:
		return errors.New("cannot scan non-string/[]byte value into JSONMap")
	}
}

// Quota constants
const (
	QuotaTypeOrganization = "organization"
	QuotaTypeTeam         = "team"
	
	QuotaStatusActive    = "active"
	QuotaStatusSuspended = "suspended"
	QuotaStatusDeleted   = "deleted"
	
	OperationAllocate   = "allocate"
	OperationDeallocate = "deallocate"
)

// QuotaCreateRequest represents a request to create a quota
type QuotaCreateRequest struct {
	ServiceID       string   `json:"service_id" binding:"required"`
	EncryptedData   string   `json:"encrypted_data" binding:"required"`
	Name            string   `json:"name" binding:"required"`
	Description     string   `json:"description"`
	Type            string   `json:"type" binding:"required"`
	TotalMB         int64    `json:"total_mb" binding:"required,min=1"`
	OrganizationID  string   `json:"organization_id,omitempty"`
	TeamID          *string  `json:"team_id,omitempty"`
}

// QuotaAllocateRequest represents a request to allocate a sub-quota
type QuotaAllocateRequest struct {
	ServiceID     string   `json:"service_id" binding:"required"`
	EncryptedData string   `json:"encrypted_data" binding:"required"`
	Name          string   `json:"name" binding:"required"`
	Description   string   `json:"description"`
	AllocateMB    int64    `json:"allocate_mb" binding:"required,min=1"`
	Type          string   `json:"type" binding:"required"`
	TargetID      string   `json:"target_id"`          // organization_id or team_id
	AdminUserIDs  []string `json:"admin_user_ids"`     // users to grant admin permission
}

// QuotaGrantPermissionRequest represents a request to grant quota permissions
type QuotaGrantPermissionRequest struct {
	ServiceID     string   `json:"service_id" binding:"required"`
	EncryptedData string   `json:"encrypted_data" binding:"required"`
	TargetUserID  string   `json:"target_user_id" binding:"required"`
	Permissions   []string `json:"permissions" binding:"required"`
}

// QuotaUsageRequest represents a request to allocate/deallocate usage
type QuotaUsageRequest struct {
	ServiceID     string `json:"service_id" binding:"required"`
	EncryptedData string `json:"encrypted_data" binding:"required"`
	ResourceID    string `json:"resource_id" binding:"required"`
	UsageMB       int64  `json:"usage_mb" binding:"required,min=1"`
	Reason        string `json:"reason"`
}

// QuotaListResponse represents a paginated list of quotas
type QuotaListResponse struct {
	Quotas     []Quota `json:"quotas"`
	TotalCount int     `json:"total_count"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	TotalPages int     `json:"total_pages"`
}

// QuotaUsageHistoryResponse represents quota usage history
type QuotaUsageHistoryResponse struct {
	Usage      []QuotaUsage `json:"usage"`
	TotalCount int          `json:"total_count"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
	TotalPages int          `json:"total_pages"`
}