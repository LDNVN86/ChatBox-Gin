package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ===========================================================================
// Request ID Middleware
// Thêm unique ID cho mỗi request để tracking và debugging
// ID được lưu trong context và trả về trong response header
// ===========================================================================

const (
	// RequestIDKey key để lưu request ID trong gin context
	RequestIDKey = "request_id"

	// RequestIDHeader tên header chứa request ID
	RequestIDHeader = "X-Request-ID"
)

// RequestID middleware thêm unique ID cho mỗi request
// Nếu client gửi header X-Request-ID thì dùng giá trị đó
// Nếu không có thì tự generate UUID mới
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Lấy từ header nếu client có gửi
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			// Generate mới nếu không có
			requestID = uuid.New().String()
		}

		// Lưu vào context để các handler/service có thể dùng
		c.Set(RequestIDKey, requestID)

		// Thêm vào response header
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

// GetRequestID lấy request ID từ gin context
// Trả về empty string nếu không có
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get(RequestIDKey); exists {
		return id.(string)
	}
	return ""
}
