package middleware

import (
	"net/http"
	"strings"

	"chatbox-gin/internal/auth"
	"chatbox-gin/internal/dto"
	"chatbox-gin/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ===========================================================================
// Auth Middleware
// Protect routes với JWT authentication
// ===========================================================================

// Context keys cho auth data
const (
	ContextKeyUserID      = "user_id"
	ContextKeyWorkspaceID = "workspace_id"
	ContextKeyUserRole    = "user_role"
	ContextKeyClaims      = "claims"
)

// AuthMiddleware tạo middleware để verify JWT from cookie or header
func AuthMiddleware(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// 1. First try to get token from cookie (httpOnly)
		if cookie, err := c.Cookie("access_token"); err == nil && cookie != "" {
			tokenString = cookie
		}

		// 2. Fallback to Authorization header (for API clients)
		if tokenString == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					tokenString = parts[1]
				}
			}
		}

		// 3. No token found
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, dto.Error("UNAUTHORIZED", "Authentication required"))
			c.Abort()
			return
		}

		// 4. Validate token
		claims, err := jwtService.ValidateAccessToken(tokenString)
		if err != nil {
			if err == auth.ErrExpiredToken {
				c.JSON(http.StatusUnauthorized, dto.Error("TOKEN_EXPIRED", "Token has expired"))
			} else {
				c.JSON(http.StatusUnauthorized, dto.Error("INVALID_TOKEN", "Invalid token"))
			}
			c.Abort()
			return
		}

		// 5. Set user info in context
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyWorkspaceID, claims.WorkspaceID)
		c.Set(ContextKeyUserRole, claims.Role)
		c.Set(ContextKeyClaims, claims)

		c.Next()
	}
}

// RequireRole middleware yêu cầu role cụ thể
func RequireRole(roles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(ContextKeyUserRole)
		if !exists {
			c.JSON(http.StatusForbidden, dto.Error("FORBIDDEN", "Access denied"))
			c.Abort()
			return
		}

		userRole := role.(models.UserRole)
		for _, r := range roles {
			if userRole == r {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, dto.Error("FORBIDDEN", "Insufficient permissions"))
		c.Abort()
	}
}

// RequireAdmin yêu cầu admin hoặc owner role
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(models.RoleAdmin, models.RoleOwner)
}

// RequireOwner yêu cầu owner role
func RequireOwner() gin.HandlerFunc {
	return RequireRole(models.RoleOwner)
}

// ===========================================================================
// Helper functions để lấy data từ context
// ===========================================================================

// GetUserID lấy user ID từ context
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	id, exists := c.Get(ContextKeyUserID)
	if !exists {
		return uuid.Nil, false
	}
	return id.(uuid.UUID), true
}

// GetWorkspaceID lấy workspace ID từ context
func GetWorkspaceID(c *gin.Context) (uuid.UUID, bool) {
	id, exists := c.Get(ContextKeyWorkspaceID)
	if !exists {
		return uuid.Nil, false
	}
	return id.(uuid.UUID), true
}

// GetUserRole lấy user role từ context
func GetUserRole(c *gin.Context) (models.UserRole, bool) {
	role, exists := c.Get(ContextKeyUserRole)
	if !exists {
		return "", false
	}
	return role.(models.UserRole), true
}

// GetClaims lấy toàn bộ claims từ context
func GetClaims(c *gin.Context) (*auth.Claims, bool) {
	claims, exists := c.Get(ContextKeyClaims)
	if !exists {
		return nil, false
	}
	return claims.(*auth.Claims), true
}
