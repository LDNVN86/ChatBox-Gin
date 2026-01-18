package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ===========================================================================
// BaseModel là struct cơ sở cho tất cả các models
// Chứa các trường chung: ID, timestamps, và soft delete
// ===========================================================================

// BaseModel chứa các trường chung cho tất cả models
type BaseModel struct {
	// ID là primary key dạng UUID, tự động generate nếu không có
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`

	// CreatedAt thời điểm tạo record
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`

	// UpdatedAt thời điểm cập nhật gần nhất
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// DeletedAt dùng cho soft delete, nếu có giá trị = đã xóa
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// BeforeCreate hook chạy trước khi insert record
// Tự động generate UUID nếu chưa có
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// GetID trả về ID của model
func (b *BaseModel) GetID() uuid.UUID {
	return b.ID
}

// IsDeleted kiểm tra model đã bị soft delete chưa
func (b *BaseModel) IsDeleted() bool {
	return b.DeletedAt.Valid
}
