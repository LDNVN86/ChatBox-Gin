package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ===========================================================================
// Conversation (Cuộc hội thoại)
// Đại diện cho một phiên chat giữa khách hàng và hệ thống
// Mỗi conversation thuộc về một participant trên một channel
// ===========================================================================

// ConversationStatus trạng thái cuộc hội thoại
type ConversationStatus string

const (
	// StatusOpen đang mở, bot có thể tự động trả lời
	StatusOpen ConversationStatus = "open"

	// StatusPending đã được assign cho agent
	StatusPending ConversationStatus = "pending"

	// StatusClosed đã đóng/hoàn thành
	StatusClosed ConversationStatus = "closed"

	// StatusBotPaused bot tạm dừng, chờ agent xử lý
	StatusBotPaused ConversationStatus = "bot_paused"
)

// Priority mức độ ưu tiên
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

// ConversationMetadata thông tin bổ sung về cuộc hội thoại
type ConversationMetadata struct {
	// Source nguồn (VD: "facebook_ad")
	Source string `json:"source,omitempty"`

	// BotHandoffReason lý do chuyển cho agent
	BotHandoffReason string `json:"bot_handoff_reason,omitempty"`

	// ClosedReason lý do đóng hội thoại
	ClosedReason string `json:"closed_reason,omitempty"`

	// SLABreached đã vi phạm SLA chưa
	SLABreached bool `json:"sla_breached,omitempty"`
}

// Value implement driver.Valuer cho JSONB
func (m ConversationMetadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implement sql.Scanner cho JSONB
func (m *ConversationMetadata) Scan(value interface{}) error {
	if value == nil {
		*m = ConversationMetadata{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, m)
}

// Conversation đại diện cho một cuộc hội thoại
type Conversation struct {
	BaseModel

	// WorkspaceID ID workspace
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index" json:"workspace_id"`

	// ChannelAccountID ID channel mà conversation diễn ra
	ChannelAccountID uuid.UUID `gorm:"type:uuid;not null;index" json:"channel_account_id"`

	// ParticipantID ID khách hàng
	ParticipantID uuid.UUID `gorm:"type:uuid;not null;index" json:"participant_id"`

	// ChannelThreadID ID thread trên channel (nếu có)
	ChannelThreadID *string `gorm:"size:255;index" json:"channel_thread_id,omitempty"`

	// Status trạng thái: open, pending, closed, bot_paused
	Status ConversationStatus `gorm:"size:50;not null;default:'open';index" json:"status"`

	// AssignedTo ID agent được assign (nullable)
	AssignedTo *uuid.UUID `gorm:"type:uuid;index" json:"assigned_to,omitempty"`

	// Priority mức độ ưu tiên
	Priority Priority `gorm:"size:20;default:'normal'" json:"priority"`

	// Subject tiêu đề/chủ đề (tùy chọn)
	Subject *string `gorm:"size:500" json:"subject,omitempty"`

	// LastMessageAt thời điểm tin nhắn cuối cùng
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`

	// LastMessagePreview preview tin nhắn cuối (max 500 ký tự)
	LastMessagePreview *string `gorm:"size:500" json:"last_message_preview,omitempty"`

	// FirstResponseAt thời điểm agent trả lời lần đầu
	FirstResponseAt *time.Time `json:"first_response_at,omitempty"`

	// ResolvedAt thời điểm đóng hội thoại
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`

	// Metadata thông tin bổ sung
	Metadata ConversationMetadata `gorm:"type:jsonb;default:'{}'" json:"metadata"`

	// Relations
	Workspace      Workspace      `gorm:"foreignKey:WorkspaceID" json:"workspace,omitempty"`
	ChannelAccount ChannelAccount `gorm:"foreignKey:ChannelAccountID" json:"channel_account,omitempty"`
	Participant    Participant    `gorm:"foreignKey:ParticipantID" json:"participant,omitempty"`
	AssignedUser   *User          `gorm:"foreignKey:AssignedTo" json:"assigned_user,omitempty"`
	Messages       []Message      `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
	Tags           []Tag          `gorm:"many2many:conversation_tags" json:"tags,omitempty"`
	Notes          []Note         `gorm:"foreignKey:ConversationID" json:"notes,omitempty"`
}

// TableName trả về tên bảng
func (Conversation) TableName() string {
	return "conversations"
}

// IsOpen kiểm tra hội thoại đang mở
func (c *Conversation) IsOpen() bool { return c.Status == StatusOpen }

// IsClosed kiểm tra hội thoại đã đóng
func (c *Conversation) IsClosed() bool { return c.Status == StatusClosed }

// IsBotPaused kiểm tra bot có đang tạm dừng không
func (c *Conversation) IsBotPaused() bool { return c.Status == StatusBotPaused }

// IsAssigned kiểm tra đã được assign cho agent chưa
func (c *Conversation) IsAssigned() bool { return c.AssignedTo != nil }

// Assign gán hội thoại cho một agent
func (c *Conversation) Assign(userID uuid.UUID) {
	c.AssignedTo = &userID
	if c.Status == StatusOpen {
		c.Status = StatusPending
	}
}

// Unassign bỏ gán agent
func (c *Conversation) Unassign() {
	c.AssignedTo = nil
}

// Close đóng hội thoại với lý do
func (c *Conversation) Close(reason string) {
	c.Status = StatusClosed
	now := time.Now()
	c.ResolvedAt = &now
	c.Metadata.ClosedReason = reason
}

// Reopen mở lại hội thoại đã đóng
func (c *Conversation) Reopen() {
	c.Status = StatusOpen
	c.ResolvedAt = nil
	c.Metadata.ClosedReason = ""
}

// PauseBot tạm dừng bot, chờ agent xử lý
func (c *Conversation) PauseBot(reason string) {
	c.Status = StatusBotPaused
	c.Metadata.BotHandoffReason = reason
}

// ResumeBot tiếp tục bot
func (c *Conversation) ResumeBot() {
	c.Status = StatusOpen
	c.Metadata.BotHandoffReason = ""
}

// UpdateLastMessage cập nhật thông tin tin nhắn cuối
func (c *Conversation) UpdateLastMessage(content string, at time.Time) {
	c.LastMessageAt = &at
	if len(content) > 500 {
		preview := content[:497] + "..."
		c.LastMessagePreview = &preview
	} else {
		c.LastMessagePreview = &content
	}
}

// SetFirstResponse đánh dấu thời điểm trả lời đầu tiên
func (c *Conversation) SetFirstResponse(at time.Time) {
	if c.FirstResponseAt == nil {
		c.FirstResponseAt = &at
	}
}
