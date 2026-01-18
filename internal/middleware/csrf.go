package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"

	"chatbox-gin/internal/dto"

	"github.com/gin-gonic/gin"
)

// ===========================================================================
// CSRF Middleware
// Double Submit Cookie pattern cho CSRF protection
// Token được set trong cookie (readable) và phải match với header
// ===========================================================================

const (
	CSRFCookieName = "csrf_token"
	CSRFHeaderName = "X-CSRF-Token"
	CSRFTokenLength = 32
)

// GenerateCSRFToken tạo random CSRF token
func GenerateCSRFToken() (string, error) {
	bytes := make([]byte, CSRFTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// SetCSRFCookie set CSRF token cookie (readable bởi JS)
func SetCSRFCookie(c *gin.Context, token string) {
	// Non-httpOnly để FE có thể đọc và gửi trong header
	c.SetCookie(
		CSRFCookieName,
		token,
		86400*7, // 7 days
		"/",
		"",    // domain empty cho localhost
		false, // secure (production nên true)
		false, // httpOnly = false để JS đọc được
	)
}

// CSRFMiddleware validates CSRF token cho state-changing requests
// Skip: GET, HEAD, OPTIONS (safe methods)
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip safe methods
		method := c.Request.Method
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			c.Next()
			return
		}

		// Get token from cookie
		cookieToken, err := c.Cookie(CSRFCookieName)
		if err != nil || cookieToken == "" {
			c.JSON(http.StatusForbidden, dto.Error("CSRF_MISSING", "CSRF token required"))
			c.Abort()
			return
		}

		// Get token from header
		headerToken := c.GetHeader(CSRFHeaderName)
		if headerToken == "" {
			c.JSON(http.StatusForbidden, dto.Error("CSRF_MISSING", "CSRF token header required"))
			c.Abort()
			return
		}

		// Compare tokens (constant-time comparison để tránh timing attack)
		if !secureCompare(cookieToken, headerToken) {
			c.JSON(http.StatusForbidden, dto.Error("CSRF_INVALID", "CSRF token mismatch"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// secureCompare so sánh 2 string trong constant time
func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	result := 0
	for i := 0; i < len(a); i++ {
		result |= int(a[i]) ^ int(b[i])
	}
	return result == 0
}

// CSRFExempt middleware để exempt specific routes khỏi CSRF check
// Usage: router.POST("/auth/login", CSRFExempt(), handler)
func CSRFExempt() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("csrf_exempt", true)
		c.Next()
	}
}

// CSRFMiddlewareWithExempt giống CSRFMiddleware nhưng check csrf_exempt flag
func CSRFMiddlewareWithExempt(exemptPaths []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip safe methods
		method := c.Request.Method
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			c.Next()
			return
		}

		// Check if path is exempt
		path := c.Request.URL.Path
		for _, exempt := range exemptPaths {
			if strings.HasPrefix(path, exempt) {
				c.Next()
				return
			}
		}

		// Check if manually exempted
		if exempt, exists := c.Get("csrf_exempt"); exists && exempt.(bool) {
			c.Next()
			return
		}

		// Get token from cookie
		cookieToken, err := c.Cookie(CSRFCookieName)
		if err != nil || cookieToken == "" {
			c.JSON(http.StatusForbidden, dto.Error("CSRF_MISSING", "CSRF token required"))
			c.Abort()
			return
		}

		// Get token from header
		headerToken := c.GetHeader(CSRFHeaderName)
		if headerToken == "" {
			c.JSON(http.StatusForbidden, dto.Error("CSRF_MISSING", "CSRF token header required"))
			c.Abort()
			return
		}

		// Validate
		if !secureCompare(cookieToken, headerToken) {
			c.JSON(http.StatusForbidden, dto.Error("CSRF_INVALID", "CSRF token mismatch"))
			c.Abort()
			return
		}

		c.Next()
	}
}
