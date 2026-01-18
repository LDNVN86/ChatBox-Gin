package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ===========================================================================
// Rule (Quy tắc bot)
// Định nghĩa cách bot tự động trả lời tin nhắn
// Có nhiều loại trigger: keyword, time_window, fallback
// ===========================================================================

// TriggerType loại trigger kích hoạt rule
type TriggerType string

const (
	// TriggerKeyword kích hoạt khi match keywords
	TriggerKeyword TriggerType = "keyword"

	// TriggerTimeWindow kích hoạt trong khung giờ cụ thể
	TriggerTimeWindow TriggerType = "time_window"

	// TriggerFallback kích hoạt khi không match rule nào
	TriggerFallback TriggerType = "fallback"

	// TriggerIntent kích hoạt theo AI intent (future)
	TriggerIntent TriggerType = "intent"
)

// ResponseType loại response
type ResponseType string

const (
	// ResponseText trả lời bằng text
	ResponseText ResponseType = "text"

	// ResponseTemplate trả lời bằng template
	ResponseTemplate ResponseType = "template"

	// ResponseHandoff chuyển cho agent
	ResponseHandoff ResponseType = "handoff"
)

// TriggerConfig cấu hình trigger
type TriggerConfig struct {
	// Keywords danh sách từ khóa
	Keywords []string `json:"keywords,omitempty"`

	// MatchType cách match: "exact" hoặc "contains"
	MatchType string `json:"match_type,omitempty"`

	// StartTime giờ bắt đầu (format "HH:mm")
	StartTime string `json:"start_time,omitempty"`

	// EndTime giờ kết thúc
	EndTime string `json:"end_time,omitempty"`

	// Timezone múi giờ
	Timezone string `json:"timezone,omitempty"`

	// Days các ngày áp dụng (0=CN, 1=T2,...)
	Days []int `json:"days,omitempty"`

	// Intent tên intent (cho AI)
	Intent string `json:"intent,omitempty"`

	// Confidence ngưỡng confidence
	Confidence float64 `json:"confidence,omitempty"`
}

// Value implement driver.Valuer cho JSONB
func (c TriggerConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implement sql.Scanner cho JSONB
func (c *TriggerConfig) Scan(value interface{}) error {
	if value == nil {
		*c = TriggerConfig{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, c)
}

// ResponseConfig cấu hình response
type ResponseConfig struct {
	// Text nội dung text trả lời
	Text string `json:"text,omitempty"`

	// TemplateID ID template
	TemplateID string `json:"template_id,omitempty"`

	// Parameters tham số cho template
	Parameters map[string]interface{} `json:"parameters,omitempty"`

	// Message tin nhắn kèm theo (cho handoff)
	Message string `json:"message,omitempty"`

	// AssignTo ID agent để assign (cho handoff)
	AssignTo *uuid.UUID `json:"assign_to,omitempty"`

	// Tags các tag để gán
	Tags []string `json:"tags,omitempty"`

	// Priority priority để set
	Priority string `json:"priority,omitempty"`
}

// Value implement driver.Valuer cho JSONB
func (c ResponseConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implement sql.Scanner cho JSONB
func (c *ResponseConfig) Scan(value interface{}) error {
	if value == nil {
		*c = ResponseConfig{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, c)
}

// Rule đại diện cho một quy tắc bot
type Rule struct {
	BaseModel

	// WorkspaceID ID workspace
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index" json:"workspace_id"`

	// Name tên rule
	Name string `gorm:"size:255;not null" json:"name"`

	// Description mô tả rule
	Description *string `gorm:"type:text" json:"description,omitempty"`

	// TriggerType loại trigger
	TriggerType TriggerType `gorm:"size:50;not null" json:"trigger_type"`

	// TriggerConfig cấu hình trigger
	TriggerConfig TriggerConfig `gorm:"type:jsonb;not null;default:'{}'" json:"trigger_config"`

	// ResponseType loại response
	ResponseType ResponseType `gorm:"size:50;not null;default:'text'" json:"response_type"`

	// ResponseConfig cấu hình response
	ResponseConfig ResponseConfig `gorm:"type:jsonb;not null;default:'{}'" json:"response_config"`

	// Priority độ ưu tiên (số lớn = ưu tiên cao hơn)
	Priority int `gorm:"not null;default:0" json:"priority"`

	// IsActive rule có đang active không
	IsActive bool `gorm:"default:true" json:"is_active"`

	// HitCount số lần rule được kích hoạt
	HitCount int64 `gorm:"default:0" json:"hit_count"`

	// LastTriggeredAt lần cuối được kích hoạt
	LastTriggeredAt *time.Time `json:"last_triggered_at,omitempty"`

	// Relations
	Workspace Workspace `gorm:"foreignKey:WorkspaceID" json:"workspace,omitempty"`
}

// TableName trả về tên bảng
func (Rule) TableName() string {
	return "rules"
}

// IsKeywordTrigger kiểm tra rule có trigger theo keyword không
func (r *Rule) IsKeywordTrigger() bool { return r.TriggerType == TriggerKeyword }

// IsTimeWindowTrigger kiểm tra rule có trigger theo time window không
func (r *Rule) IsTimeWindowTrigger() bool { return r.TriggerType == TriggerTimeWindow }

// IsFallbackTrigger kiểm tra rule có phải fallback không
func (r *Rule) IsFallbackTrigger() bool { return r.TriggerType == TriggerFallback }

// IsHandoffResponse kiểm tra rule có chuyển cho agent không
func (r *Rule) IsHandoffResponse() bool { return r.ResponseType == ResponseHandoff }

// MatchesKeyword kiểm tra nội dung có match keywords không
func (r *Rule) MatchesKeyword(content string) bool {
	if !r.IsKeywordTrigger() {
		return false
	}

	contentLower := strings.ToLower(content)
	matchType := r.TriggerConfig.MatchType
	if matchType == "" {
		matchType = "contains" // Mặc định là contains
	}

	for _, keyword := range r.TriggerConfig.Keywords {
		keywordLower := strings.ToLower(keyword)
		switch matchType {
		case "exact":
			if contentLower == keywordLower {
				return true
			}
		case "contains":
			if strings.Contains(contentLower, keywordLower) {
				return true
			}
		}
	}
	return false
}

// MatchesTimeWindow kiểm tra thời gian có trong time window không
func (r *Rule) MatchesTimeWindow(now time.Time) bool {
	if !r.IsTimeWindowTrigger() {
		return false
	}

	tc := r.TriggerConfig

	// Load timezone
	loc, err := time.LoadLocation(tc.Timezone)
	if err != nil {
		loc = time.UTC
	}
	localNow := now.In(loc)

	// Kiểm tra ngày
	if len(tc.Days) > 0 {
		weekday := int(localNow.Weekday())
		dayMatch := false
		for _, day := range tc.Days {
			if day == weekday {
				dayMatch = true
				break
			}
		}
		if !dayMatch {
			return false
		}
	}

	// Kiểm tra giờ
	currentTime := localNow.Format("15:04")

	// Xử lý trường hợp qua đêm (VD: 22:00 - 06:00)
	if tc.StartTime > tc.EndTime {
		return currentTime >= tc.StartTime || currentTime <= tc.EndTime
	}

	return currentTime >= tc.StartTime && currentTime <= tc.EndTime
}

// IncrementHitCount tăng số lần kích hoạt
func (r *Rule) IncrementHitCount() {
	r.HitCount++
	now := time.Now()
	r.LastTriggeredAt = &now
}

// GetResponseText trả về text response
func (r *Rule) GetResponseText() string {
	return r.ResponseConfig.Text
}
