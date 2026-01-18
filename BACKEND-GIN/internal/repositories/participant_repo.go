package repositories

import (
	"context"

	"chatbox-gin/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ===========================================================================
// Participant Repository GORM Implementation
// ===========================================================================

// participantRepo triển khai ParticipantRepository với GORM
type participantRepo struct {
	db *gorm.DB
}

// NewParticipantRepository tạo instance mới của ParticipantRepository
func NewParticipantRepository(db *gorm.DB) ParticipantRepository {
	return &participantRepo{db: db}
}

// FindByID tìm participant theo ID
func (r *participantRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Participant, error) {
	var participant models.Participant
	if err := r.db.WithContext(ctx).First(&participant, id).Error; err != nil {
		return nil, err
	}
	return &participant, nil
}

// FindByChannelUserID tìm participant theo channel user ID
func (r *participantRepo) FindByChannelUserID(ctx context.Context, channelAccountID uuid.UUID, channelUserID string) (*models.Participant, error) {
	var participant models.Participant
	err := r.db.WithContext(ctx).
		Where("channel_account_id = ? AND channel_user_id = ?", channelAccountID, channelUserID).
		First(&participant).Error
	if err != nil {
		return nil, err
	}
	return &participant, nil
}

// FindOrCreate tìm hoặc tạo mới participant
func (r *participantRepo) FindOrCreate(ctx context.Context, participant *models.Participant) (*models.Participant, bool, error) {
	// Thử tìm trước
	existing, err := r.FindByChannelUserID(ctx, participant.ChannelAccountID, participant.ChannelUserID)
	if err == nil {
		return existing, false, nil
	}

	// Không tìm thấy, tạo mới
	if err := r.db.WithContext(ctx).Create(participant).Error; err != nil {
		return nil, false, err
	}

	return participant, true, nil
}

// Update cập nhật participant
func (r *participantRepo) Update(ctx context.Context, participant *models.Participant) error {
	return r.db.WithContext(ctx).Save(participant).Error
}
