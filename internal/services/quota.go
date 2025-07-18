package services

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/emagen-ai/cagen-quota/internal/auth"
	"github.com/emagen-ai/cagen-quota/internal/database"
	"github.com/emagen-ai/cagen-quota/internal/models"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// QuotaService handles quota operations
type QuotaService struct {
	db         *database.DB
	authClient *auth.AuthClient
	logger     *logrus.Logger
}

// NewQuotaService creates a new quota service
func NewQuotaService(db *database.DB, authClient *auth.AuthClient, logger *logrus.Logger) *QuotaService {
	return &QuotaService{
		db:         db,
		authClient: authClient,
		logger:     logger,
	}
}

// CreateQuota creates a root quota
func (qs *QuotaService) CreateQuota(userInfo *auth.UserInfo, request *models.QuotaCreateRequest) (*models.Quota, error) {
	// Validate request
	if request.TotalMB <= 0 {
		return nil, fmt.Errorf("total_mb must be greater than 0")
	}

	if request.Type != models.QuotaTypeOrganization && request.Type != models.QuotaTypeTeam {
		return nil, fmt.Errorf("invalid quota type: %s", request.Type)
	}

	// For team quotas, team_id must be specified
	if request.Type == models.QuotaTypeTeam && (request.TeamID == nil || *request.TeamID == "") {
		return nil, fmt.Errorf("team_id is required for team quota")
	}

	// Generate quota ID
	quotaID := fmt.Sprintf("quota_%s", strings.ToLower(uuid.New().String()[:13]))

	// Create quota within transaction
	quota := &models.Quota{}
	err := qs.db.WithTransaction(func(tx *sql.Tx) error {
		// 1. Create quota record
		quota = &models.Quota{
			ID:             quotaID,
			Name:           request.Name,
			Description:    request.Description,
			Type:           request.Type,
			TotalMB:        request.TotalMB,
			UsedMB:         0,
			AllocatedMB:    0,
			ParentQuotaID:  nil, // Root quota
			Level:          0,   // Root level
			Path:           "/" + quotaID,
			OwnerID:        userInfo.UserID,
			OrganizationID: userInfo.OrganizationID,
			TeamID:         request.TeamID,
			Status:         models.QuotaStatusActive,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		insertQuery := `
			INSERT INTO quotas (id, name, description, type, total_mb, used_mb, allocated_mb, 
			                   parent_quota_id, level, path, owner_id, organization_id, team_id, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		`

		_, err := tx.Exec(insertQuery, quota.ID, quota.Name, quota.Description, quota.Type,
			quota.TotalMB, quota.UsedMB, quota.AllocatedMB, quota.ParentQuotaID, quota.Level,
			quota.Path, quota.OwnerID, quota.OrganizationID, quota.TeamID, quota.Status,
			quota.CreatedAt, quota.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to create quota: %w", err)
		}

		// 2. Create quota resource in auth service (disabled for now)
		// TODO: Re-enable when auth service is fully configured
		/*
		err = qs.authClient.CreateResource(userInfo, quotaID, "quota", quota.Name, quota.Description)
		if err != nil {
			return fmt.Errorf("failed to create quota resource in auth service: %w", err)
		}
		*/
		qs.logger.WithField("quota_id", quotaID).Info("Skipped auth service resource creation for testing")

		// 3. Create audit log
		err = qs.createAuditLogTx(tx, quotaID, "create", userInfo.UserID, nil, map[string]interface{}{
			"name":     quota.Name,
			"type":     quota.Type,
			"total_mb": quota.TotalMB,
		})
		if err != nil {
			qs.logger.WithError(err).Warn("Failed to create audit log")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	qs.logger.WithFields(logrus.Fields{
		"quota_id":        quota.ID,
		"name":            quota.Name,
		"type":            quota.Type,
		"total_mb":        quota.TotalMB,
		"organization_id": quota.OrganizationID,
		"team_id":         quota.TeamID,
		"owner_id":        quota.OwnerID,
	}).Info("Root quota created successfully")

	return quota, nil
}

// ListQuotas lists quotas for a user with pagination and filtering
func (qs *QuotaService) ListQuotas(userInfo *auth.UserInfo, page, pageSize int, quotaType string) (*models.QuotaListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Build query with filters
	whereClause := "WHERE organization_id = $1 AND status = 'active'"
	args := []interface{}{userInfo.OrganizationID}
	argIndex := 2

	if quotaType != "" {
		whereClause += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, quotaType)
		argIndex++
	}

	// Count total items
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM quotas %s", whereClause)
	var totalCount int
	err := qs.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count quotas: %w", err)
	}

	// Get quotas with pagination
	query := fmt.Sprintf(`
		SELECT id, name, description, type, total_mb, used_mb, allocated_mb, 
		       parent_quota_id, level, path, owner_id, organization_id, team_id, 
		       status, created_at, updated_at, deleted_at
		FROM quotas %s 
		ORDER BY created_at DESC 
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)
	
	args = append(args, pageSize, offset)

	rows, err := qs.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query quotas: %w", err)
	}
	defer rows.Close()

	var quotas []models.Quota
	for rows.Next() {
		var quota models.Quota
		var parentQuotaID sql.NullString
		var teamID sql.NullString
		var deletedAt sql.NullTime

		err := rows.Scan(
			&quota.ID, &quota.Name, &quota.Description, &quota.Type,
			&quota.TotalMB, &quota.UsedMB, &quota.AllocatedMB,
			&parentQuotaID, &quota.Level, &quota.Path,
			&quota.OwnerID, &quota.OrganizationID, &teamID,
			&quota.Status, &quota.CreatedAt, &quota.UpdatedAt, &deletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan quota: %w", err)
		}

		// Handle nullable fields
		if parentQuotaID.Valid {
			quota.ParentQuotaID = &parentQuotaID.String
		}
		if teamID.Valid {
			quota.TeamID = &teamID.String
		}
		if deletedAt.Valid {
			quota.DeletedAt = &deletedAt.Time
		}

		// Calculate available_mb
		quota.AvailableMB = quota.TotalMB - quota.UsedMB - quota.AllocatedMB

		quotas = append(quotas, quota)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating quota rows: %w", err)
	}

	// Calculate total pages
	totalPages := (totalCount + pageSize - 1) / pageSize

	qs.logger.WithFields(logrus.Fields{
		"user_id":     userInfo.UserID,
		"org_id":      userInfo.OrganizationID,
		"total_count": totalCount,
		"page":        page,
		"page_size":   pageSize,
		"quota_type":  quotaType,
		"found":       len(quotas),
	}).Info("Listed quotas successfully")

	return &models.QuotaListResponse{
		Quotas:     quotas,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// AllocateQuota allocates a sub-quota from a parent quota
func (qs *QuotaService) AllocateQuota(userInfo *auth.UserInfo, parentQuotaID string, request *models.QuotaAllocateRequest) (*models.Quota, error) {
	// Check admin permission on parent quota
	hasPermission, err := qs.authClient.CheckPermission(userInfo, parentQuotaID, []string{auth.QuotaPermissionAdmin})
	if err != nil {
		return nil, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return nil, fmt.Errorf("insufficient permissions to allocate quota")
	}

	// Validate request
	if request.AllocateMB <= 0 {
		return nil, fmt.Errorf("allocate_mb must be greater than 0")
	}

	// Generate child quota ID
	childQuotaID := fmt.Sprintf("quota_%s", strings.ToLower(uuid.New().String()[:13]))

	// Allocate quota within transaction
	childQuota := &models.Quota{}
	err = qs.db.WithTransaction(func(tx *sql.Tx) error {
		// 1. Get parent quota with lock
		parentQuota, err := qs.getQuotaForUpdateTx(tx, parentQuotaID)
		if err != nil {
			return fmt.Errorf("failed to get parent quota: %w", err)
		}

		// 2. Check available capacity
		if parentQuota.AvailableMB < request.AllocateMB {
			return fmt.Errorf("insufficient quota: available %d MB, requested %d MB",
				parentQuota.AvailableMB, request.AllocateMB)
		}

		// 3. Validate hierarchy rules
		err = qs.validateAllocationRules(parentQuota, request)
		if err != nil {
			return err
		}

		// 4. Create child quota
		childQuota = &models.Quota{
			ID:             childQuotaID,
			Name:           request.Name,
			Description:    request.Description,
			Type:           request.Type,
			TotalMB:        request.AllocateMB,
			UsedMB:         0,
			AllocatedMB:    0,
			ParentQuotaID:  &parentQuotaID,
			Level:          parentQuota.Level + 1,
			Path:           parentQuota.Path + "/" + childQuotaID,
			OwnerID:        parentQuota.OwnerID, // Inherit owner from parent
			OrganizationID: parentQuota.OrganizationID,
			TeamID:         qs.determineTeamID(parentQuota, request),
			Status:         models.QuotaStatusActive,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		insertQuery := `
			INSERT INTO quotas (id, name, description, type, total_mb, used_mb, allocated_mb, 
			                   parent_quota_id, level, path, owner_id, organization_id, team_id, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		`

		_, err = tx.Exec(insertQuery, childQuota.ID, childQuota.Name, childQuota.Description, childQuota.Type,
			childQuota.TotalMB, childQuota.UsedMB, childQuota.AllocatedMB, childQuota.ParentQuotaID, childQuota.Level,
			childQuota.Path, childQuota.OwnerID, childQuota.OrganizationID, childQuota.TeamID, childQuota.Status,
			childQuota.CreatedAt, childQuota.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to create child quota: %w", err)
		}

		// 5. Update parent quota allocated_mb
		updateQuery := `UPDATE quotas SET allocated_mb = allocated_mb + $1, updated_at = NOW() WHERE id = $2`
		_, err = tx.Exec(updateQuery, request.AllocateMB, parentQuotaID)
		if err != nil {
			return fmt.Errorf("failed to update parent quota: %w", err)
		}

		// 6. Create quota resource in auth service
		err = qs.authClient.CreateResource(userInfo, childQuotaID, "quota", childQuota.Name, childQuota.Description)
		if err != nil {
			return fmt.Errorf("failed to create child quota resource in auth service: %w", err)
		}

		// 7. Grant admin permissions to specified users
		for _, adminUserID := range request.AdminUserIDs {
			err = qs.authClient.GrantPermission(userInfo, adminUserID, childQuotaID, []string{auth.QuotaPermissionAdmin})
			if err != nil {
				qs.logger.WithError(err).WithFields(logrus.Fields{
					"child_quota_id": childQuotaID,
					"admin_user_id":  adminUserID,
				}).Warn("Failed to grant admin permission")
			}
		}

		// 8. Create audit log
		err = qs.createAuditLogTx(tx, childQuotaID, "allocate", userInfo.UserID, nil, map[string]interface{}{
			"parent_quota_id": parentQuotaID,
			"allocated_mb":    request.AllocateMB,
			"name":            childQuota.Name,
			"type":            childQuota.Type,
		})
		if err != nil {
			qs.logger.WithError(err).Warn("Failed to create audit log")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	qs.logger.WithFields(logrus.Fields{
		"parent_quota_id": parentQuotaID,
		"child_quota_id":  childQuota.ID,
		"allocated_mb":    request.AllocateMB,
		"admin_user_ids":  request.AdminUserIDs,
	}).Info("Quota allocated successfully")

	return childQuota, nil
}

// ReleaseQuota releases a quota and returns its capacity to parent
func (qs *QuotaService) ReleaseQuota(userInfo *auth.UserInfo, quotaID string) error {
	// Check admin permission
	hasPermission, err := qs.authClient.CheckPermission(userInfo, quotaID, []string{auth.QuotaPermissionAdmin})
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return fmt.Errorf("insufficient permissions to release quota")
	}

	return qs.db.WithTransaction(func(tx *sql.Tx) error {
		// 1. Get quota with lock
		quota, err := qs.getQuotaForUpdateTx(tx, quotaID)
		if err != nil {
			return fmt.Errorf("failed to get quota: %w", err)
		}

		// 2. Check if quota can be released
		if quota.UsedMB > 0 || quota.AllocatedMB > 0 {
			return fmt.Errorf("cannot release quota with active usage (%d MB) or allocations (%d MB)",
				quota.UsedMB, quota.AllocatedMB)
		}

		// 3. Return capacity to parent (if exists)
		if quota.ParentQuotaID != nil {
			updateParentQuery := `UPDATE quotas SET allocated_mb = allocated_mb - $1, updated_at = NOW() WHERE id = $2`
			_, err = tx.Exec(updateParentQuery, quota.TotalMB, *quota.ParentQuotaID)
			if err != nil {
				return fmt.Errorf("failed to update parent quota: %w", err)
			}
		}

		// 4. Soft delete quota
		deleteQuery := `UPDATE quotas SET status = $1, deleted_at = NOW(), updated_at = NOW() WHERE id = $2`
		_, err = tx.Exec(deleteQuery, models.QuotaStatusDeleted, quotaID)
		if err != nil {
			return fmt.Errorf("failed to delete quota: %w", err)
		}

		// 5. Create audit log
		err = qs.createAuditLogTx(tx, quotaID, "release", userInfo.UserID, nil, map[string]interface{}{
			"parent_quota_id": quota.ParentQuotaID,
			"returned_mb":     quota.TotalMB,
		})
		if err != nil {
			qs.logger.WithError(err).Warn("Failed to create audit log")
		}

		return nil
	})
}

// AllocateUsage allocates usage to a quota
func (qs *QuotaService) AllocateUsage(userInfo *auth.UserInfo, quotaID string, request *models.QuotaUsageRequest) error {
	// Check read permission
	hasPermission, err := qs.authClient.CheckPermission(userInfo, quotaID, []string{auth.QuotaPermissionRead})
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return fmt.Errorf("insufficient permissions to use quota")
	}

	return qs.db.WithTransaction(func(tx *sql.Tx) error {
		// 1. Get quota with lock
		quota, err := qs.getQuotaForUpdateTx(tx, quotaID)
		if err != nil {
			return fmt.Errorf("failed to get quota: %w", err)
		}

		// 2. Check available capacity
		availableForUsage := quota.TotalMB - quota.UsedMB - quota.AllocatedMB
		if availableForUsage < request.UsageMB {
			return fmt.Errorf("insufficient quota: available %d MB, requested %d MB",
				availableForUsage, request.UsageMB)
		}

		// 3. Update quota usage
		updateQuery := `UPDATE quotas SET used_mb = used_mb + $1, updated_at = NOW() WHERE id = $2`
		_, err = tx.Exec(updateQuery, request.UsageMB, quotaID)
		if err != nil {
			return fmt.Errorf("failed to update quota usage: %w", err)
		}

		// 4. Record usage
		usageID := fmt.Sprintf("usage_%s", strings.ToLower(uuid.New().String()[:13]))
		usageQuery := `
			INSERT INTO quota_usage (id, quota_id, user_id, resource_id, usage_mb, operation, reason, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		`
		_, err = tx.Exec(usageQuery, usageID, quotaID, userInfo.UserID, request.ResourceID,
			request.UsageMB, models.OperationAllocate, request.Reason)
		if err != nil {
			return fmt.Errorf("failed to record usage: %w", err)
		}

		// 5. Create audit log
		err = qs.createAuditLogTx(tx, quotaID, "usage_allocate", userInfo.UserID, nil, map[string]interface{}{
			"resource_id": request.ResourceID,
			"usage_mb":    request.UsageMB,
			"reason":      request.Reason,
		})
		if err != nil {
			qs.logger.WithError(err).Warn("Failed to create audit log")
		}

		return nil
	})
}

// DeallocateUsage deallocates usage from a quota
func (qs *QuotaService) DeallocateUsage(userInfo *auth.UserInfo, quotaID string, request *models.QuotaUsageRequest) error {
	// Check read permission
	hasPermission, err := qs.authClient.CheckPermission(userInfo, quotaID, []string{auth.QuotaPermissionRead})
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return fmt.Errorf("insufficient permissions to deallocate quota usage")
	}

	return qs.db.WithTransaction(func(tx *sql.Tx) error {
		// 1. Get quota with lock
		quota, err := qs.getQuotaForUpdateTx(tx, quotaID)
		if err != nil {
			return fmt.Errorf("failed to get quota: %w", err)
		}

		// 2. Check if enough usage to deallocate
		if quota.UsedMB < request.UsageMB {
			return fmt.Errorf("cannot deallocate %d MB, only %d MB in use", request.UsageMB, quota.UsedMB)
		}

		// 3. Update quota usage
		updateQuery := `UPDATE quotas SET used_mb = used_mb - $1, updated_at = NOW() WHERE id = $2`
		_, err = tx.Exec(updateQuery, request.UsageMB, quotaID)
		if err != nil {
			return fmt.Errorf("failed to update quota usage: %w", err)
		}

		// 4. Record usage
		usageID := fmt.Sprintf("usage_%s", strings.ToLower(uuid.New().String()[:13]))
		usageQuery := `
			INSERT INTO quota_usage (id, quota_id, user_id, resource_id, usage_mb, operation, reason, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		`
		_, err = tx.Exec(usageQuery, usageID, quotaID, userInfo.UserID, request.ResourceID,
			request.UsageMB, models.OperationDeallocate, request.Reason)
		if err != nil {
			return fmt.Errorf("failed to record usage: %w", err)
		}

		// 5. Create audit log
		err = qs.createAuditLogTx(tx, quotaID, "usage_deallocate", userInfo.UserID, nil, map[string]interface{}{
			"resource_id": request.ResourceID,
			"usage_mb":    request.UsageMB,
			"reason":      request.Reason,
		})
		if err != nil {
			qs.logger.WithError(err).Warn("Failed to create audit log")
		}

		return nil
	})
}

// GetQuota retrieves a quota by ID
func (qs *QuotaService) GetQuota(userInfo *auth.UserInfo, quotaID string) (*models.Quota, error) {
	// Check read permission
	hasPermission, err := qs.authClient.CheckPermission(userInfo, quotaID, []string{auth.QuotaPermissionRead})
	if err != nil {
		return nil, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return nil, fmt.Errorf("insufficient permissions to view quota")
	}

	query := `
		SELECT id, name, description, type, total_mb, used_mb, allocated_mb, 
		       parent_quota_id, level, path, owner_id, organization_id, team_id, 
		       status, created_at, updated_at, deleted_at
		FROM quotas 
		WHERE id = $1 AND status != $2
	`

	quota := &models.Quota{}
	row := qs.db.QueryRow(query, quotaID, models.QuotaStatusDeleted)

	err = row.Scan(&quota.ID, &quota.Name, &quota.Description, &quota.Type,
		&quota.TotalMB, &quota.UsedMB, &quota.AllocatedMB, &quota.ParentQuotaID,
		&quota.Level, &quota.Path, &quota.OwnerID, &quota.OrganizationID, &quota.TeamID,
		&quota.Status, &quota.CreatedAt, &quota.UpdatedAt, &quota.DeletedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("quota not found")
		}
		return nil, fmt.Errorf("failed to get quota: %w", err)
	}

	// Calculate available MB
	quota.AvailableMB = quota.TotalMB - quota.UsedMB - quota.AllocatedMB

	return quota, nil
}

// Helper functions

func (qs *QuotaService) getQuotaForUpdateTx(tx *sql.Tx, quotaID string) (*models.Quota, error) {
	query := `
		SELECT id, name, description, type, total_mb, used_mb, allocated_mb, 
		       parent_quota_id, level, path, owner_id, organization_id, team_id, 
		       status, created_at, updated_at, deleted_at
		FROM quotas 
		WHERE id = $1 AND status != $2
		FOR UPDATE
	`

	quota := &models.Quota{}
	row := tx.QueryRow(query, quotaID, models.QuotaStatusDeleted)

	err := row.Scan(&quota.ID, &quota.Name, &quota.Description, &quota.Type,
		&quota.TotalMB, &quota.UsedMB, &quota.AllocatedMB, &quota.ParentQuotaID,
		&quota.Level, &quota.Path, &quota.OwnerID, &quota.OrganizationID, &quota.TeamID,
		&quota.Status, &quota.CreatedAt, &quota.UpdatedAt, &quota.DeletedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("quota not found")
		}
		return nil, fmt.Errorf("failed to get quota: %w", err)
	}

	// Calculate available MB
	quota.AvailableMB = quota.TotalMB - quota.UsedMB - quota.AllocatedMB

	return quota, nil
}

func (qs *QuotaService) validateAllocationRules(parentQuota *models.Quota, request *models.QuotaAllocateRequest) error {
	// Organization quota can allocate to team quota
	if parentQuota.Type == models.QuotaTypeOrganization && request.Type == models.QuotaTypeTeam {
		return nil
	}

	// Organization quota can allocate to organization quota
	if parentQuota.Type == models.QuotaTypeOrganization && request.Type == models.QuotaTypeOrganization {
		return nil
	}

	// Team quota can only allocate to same team
	if parentQuota.Type == models.QuotaTypeTeam && request.Type == models.QuotaTypeTeam {
		if parentQuota.TeamID == nil || request.TargetID != *parentQuota.TeamID {
			return fmt.Errorf("team quota can only allocate to the same team")
		}
		return nil
	}

	return fmt.Errorf("invalid allocation: %s quota cannot allocate to %s quota",
		parentQuota.Type, request.Type)
}

func (qs *QuotaService) determineTeamID(parentQuota *models.Quota, request *models.QuotaAllocateRequest) *string {
	if request.Type == models.QuotaTypeTeam {
		return &request.TargetID
	}
	return parentQuota.TeamID
}

func (qs *QuotaService) createAuditLogTx(tx *sql.Tx, quotaID, actionType, actorUserID string, targetUserID *string, details map[string]interface{}) error {
	auditID := fmt.Sprintf("audit_%s", strings.ToLower(uuid.New().String()[:13]))
	
	detailsJSON := "{}"
	if details != nil {
		if jsonBytes, err := models.JSONMap(details).Value(); err == nil {
			if str, ok := jsonBytes.([]byte); ok {
				detailsJSON = string(str)
			}
		}
	}

	query := `
		INSERT INTO quota_audit_logs (id, quota_id, action_type, actor_user_id, target_user_id, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`

	_, err := tx.Exec(query, auditID, quotaID, actionType, actorUserID, targetUserID, detailsJSON)
	return err
}