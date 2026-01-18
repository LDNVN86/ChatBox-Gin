package bot

import (
	"chatbox-gin/internal/channel"
	"chatbox-gin/internal/models"
)

// ===========================================================================
// Response Builder
// Tạo OutboundMessage từ rule response config
// Tách riêng để dễ test và tái sử dụng
// ===========================================================================

// ResponseBuilder interface xây dựng response từ rule
type ResponseBuilder interface {
	// BuildFromRule tạo OutboundMessage từ rule
	BuildFromRule(rule *models.Rule, recipientID string) *channel.OutboundMessage
}

// ===========================================================================
// Response Builder Implementation
// ===========================================================================

// responseBuilder triển khai ResponseBuilder
type responseBuilder struct{}

// NewResponseBuilder tạo instance mới của ResponseBuilder
func NewResponseBuilder() ResponseBuilder {
	return &responseBuilder{}
}

// BuildFromRule tạo OutboundMessage từ rule response config
func (b *responseBuilder) BuildFromRule(rule *models.Rule, recipientID string) *channel.OutboundMessage {
	msg := &channel.OutboundMessage{
		RecipientID: recipientID,
		ContentType: "text",
	}

	switch rule.ResponseType {
	case models.ResponseText:
		msg.Content = rule.ResponseConfig.Text

	case models.ResponseTemplate:
		// TODO: Xử lý template rendering
		msg.Content = rule.ResponseConfig.Text

	case models.ResponseHandoff:
		// Tin nhắn khi handoff
		msg.Content = rule.ResponseConfig.Message
		if msg.Content == "" {
			msg.Content = "Đang chuyển bạn đến nhân viên hỗ trợ..."
		}
	}

	return msg
}
