package models

import (
	"time"

	"github.com/google/uuid"
)

// ===========================================================================
// Tag (Nhãn)
// Dùng để phân loại và tổ chức các cuộc hội thoại
// ===========================================================================

// Tag đại diện cho một nhãn/label
type Tag struct {
	BaseModel

	// WorkspaceID ID workspace sở hữu tag
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index" json:"workspace_id"`

	// Name tên tag (VD: "VIP", "Urgent")
	Name string `gorm:"size:100;not null" json:"name"`

	// Color màu hiển thị (hex format, VD: "#f59e0b")
	Color string `gorm:"size:20;default:'#6366f1'" json:"color"`

	// Description mô tả tag
	Description *string `gorm:"type:text" json:"description,omitempty"`

	// Relations
	Workspace     Workspace      `gorm:"foreignKey:WorkspaceID" json:"workspace,omitempty"`
	Conversations []Conversation `gorm:"many2many:conversation_tags" json:"conversations,omitempty"`
}

// TableName trả về tên bảng
func (Tag) TableName() string {
	return "tags"
}

// ===========================================================================
// ConversationTag (Bảng trung gian)
// Liên kết nhiều-nhiều giữa Conversation và Tag
// ===========================================================================

// ConversationTag bảng junction cho quan hệ conversation-tag
type ConversationTag struct {
	// ID primary key
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`

	// ConversationID ID cuộc hội thoại
	ConversationID uuid.UUID `gorm:"type:uuid;not null;index" json:"conversation_id"`

	// TagID ID tag
	TagID uuid.UUID `gorm:"type:uuid;not null;index" json:"tag_id"`

	// CreatedBy người gán tag (nullable nếu system tự gán)
	CreatedBy *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`

	// CreatedAt thời điểm gán tag
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`

	// Relations
	Conversation Conversation `gorm:"foreignKey:ConversationID" json:"conversation,omitempty"`
	Tag          Tag          `gorm:"foreignKey:TagID" json:"tag,omitempty"`
	Creator      *User        `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

// TableName trả về tên bảng
func (ConversationTag) TableName() string {
	return "conversation_tags"
}
