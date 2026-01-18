package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ===========================================================================
// Message (Tin nhắn)
// Đại diện cho một tin nhắn trong cuộc hội thoại
// Có thể từ khách hàng (in), bot (out), hoặc agent (out)
// ===========================================================================

// MessageDirection hướng tin nhắn
type MessageDirection string

const (
	// DirectionIn tin nhắn từ khách hàng đến hệ thống
	DirectionIn MessageDirection = "in"

	// DirectionOut tin nhắn từ hệ thống đến khách hàng
	DirectionOut MessageDirection = "out"
)

// SenderType loại người gửi
type SenderType string

const (
	// SenderCustomer tin nhắn từ khách hàng
	SenderCustomer SenderType = "customer"

	// SenderBot tin nhắn từ bot tự động
	SenderBot SenderType = "bot"

	// SenderAgent tin nhắn từ nhân viên
	SenderAgent SenderType = "agent"
)

// ContentType loại nội dung
type ContentType string

const (
	ContentText       ContentType = "text"
	ContentImage      ContentType = "image"
	ContentFile       ContentType = "file"
	ContentTemplate   ContentType = "template"
	ContentQuickReply ContentType = "quick_reply"
)

// Attachment file đính kèm trong tin nhắn
type Attachment struct {
	Type     string `json:"type"`      // image, file, video, audio
	URL      string `json:"url"`       // URL download
	Name     string `json:"name"`      // Tên file
	Size     int64  `json:"size"`      // Kích thước (bytes)
	MimeType string `json:"mime_type"` // MIME type
}

// Attachments danh sách attachments cho JSONB
type Attachments []Attachment

// Value implement driver.Valuer cho JSONB
func (a Attachments) Value() (driver.Value, error) {
	if a == nil {
		return json.Marshal([]Attachment{})
	}
	return json.Marshal(a)
}

// Scan implement sql.Scanner cho JSONB
func (a *Attachments) Scan(value interface{}) error {
	if value == nil {
		*a = []Attachment{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, a)
}

// QuickReply nút quick reply
type QuickReply struct {
	Title   string `json:"title"`   // Text hiển thị
	Payload string `json:"payload"` // Giá trị gửi lại khi bấm
}

// Button nút bấm trong tin nhắn
type Button struct {
	Type    string `json:"type"`    // postback, web_url, phone_number
	Title   string `json:"title"`   // Text hiển thị
	Payload string `json:"payload"` // Giá trị hoặc URL
	URL     string `json:"url"`
}

// MessageMetadata thông tin bổ sung về tin nhắn
type MessageMetadata struct {
	// QuickReplies danh sách quick reply buttons
	QuickReplies []QuickReply `json:"quick_replies,omitempty"`

	// Buttons danh sách buttons
	Buttons []Button `json:"buttons,omitempty"`

	// MatchedRuleID ID rule đã match (nếu bot trả lời)
	MatchedRuleID *uuid.UUID `json:"matched_rule_id,omitempty"`

	// MatchedKeyword keyword đã match
	MatchedKeyword string `json:"matched_keyword,omitempty"`

	// Confidence độ tin cậy của match
	Confidence float64 `json:"confidence,omitempty"`

	// DeliveredAt thời điểm đã gửi đến channel
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`

	// FailedAt thời điểm gửi thất bại
	FailedAt *time.Time `json:"failed_at,omitempty"`

	// FailReason lý do gửi thất bại
	FailReason string `json:"fail_reason,omitempty"`

	// RetryCount số lần đã retry
	RetryCount int `json:"retry_count,omitempty"`
}

// Value implement driver.Valuer cho JSONB
func (m MessageMetadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implement sql.Scanner cho JSONB
func (m *MessageMetadata) Scan(value interface{}) error {
	if value == nil {
		*m = MessageMetadata{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, m)
}

// Message đại diện cho một tin nhắn
type Message struct {
	BaseModel

	// ConversationID ID cuộc hội thoại
	ConversationID uuid.UUID `gorm:"type:uuid;not null;index" json:"conversation_id"`

	// Direction hướng: in (từ khách) hoặc out (từ hệ thống)
	Direction MessageDirection `gorm:"size:10;not null" json:"direction"`

	// SenderType loại người gửi: customer, bot, agent
	SenderType SenderType `gorm:"size:20;not null" json:"sender_type"`

	// SenderID ID user nếu sender là agent (nullable)
	SenderID *uuid.UUID `gorm:"type:uuid" json:"sender_id,omitempty"`

	// Content nội dung text
	Content *string `gorm:"type:text" json:"content,omitempty"`

	// ContentType loại nội dung
	ContentType ContentType `gorm:"size:50;default:'text'" json:"content_type"`

	// Attachments danh sách file đính kèm
	Attachments Attachments `gorm:"type:jsonb;default:'[]'" json:"attachments"`

	// ChannelMessageID ID tin nhắn trên channel (để dedup)
	ChannelMessageID *string `gorm:"size:255;index" json:"channel_message_id,omitempty"`

	// Metadata thông tin bổ sung
	Metadata MessageMetadata `gorm:"type:jsonb;default:'{}'" json:"metadata"`

	// IsRead tin nhắn đã được đọc chưa
	IsRead bool `gorm:"default:false" json:"is_read"`

	// ReadAt thời điểm đọc
	ReadAt *time.Time `json:"read_at,omitempty"`

	// Relations
	Conversation Conversation `gorm:"foreignKey:ConversationID" json:"conversation,omitempty"`
	Sender       *User        `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
}

// TableName trả về tên bảng
func (Message) TableName() string {
	return "messages"
}

// IsInbound kiểm tra tin nhắn từ khách hàng
func (m *Message) IsInbound() bool { return m.Direction == DirectionIn }

// IsOutbound kiểm tra tin nhắn từ hệ thống
func (m *Message) IsOutbound() bool { return m.Direction == DirectionOut }

// IsFromBot kiểm tra tin nhắn từ bot
func (m *Message) IsFromBot() bool { return m.SenderType == SenderBot }

// IsFromAgent kiểm tra tin nhắn từ agent
func (m *Message) IsFromAgent() bool { return m.SenderType == SenderAgent }

// HasAttachments kiểm tra có file đính kèm không
func (m *Message) HasAttachments() bool { return len(m.Attachments) > 0 }

// MarkAsRead đánh dấu tin nhắn đã đọc
func (m *Message) MarkAsRead() {
	if !m.IsRead {
		m.IsRead = true
		now := time.Now()
		m.ReadAt = &now
	}
}

// GetContentPreview trả về preview nội dung
func (m *Message) GetContentPreview(maxLen int) string {
	if m.Content == nil {
		if m.HasAttachments() {
			return "[Attachment]"
		}
		return ""
	}
	content := *m.Content
	if len(content) > maxLen {
		return content[:maxLen-3] + "..."
	}
	return content
}
