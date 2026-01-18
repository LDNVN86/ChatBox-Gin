package repositories

import (
	"context"

	"chatbox-gin/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ===========================================================================
// ChannelAccountRepository Implementation
// ===========================================================================

// channelAccountRepoImpl implementation của ChannelAccountRepository
type channelAccountRepoImpl struct {
	db *gorm.DB
}

// NewChannelAccountRepository tạo repository mới
func NewChannelAccountRepository(db *gorm.DB) ChannelAccountRepository {
	return &channelAccountRepoImpl{db: db}
}

// FindByID tìm channel account theo ID
func (r *channelAccountRepoImpl) FindByID(ctx context.Context, id uuid.UUID) (*models.ChannelAccount, error) {
	var account models.ChannelAccount
	err := r.db.WithContext(ctx).First(&account, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// FindByChannelID tìm channel account theo channel_id (Page ID, OA ID, etc.)
func (r *channelAccountRepoImpl) FindByChannelID(ctx context.Context, channelID string, channelType models.ChannelType) (*models.ChannelAccount, error) {
	var account models.ChannelAccount
	err := r.db.WithContext(ctx).
		Where("channel_id = ? AND channel_type = ?", channelID, channelType).
		First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// FindByWorkspace lấy danh sách channels trong workspace
func (r *channelAccountRepoImpl) FindByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]models.ChannelAccount, error) {
	var accounts []models.ChannelAccount
	err := r.db.WithContext(ctx).
		Where("workspace_id = ?", workspaceID).
		Order("created_at DESC").
		Find(&accounts).Error
	return accounts, err
}

// FindByWorkspaceAndType tìm channel theo workspace và type
func (r *channelAccountRepoImpl) FindByWorkspaceAndType(ctx context.Context, workspaceID uuid.UUID, channelType models.ChannelType) (*models.ChannelAccount, error) {
	var account models.ChannelAccount
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND channel_type = ?", workspaceID, channelType).
		First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// Create tạo channel account mới
func (r *channelAccountRepoImpl) Create(ctx context.Context, account *models.ChannelAccount) error {
	return r.db.WithContext(ctx).Create(account).Error
}

// Update cập nhật channel account
func (r *channelAccountRepoImpl) Update(ctx context.Context, account *models.ChannelAccount) error {
	return r.db.WithContext(ctx).Save(account).Error
}
