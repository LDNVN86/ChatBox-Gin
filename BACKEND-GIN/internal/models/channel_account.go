package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ===========================================================================
// ChannelAccount (Tài khoản kênh chat)
// Đại diện cho một kết nối với Facebook Page, Zalo OA, hoặc Mock channel
// Mỗi workspace có thể có nhiều channel accounts
// ===========================================================================

// ChannelType loại kênh chat
type ChannelType string

const (
	// ChannelFacebook Facebook Messenger
	ChannelFacebook ChannelType = "facebook"

	// ChannelZalo Zalo OA
	ChannelZalo ChannelType = "zalo"

	// ChannelWeb Web widget
	ChannelWeb ChannelType = "web"

	// ChannelMock Mock channel cho testing
	ChannelMock ChannelType = "mock"
)

// ChannelCredentials thông tin xác thực cho từng loại channel
// QUAN TRỌNG: Không bao giờ expose trong JSON response
type ChannelCredentials struct {
	// Facebook credentials
	PageAccessToken string `json:"page_access_token,omitempty"`
	AppSecret       string `json:"app_secret,omitempty"`

	// Zalo credentials
	OAAccessToken  string `json:"oa_access_token,omitempty"`
	OARefreshToken string `json:"oa_refresh_token,omitempty"`
	OASecretKey    string `json:"oa_secret_key,omitempty"`
	AppID          string `json:"app_id,omitempty"`

	// Common
	WebhookSecret string `json:"webhook_secret,omitempty"`
}

// Value implement driver.Valuer cho JSONB
func (c ChannelCredentials) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implement sql.Scanner cho JSONB
func (c *ChannelCredentials) Scan(value interface{}) error {
	if value == nil {
		*c = ChannelCredentials{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, c)
}

// ChannelSettings cấu hình riêng cho từng channel
type ChannelSettings struct {
	// AutoReply có tự động trả lời không
	AutoReply bool `json:"auto_reply"`

	// WelcomeMsg tin nhắn chào mừng
	WelcomeMsg string `json:"welcome_message,omitempty"`

	// OfflineMsg tin nhắn khi ngoài giờ làm việc
	OfflineMsg string `json:"offline_message,omitempty"`

	// HandoffMsg tin nhắn khi chuyển cho nhân viên
	HandoffMsg string `json:"handoff_message,omitempty"`

	// BotEnabled có bật bot cho channel này không
	BotEnabled bool `json:"bot_enabled"`

	// MaxRetries số lần retry khi gửi thất bại
	MaxRetries int `json:"max_retries"`

	// RetryDelayMs delay giữa các lần retry (milliseconds)
	RetryDelayMs int `json:"retry_delay_ms"`
}

// Value implement driver.Valuer cho JSONB
func (s ChannelSettings) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan implement sql.Scanner cho JSONB
func (s *ChannelSettings) Scan(value interface{}) error {
	if value == nil {
		*s = ChannelSettings{AutoReply: true, BotEnabled: true, MaxRetries: 3, RetryDelayMs: 1000}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, s)
}

// ChannelAccount đại diện cho một kết nối channel
type ChannelAccount struct {
	BaseModel

	// WorkspaceID ID workspace sở hữu channel
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index" json:"workspace_id"`

	// ChannelType loại channel (facebook/zalo/web/mock)
	ChannelType ChannelType `gorm:"size:50;not null" json:"channel_type"`

	// Name tên hiển thị (VD: "FB Page Chính")
	Name string `gorm:"size:255;not null" json:"name"`

	// ChannelID ID trên platform (FB Page ID, Zalo OA ID)
	ChannelID *string `gorm:"size:255" json:"channel_id,omitempty"`

	// Credentials thông tin xác thực (KHÔNG expose trong JSON)
	Credentials ChannelCredentials `gorm:"type:jsonb;default:'{}'" json:"-"`

	// Settings cấu hình channel
	Settings ChannelSettings `gorm:"type:jsonb;default:'{}'" json:"settings"`

	// IsActive channel có đang active không
	IsActive bool `gorm:"default:true" json:"is_active"`

	// ConnectedAt thời điểm kết nối
	ConnectedAt *time.Time `json:"connected_at,omitempty"`

	// Relations
	Workspace     Workspace      `gorm:"foreignKey:WorkspaceID" json:"workspace,omitempty"`
	Conversations []Conversation `gorm:"foreignKey:ChannelAccountID" json:"conversations,omitempty"`
	Participants  []Participant  `gorm:"foreignKey:ChannelAccountID" json:"participants,omitempty"`
}

// TableName trả về tên bảng
func (ChannelAccount) TableName() string {
	return "channel_accounts"
}

// IsFacebook kiểm tra có phải Facebook channel không
func (c *ChannelAccount) IsFacebook() bool { return c.ChannelType == ChannelFacebook }

// IsZalo kiểm tra có phải Zalo channel không
func (c *ChannelAccount) IsZalo() bool { return c.ChannelType == ChannelZalo }

// IsMock kiểm tra có phải Mock channel không
func (c *ChannelAccount) IsMock() bool { return c.ChannelType == ChannelMock }

// SetConnected đánh dấu channel đã kết nối
func (c *ChannelAccount) SetConnected() {
	now := time.Now()
	c.ConnectedAt = &now
	c.IsActive = true
}
