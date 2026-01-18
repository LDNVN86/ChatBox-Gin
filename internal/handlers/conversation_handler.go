package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"chatbox-gin/internal/channel"
	"chatbox-gin/internal/dto"
	"chatbox-gin/internal/middleware"
	"chatbox-gin/internal/models"
	"chatbox-gin/internal/realtime"
	"chatbox-gin/internal/repositories"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ===========================================================================
// Conversation Handler
// Quản lý API cho conversations và messages
// ===========================================================================

// ConversationHandler xử lý các endpoint conversation
type ConversationHandler struct {
	conversationRepo   repositories.ConversationRepository
	messageRepo        repositories.MessageRepository
	participantRepo    repositories.ParticipantRepository
	channelAccountRepo repositories.ChannelAccountRepository
	channelRegistry    *channel.Registry
	publisher          realtime.Publisher
	logger             *zap.Logger
}

// NewConversationHandler tạo handler mới
func NewConversationHandler(
	conversationRepo repositories.ConversationRepository,
	messageRepo repositories.MessageRepository,
	participantRepo repositories.ParticipantRepository,
	channelAccountRepo repositories.ChannelAccountRepository,
	channelRegistry *channel.Registry,
	publisher realtime.Publisher,
	logger *zap.Logger,
) *ConversationHandler {
	return &ConversationHandler{
		conversationRepo:   conversationRepo,
		messageRepo:        messageRepo,
		participantRepo:    participantRepo,
		channelAccountRepo: channelAccountRepo,
		channelRegistry:    channelRegistry,
		publisher:          publisher,
		logger:             logger,
	}
}

// ===========================================================================
// Error Helper
// Xử lý lỗi DB và trả về response phù hợp
// ===========================================================================

// handleDBError xử lý lỗi từ database và trả về error response
// Giúp user hiểu được vấn đề thay vì thấy lỗi kỹ thuật
func (h *ConversationHandler) handleDBError(c *gin.Context, requestID string, err error, entity string) {
	// Ghi log lỗi chi tiết cho developer
	h.logger.Error("database error",
		zap.String("request_id", requestID),
		zap.String("entity", entity),
		zap.Error(err),
	)

	// Phân loại lỗi để trả về message phù hợp cho user
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, dto.Error(
			"NOT_FOUND",
			"Không tìm thấy "+entity+" yêu cầu",
		))
	case errors.Is(err, gorm.ErrDuplicatedKey):
		c.JSON(http.StatusConflict, dto.Error(
			"DUPLICATE",
			entity+" đã tồn tại",
		))
	default:
		c.JSON(http.StatusInternalServerError, dto.Error(
			"DB_ERROR",
			"Có lỗi khi truy vấn dữ liệu. Vui lòng thử lại sau.",
		))
	}
}

// ===========================================================================
// Request DTOs
// ===========================================================================

// ListConversationsQuery query params cho list conversations
type ListConversationsQuery struct {
	WorkspaceID string `form:"workspace_id" binding:"required"`
	Status      string `form:"status" binding:"omitempty,oneof=open pending closed bot_paused"`
	AssignedTo  string `form:"assigned_to"`
	Page        int    `form:"page"`
	Limit       int    `form:"limit"`
}

// UpdateConversationBody body cho update conversation
type UpdateConversationBody struct {
	Status     *string    `json:"status" binding:"omitempty,oneof=open pending closed bot_paused"`
	AssignedTo *uuid.UUID `json:"assigned_to"`
	Priority   *string    `json:"priority" binding:"omitempty,oneof=low normal high urgent"`
}

// SendMessageBody body cho gửi tin nhắn
type SendMessageBody struct {
	Content     string `json:"content" binding:"required,min=1,max=5000"`
	ContentType string `json:"content_type" binding:"omitempty,oneof=text image file"`
}

// ===========================================================================
// Handlers
// ===========================================================================

// List lấy danh sách conversations
// GET /api/v1/conversations?workspace_id=xxx&status=open&page=1&limit=20
func (h *ConversationHandler) List(c *gin.Context) {
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	var query ListConversationsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", "workspace_id là bắt buộc"))
		return
	}

	workspaceID, err := uuid.Parse(query.WorkspaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", "workspace_id không hợp lệ"))
		return
	}

	// Set defaults
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 || query.Limit > 100 {
		query.Limit = 20
	}

	// Build find options
	opts := repositories.FindOptions{
		Offset:   (query.Page - 1) * query.Limit,
		Limit:    query.Limit,
		OrderBy:  "last_message_at",
		OrderDir: "desc",
		Filters:  make(map[string]interface{}),
	}

	if query.Status != "" {
		opts.Filters["status"] = query.Status
	}
	if query.AssignedTo != "" {
		if assignedID, err := uuid.Parse(query.AssignedTo); err == nil {
			opts.Filters["assigned_to"] = assignedID
		}
	}

	conversations, total, err := h.conversationRepo.FindByWorkspace(ctx, workspaceID, opts)
	if err != nil {
		h.handleDBError(c, requestID, err, "conversations")
		return
	}

	c.JSON(http.StatusOK, dto.SuccessWithMeta(
		conversations,
		dto.NewMeta(query.Page, query.Limit, total),
	))
}

// Get lấy chi tiết conversation
// GET /api/v1/conversations/:id
func (h *ConversationHandler) Get(c *gin.Context) {
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", "Conversation ID không hợp lệ"))
		return
	}

	conversation, err := h.conversationRepo.FindByID(ctx, conversationID)
	if err != nil {
		h.handleDBError(c, requestID, err, "conversation")
		return
	}

	c.JSON(http.StatusOK, dto.Success(conversation))
}

// Update cập nhật conversation
// PATCH /api/v1/conversations/:id
func (h *ConversationHandler) Update(c *gin.Context) {
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", "Conversation ID không hợp lệ"))
		return
	}

	conversation, err := h.conversationRepo.FindByID(ctx, conversationID)
	if err != nil {
		h.handleDBError(c, requestID, err, "conversation")
		return
	}

	var body UpdateConversationBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", err.Error()))
		return
	}

	// Cập nhật các fields
	if body.Status != nil {
		conversation.Status = models.ConversationStatus(*body.Status)
	}
	if body.AssignedTo != nil {
		conversation.AssignedTo = body.AssignedTo
	}
	if body.Priority != nil {
		conversation.Priority = models.Priority(*body.Priority)
	}

	if err := h.conversationRepo.Update(ctx, conversation); err != nil {
		h.handleDBError(c, requestID, err, "conversation")
		return
	}

	h.logger.Info("conversation updated",
		zap.String("request_id", requestID),
		zap.String("conversation_id", conversationID.String()),
	)

	c.JSON(http.StatusOK, dto.Success(conversation))
}

// ListMessages lấy danh sách messages của conversation
// GET /api/v1/conversations/:id/messages
func (h *ConversationHandler) ListMessages(c *gin.Context) {
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", "Conversation ID không hợp lệ"))
		return
	}

	// Kiểm tra conversation tồn tại
	_, err = h.conversationRepo.FindByID(ctx, conversationID)
	if err != nil {
		h.handleDBError(c, requestID, err, "conversation")
		return
	}

	// Parse pagination
	page := 1
	limit := 50
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	opts := repositories.FindOptions{
		Offset:   (page - 1) * limit,
		Limit:    limit,
		OrderBy:  "created_at",
		OrderDir: "asc", // Messages theo thứ tự thời gian
	}

	messages, total, err := h.messageRepo.FindByConversation(ctx, conversationID, opts)
	if err != nil {
		h.handleDBError(c, requestID, err, "messages")
		return
	}

	c.JSON(http.StatusOK, dto.SuccessWithMeta(
		messages,
		dto.NewMeta(page, limit, total),
	))
}

// SendMessage gửi tin nhắn từ agent
// POST /api/v1/conversations/:id/messages
func (h *ConversationHandler) SendMessage(c *gin.Context) {
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", "Conversation ID không hợp lệ"))
		return
	}

	// Kiểm tra conversation tồn tại
	conversation, err := h.conversationRepo.FindByID(ctx, conversationID)
	if err != nil {
		h.handleDBError(c, requestID, err, "conversation")
		return
	}

	var body SendMessageBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", err.Error()))
		return
	}

	// Tạo message mới
	contentType := models.ContentText
	if body.ContentType != "" {
		contentType = models.ContentType(body.ContentType)
	}

	message := &models.Message{
		ConversationID: conversationID,
		Direction:      models.DirectionOut,
		SenderType:     models.SenderAgent,
		Content:        &body.Content,
		ContentType:    contentType,
		// TODO: Lấy SenderID từ JWT auth
	}

	if err := h.messageRepo.Create(ctx, message); err != nil {
		h.handleDBError(c, requestID, err, "message")
		return
	}

	// Gửi message qua channel (Facebook, Zalo, etc.)
	go h.sendToChannel(context.Background(), conversation, message)

	// Cập nhật last message
	conversation.UpdateLastMessage(body.Content, message.CreatedAt)
	_ = h.conversationRepo.Update(ctx, conversation)

	// Publish realtime event
	if h.publisher != nil {
		go func() {
			content := ""
			if message.Content != nil {
				content = *message.Content
			}
			event := &realtime.MessageEvent{
				MessageID:      message.ID,
				ConversationID: conversationID,
				Direction:      string(message.Direction),
				SenderType:     string(message.SenderType),
				Content:        content,
				CreatedAt:      message.CreatedAt,
			}
			if err := h.publisher.PublishNewMessage(conversation.WorkspaceID, event); err != nil {
				h.logger.Warn("failed to publish agent message event", zap.Error(err))
			}
		}()
	}

	h.logger.Info("message sent",
		zap.String("request_id", requestID),
		zap.String("message_id", message.ID.String()),
	)

	c.JSON(http.StatusCreated, dto.Success(message))
}

// sendToChannel gửi message qua channel tương ứng (FB, Zalo, etc.)
func (h *ConversationHandler) sendToChannel(ctx context.Context, conv *models.Conversation, msg *models.Message) {
	// Lấy participant để có recipient ID
	participant, err := h.participantRepo.FindByID(ctx, conv.ParticipantID)
	if err != nil {
		h.logger.Warn("sendToChannel: failed to get participant", zap.Error(err))
		return
	}

	// Lấy channel account để có credentials và channel type
	channelAccount, err := h.channelAccountRepo.FindByID(ctx, conv.ChannelAccountID)
	if err != nil {
		h.logger.Warn("sendToChannel: failed to get channel account", zap.Error(err))
		return
	}

	// Lấy channel từ registry
	ch, err := h.channelRegistry.Get(string(channelAccount.ChannelType))
	if err != nil {
		h.logger.Warn("sendToChannel: channel not found", zap.String("type", string(channelAccount.ChannelType)))
		return
	}

	// Tạo outbound message
	content := ""
	if msg.Content != nil {
		content = *msg.Content
	}
	outbound := &channel.OutboundMessage{
		RecipientID: participant.ChannelUserID,
		Content:     content,
		ContentType: string(msg.ContentType),
	}

	// Chuẩn bị credentials
	credentials := map[string]string{
		"page_access_token": channelAccount.Credentials.PageAccessToken,
	}

	// Gửi qua channel
	result, err := ch.Send(ctx, outbound, credentials)
	if err != nil {
		h.logger.Warn("sendToChannel: send failed", zap.Error(err))
		return
	}

	if result.Success {
		// Cập nhật channel_message_id cho message
		msg.ChannelMessageID = &result.ChannelMessageID
		if updateErr := h.messageRepo.Update(ctx, msg); updateErr != nil {
			h.logger.Warn("sendToChannel: failed to update message ID", zap.Error(updateErr))
		}
		h.logger.Info("message sent to channel",
			zap.String("channel", string(channelAccount.ChannelType)),
			zap.String("channel_message_id", result.ChannelMessageID),
		)
	} else {
		h.logger.Warn("sendToChannel: channel returned error", zap.Error(result.Error))
	}
}

// ===========================================================================
// Bot Control
// ===========================================================================

// ToggleBotRequest body for toggling bot
type ToggleBotRequest struct {
	Enabled bool    `json:"enabled"`
	Reason  *string `json:"reason,omitempty"`
}

// ToggleBot enables or disables bot for a conversation
// POST /api/v1/conversations/:id/bot
func (h *ConversationHandler) ToggleBot(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_ID", "Invalid conversation ID"))
		return
	}

	var req ToggleBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", err.Error()))
		return
	}

	conversation, err := h.conversationRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, dto.Error("NOT_FOUND", "Conversation not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.Error("INTERNAL_ERROR", "Failed to find conversation"))
		return
	}

	if req.Enabled {
		conversation.ResumeBot()
	} else {
		reason := "Paused by agent"
		if req.Reason != nil {
			reason = *req.Reason
		}
		conversation.PauseBot(reason)
	}

	if err := h.conversationRepo.Update(c.Request.Context(), conversation); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error("INTERNAL_ERROR", "Failed to update conversation"))
		return
	}

	h.logger.Info("bot toggled",
		zap.String("conversation_id", id.String()),
		zap.Bool("enabled", req.Enabled),
	)

	c.JSON(http.StatusOK, dto.Success(gin.H{
		"bot_enabled": req.Enabled,
		"status":      conversation.Status,
	}))
}

// ===========================================================================
// Route Registration
// ===========================================================================

// RegisterRoutes đăng ký routes
func (h *ConversationHandler) RegisterRoutes(rg *gin.RouterGroup) {
	conversations := rg.Group("/conversations")
	{
		conversations.GET("", h.List)
		conversations.GET("/:id", h.Get)
		conversations.PATCH("/:id", h.Update)
		conversations.GET("/:id/messages", h.ListMessages)
		conversations.POST("/:id/messages", h.SendMessage)
		conversations.POST("/:id/bot", h.ToggleBot)
	}
}