package repositories

import (
	"context"

	"chatbox-gin/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ===========================================================================
// User Repository Implementation
// Database operations for User model
// (interface defined in interfaces.go)
// ===========================================================================

// userRepo implementation
type userRepo struct {
	db *gorm.DB
}

// NewUserRepository tạo user repository mới
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepo{db: db}
}

// FindByID tìm user theo ID
func (r *userRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).
		Preload("Workspace").
		First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail tìm user theo email trong workspace (cho login)
func (r *userRepo) FindByEmail(ctx context.Context, workspaceID uuid.UUID, email string) (*models.User, error) {
	var user models.User
	query := r.db.WithContext(ctx).Preload("Workspace").Where("is_active = ?", true)

	// Nếu workspaceID không phải nil UUID, filter theo workspace
	if workspaceID != uuid.Nil {
		query = query.Where("workspace_id = ?", workspaceID)
	}

	if err := query.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByWorkspace lấy danh sách users trong workspace
func (r *userRepo) FindByWorkspace(ctx context.Context, workspaceID uuid.UUID, opts FindOptions) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	query := r.db.WithContext(ctx).
		Where("workspace_id = ? AND is_active = ?", workspaceID, true)

	// Count total
	if err := query.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit).Offset(opts.Offset)
	}

	if err := query.Order("created_at ASC").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// Create tạo user mới
func (r *userRepo) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Update cập nhật user
func (r *userRepo) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}
