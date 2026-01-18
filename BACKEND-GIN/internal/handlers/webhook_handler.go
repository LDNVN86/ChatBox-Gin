package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"chatbox-gin/internal/channel"
	"chatbox-gin/internal/dto"
	"chatbox-gin/internal/middleware"
	"chatbox-gin/internal/models"
	"chatbox-gin/internal/repositories"
	"chatbox-gin/internal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ===========================================================================
// Webhook Handler
// Xử lý webhooks từ Facebook, Zalo, etc.
// ===========================================================================

// WebhookHandler xử lý webhook endpoints
type WebhookHandler struct {
	channelRegistry  *channel.Registry
	channelAcctRepo  repositories.ChannelAccountRepository
	messageService   services.MessageService
	fbVerifyToken    string
	logger           *zap.Logger
}

// NewWebhookHandler tạo handler mới
func NewWebhookHandler(
	registry *channel.Registry,
	channelAcctRepo repositories.ChannelAccountRepository,
	messageService services.MessageService,
	fbVerifyToken string,
	logger *zap.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		channelRegistry:  registry,
		channelAcctRepo:  channelAcctRepo,
		messageService:   messageService,
		fbVerifyToken:    fbVerifyToken,
		logger:           logger,
	}
}

// ===========================================================================
// Facebook Webhook
// ===========================================================================

// FacebookVerify xử lý GET request để verify webhook
// GET /webhook/facebook
func (h *WebhookHandler) FacebookVerify(c *gin.Context) {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	h.logger.Info("fb webhook verify",
		zap.String("mode", mode),
		zap.String("token", token),
	)

	if mode == "subscribe" && token == h.fbVerifyToken {
		c.String(http.StatusOK, challenge)
		return
	}

	c.JSON(http.StatusForbidden, dto.Error("FORBIDDEN", "Invalid verify token"))
}

// FacebookWebhook xử lý POST request nhận tin nhắn
// POST /webhook/facebook
func (h *WebhookHandler) FacebookWebhook(c *gin.Context) {
	requestID := middleware.GetRequestID(c)
	ctx := c.Request.Context()

	// Read body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("BAD_REQUEST", "Cannot read body"))
		return
	}

	// Parse payload
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error("BAD_REQUEST", "Invalid JSON"))
		return
	}

	h.logger.Info("fb webhook received",
		zap.String("request_id", requestID),
		zap.Any("payload", payload),
	)

	// Get Facebook channel
	fbChannel, err := h.channelRegistry.Get("facebook")
	if err != nil {
		h.logger.Error("facebook channel not registered", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{"status": "ok"}) // Always return 200 to FB
		return
	}

	// Get channel account from payload (Page ID)
	pageID := h.extractPageID(payload)
	if pageID == "" {
		h.logger.Warn("no page id in payload")
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// Find channel account by channel_id (Page ID)
	channelAcct, err := h.channelAcctRepo.FindByChannelID(ctx, pageID, models.ChannelFacebook)
	if err != nil {
		h.logger.Warn("channel account not found",
			zap.String("page_id", pageID),
			zap.Error(err),
		)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// Normalize message
	inbound, err := fbChannel.Normalize(ctx, channelAcct.ID, payload)
	if err != nil {
		h.logger.Warn("normalize failed", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// Process inbound message
	result, err := h.messageService.ProcessInbound(ctx, channelAcct.WorkspaceID, channelAcct.ID, inbound)
	if err != nil {
		h.logger.Error("process inbound failed",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
	} else {
		h.logger.Info("fb message processed",
			zap.String("request_id", requestID),
			zap.String("message_id", result.MessageID.String()),
			zap.Bool("bot_replied", result.BotReplied),
		)
	}

	// Always return 200 to FB
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// extractPageID lấy Page ID từ FB webhook payload
func (h *WebhookHandler) extractPageID(payload map[string]interface{}) string {
	entries, ok := payload["entry"].([]interface{})
	if !ok || len(entries) == 0 {
		return ""
	}
	
	entry, ok := entries[0].(map[string]interface{})
	if !ok {
		return ""
	}
	
	pageID, _ := entry["id"].(string)
	return pageID
}

// ===========================================================================
// Route Registration
// ===========================================================================

// RegisterRoutes đăng ký webhook routes
func (h *WebhookHandler) RegisterRoutes(rg *gin.RouterGroup) {
	webhook := rg.Group("/webhook")
	{
		// Facebook webhooks
		webhook.GET("/facebook", h.FacebookVerify)
		webhook.POST("/facebook", h.FacebookWebhook)

		// TODO: Zalo webhooks
		// webhook.GET("/zalo", h.ZaloVerify)
		// webhook.POST("/zalo", h.ZaloWebhook)
	}
}
