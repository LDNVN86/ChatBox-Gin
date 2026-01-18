package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ===========================================================================
// User (Người dùng hệ thống)
// Đại diện cho agents, admins, owners của workspace
// KHÔNG phải khách hàng chat (khách hàng là Participant)
// ===========================================================================

// UserRole các vai trò người dùng
type UserRole string

const (
	// RoleOwner chủ workspace, có toàn quyền
	RoleOwner UserRole = "owner"

	// RoleAdmin quản trị viên, có thể quản lý users và settings
	RoleAdmin UserRole = "admin"

	// RoleAgent nhân viên hỗ trợ, chỉ có thể chat với khách
	RoleAgent UserRole = "agent"
)

// User đại diện cho người dùng hệ thống (agent, admin, owner)
type User struct {
	BaseModel

	// WorkspaceID ID workspace mà user thuộc về
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index" json:"workspace_id"`

	// Email địa chỉ email (unique trong workspace)
	Email string `gorm:"size:255;not null" json:"email"`

	// PasswordHash mật khẩu đã hash (KHÔNG bao giờ trả về trong JSON)
	PasswordHash string `gorm:"size:255;not null" json:"-"`

	// RefreshTokenHash hash của refresh token hiện tại (KHÔNG trả về trong JSON)
	// Dùng để validate và revoke refresh token
	RefreshTokenHash *string `gorm:"size:255" json:"-"`

	// Name tên hiển thị
	Name string `gorm:"size:255;not null" json:"name"`

	// AvatarURL URL avatar
	AvatarURL *string `gorm:"size:500" json:"avatar_url,omitempty"`

	// Role vai trò: owner, admin, agent
	Role UserRole `gorm:"size:50;not null;default:'agent'" json:"role"`

	// IsActive tài khoản có active không
	IsActive bool `gorm:"default:true" json:"is_active"`

	// LastSeenAt lần cuối online
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`

	// Relations
	Workspace             Workspace      `gorm:"foreignKey:WorkspaceID" json:"workspace,omitempty"`
	AssignedConversations []Conversation `gorm:"foreignKey:AssignedTo" json:"assigned_conversations,omitempty"`
}

// TableName trả về tên bảng
func (User) TableName() string {
	return "users"
}

// SetPassword hash và set password
// Sử dụng bcrypt với cost mặc định
func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword kiểm tra password có đúng không
// Trả về true nếu đúng, false nếu sai
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// IsOwner kiểm tra user có phải owner không
func (u *User) IsOwner() bool {
	return u.Role == RoleOwner
}

// IsAdmin kiểm tra user có quyền admin không (owner hoặc admin)
func (u *User) IsAdmin() bool {
	return u.Role == RoleOwner || u.Role == RoleAdmin
}

// UpdateLastSeen cập nhật thời gian online gần nhất
func (u *User) UpdateLastSeen() {
	now := time.Now()
	u.LastSeenAt = &now
}
