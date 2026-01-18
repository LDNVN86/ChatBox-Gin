package handlers

import (
	"net/http"

	"chatbox-gin/internal/dto"
	"chatbox-gin/internal/middleware"
	"chatbox-gin/internal/models"
	"chatbox-gin/internal/repositories"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ===========================================================================
// Rule Handler
// Quản lý CRUD cho bot rules từ dashboard
// Shop owner có thể tạo/sửa/xóa câu trả lời tự động
// ===========================================================================

// RuleHandler xử lý các endpoint liên quan đến Rules
type RuleHandler struct {
	ruleRepo repositories.RuleRepository
	logger   *zap.Logger
}

// NewRuleHandler tạo RuleHandler mới
func NewRuleHandler(ruleRepo repositories.RuleRepository, logger *zap.Logger) *RuleHandler {
	return &RuleHandler{
		ruleRepo: ruleRepo,
		logger:   logger,
	}
}

// ===========================================================================
// Request/Response DTOs
// ===========================================================================

// CreateRuleRequest tạo rule mới
type CreateRuleRequest struct {
	Name           string                 `json:"name" binding:"required,min=1,max=255"`
	Description    string                 `json:"description"`
	TriggerType    models.TriggerType     `json:"trigger_type" binding:"required,oneof=keyword time_window fallback"`
	TriggerConfig  models.TriggerConfig   `json:"trigger_config" binding:"required"`
	ResponseType   models.ResponseType    `json:"response_type" binding:"required,oneof=text template handoff"`
	ResponseConfig models.ResponseConfig  `json:"response_config" binding:"required"`
	Priority       int                    `json:"priority"`
	IsActive       bool                   `json:"is_active"`
}

// UpdateRuleRequest cập nhật rule
type UpdateRuleRequest struct {
	Name           *string                 `json:"name" binding:"omitempty,min=1,max=255"`
	Description    *string                 `json:"description"`
	TriggerType    *models.TriggerType     `json:"trigger_type" binding:"omitempty,oneof=keyword time_window fallback"`
	TriggerConfig  *models.TriggerConfig   `json:"trigger_config"`
	ResponseType   *models.ResponseType    `json:"response_type" binding:"omitempty,oneof=text template handoff"`
	ResponseConfig *models.ResponseConfig  `json:"response_config"`
	Priority       *int                    `json:"priority"`
	IsActive       *bool                   `json:"is_active"`
}

// ===========================================================================
// Handlers
// ===========================================================================

// List lấy danh sách rules của workspace
// GET /api/v1/rules?workspace_id=xxx
func (h *RuleHandler) List(c *gin.Context) {
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	// Lấy workspace_id từ query
	workspaceIDStr := c.Query("workspace_id")
	if workspaceIDStr == "" {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", "workspace_id là bắt buộc"))
		return
	}

	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", "workspace_id không hợp lệ"))
		return
	}

	// Lấy tất cả rules (cả active và inactive)
	rules, err := h.ruleRepo.FindActiveByWorkspace(ctx, workspaceID)
	if err != nil {
		h.logger.Error("failed to get rules",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.Error("DB_ERROR", "Không thể lấy danh sách rules"))
		return
	}

	c.JSON(http.StatusOK, dto.Success(gin.H{
		"rules": rules,
		"total": len(rules),
	}))
}

// Get lấy chi tiết rule
// GET /api/v1/rules/:id
func (h *RuleHandler) Get(c *gin.Context) {
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", "Rule ID không hợp lệ"))
		return
	}

	rule, err := h.ruleRepo.FindByID(ctx, ruleID)
	if err != nil {
		h.logger.Warn("rule not found",
			zap.String("request_id", requestID),
			zap.String("rule_id", ruleID.String()),
		)
		c.JSON(http.StatusNotFound, dto.Error("NOT_FOUND", "Không tìm thấy rule"))
		return
	}

	c.JSON(http.StatusOK, dto.Success(rule))
}

// Create tạo rule mới
// POST /api/v1/rules
func (h *RuleHandler) Create(c *gin.Context) {
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	// Lấy workspace_id từ query hoặc body
	workspaceIDStr := c.Query("workspace_id")
	if workspaceIDStr == "" {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", "workspace_id là bắt buộc"))
		return
	}

	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", "workspace_id không hợp lệ"))
		return
	}

	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", err.Error()))
		return
	}

	// Tạo rule mới
	rule := &models.Rule{
		WorkspaceID:    workspaceID,
		Name:           req.Name,
		TriggerType:    req.TriggerType,
		TriggerConfig:  req.TriggerConfig,
		ResponseType:   req.ResponseType,
		ResponseConfig: req.ResponseConfig,
		Priority:       req.Priority,
		IsActive:       req.IsActive,
	}

	if req.Description != "" {
		rule.Description = &req.Description
	}

	if err := h.ruleRepo.Create(ctx, rule); err != nil {
		h.logger.Error("failed to create rule",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.Error("DB_ERROR", "Không thể tạo rule"))
		return
	}

	h.logger.Info("rule created",
		zap.String("request_id", requestID),
		zap.String("rule_id", rule.ID.String()),
		zap.String("name", rule.Name),
	)

	c.JSON(http.StatusCreated, dto.Success(rule))
}

// Update cập nhật rule
// PUT /api/v1/rules/:id
func (h *RuleHandler) Update(c *gin.Context) {
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", "Rule ID không hợp lệ"))
		return
	}

	// Tìm rule hiện tại
	rule, err := h.ruleRepo.FindByID(ctx, ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Error("NOT_FOUND", "Không tìm thấy rule"))
		return
	}

	var req UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", err.Error()))
		return
	}

	// Cập nhật các fields được gửi
	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Description != nil {
		rule.Description = req.Description
	}
	if req.TriggerType != nil {
		rule.TriggerType = *req.TriggerType
	}
	if req.TriggerConfig != nil {
		rule.TriggerConfig = *req.TriggerConfig
	}
	if req.ResponseType != nil {
		rule.ResponseType = *req.ResponseType
	}
	if req.ResponseConfig != nil {
		rule.ResponseConfig = *req.ResponseConfig
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.IsActive != nil {
		rule.IsActive = *req.IsActive
	}

	if err := h.ruleRepo.Update(ctx, rule); err != nil {
		h.logger.Error("failed to update rule",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.Error("DB_ERROR", "Không thể cập nhật rule"))
		return
	}

	h.logger.Info("rule updated",
		zap.String("request_id", requestID),
		zap.String("rule_id", rule.ID.String()),
	)

	c.JSON(http.StatusOK, dto.Success(rule))
}

// Delete xóa rule (soft delete)
// DELETE /api/v1/rules/:id
func (h *RuleHandler) Delete(c *gin.Context) {
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", "Rule ID không hợp lệ"))
		return
	}

	// Kiểm tra rule tồn tại
	_, err = h.ruleRepo.FindByID(ctx, ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Error("NOT_FOUND", "Không tìm thấy rule"))
		return
	}

	// Soft delete sử dụng GORM (set deleted_at)
	if err := h.ruleRepo.Delete(ctx, ruleID); err != nil {
		h.logger.Error("failed to delete rule",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.Error("DB_ERROR", "Không thể xóa rule"))
		return
	}

	h.logger.Info("rule deleted",
		zap.String("request_id", requestID),
		zap.String("rule_id", ruleID.String()),
	)

	c.JSON(http.StatusOK, dto.Success(gin.H{
		"message": "Rule đã được chuyển vào thùng rác",
	}))
}

// ListDeleted lấy danh sách rules đã xóa
// GET /api/v1/rules/trash?workspace_id=xxx
func (h *RuleHandler) ListDeleted(c *gin.Context) {
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	workspaceIDStr := c.Query("workspace_id")
	if workspaceIDStr == "" {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", "workspace_id là bắt buộc"))
		return
	}

	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", "workspace_id không hợp lệ"))
		return
	}

	rules, err := h.ruleRepo.FindDeletedByWorkspace(ctx, workspaceID)
	if err != nil {
		h.logger.Error("failed to get deleted rules",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.Error("DB_ERROR", "Không thể lấy danh sách rules đã xóa"))
		return
	}

	c.JSON(http.StatusOK, dto.Success(gin.H{
		"rules": rules,
		"total": len(rules),
	}))
}

// Restore khôi phục rule đã xóa
// POST /api/v1/rules/:id/restore
func (h *RuleHandler) Restore(c *gin.Context) {
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", "Rule ID không hợp lệ"))
		return
	}

	// Kiểm tra rule đã xóa tồn tại (dùng Unscoped)
	rule, err := h.ruleRepo.FindByIDUnscoped(ctx, ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Error("NOT_FOUND", "Không tìm thấy rule"))
		return
	}

	// Kiểm tra rule thực sự đã bị xóa
	if !rule.DeletedAt.Valid {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", "Rule chưa bị xóa"))
		return
	}

	// Restore rule
	if err := h.ruleRepo.Restore(ctx, ruleID); err != nil {
		h.logger.Error("failed to restore rule",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.Error("DB_ERROR", "Không thể khôi phục rule"))
		return
	}

	h.logger.Info("rule restored",
		zap.String("request_id", requestID),
		zap.String("rule_id", ruleID.String()),
	)

	c.JSON(http.StatusOK, dto.Success(gin.H{
		"message": "Rule đã được khôi phục",
	}))
}

// Toggle bật/tắt rule
// PATCH /api/v1/rules/:id/toggle
func (h *RuleHandler) Toggle(c *gin.Context) {
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", "Rule ID không hợp lệ"))
		return
	}

	rule, err := h.ruleRepo.FindByID(ctx, ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Error("NOT_FOUND", "Không tìm thấy rule"))
		return
	}

	// Toggle trạng thái
	rule.IsActive = !rule.IsActive

	if err := h.ruleRepo.Update(ctx, rule); err != nil {
		h.logger.Error("failed to toggle rule",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.Error("DB_ERROR", "Không thể thay đổi trạng thái rule"))
		return
	}

	h.logger.Info("rule toggled",
		zap.String("request_id", requestID),
		zap.String("rule_id", ruleID.String()),
		zap.Bool("is_active", rule.IsActive),
	)

	c.JSON(http.StatusOK, dto.Success(gin.H{
		"message":   "Đã thay đổi trạng thái rule",
		"is_active": rule.IsActive,
	}))
}

// ===========================================================================
// Route Registration
// ===========================================================================

// RegisterRoutes đăng ký routes cho rule handler
func (h *RuleHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rules := rg.Group("/rules")
	{
		rules.GET("", h.List)                  // Danh sách rules
		rules.GET("/trash", h.ListDeleted)     // Danh sách rules đã xóa
		rules.GET("/:id", h.Get)               // Chi tiết rule
		rules.POST("", h.Create)               // Tạo rule mới
		rules.PUT("/:id", h.Update)            // Cập nhật rule
		rules.DELETE("/:id", h.Delete)         // Xóa rule (soft delete)
		rules.PATCH("/:id/toggle", h.Toggle)   // Bật/tắt rule
		rules.POST("/:id/restore", h.Restore)  // Khôi phục rule đã xóa
	}
}
