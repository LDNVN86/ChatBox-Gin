package repositories

import (
	"context"
	"time"

	"chatbox-gin/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ===========================================================================
// Message Repository GORM Implementation
// ===========================================================================

// messageRepo triển khai MessageRepository với GORM
type messageRepo struct {
	db *gorm.DB
}

// NewMessageRepository tạo instance mới của MessageRepository
func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepo{db: db}
}

// FindByID tìm message theo ID
func (r *messageRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Message, error) {
	var msg models.Message
	if err := r.db.WithContext(ctx).First(&msg, id).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}

// FindByConversation lấy danh sách messages trong conversation
func (r *messageRepo) FindByConversation(ctx context.Context, conversationID uuid.UUID, opts FindOptions) ([]models.Message, int64, error) {
	opts.SetDefaults()

	var messages []models.Message
	var total int64

	query := r.db.WithContext(ctx).
		Model(&models.Message{}).
		Where("conversation_id = ?", conversationID)

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Default order by created_at asc for messages (chronological)
	if opts.OrderBy == "created_at" && opts.OrderDir == "desc" {
		opts.OrderDir = "asc" // Messages thường được sort theo thứ tự thời gian
	}

	// Get records
	err := query.
		Order(opts.GetOrderClause()).
		Offset(opts.Offset).
		Limit(opts.Limit).
		Find(&messages).Error

	return messages, total, err
}

// FindByChannelMessageID tìm message theo channel message ID
func (r *messageRepo) FindByChannelMessageID(ctx context.Context, channelMessageID string) (*models.Message, error) {
	var msg models.Message
	err := r.db.WithContext(ctx).
		Where("channel_message_id = ?", channelMessageID).
		First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// Create tạo message mới
func (r *messageRepo) Create(ctx context.Context, msg *models.Message) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

// Update cập nhật message
func (r *messageRepo) Update(ctx context.Context, msg *models.Message) error {
	return r.db.WithContext(ctx).Save(msg).Error
}

// MarkAsRead đánh dấu message đã đọc
func (r *messageRepo) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.Message{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}
