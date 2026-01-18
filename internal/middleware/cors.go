package middleware

import (
	"github.com/gin-gonic/gin"
)

// ===========================================================================
// CORS Middleware
// Xử lý Cross-Origin Resource Sharing cho frontend clients
// Cho phép browser gọi API từ domain khác
// ===========================================================================

// CORS middleware xử lý CORS headers
// allowedOrigins: danh sách origins được phép (dùng "*" cho tất cả)
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Kiểm tra origin có được phép không
		allowed := false
		for _, o := range allowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		// Thêm CORS headers nếu origin được phép
		if allowed {
			// Cho phép origin này
			c.Header("Access-Control-Allow-Origin", origin)

			// Các method được phép
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

			// Các header được phép
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Request-ID")

			// Cho phép gửi cookies
			c.Header("Access-Control-Allow-Credentials", "true")

			// Cache preflight request 24 giờ
			c.Header("Access-Control-Max-Age", "86400")
		}

		// Handle preflight request (OPTIONS)
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}