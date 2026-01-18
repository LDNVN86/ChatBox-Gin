package handlers

import (
	"net/http"

	"chatbox-gin/internal/channel"
	"chatbox-gin/internal/dto"
	"chatbox-gin/internal/middleware"
	"chatbox-gin/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ===========================================================================
// MockHandler xử lý các endpoint liên quan đến Mock Channel
// Dùng để testing mà không cần FB/Zalo credentials thật
// ===========================================================================

// MockHandler chứa dependencies cần thiết
type MockHandler struct {
	registry       *channel.Registry
	messageService services.MessageService
	logger         *zap.Logger
}

// NewMockHandler tạo MockHandler mới
func NewMockHandler(registry *channel.Registry, messageService services.MessageService, logger *zap.Logger) *MockHandler {
	return &MockHandler{
		registry:       registry,
		messageService: messageService,
		logger:         logger,
	}
}

// ===========================================================================
// Request/Response DTOs
// ===========================================================================

// MockInboundRequest là payload gửi đến mock endpoint
// Simulate việc khách hàng gửi tin nhắn
type MockInboundRequest struct {
	// WorkspaceID ID của workspace
	WorkspaceID uuid.UUID `json:"workspace_id" binding:"required"`

	// ChannelAccountID ID của channel account (mock channel)
	ChannelAccountID uuid.UUID `json:"channel_account_id" binding:"required"`

	// SenderID ID của người gửi (mock user)
	SenderID string `json:"sender_id" binding:"required"`

	// SenderName tên hiển thị (tùy chọn)
	SenderName string `json:"sender_name"`

	// Message nội dung tin nhắn
	Message string `json:"message" binding:"required"`

	// MessageID ID tin nhắn (tùy chọn, sẽ auto generate nếu không có)
	MessageID string `json:"message_id"`
}

// MockOutboundRequest là payload để gửi tin nhắn từ bot/agent
type MockOutboundRequest struct {
	// RecipientID ID người nhận
	RecipientID string `json:"recipient_id" binding:"required"`

	// Message nội dung tin nhắn
	Message string `json:"message" binding:"required"`

	// QuickReplies các nút quick reply (tùy chọn)
	QuickReplies []channel.QuickReplyData `json:"quick_replies"`
}

// ===========================================================================
// Handlers
// ===========================================================================

// SimulateInbound xử lý POST /api/mock/inbound
// Simulate việc khách hàng gửi tin nhắn đến hệ thống
//
// Flow:
// 1. Validate request
// 2. Lấy mock channel từ registry
// 3. Normalize payload thành InboundMessage
// 4. Gọi MessageService.ProcessInbound để xử lý đầy đủ flow:
//    - Tìm/tạo Participant
//    - Tìm/tạo Conversation
//    - Lưu Message vào DB
//    - Chạy Bot Rule Engine
//    - Gửi response (nếu có)
// 5. Trả về kết quả
func (h *MockHandler) SimulateInbound(c *gin.Context) {
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	// Bind và validate request
	var req MockInboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("mock inbound: invalid request",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", err.Error()))
		return
	}

	// Lấy mock channel từ registry
	ch, err := h.registry.Get("mock")
	if err != nil {
		h.logger.Error("mock inbound: channel not found",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.Error("CHANNEL_NOT_FOUND", "Mock channel chưa được đăng ký"))
		return
	}

	// Tạo payload để normalize
	payload := map[string]interface{}{
		"sender_id":   req.SenderID,
		"sender_name": req.SenderName,
		"message":     req.Message,
		"message_id":  req.MessageID,
	}

	// Normalize thành InboundMessage chuẩn
	inbound, err := ch.Normalize(ctx, req.ChannelAccountID, payload)
	if err != nil {
		h.logger.Error("mock inbound: normalize failed",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, dto.Error("NORMALIZE_FAILED", err.Error()))
		return
	}

	h.logger.Info("mock inbound: message received",
		zap.String("request_id", requestID),
		zap.String("sender_id", inbound.SenderID),
		zap.String("message_id", inbound.ChannelMessageID),
		zap.String("content", inbound.Content),
	)

	// Xử lý message qua MessageService (bao gồm bot response)
	result, err := h.messageService.ProcessInbound(ctx, req.WorkspaceID, req.ChannelAccountID, inbound)
	if err != nil {
		h.logger.Error("mock inbound: process failed",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.Error("PROCESS_FAILED", err.Error()))
		return
	}

	h.logger.Info("mock inbound: message processed",
		zap.String("request_id", requestID),
		zap.String("message_id", result.MessageID.String()),
		zap.Bool("participant_created", result.ParticipantCreated),
		zap.Bool("conversation_created", result.ConversationCreated),
		zap.Bool("bot_replied", result.BotReplied),
		zap.Bool("bot_handoff", result.BotHandoff),
	)

	// Trả về kết quả
	response := gin.H{
		"message":              "Tin nhắn đã được xử lý",
		"workspace_id":         req.WorkspaceID,
		"participant_id":       result.ParticipantID,
		"participant_created":  result.ParticipantCreated,
		"conversation_id":      result.ConversationID,
		"conversation_created": result.ConversationCreated,
		"message_id":           result.MessageID,
		"bot_replied":          result.BotReplied,
		"bot_handoff":          result.BotHandoff,
	}

	// Thêm bot response nếu có
	if result.ResponseSent != nil {
		response["bot_response"] = result.ResponseSent.Content
	}

	c.JSON(http.StatusOK, dto.Success(response))
}

// SimulateOutbound xử lý POST /api/mock/outbound
// Simulate việc hệ thống gửi tin nhắn cho khách hàng
func (h *MockHandler) SimulateOutbound(c *gin.Context) {
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	// Bind và validate request
	var req MockOutboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("mock outbound: invalid request",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, dto.Error("INVALID_REQUEST", err.Error()))
		return
	}

	// Lấy mock channel từ registry
	ch, err := h.registry.Get("mock")
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error("CHANNEL_NOT_FOUND", "Mock channel chưa được đăng ký"))
		return
	}

	// Tạo OutboundMessage
	outbound := &channel.OutboundMessage{
		RecipientID:  req.RecipientID,
		Content:      req.Message,
		ContentType:  "text",
		QuickReplies: req.QuickReplies,
	}

	// "Gửi" tin nhắn (mock channel chỉ log và lưu lại)
	result, err := ch.Send(ctx, outbound, nil)
	if err != nil {
		h.logger.Error("mock outbound: send failed",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, dto.Error("SEND_FAILED", err.Error()))
		return
	}

	if !result.Success {
		c.JSON(http.StatusBadRequest, dto.Error("SEND_FAILED", result.Error.Error()))
		return
	}

	h.logger.Info("mock outbound: message sent",
		zap.String("request_id", requestID),
		zap.String("recipient_id", req.RecipientID),
		zap.String("message_id", result.ChannelMessageID),
	)

	c.JSON(http.StatusOK, dto.Success(gin.H{
		"message":    "Tin nhắn đã được gửi",
		"message_id": result.ChannelMessageID,
	}))
}

// GetSentMessages xử lý GET /api/mock/sent
// Trả về danh sách tin nhắn đã gửi (để testing/debug)
func (h *MockHandler) GetSentMessages(c *gin.Context) {
	ch, err := h.registry.Get("mock")
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error("CHANNEL_NOT_FOUND", "Mock channel chưa được đăng ký"))
		return
	}

	// Type assertion để lấy MockChannel
	mockCh, ok := ch.(*channel.MockChannel)
	if !ok {
		c.JSON(http.StatusInternalServerError, dto.Error("INVALID_CHANNEL", "Channel không phải MockChannel"))
		return
	}

	messages := mockCh.GetSentMessages()
	c.JSON(http.StatusOK, dto.Success(gin.H{
		"count":    len(messages),
		"messages": messages,
	}))
}

// ClearSentMessages xử lý DELETE /api/mock/sent
// Xóa danh sách tin nhắn đã gửi
func (h *MockHandler) ClearSentMessages(c *gin.Context) {
	ch, err := h.registry.Get("mock")
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error("CHANNEL_NOT_FOUND", "Mock channel chưa được đăng ký"))
		return
	}

	mockCh, ok := ch.(*channel.MockChannel)
	if !ok {
		c.JSON(http.StatusInternalServerError, dto.Error("INVALID_CHANNEL", "Channel không phải MockChannel"))
		return
	}

	mockCh.ClearSentMessages()
	c.JSON(http.StatusOK, dto.Success(gin.H{
		"message": "Đã xóa tất cả tin nhắn",
	}))
}

// ===========================================================================
// Route registration helper
// ===========================================================================

// RegisterRoutes đăng ký các routes cho mock handler
func (h *MockHandler) RegisterRoutes(rg *gin.RouterGroup) {
	mock := rg.Group("/mock")
	{
		// Simulate khách hàng gửi tin nhắn
		mock.POST("/inbound", h.SimulateInbound)

		// Simulate hệ thống gửi tin nhắn
		mock.POST("/outbound", h.SimulateOutbound)

		// Debug: xem tin nhắn đã gửi
		mock.GET("/sent", h.GetSentMessages)

		// Debug: xóa tin nhắn đã gửi
		mock.DELETE("/sent", h.ClearSentMessages)
	}
}
