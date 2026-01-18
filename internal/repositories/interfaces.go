package repositories

import (
	"context"

	"chatbox-gin/internal/models"

	"github.com/google/uuid"
)

// ===========================================================================
// Workspace Repository Interface
// Quản lý CRUD cho workspaces
// ===========================================================================

// WorkspaceRepository interface cho workspace data access
type WorkspaceRepository interface {
	// FindByID tìm workspace theo ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.Workspace, error)

	// FindBySlug tìm workspace theo slug
	FindBySlug(ctx context.Context, slug string) (*models.Workspace, error)

	// Create tạo workspace mới
	Create(ctx context.Context, workspace *models.Workspace) error

	// Update cập nhật workspace
	Update(ctx context.Context, workspace *models.Workspace) error
}

// ===========================================================================
// User Repository Interface
// Quản lý CRUD cho users
// ===========================================================================

// UserRepository interface cho user data access
type UserRepository interface {
	// FindByID tìm user theo ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)

	// FindByEmail tìm user theo email trong workspace
	FindByEmail(ctx context.Context, workspaceID uuid.UUID, email string) (*models.User, error)

	// FindByWorkspace lấy danh sách users trong workspace
	FindByWorkspace(ctx context.Context, workspaceID uuid.UUID, opts FindOptions) ([]models.User, int64, error)

	// Create tạo user mới
	Create(ctx context.Context, user *models.User) error

	// Update cập nhật user
	Update(ctx context.Context, user *models.User) error
}

// ===========================================================================
// ChannelAccount Repository Interface
// Quản lý CRUD cho channel accounts
// ===========================================================================

// ChannelAccountRepository interface cho channel account data access
type ChannelAccountRepository interface {
	// FindByID tìm channel account theo ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.ChannelAccount, error)

	// FindByChannelID tìm channel account theo channel_id (Page ID, OA ID)
	FindByChannelID(ctx context.Context, channelID string, channelType models.ChannelType) (*models.ChannelAccount, error)

	// FindByWorkspace lấy danh sách channels trong workspace
	FindByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]models.ChannelAccount, error)

	// FindByWorkspaceAndType tìm channel theo workspace và type
	FindByWorkspaceAndType(ctx context.Context, workspaceID uuid.UUID, channelType models.ChannelType) (*models.ChannelAccount, error)

	// Create tạo channel account mới
	Create(ctx context.Context, account *models.ChannelAccount) error

	// Update cập nhật channel account
	Update(ctx context.Context, account *models.ChannelAccount) error
}
