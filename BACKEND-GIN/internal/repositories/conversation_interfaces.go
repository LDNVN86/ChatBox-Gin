package repositories

import (
	"context"

	"chatbox-gin/internal/models"

	"github.com/google/uuid"
)

// ===========================================================================
// Participant Repository Interface
// Quản lý CRUD cho participants (khách hàng)
// ===========================================================================

// ParticipantRepository interface cho participant data access
type ParticipantRepository interface {
	// FindByID tìm participant theo ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.Participant, error)

	// FindByChannelUserID tìm participant theo channel user ID
	// Dùng để identify khách hàng từ channel
	FindByChannelUserID(ctx context.Context, channelAccountID uuid.UUID, channelUserID string) (*models.Participant, error)

	// FindOrCreate tìm hoặc tạo mới participant
	// Trả về participant và bool (true nếu mới tạo)
	FindOrCreate(ctx context.Context, participant *models.Participant) (*models.Participant, bool, error)

	// Update cập nhật participant
	Update(ctx context.Context, participant *models.Participant) error
}

// ===========================================================================
// Conversation Repository Interface
// Quản lý CRUD cho conversations
// ===========================================================================

// ConversationRepository interface cho conversation data access
type ConversationRepository interface {
	// FindByID tìm conversation theo ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.Conversation, error)

	// FindByThreadID tìm conversation theo channel thread ID
	FindByThreadID(ctx context.Context, channelAccountID uuid.UUID, threadID string) (*models.Conversation, error)

	// FindByWorkspace lấy danh sách conversations trong workspace
	FindByWorkspace(ctx context.Context, workspaceID uuid.UUID, opts FindOptions) ([]models.Conversation, int64, error)

	// FindOrCreate tìm hoặc tạo mới conversation
	// Trả về conversation, bool (true nếu mới tạo), error
	FindOrCreate(ctx context.Context, conv *models.Conversation) (*models.Conversation, bool, error)

	// FindOpenByParticipant tìm conversation đang mở của participant
	FindOpenByParticipant(ctx context.Context, participantID uuid.UUID) (*models.Conversation, error)

	// Create tạo conversation mới
	Create(ctx context.Context, conv *models.Conversation) error

	// Update cập nhật conversation
	Update(ctx context.Context, conv *models.Conversation) error
}

// ===========================================================================
// Message Repository Interface
// Quản lý CRUD cho messages
// ===========================================================================

// MessageRepository interface cho message data access
type MessageRepository interface {
	// FindByID tìm message theo ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.Message, error)

	// FindByConversation lấy danh sách messages trong conversation
	FindByConversation(ctx context.Context, conversationID uuid.UUID, opts FindOptions) ([]models.Message, int64, error)

	// FindByChannelMessageID tìm message theo channel message ID
	// Dùng để check duplicate
	FindByChannelMessageID(ctx context.Context, channelMessageID string) (*models.Message, error)

	// Create tạo message mới
	Create(ctx context.Context, msg *models.Message) error

	// Update cập nhật message
	Update(ctx context.Context, msg *models.Message) error

	// MarkAsRead đánh dấu message đã đọc
	MarkAsRead(ctx context.Context, id uuid.UUID) error
}
