package channel

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ===========================================================================
// MockChannel là channel adapter dùng để testing
// Không cần credentials thật, simulate việc gửi/nhận tin nhắn
// ===========================================================================

// MockChannel implement Channel interface cho mục đích testing
type MockChannel struct {
	logger *zap.Logger

	// sentMessages lưu các tin nhắn đã "gửi" (để testing)
	sentMessages []*OutboundMessage
}

// NewMockChannel tạo một MockChannel mới
func NewMockChannel(logger *zap.Logger) *MockChannel {
	return &MockChannel{
		logger:       logger,
		sentMessages: make([]*OutboundMessage, 0),
	}
}

// Type trả về loại channel - "mock"
func (m *MockChannel) Type() string {
	return "mock"
}

// ===========================================================================
// Normalizer implementation
// ===========================================================================

// Normalize chuyển đổi mock webhook payload thành InboundMessage
// Mock payload có cấu trúc đơn giản để dễ test
//
// Expected payload format:
//
//	{
//	    "sender_id": "mock_user_123",
//	    "sender_name": "Test User",
//	    "message": "Hello bot!",
//	    "message_id": "msg_001",
//	    "timestamp": 1705487400
//	}
func (m *MockChannel) Normalize(
	ctx context.Context,
	channelAccountID uuid.UUID,
	payload map[string]interface{},
) (*InboundMessage, error) {
	// Lấy sender_id (bắt buộc)
	senderID, ok := payload["sender_id"].(string)
	if !ok || senderID == "" {
		return nil, fmt.Errorf("mock payload thiếu 'sender_id'")
	}

	// Lấy message content (bắt buộc)
	content, ok := payload["message"].(string)
	if !ok {
		content = "" // Cho phép message rỗng (có thể là attachment only)
	}

	// Lấy message_id (tùy chọn, tự generate nếu không có)
	messageID, ok := payload["message_id"].(string)
	if !ok || messageID == "" {
		messageID = fmt.Sprintf("mock_%s_%d", senderID, time.Now().UnixNano())
	}

	// Lấy sender_name (tùy chọn)
	senderName, _ := payload["sender_name"].(string)

	// Lấy timestamp (tùy chọn, dùng now nếu không có)
	timestamp := time.Now()
	if ts, ok := payload["timestamp"].(float64); ok {
		timestamp = time.Unix(int64(ts), 0)
	}

	// Xử lý attachments nếu có
	var attachments []AttachmentData
	if rawAttachments, ok := payload["attachments"].([]interface{}); ok {
		for _, raw := range rawAttachments {
			if att, ok := raw.(map[string]interface{}); ok {
				attachments = append(attachments, AttachmentData{
					Type:     getString(att, "type"),
					URL:      getString(att, "url"),
					Name:     getString(att, "name"),
					MimeType: getString(att, "mime_type"),
				})
			}
		}
	}

	// Xác định content type
	contentType := "text"
	if len(attachments) > 0 && content == "" {
		contentType = attachments[0].Type
	}

	// Log để debug
	m.logger.Debug("mock channel: đã normalize message",
		zap.String("sender_id", senderID),
		zap.String("message_id", messageID),
		zap.String("content", truncate(content, 50)),
	)

	return &InboundMessage{
		ChannelType:      "mock",
		ChannelMessageID: messageID,
		SenderID:         senderID,
		SenderName:       senderName,
		RecipientID:      channelAccountID.String(),
		Content:          content,
		ContentType:      contentType,
		Attachments:      attachments,
		Timestamp:        timestamp,
		RawPayload:       payload,
	}, nil
}

// ===========================================================================
// Sender implementation
// ===========================================================================

// Send "gửi" tin nhắn (thực tế chỉ log và lưu lại để testing)
// Trong mock channel, không có API thật để gọi
func (m *MockChannel) Send(
	ctx context.Context,
	msg *OutboundMessage,
	credentials map[string]string,
) (*SendResult, error) {
	// Validate input
	if msg.RecipientID == "" {
		return &SendResult{
			Success: false,
			Error:   fmt.Errorf("recipient_id không được để trống"),
		}, nil
	}

	// Generate message ID cho response
	messageID := fmt.Sprintf("mock_sent_%d", time.Now().UnixNano())

	// Log tin nhắn đã gửi
	m.logger.Info("mock channel: đã gửi tin nhắn",
		zap.String("recipient_id", msg.RecipientID),
		zap.String("message_id", messageID),
		zap.String("content", truncate(msg.Content, 100)),
		zap.Int("quick_replies", len(msg.QuickReplies)),
		zap.Int("buttons", len(msg.Buttons)),
	)

	// Lưu vào list để có thể verify trong tests
	m.sentMessages = append(m.sentMessages, msg)

	return &SendResult{
		Success:          true,
		ChannelMessageID: messageID,
	}, nil
}

// ===========================================================================
// SignatureVerifier implementation
// ===========================================================================

// Verify luôn trả về true cho mock channel (không cần xác thực)
func (m *MockChannel) Verify(signature string, body []byte, secret string) bool {
	// Mock channel không cần verify signature
	// Trong môi trường development/testing, chấp nhận mọi request
	return true
}

// ===========================================================================
// Testing helpers
// ===========================================================================

// GetSentMessages trả về danh sách tin nhắn đã gửi (để testing)
func (m *MockChannel) GetSentMessages() []*OutboundMessage {
	return m.sentMessages
}

// ClearSentMessages xóa danh sách tin nhắn đã gửi
func (m *MockChannel) ClearSentMessages() {
	m.sentMessages = make([]*OutboundMessage, 0)
}

// GetLastSentMessage trả về tin nhắn cuối cùng đã gửi
func (m *MockChannel) GetLastSentMessage() *OutboundMessage {
	if len(m.sentMessages) == 0 {
		return nil
	}
	return m.sentMessages[len(m.sentMessages)-1]
}

// ===========================================================================
// Helper functions
// ===========================================================================

// getString lấy string value từ map, trả về empty string nếu không tìm thấy
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// truncate cắt ngắn string nếu dài hơn maxLen
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
