package dto

import "math"

// ===========================================================================
// Response DTOs (Data Transfer Objects)
// Các struct chuẩn hóa response format
// ===========================================================================

// Response cấu trúc response chuẩn cho tất cả API
type Response struct {
	// Success request thành công hay không
	Success bool `json:"success"`

	// Data dữ liệu trả về (nếu thành công)
	Data interface{} `json:"data,omitempty"`

	// Error thông tin lỗi (nếu thất bại)
	Error *APIError `json:"error,omitempty"`

	// Meta thông tin phân trang (cho list API)
	Meta *Meta `json:"meta,omitempty"`
}

// APIError cấu trúc lỗi chuẩn
type APIError struct {
	// Code mã lỗi (VD: "NOT_FOUND", "INVALID_INPUT")
	Code string `json:"code"`

	// Message thông báo lỗi chi tiết
	Message string `json:"message"`
}

// Meta thông tin phân trang
type Meta struct {
	// Total tổng số records
	Total int64 `json:"total"`

	// Page trang hiện tại
	Page int `json:"page"`

	// Limit số records mỗi trang
	Limit int `json:"limit"`

	// TotalPages tổng số trang
	TotalPages int `json:"total_pages"`
}

// NewMeta tạo Meta từ thông tin phân trang
func NewMeta(page, limit int, total int64) *Meta {
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	return &Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}

// ===========================================================================
// Response Builders
// Helper functions để tạo response
// ===========================================================================

// Success tạo response thành công
func Success(data interface{}) Response {
	return Response{
		Success: true,
		Data:    data,
	}
}

// SuccessWithMeta tạo response thành công với thông tin phân trang
func SuccessWithMeta(data interface{}, meta *Meta) Response {
	return Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	}
}

// Error tạo response lỗi
func Error(code, message string) Response {
	return Response{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	}
}

// ErrorFromErr tạo response lỗi từ error object
func ErrorFromErr(err error) Response {
	return Response{
		Success: false,
		Error: &APIError{
			Code:    "ERROR",
			Message: err.Error(),
		},
	}
}