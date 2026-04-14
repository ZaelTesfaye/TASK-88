package masterdata

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	appErrors "backend/internal/errors"
	"backend/internal/models"
)

// Version state constants.
const (
	StateDraft    = "draft"
	StateReview   = "review"
	StateActive   = "active"
	StateArchived = "archived"
)

// VersionService provides master data version management operations.
type VersionService struct {
	db *gorm.DB
}

// NewVersionService creates a new VersionService.
func NewVersionService(db *gorm.DB) *VersionService {
	return &VersionService{db: db}
}

// CreateDraft creates a new draft version for the given entity type and scope.
func (s *VersionService) CreateDraft(entityType, scopeKey string, createdBy uint) (*models.MasterVersion, error) {
	if !validEntityTypes[entityType] {
		return nil, appErrors.ValidationError(
			fmt.Sprintf("invalid entity type %q", entityType),
			map[string]interface{}{"valid_types": getValidEntityTypes()},
		)
	}
	if scopeKey == "" {
		return nil, appErrors.ValidationError("scope_key is required", nil)
	}

	// Determine the next version number.
	var maxVersion int
	err := s.db.Model(&models.MasterVersion{}).
		Where("entity_type = ? AND scope_key = ?", entityType, scopeKey).
		Select("COALESCE(MAX(version_no), 0)").
		Scan(&maxVersion).Error
	if err != nil {
		return nil, appErrors.InternalError(fmt.Sprintf("failed to determine next version number: %v", err))
	}

	version := models.MasterVersion{
		EntityType: entityType,
		ScopeKey:   scopeKey,
		VersionNo:  maxVersion + 1,
		State:      StateDraft,
		CreatedBy:  createdBy,
	}

	if err := s.db.Create(&version).Error; err != nil {
		return nil, appErrors.InternalError(fmt.Sprintf("failed to create draft version: %v", err))
	}

	return &version, nil
}

// SubmitForReview transitions a draft version to the review state.
func (s *VersionService) SubmitForReview(versionID uint, reviewerID uint) (*models.MasterVersion, error) {
	var version models.MasterVersion
	if err := s.db.First(&version, versionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, appErrors.NotFound(fmt.Sprintf("version %d not found", versionID))
		}
		return nil, appErrors.InternalError(fmt.Sprintf("failed to load version: %v", err))
	}

	// Only drafts can be submitted.
	if version.State != StateDraft {
		return nil, appErrors.ValidationError(
			fmt.Sprintf("only draft versions can be submitted for review; current state is %q", version.State),
			map[string]interface{}{"current_state": version.State, "required_state": StateDraft},
		)
	}

	// Reviewer must be different from creator.
	if reviewerID == version.CreatedBy {
		return nil, appErrors.ValidationError(
			"reviewer must be different from the creator",
			map[string]interface{}{"creator_id": version.CreatedBy, "reviewer_id": reviewerID},
		)
	}

	version.State = StateReview
	version.ReviewedBy = &reviewerID

	if err := s.db.Save(&version).Error; err != nil {
		return nil, appErrors.InternalError(fmt.Sprintf("failed to submit version for review: %v", err))
	}

	return &version, nil
}

// Activate transitions a reviewed version to the active state.
// Only reviewed versions can be activated. The activator must have SystemAdmin role.
// Deactivates any currently active version for the same entity+scope atomically.
func (s *VersionService) Activate(versionID uint, activatorID uint) (*models.MasterVersion, error) {
	var version models.MasterVersion

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Lock the row for update to prevent race conditions.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&version, versionID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return appErrors.NotFound(fmt.Sprintf("version %d not found", versionID))
			}
			return appErrors.InternalError(fmt.Sprintf("failed to load version: %v", err))
		}

		// Only reviewed versions can be activated.
		if version.State != StateReview {
			return appErrors.ValidationError(
				fmt.Sprintf("only reviewed versions can be activated; current state is %q", version.State),
				map[string]interface{}{"current_state": version.State, "required_state": StateReview},
			)
		}

		// Verify activator is a SystemAdmin by looking up the user.
		var activator models.User
		if err := tx.First(&activator, activatorID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return appErrors.NotFound(fmt.Sprintf("activator user %d not found", activatorID))
			}
			return appErrors.InternalError(fmt.Sprintf("failed to load activator: %v", err))
		}
		if activator.Role != "system_admin" {
			return appErrors.Forbidden("only system administrators can activate versions")
		}

		// Deactivate any currently active version for the same entity+scope.
		if err := tx.Model(&models.MasterVersion{}).
			Where("entity_type = ? AND scope_key = ? AND state = ? AND id != ?",
				version.EntityType, version.ScopeKey, StateActive, versionID).
			Updates(map[string]interface{}{
				"state": StateArchived,
			}).Error; err != nil {
			return appErrors.InternalError(fmt.Sprintf("failed to archive active version: %v", err))
		}

		// Activate this version.
		now := time.Now()
		version.State = StateActive
		version.ActivatedBy = &activatorID
		version.ActivatedAt = &now

		if err := tx.Save(&version).Error; err != nil {
			return appErrors.InternalError(fmt.Sprintf("failed to activate version: %v", err))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &version, nil
}

// Rollback reverts to a previous version by archiving the current active and reactivating the target.
func (s *VersionService) Rollback(versionID uint, actorID uint) (*models.MasterVersion, error) {
	var version models.MasterVersion

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Load the target version.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&version, versionID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return appErrors.NotFound(fmt.Sprintf("version %d not found", versionID))
			}
			return appErrors.InternalError(fmt.Sprintf("failed to load version: %v", err))
		}

		// Target must be archived to be rolled back to.
		if version.State != StateArchived {
			return appErrors.ValidationError(
				fmt.Sprintf("can only rollback to archived versions; target version state is %q", version.State),
				map[string]interface{}{"current_state": version.State, "required_state": StateArchived},
			)
		}

		// Archive any currently active version for same entity+scope.
		if err := tx.Model(&models.MasterVersion{}).
			Where("entity_type = ? AND scope_key = ? AND state = ?",
				version.EntityType, version.ScopeKey, StateActive).
			Updates(map[string]interface{}{
				"state": StateArchived,
			}).Error; err != nil {
			return appErrors.InternalError(fmt.Sprintf("failed to archive current active version: %v", err))
		}

		// Reactivate the target version.
		now := time.Now()
		version.State = StateActive
		version.ActivatedBy = &actorID
		version.ActivatedAt = &now

		if err := tx.Save(&version).Error; err != nil {
			return appErrors.InternalError(fmt.Sprintf("failed to reactivate version: %v", err))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &version, nil
}

// AddItems adds master record IDs to a version.
func (s *VersionService) AddItems(versionID uint, recordIDs []uint) error {
	var version models.MasterVersion
	if err := s.db.First(&version, versionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return appErrors.NotFound(fmt.Sprintf("version %d not found", versionID))
		}
		return appErrors.InternalError(fmt.Sprintf("failed to load version: %v", err))
	}

	// Only draft versions can have items added.
	if version.State != StateDraft {
		return appErrors.ValidationError(
			fmt.Sprintf("items can only be added to draft versions; current state is %q", version.State),
			map[string]interface{}{"current_state": version.State},
		)
	}

	if len(recordIDs) == 0 {
		return appErrors.ValidationError("at least one record ID is required", nil)
	}

	// Verify all records exist.
	var existingCount int64
	if err := s.db.Model(&models.MasterRecord{}).Where("id IN ?", recordIDs).Count(&existingCount).Error; err != nil {
		return appErrors.InternalError(fmt.Sprintf("failed to verify records: %v", err))
	}
	if int(existingCount) != len(recordIDs) {
		return appErrors.ValidationError(
			fmt.Sprintf("some record IDs do not exist; expected %d, found %d", len(recordIDs), existingCount),
			nil,
		)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, recordID := range recordIDs {
			// Check if already linked.
			var count int64
			if err := tx.Model(&models.MasterVersionItem{}).
				Where("version_id = ? AND master_record_id = ?", versionID, recordID).
				Count(&count).Error; err != nil {
				return appErrors.InternalError(fmt.Sprintf("failed to check existing item: %v", err))
			}
			if count > 0 {
				continue // Already linked, skip.
			}

			item := models.MasterVersionItem{
				VersionID:      versionID,
				MasterRecordID: recordID,
			}
			if err := tx.Create(&item).Error; err != nil {
				return appErrors.InternalError(fmt.Sprintf("failed to add item to version: %v", err))
			}
		}
		return nil
	})
}

// RemoveItem removes a specific item from a version.
func (s *VersionService) RemoveItem(versionID uint, itemID uint) error {
	var version models.MasterVersion
	if err := s.db.First(&version, versionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return appErrors.NotFound(fmt.Sprintf("version %d not found", versionID))
		}
		return appErrors.InternalError(fmt.Sprintf("failed to load version: %v", err))
	}

	// Only draft versions can have items removed.
	if version.State != StateDraft {
		return appErrors.ValidationError(
			fmt.Sprintf("items can only be removed from draft versions; current state is %q", version.State),
			map[string]interface{}{"current_state": version.State},
		)
	}

	result := s.db.Where("id = ? AND version_id = ?", itemID, versionID).Delete(&models.MasterVersionItem{})
	if result.Error != nil {
		return appErrors.InternalError(fmt.Sprintf("failed to remove item: %v", result.Error))
	}
	if result.RowsAffected == 0 {
		return appErrors.NotFound(fmt.Sprintf("item %d not found in version %d", itemID, versionID))
	}

	return nil
}

// GetVersionHistory returns the version history for an entity+scope, ordered by version number descending.
func (s *VersionService) GetVersionHistory(entityType, scopeKey string) ([]models.MasterVersion, error) {
	if !validEntityTypes[entityType] {
		return nil, appErrors.ValidationError(
			fmt.Sprintf("invalid entity type %q", entityType),
			map[string]interface{}{"valid_types": getValidEntityTypes()},
		)
	}

	var versions []models.MasterVersion
	query := s.db.Where("entity_type = ?", entityType)
	if scopeKey != "" {
		query = query.Where("scope_key = ?", scopeKey)
	}

	if err := query.Order("version_no DESC").Find(&versions).Error; err != nil {
		return nil, appErrors.InternalError(fmt.Sprintf("failed to load version history: %v", err))
	}

	if versions == nil {
		versions = []models.MasterVersion{}
	}

	return versions, nil
}

// GetVersion returns a single version by ID.
func (s *VersionService) GetVersion(id uint) (*models.MasterVersion, error) {
	var version models.MasterVersion
	if err := s.db.First(&version, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, appErrors.NotFound(fmt.Sprintf("version %d not found", id))
		}
		return nil, appErrors.InternalError(fmt.Sprintf("failed to load version: %v", err))
	}
	return &version, nil
}

// GetVersionItems returns all items in a version.
func (s *VersionService) GetVersionItems(versionID uint) ([]models.MasterVersionItem, error) {
	var version models.MasterVersion
	if err := s.db.First(&version, versionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, appErrors.NotFound(fmt.Sprintf("version %d not found", versionID))
		}
		return nil, appErrors.InternalError(fmt.Sprintf("failed to load version: %v", err))
	}

	var items []models.MasterVersionItem
	if err := s.db.Where("version_id = ?", versionID).Find(&items).Error; err != nil {
		return nil, appErrors.InternalError(fmt.Sprintf("failed to load version items: %v", err))
	}

	if items == nil {
		items = []models.MasterVersionItem{}
	}

	return items, nil
}

// DiffVersions computes the diff between two versions.
func (s *VersionService) DiffVersions(versionAID, versionBID uint) (map[string]interface{}, error) {
	// Load items from both versions.
	var itemsA []models.MasterVersionItem
	if err := s.db.Where("version_id = ?", versionAID).Find(&itemsA).Error; err != nil {
		return nil, appErrors.InternalError(fmt.Sprintf("failed to load version A items: %v", err))
	}

	var itemsB []models.MasterVersionItem
	if err := s.db.Where("version_id = ?", versionBID).Find(&itemsB).Error; err != nil {
		return nil, appErrors.InternalError(fmt.Sprintf("failed to load version B items: %v", err))
	}

	setA := make(map[uint]bool, len(itemsA))
	for _, item := range itemsA {
		setA[item.MasterRecordID] = true
	}

	setB := make(map[uint]bool, len(itemsB))
	for _, item := range itemsB {
		setB[item.MasterRecordID] = true
	}

	var added, removed, unchanged []uint
	for id := range setB {
		if !setA[id] {
			added = append(added, id)
		} else {
			unchanged = append(unchanged, id)
		}
	}
	for id := range setA {
		if !setB[id] {
			removed = append(removed, id)
		}
	}

	return map[string]interface{}{
		"version_a_id": versionAID,
		"version_b_id": versionBID,
		"added":        added,
		"removed":      removed,
		"unchanged":    unchanged,
	}, nil
}
