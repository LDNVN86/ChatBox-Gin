package repositories

import (
	"context"

	"github.com/google/uuid"
)

// ===========================================================================
// Repository Base Interfaces và Types
// Các interface và struct dùng chung cho tất cả repositories
// ===========================================================================

// FindOptions tùy chọn query cho các method Find
type FindOptions struct {
	// Offset vị trí bắt đầu (cho phân trang)
	Offset int

	// Limit số lượng records tối đa
	Limit int

	// OrderBy cột để sắp xếp
	OrderBy string

	// OrderDir hướng sắp xếp: "asc" hoặc "desc"
	OrderDir string

	// Preloads các quan hệ cần eager load
	Preloads []string

	// Filters các điều kiện filter
	Filters map[string]interface{}
}

// SetDefaults thiết lập giá trị mặc định cho FindOptions
func (o *FindOptions) SetDefaults() {
	if o.Limit == 0 {
		o.Limit = 20
	}
	if o.OrderBy == "" {
		o.OrderBy = "created_at"
	}
	if o.OrderDir == "" {
		o.OrderDir = "desc"
	}
}

// GetOrderClause trả về chuỗi ORDER BY
func (o *FindOptions) GetOrderClause() string {
	return o.OrderBy + " " + o.OrderDir
}

// ===========================================================================
// Generic Repository Interface
// Interface cơ bản với các method CRUD chung
// ===========================================================================

// Repository interface cơ bản cho tất cả repositories
type Repository[T any] interface {
	// FindByID tìm record theo ID
	FindByID(ctx context.Context, id uuid.UUID) (*T, error)

	// Create tạo record mới
	Create(ctx context.Context, entity *T) error

	// Update cập nhật record
	Update(ctx context.Context, entity *T) error

	// Delete xóa record (soft delete nếu có DeletedAt)
	Delete(ctx context.Context, id uuid.UUID) error
}
