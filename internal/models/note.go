package models

import (
	"github.com/google/uuid"
)

// ===========================================================================
// Note (Ghi chú nội bộ)
// Nhân viên có thể ghi chú về cuộc hội thoại
// Chỉ nhân viên thấy được, khách hàng không thấy
// ===========================================================================

// Note đại diện cho một ghi chú nội bộ
type Note struct {
	BaseModel

	// ConversationID ID cuộc hội thoại
	ConversationID uuid.UUID `gorm:"type:uuid;not null;index" json:"conversation_id"`

	// UserID ID người tạo ghi chú
	UserID uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`

	// Content nội dung ghi chú
	Content string `gorm:"type:text;not null" json:"content"`

	// IsPinned có ghim ghi chú không
	IsPinned bool `gorm:"default:false" json:"is_pinned"`

	// Relations
	Conversation Conversation `gorm:"foreignKey:ConversationID" json:"conversation,omitempty"`
	User         User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName trả về tên bảng
func (Note) TableName() string {
	return "notes"
}

// Pin ghim ghi chú
func (n *Note) Pin() { n.IsPinned = true }

// Unpin bỏ ghim ghi chú
func (n *Note) Unpin() { n.IsPinned = false }
