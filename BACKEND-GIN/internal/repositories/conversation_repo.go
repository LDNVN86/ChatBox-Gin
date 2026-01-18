package repositories

import (
	"context"

	"chatbox-gin/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ===========================================================================
// Conversation Repository GORM Implementation
// ===========================================================================

// conversationRepo triển khai ConversationRepository với GORM
type conversationRepo struct {
	db *gorm.DB
}

// NewConversationRepository tạo instance mới của ConversationRepository
func NewConversationRepository(db *gorm.DB) ConversationRepository {
	return &conversationRepo{db: db}
}

// FindByID tìm conversation theo ID
func (r *conversationRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Conversation, error) {
	var conv models.Conversation
	if err := r.db.WithContext(ctx).
		Preload("Participant").
		Preload("ChannelAccount").
		First(&conv, id).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

// FindByThreadID tìm conversation theo channel thread ID
func (r *conversationRepo) FindByThreadID(ctx context.Context, channelAccountID uuid.UUID, threadID string) (*models.Conversation, error) {
	var conv models.Conversation
	err := r.db.WithContext(ctx).
		Where("channel_account_id = ? AND channel_thread_id = ?", channelAccountID, threadID).
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// FindByWorkspace lấy danh sách conversations trong workspace
func (r *conversationRepo) FindByWorkspace(ctx context.Context, workspaceID uuid.UUID, opts FindOptions) ([]models.Conversation, int64, error) {
	opts.SetDefaults()

	var conversations []models.Conversation
	var total int64

	query := r.db.WithContext(ctx).
		Model(&models.Conversation{}).
		Where("workspace_id = ?", workspaceID)

	// Apply filters
	if opts.Filters != nil {
		if status, ok := opts.Filters["status"]; ok {
			query = query.Where("status = ?", status)
		}
		if assignedTo, ok := opts.Filters["assigned_to"]; ok {
			query = query.Where("assigned_to = ?", assignedTo)
		}
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get records
	err := query.
		Preload("Participant").
		Preload("ChannelAccount").
		Order(opts.GetOrderClause()).
		Offset(opts.Offset).
		Limit(opts.Limit).
		Find(&conversations).Error

	return conversations, total, err
}

// FindOrCreate tìm hoặc tạo mới conversation
func (r *conversationRepo) FindOrCreate(ctx context.Context, conv *models.Conversation) (*models.Conversation, bool, error) {
	// Thử tìm conversation đang mở của participant
	existing, err := r.FindOpenByParticipant(ctx, conv.ParticipantID)
	if err == nil {
		return existing, false, nil
	}

	// Không tìm thấy, tạo mới
	if err := r.db.WithContext(ctx).Create(conv).Error; err != nil {
		return nil, false, err
	}

	return conv, true, nil
}

// FindOpenByParticipant tìm conversation đang mở của participant
func (r *conversationRepo) FindOpenByParticipant(ctx context.Context, participantID uuid.UUID) (*models.Conversation, error) {
	var conv models.Conversation
	err := r.db.WithContext(ctx).
		Where("participant_id = ?", participantID).
		Where("status IN ?", []models.ConversationStatus{models.StatusOpen, models.StatusPending, models.StatusBotPaused}).
		Order("created_at DESC").
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// Create tạo conversation mới
func (r *conversationRepo) Create(ctx context.Context, conv *models.Conversation) error {
	return r.db.WithContext(ctx).Create(conv).Error
}

// Update cập nhật conversation
func (r *conversationRepo) Update(ctx context.Context, conv *models.Conversation) error {
	return r.db.WithContext(ctx).Save(conv).Error
}
