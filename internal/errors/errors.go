package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// ===========================================================================
// Custom Errors
// Định nghĩa các lỗi chuẩn cho ứng dụng
// Mỗi lỗi được map với HTTP status code tương ứng
// ===========================================================================

// Sentinel errors - các lỗi chuẩn để dùng với errors.Is()
var (
	// ErrNotFound resource không tồn tại
	ErrNotFound = errors.New("not found")

	// ErrUnauthorized chưa đăng nhập/token không hợp lệ
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden không có quyền truy cập
	ErrForbidden = errors.New("forbidden")

	// ErrInvalidInput dữ liệu đầu vào không hợp lệ
	ErrInvalidInput = errors.New("invalid input")

	// ErrDuplicateEntry dữ liệu đã tồn tại (unique constraint)
	ErrDuplicateEntry = errors.New("duplicate entry")

	// ErrConflict xung đột dữ liệu (VD: concurrent update)
	ErrConflict = errors.New("conflict")

	// ErrInternal lỗi server nội bộ
	ErrInternal = errors.New("internal server error")

	// ErrExternal lỗi từ service bên ngoài (FB API, Zalo API)
	ErrExternal = errors.New("external service error")

	// ErrTimeout request timeout
	ErrTimeout = errors.New("timeout")

	// Auth errors
	// ErrInvalidCredentials email hoặc password không đúng
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrTokenExpired token đã hết hạn
	ErrTokenExpired = errors.New("token expired")

	// ErrInvalidToken token không hợp lệ
	ErrInvalidToken = errors.New("invalid token")
)

// ===========================================================================
// AppError
// Custom error type cho ứng dụng
// ===========================================================================

// AppError cấu trúc lỗi chi tiết
type AppError struct {
	// Err lỗi gốc (wrapped error)
	Err error

	// Message thông báo lỗi cho user
	Message string

	// Code mã lỗi (VD: "NOT_FOUND")
	Code string

	// StatusCode HTTP status code
	StatusCode int
}

// Error implement error interface
func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

// Unwrap trả về wrapped error (cho errors.Is/As)
func (e *AppError) Unwrap() error {
	return e.Err
}

// New tạo AppError mới từ sentinel error
func New(err error, message string) *AppError {
	return &AppError{
		Err:        err,
		Message:    message,
		StatusCode: StatusCode(err),
		Code:       ErrorCode(err),
	}
}

// Wrap wrap error với message bổ sung
// Dùng %w để giữ nguyên wrapped error chain
func Wrap(err error, message string) error {
	return fmt.Errorf("%s: %w", message, err)
}

// ===========================================================================
// Error Mapping Functions
// Map từ error sang HTTP status code và error code
// ===========================================================================

// StatusCode trả về HTTP status code tương ứng với error
func StatusCode(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, ErrDuplicateEntry):
		return http.StatusConflict
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	case errors.Is(err, ErrTimeout):
		return http.StatusGatewayTimeout
	case errors.Is(err, ErrInvalidCredentials):
		return http.StatusUnauthorized
	case errors.Is(err, ErrTokenExpired):
		return http.StatusUnauthorized
	case errors.Is(err, ErrInvalidToken):
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// ErrorCode trả về error code string tương ứng với error
func ErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, ErrUnauthorized):
		return "UNAUTHORIZED"
	case errors.Is(err, ErrForbidden):
		return "FORBIDDEN"
	case errors.Is(err, ErrInvalidInput):
		return "INVALID_INPUT"
	case errors.Is(err, ErrDuplicateEntry):
		return "DUPLICATE_ENTRY"
	case errors.Is(err, ErrConflict):
		return "CONFLICT"
	case errors.Is(err, ErrTimeout):
		return "TIMEOUT"
	case errors.Is(err, ErrInvalidCredentials):
		return "INVALID_CREDENTIALS"
	case errors.Is(err, ErrTokenExpired):
		return "TOKEN_EXPIRED"
	case errors.Is(err, ErrInvalidToken):
		return "INVALID_TOKEN"
	default:
		return "INTERNAL_ERROR"
	}
}

// Is helper function cho errors.Is()
func Is(err, target error) bool {
	return errors.Is(err, target)
}