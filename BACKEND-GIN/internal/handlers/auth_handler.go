package handlers

import (
	"net/http"

	"chatbox-gin/internal/dto"
	apperrors "chatbox-gin/internal/errors"
	"chatbox-gin/internal/middleware"
	"chatbox-gin/internal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ===========================================================================
// Auth Handler
// Handle authentication endpoints: login, refresh, me, logout
// ===========================================================================

// AuthHandler xử lý các endpoint auth
type AuthHandler struct {
	authService services.AuthService
	logger      *zap.Logger
}

// NewAuthHandler tạo auth handler mới
func NewAuthHandler(
	authService services.AuthService,
	logger *zap.Logger,
) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// ===========================================================================
// Request/Response DTOs
// ===========================================================================

// LoginRequest body cho đăng nhập
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginResponse response sau đăng nhập
type LoginResponse struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresAt    int64         `json:"expires_at"`
	User         *UserResponse `json:"user"`
}

// UserResponse user data (không có password)
type UserResponse struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	Name        string  `json:"name"`
	Role        string  `json:"role"`
	WorkspaceID string  `json:"workspace_id"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// RefreshRequest body cho refresh token
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ===========================================================================
// Handlers
// ===========================================================================

// Login đăng nhập user
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", err.Error()))
		return
	}

	// Call auth service
	result, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if err == apperrors.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, dto.Error("INVALID_CREDENTIALS", "Email hoặc mật khẩu không đúng"))
			return
		}
		h.logger.Error("login failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.Error("INTERNAL_ERROR", "Đã có lỗi xảy ra"))
		return
	}

	// Set httpOnly cookies with SameSite=Lax
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", result.Tokens.AccessToken, result.Tokens.ExpiresIn, "/", "", false, true)
	c.SetCookie("refresh_token", result.Tokens.RefreshToken, 604800, "/", "", false, true)

	// Set CSRF token (readable by JS)
	csrfToken, err := middleware.GenerateCSRFToken()
	if err != nil {
		h.logger.Error("generate csrf token failed", zap.Error(err))
	} else {
		middleware.SetCSRFCookie(c, csrfToken)
	}

	c.JSON(http.StatusOK, dto.Success(&UserResponse{
		ID:          result.User.ID.String(),
		Email:       result.User.Email,
		Name:        result.User.Name,
		Role:        string(result.User.Role),
		WorkspaceID: result.User.WorkspaceID.String(),
		AvatarURL:   result.User.AvatarURL,
	}))
}

// Refresh làm mới tokens
// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	// Read refresh token from cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		c.JSON(http.StatusUnauthorized, dto.Error("NO_TOKEN", "Refresh token không tồn tại"))
		return
	}

	// Call auth service
	result, err := h.authService.RefreshTokens(c.Request.Context(), refreshToken)
	if err != nil {
		if err == apperrors.ErrTokenExpired {
			c.SetCookie("access_token", "", -1, "/", "", false, true)
			c.SetCookie("refresh_token", "", -1, "/", "", false, true)
			c.JSON(http.StatusUnauthorized, dto.Error("TOKEN_EXPIRED", "Refresh token đã hết hạn"))
			return
		}
		if err == apperrors.ErrInvalidToken {
			c.JSON(http.StatusUnauthorized, dto.Error("INVALID_TOKEN", "Refresh token không hợp lệ"))
			return
		}
		h.logger.Error("refresh failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.Error("INTERNAL_ERROR", "Đã có lỗi xảy ra"))
		return
	}

	// Set new httpOnly cookies with SameSite=Lax
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", result.Tokens.AccessToken, result.Tokens.ExpiresIn, "/", "", false, true)
	c.SetCookie("refresh_token", result.Tokens.RefreshToken, 604800, "/", "", false, true)

	// Refresh CSRF token too
	csrfToken, err := middleware.GenerateCSRFToken()
	if err != nil {
		h.logger.Error("generate csrf token failed", zap.Error(err))
	} else {
		middleware.SetCSRFCookie(c, csrfToken)
	}

	c.JSON(http.StatusOK, dto.Success(&UserResponse{
		ID:          result.User.ID.String(),
		Email:       result.User.Email,
		Name:        result.User.Name,
		Role:        string(result.User.Role),
		WorkspaceID: result.User.WorkspaceID.String(),
		AvatarURL:   result.User.AvatarURL,
	}))
}

// Me lấy thông tin user hiện tại
// GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.Error("UNAUTHORIZED", "Chưa đăng nhập"))
		return
	}

	user, err := h.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Error("USER_NOT_FOUND", "Người dùng không tồn tại"))
		return
	}

	c.JSON(http.StatusOK, dto.Success(&UserResponse{
		ID:          user.ID.String(),
		Email:       user.Email,
		Name:        user.Name,
		Role:        string(user.Role),
		WorkspaceID: user.WorkspaceID.String(),
		AvatarURL:   user.AvatarURL,
	}))
}

// Logout đăng xuất - Revoke token và clear cookies
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// Revoke refresh token từ DB
	userID, ok := middleware.GetUserID(c)
	if ok {
		if err := h.authService.RevokeRefreshToken(c.Request.Context(), userID); err != nil {
			h.logger.Warn("revoke refresh token failed", zap.Error(err))
		}
	}

	// Clear all auth cookies
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)
	c.SetCookie("csrf_token", "", -1, "/", "", false, false)

	c.JSON(http.StatusOK, dto.Success(gin.H{"message": "Đăng xuất thành công"}))
}

// ===========================================================================
// Route Registration
// ===========================================================================

// RegisterRoutes đăng ký routes cho auth
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	auth := rg.Group("/auth")
	{
		// Public routes (không cần auth)
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.Refresh)

		// Protected routes (cần auth)
		auth.GET("/me", authMiddleware, h.Me)
		auth.POST("/logout", authMiddleware, h.Logout)
	}
}
