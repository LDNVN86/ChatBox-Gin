package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ===========================================================================
// Logging Middleware
// Log thông tin mỗi HTTP request (method, path, status, latency)
// Sử dụng structured logging với zap
// ===========================================================================

// Logging middleware log thông tin mỗi request
// Log level phụ thuộc vào status code:
// - >= 500: Error
// - >= 400: Warn
// - < 400: Info
func Logging(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ghi nhận thời điểm bắt đầu
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Xử lý request
		c.Next()

		// Tính thời gian xử lý
		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// Tạo log fields
		fields := []zap.Field{
			zap.String("request_id", GetRequestID(c)),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int("body_size", c.Writer.Size()),
		}

		// Thêm errors nếu có
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		// Log với level phù hợp theo status code
		switch {
		case statusCode >= 500:
			logger.Error("request completed", fields...)
		case statusCode >= 400:
			logger.Warn("request completed", fields...)
		default:
			logger.Info("request completed", fields...)
		}
	}
}