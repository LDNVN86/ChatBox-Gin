package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ===========================================================================
// Participant (Khách hàng)
// Đại diện cho một khách hàng chat với hệ thống
// Mỗi participant được identify bằng ChannelUserID từ channel tương ứng
// ===========================================================================

// ParticipantMetadata thông tin bổ sung về khách hàng
type ParticipantMetadata struct {
	// Source nguồn khách hàng (VD: "facebook_ad", "organic")
	Source string `json:"source,omitempty"`

	// FirstMessage tin nhắn đầu tiên
	FirstMessage string `json:"first_message,omitempty"`

	// Tags các tag được gán
	Tags []string `json:"tags,omitempty"`

	// CustomFields các trường tùy chỉnh
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}

// Value implement driver.Valuer cho JSONB
func (m ParticipantMetadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implement sql.Scanner cho JSONB
func (m *ParticipantMetadata) Scan(value interface{}) error {
	if value == nil {
		*m = ParticipantMetadata{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, m)
}

// Participant đại diện cho khách hàng chat
type Participant struct {
	BaseModel

	// WorkspaceID ID workspace
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index" json:"workspace_id"`

	// ChannelAccountID ID channel mà khách hàng đến từ
	ChannelAccountID uuid.UUID `gorm:"type:uuid;not null;index" json:"channel_account_id"`

	// ChannelUserID ID của khách hàng trên channel (FB PSID, Zalo UID)
	ChannelUserID string `gorm:"size:255;not null;index" json:"channel_user_id"`

	// Name tên khách hàng (lấy từ FB/Zalo profile)
	Name *string `gorm:"size:255" json:"name,omitempty"`

	// AvatarURL URL avatar
	AvatarURL *string `gorm:"size:500" json:"avatar_url,omitempty"`

	// Email email (nếu khách cung cấp)
	Email *string `gorm:"size:255" json:"email,omitempty"`

	// Phone số điện thoại (nếu khách cung cấp)
	Phone *string `gorm:"size:50" json:"phone,omitempty"`

	// Metadata thông tin bổ sung
	Metadata ParticipantMetadata `gorm:"type:jsonb;default:'{}'" json:"metadata"`

	// FirstSeenAt lần đầu tiên liên hệ
	FirstSeenAt time.Time `gorm:"not null;default:now()" json:"first_seen_at"`

	// LastSeenAt lần cuối cùng liên hệ
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`

	// Relations
	Workspace      Workspace      `gorm:"foreignKey:WorkspaceID" json:"workspace,omitempty"`
	ChannelAccount ChannelAccount `gorm:"foreignKey:ChannelAccountID" json:"channel_account,omitempty"`
	Conversations  []Conversation `gorm:"foreignKey:ParticipantID" json:"conversations,omitempty"`
}

// TableName trả về tên bảng
func (Participant) TableName() string {
	return "participants"
}

// UpdateLastSeen cập nhật thời gian hoạt động gần nhất
func (p *Participant) UpdateLastSeen() {
	now := time.Now()
	p.LastSeenAt = &now
}

// GetDisplayName trả về tên hiển thị
// Nếu không có tên thì trả về "Unknown"
func (p *Participant) GetDisplayName() string {
	if p.Name != nil {
		return *p.Name
	}
	return "Unknown"
}

// HasContactInfo kiểm tra khách có thông tin liên hệ không
func (p *Participant) HasContactInfo() bool {
	return (p.Email != nil && *p.Email != "") || (p.Phone != nil && *p.Phone != "")
}
