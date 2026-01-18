package middleware

import (
	"net/http"
	"runtime/debug"

	"chatbox-gin/internal/dto"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ===========================================================================
// Recovery Middleware
// Bắt panic và trả về response lỗi thay vì crash server
// Log stack trace để debugging
// ===========================================================================

// Recovery middleware bắt panic trong handlers
// Khi có panic, nó sẽ:
// 1. Log error và stack trace
// 2. Trả về 500 Internal Server Error
// 3. Không crash server
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log panic với stack trace
				logger.Error("panic recovered",
					zap.String("request_id", GetRequestID(c)),
					zap.Any("error", err),
					zap.String("stack", string(debug.Stack())),
				)

				// Trả về error response
				c.AbortWithStatusJSON(http.StatusInternalServerError, dto.Error(
					"INTERNAL_ERROR",
					"An internal error occurred",
				))
			}
		}()

		c.Next()
	}
}