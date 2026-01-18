package dto

import "github.com/google/uuid"

// ===========================================================================
// Request DTOs (Data Transfer Objects)
// Các struct dùng để validate và parse request body/query
// ===========================================================================

// PaginationRequest phân trang cho các API list
type PaginationRequest struct {
	// Page số trang hiện tại (bắt đầu từ 1)
	Page int `form:"page" binding:"min=0"`

	// Limit số record mỗi trang (tối đa 100)
	Limit int `form:"limit" binding:"min=0,max=100"`
}

// SetDefaults set giá trị mặc định cho pagination
func (p *PaginationRequest) SetDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.Limit <= 0 {
		p.Limit = 20
	}
}

// Offset tính offset cho database query
func (p *PaginationRequest) Offset() int {
	return (p.Page - 1) * p.Limit
}

// ===========================================================================
// Conversation Requests
// ===========================================================================

// ListConversationsRequest request lấy danh sách hội thoại
type ListConversationsRequest struct {
	PaginationRequest

	// Status filter theo trạng thái
	Status string `form:"status" binding:"omitempty,oneof=open pending closed bot_paused"`

	// AssignedTo filter theo agent được assign
	AssignedTo *uuid.UUID `form:"assigned_to"`

	// Search từ khóa tìm kiếm
	Search string `form:"q" binding:"max=100"`
}

// UpdateConversationRequest request cập nhật hội thoại
type UpdateConversationRequest struct {
	// Status trạng thái mới (nullable = không đổi)
	Status *string `json:"status" binding:"omitempty,oneof=open pending closed bot_paused"`

	// AssignedTo ID agent mới
	AssignedTo *uuid.UUID `json:"assigned_to"`

	// Priority mức độ ưu tiên
	Priority *string `json:"priority" binding:"omitempty,oneof=low normal high urgent"`
}

// ===========================================================================
// Message Requests
// ===========================================================================

// CreateMessageRequest request tạo tin nhắn mới
type CreateMessageRequest struct {
	// Content nội dung text (bắt buộc, 1-5000 ký tự)
	Content string `json:"content" binding:"required,min=1,max=5000"`

	// ContentType loại nội dung (mặc định: text)
	ContentType string `json:"content_type" binding:"omitempty,oneof=text image file"`

	// Attachments file đính kèm
	Attachments []Attachment `json:"attachments" binding:"dive"`
}

// Attachment file đính kèm trong request
type Attachment struct {
	// Type loại file: image, file
	Type string `json:"type" binding:"required,oneof=image file"`

	// URL đường dẫn file
	URL string `json:"url" binding:"required,url"`

	// Name tên file
	Name string `json:"name"`

	// Size kích thước (bytes)
	Size int64 `json:"size"`
}