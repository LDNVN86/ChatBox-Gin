package repositories

import (
	"context"
	"time"

	"chatbox-gin/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ===========================================================================
// Rule Repository GORM Implementation
// ===========================================================================

// ruleRepo triển khai RuleRepository với GORM
type ruleRepo struct {
	db *gorm.DB
}

// NewRuleRepository tạo instance mới của RuleRepository
func NewRuleRepository(db *gorm.DB) RuleRepository {
	return &ruleRepo{db: db}
}

// FindByID tìm rule theo ID
func (r *ruleRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Rule, error) {
	var rule models.Rule
	if err := r.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// FindActiveByWorkspace lấy tất cả rules active, sắp xếp theo priority DESC
func (r *ruleRepo) FindActiveByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]models.Rule, error) {
	var rules []models.Rule
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND is_active = ?", workspaceID, true).
		Order("priority DESC").
		Find(&rules).Error
	return rules, err
}

// FindByTriggerType tìm rules theo trigger type
func (r *ruleRepo) FindByTriggerType(ctx context.Context, workspaceID uuid.UUID, triggerType models.TriggerType) ([]models.Rule, error) {
	var rules []models.Rule
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND is_active = ? AND trigger_type = ?", workspaceID, true, triggerType).
		Order("priority DESC").
		Find(&rules).Error
	return rules, err
}

// Create tạo rule mới
func (r *ruleRepo) Create(ctx context.Context, rule *models.Rule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

// Update cập nhật rule
func (r *ruleRepo) Update(ctx context.Context, rule *models.Rule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

// IncrementHitCount tăng hit count và update last_triggered_at
func (r *ruleRepo) IncrementHitCount(ctx context.Context, ruleID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.Rule{}).
		Where("id = ?", ruleID).
		Updates(map[string]interface{}{
			"hit_count":         gorm.Expr("hit_count + 1"),
			"last_triggered_at": now,
		}).Error
}

// FindByIDUnscoped tìm rule theo ID, bao gồm cả deleted
func (r *ruleRepo) FindByIDUnscoped(ctx context.Context, id uuid.UUID) (*models.Rule, error) {
	var rule models.Rule
	if err := r.db.WithContext(ctx).Unscoped().First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// FindAllByWorkspace lấy tất cả rules (kể cả inactive) trong workspace
func (r *ruleRepo) FindAllByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]models.Rule, error) {
	var rules []models.Rule
	err := r.db.WithContext(ctx).
		Where("workspace_id = ?", workspaceID).
		Order("priority DESC").
		Find(&rules).Error
	return rules, err
}

// FindDeletedByWorkspace lấy tất cả rules đã xóa trong workspace
func (r *ruleRepo) FindDeletedByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]models.Rule, error) {
	var rules []models.Rule
	err := r.db.WithContext(ctx).
		Unscoped().
		Where("workspace_id = ? AND deleted_at IS NOT NULL", workspaceID).
		Order("deleted_at DESC").
		Find(&rules).Error
	return rules, err
}

// Delete soft delete rule (set deleted_at)
func (r *ruleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Rule{}, id).Error
}

// Restore khôi phục rule đã xóa (clear deleted_at)
func (r *ruleRepo) Restore(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Unscoped().
		Model(&models.Rule{}).
		Where("id = ?", id).
		Update("deleted_at", nil).Error
}
