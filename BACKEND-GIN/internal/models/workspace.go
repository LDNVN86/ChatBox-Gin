package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// ===========================================================================
// Workspace (Không gian làm việc)
// Đại diện cho một business/shop trong hệ thống multi-tenant
// Tất cả entities khác đều thuộc về một workspace
// ===========================================================================

// WorkspaceSettings cấu hình cho workspace
type WorkspaceSettings struct {
	// Timezone múi giờ (VD: "Asia/Ho_Chi_Minh")
	Timezone string `json:"timezone"`

	// WorkingHours giờ làm việc
	WorkingHours *WorkingHours `json:"working_hours,omitempty"`

	// BotEnabled có bật bot tự động không
	BotEnabled bool `json:"bot_enabled"`

	// Language ngôn ngữ mặc định (vi, en)
	Language string `json:"language"`
}

// WorkingHours cấu hình giờ làm việc
type WorkingHours struct {
	// Start giờ bắt đầu (format "HH:mm", VD: "09:00")
	Start string `json:"start"`

	// End giờ kết thúc (format "HH:mm", VD: "18:00")
	End string `json:"end"`

	// Days các ngày làm việc (0=Chủ nhật, 1=Thứ 2, ..., 6=Thứ 7)
	Days []int `json:"days"`
}

// Value implement driver.Valuer để lưu JSONB vào PostgreSQL
func (s WorkspaceSettings) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan implement sql.Scanner để đọc JSONB từ PostgreSQL
func (s *WorkspaceSettings) Scan(value interface{}) error {
	if value == nil {
		*s = WorkspaceSettings{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, s)
}

// Workspace đại diện cho một không gian làm việc (business/shop)
type Workspace struct {
	BaseModel

	// Name tên workspace (VD: "Game Shop ABC")
	Name string `gorm:"size:255;not null" json:"name"`

	// Slug URL-friendly identifier (VD: "game-shop-abc")
	Slug string `gorm:"size:100;uniqueIndex;not null" json:"slug"`

	// Settings cấu hình workspace (JSONB)
	Settings WorkspaceSettings `gorm:"type:jsonb;default:'{}'" json:"settings"`

	// IsActive workspace có đang hoạt động không
	IsActive bool `gorm:"default:true" json:"is_active"`

	// Relations - Các quan hệ với bảng khác
	Users           []User           `gorm:"foreignKey:WorkspaceID" json:"users,omitempty"`
	ChannelAccounts []ChannelAccount `gorm:"foreignKey:WorkspaceID" json:"channel_accounts,omitempty"`
	Conversations   []Conversation   `gorm:"foreignKey:WorkspaceID" json:"conversations,omitempty"`
	Rules           []Rule           `gorm:"foreignKey:WorkspaceID" json:"rules,omitempty"`
	Tags            []Tag            `gorm:"foreignKey:WorkspaceID" json:"tags,omitempty"`
}

// TableName trả về tên bảng trong database
func (Workspace) TableName() string {
	return "workspaces"
}

// IsWithinWorkingHours kiểm tra thời điểm hiện tại có trong giờ làm việc không
// Trả về true nếu không có cấu hình giờ làm việc (luôn available)
func (w *Workspace) IsWithinWorkingHours(now time.Time) bool {
	if w.Settings.WorkingHours == nil {
		return true // Không có cấu hình = luôn available
	}

	wh := w.Settings.WorkingHours

	// Kiểm tra có phải ngày làm việc không
	weekday := int(now.Weekday())
	isWorkingDay := false
	for _, day := range wh.Days {
		if day == weekday {
			isWorkingDay = true
			break
		}
	}
	if !isWorkingDay {
		return false
	}

	// Kiểm tra giờ
	currentTime := now.Format("15:04")
	return currentTime >= wh.Start && currentTime <= wh.End
}
