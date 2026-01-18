package repositories

import (
	"context"

	"chatbox-gin/internal/models"

	"github.com/google/uuid"
)

// ===========================================================================
// Rule Repository Interface
// Quản lý CRUD cho bot rules
// ===========================================================================

// RuleRepository interface cho rule data access
type RuleRepository interface {
	// FindByID tìm rule theo ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.Rule, error)

	// FindByIDUnscoped tìm rule theo ID, bao gồm cả deleted
	FindByIDUnscoped(ctx context.Context, id uuid.UUID) (*models.Rule, error)

	// FindActiveByWorkspace lấy tất cả rules active trong workspace
	// Sắp xếp theo priority DESC
	FindActiveByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]models.Rule, error)

	// FindAllByWorkspace lấy tất cả rules (kể cả inactive) trong workspace
	FindAllByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]models.Rule, error)

	// FindDeletedByWorkspace lấy tất cả rules đã xóa trong workspace
	FindDeletedByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]models.Rule, error)

	// FindByTriggerType tìm rules theo trigger type
	FindByTriggerType(ctx context.Context, workspaceID uuid.UUID, triggerType models.TriggerType) ([]models.Rule, error)

	// Create tạo rule mới
	Create(ctx context.Context, rule *models.Rule) error

	// Update cập nhật rule
	Update(ctx context.Context, rule *models.Rule) error

	// Delete soft delete rule
	Delete(ctx context.Context, id uuid.UUID) error

	// Restore khôi phục rule đã xóa
	Restore(ctx context.Context, id uuid.UUID) error

	// IncrementHitCount tăng hit count và update last_triggered_at
	IncrementHitCount(ctx context.Context, ruleID uuid.UUID) error
}

// ===========================================================================
// Tag Repository Interface
// Quản lý CRUD cho tags
// ===========================================================================

// TagRepository interface cho tag data access
type TagRepository interface {
	// FindByID tìm tag theo ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.Tag, error)

	// FindByWorkspace lấy danh sách tags trong workspace
	FindByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]models.Tag, error)

	// FindByName tìm tag theo tên trong workspace
	FindByName(ctx context.Context, workspaceID uuid.UUID, name string) (*models.Tag, error)

	// Create tạo tag mới
	Create(ctx context.Context, tag *models.Tag) error

	// Update cập nhật tag
	Update(ctx context.Context, tag *models.Tag) error

	// Delete xóa tag
	Delete(ctx context.Context, id uuid.UUID) error

	// AddToConversation thêm tag vào conversation
	AddToConversation(ctx context.Context, conversationID, tagID uuid.UUID, createdBy *uuid.UUID) error

	// RemoveFromConversation xóa tag khỏi conversation
	RemoveFromConversation(ctx context.Context, conversationID, tagID uuid.UUID) error
}

// ===========================================================================
// WebhookEvent Repository Interface
// Quản lý webhook events cho idempotency và retry
// ===========================================================================

// WebhookEventRepository interface cho webhook event data access
type WebhookEventRepository interface {
	// FindByEventID tìm event theo channel type và event ID
	// Dùng để check duplicate
	FindByEventID(ctx context.Context, channelType models.ChannelType, eventID string) (*models.WebhookEvent, error)

	// Create tạo webhook event mới
	Create(ctx context.Context, event *models.WebhookEvent) error

	// Update cập nhật webhook event
	Update(ctx context.Context, event *models.WebhookEvent) error

	// FindPendingForRetry tìm các events failed có thể retry
	FindPendingForRetry(ctx context.Context, maxRetries int, limit int) ([]models.WebhookEvent, error)
}

// ===========================================================================
// Note Repository Interface
// Quản lý CRUD cho internal notes
// ===========================================================================

// NoteRepository interface cho note data access
type NoteRepository interface {
	// FindByConversation lấy danh sách notes của conversation
	FindByConversation(ctx context.Context, conversationID uuid.UUID) ([]models.Note, error)

	// Create tạo note mới
	Create(ctx context.Context, note *models.Note) error

	// Update cập nhật note
	Update(ctx context.Context, note *models.Note) error

	// Delete xóa note
	Delete(ctx context.Context, id uuid.UUID) error
}
