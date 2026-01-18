package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ===========================================================================
// WebhookEvent (Sự kiện Webhook)
// Lưu trữ các webhook events để tracking và retry
// Đảm bảo idempotency (không xử lý trùng lặp)
// ===========================================================================

// WebhookEventStatus trạng thái xử lý webhook
type WebhookEventStatus string

const (
	// WebhookStatusPending đang chờ xử lý
	WebhookStatusPending WebhookEventStatus = "pending"

	// WebhookStatusProcessing đang xử lý
	WebhookStatusProcessing WebhookEventStatus = "processing"

	// WebhookStatusProcessed đã xử lý thành công
	WebhookStatusProcessed WebhookEventStatus = "processed"

	// WebhookStatusFailed xử lý thất bại
	WebhookStatusFailed WebhookEventStatus = "failed"
)

// WebhookPayload wrap raw payload từ webhook
type WebhookPayload map[string]interface{}

// Value implement driver.Valuer cho JSONB
func (p WebhookPayload) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan implement sql.Scanner cho JSONB
func (p *WebhookPayload) Scan(value interface{}) error {
	if value == nil {
		*p = WebhookPayload{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, p)
}

// WebhookEvent lưu trữ webhook event
type WebhookEvent struct {
	// ID primary key
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`

	// ChannelType loại channel (facebook/zalo)
	ChannelType ChannelType `gorm:"size:50;not null" json:"channel_type"`

	// EventID ID event từ channel (để dedup)
	EventID string `gorm:"size:255;not null;uniqueIndex" json:"event_id"`

	// EventType loại event (message, postback, etc.)
	EventType *string `gorm:"size:100" json:"event_type,omitempty"`

	// Payload raw payload từ webhook
	Payload WebhookPayload `gorm:"type:jsonb;not null;default:'{}'" json:"payload"`

	// Status trạng thái xử lý
	Status WebhookEventStatus `gorm:"size:50;not null;default:'pending'" json:"status"`

	// RetryCount số lần đã retry
	RetryCount int `gorm:"default:0" json:"retry_count"`

	// ErrorMessage lỗi nếu có
	ErrorMessage *string `gorm:"type:text" json:"error_message,omitempty"`

	// ProcessedAt thời điểm xử lý thành công
	ProcessedAt *time.Time `json:"processed_at,omitempty"`

	// Timestamps
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName trả về tên bảng
func (WebhookEvent) TableName() string {
	return "webhook_events"
}

// MarkProcessing đánh dấu đang xử lý
func (e *WebhookEvent) MarkProcessing() {
	e.Status = WebhookStatusProcessing
}

// MarkProcessed đánh dấu đã xử lý thành công
func (e *WebhookEvent) MarkProcessed() {
	e.Status = WebhookStatusProcessed
	now := time.Now()
	e.ProcessedAt = &now
}

// MarkFailed đánh dấu xử lý thất bại
func (e *WebhookEvent) MarkFailed(err error) {
	e.Status = WebhookStatusFailed
	errMsg := err.Error()
	e.ErrorMessage = &errMsg
	e.RetryCount++
}

// CanRetry kiểm tra có thể retry không
func (e *WebhookEvent) CanRetry(maxRetries int) bool {
	return e.Status == WebhookStatusFailed && e.RetryCount < maxRetries
}

// ResetForRetry reset để retry lại
func (e *WebhookEvent) ResetForRetry() {
	e.Status = WebhookStatusPending
	e.ErrorMessage = nil
}
