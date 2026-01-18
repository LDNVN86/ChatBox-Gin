package channel

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ===========================================================================
// Các interfaces cho hệ thống channel messaging
// Channel là một kênh giao tiếp (Facebook, Zalo, Web, Mock, etc.)
// ===========================================================================

// InboundMessage đại diện cho tin nhắn nhận được từ khách hàng
// Đây là cấu trúc chuẩn hóa, bất kể nguồn gốc từ channel nào
type InboundMessage struct {
	// ChannelType loại channel (facebook/zalo/web/mock)
	ChannelType string

	// ChannelMessageID là ID tin nhắn gốc từ channel (dùng cho deduplication)
	ChannelMessageID string

	// SenderID là ID người gửi trên channel đó
	SenderID string

	// SenderName tên hiển thị của người gửi (nếu có)
	SenderName string

	// SenderAvatar URL avatar của người gửi (nếu có)
	SenderAvatar string

	// RecipientID là ID page/OA nhận tin nhắn
	RecipientID string

	// ThreadID là ID cuộc hội thoại trên channel (nếu có)
	ThreadID string

	// Content nội dung text của tin nhắn
	Content string

	// ContentType loại nội dung (text/image/file/etc.)
	ContentType string

	// Attachments danh sách file đính kèm
	Attachments []AttachmentData

	// Timestamp thời điểm gửi tin nhắn
	Timestamp time.Time

	// RawPayload dữ liệu gốc từ webhook (để debug)
	RawPayload map[string]interface{}
}

// AttachmentData đại diện cho file đính kèm trong tin nhắn
type AttachmentData struct {
	Type     string `json:"type"`      // image, file, video, audio
	URL      string `json:"url"`       // URL download
	Name     string `json:"name"`      // Tên file
	Size     int64  `json:"size"`      // Kích thước (bytes)
	MimeType string `json:"mime_type"` // MIME type
}

// OutboundMessage đại diện cho tin nhắn gửi đi cho khách hàng
type OutboundMessage struct {
	// RecipientID là ID người nhận trên channel
	RecipientID string

	// ThreadID là ID cuộc hội thoại (nếu cần)
	ThreadID string

	// Content nội dung text
	Content string

	// ContentType loại nội dung
	ContentType string

	// Attachments file đính kèm
	Attachments []AttachmentData

	// QuickReplies các nút quick reply
	QuickReplies []QuickReplyData

	// Buttons các nút bấm
	Buttons []ButtonData

	// Metadata thông tin bổ sung
	Metadata map[string]interface{}
}

// QuickReplyData đại diện cho nút quick reply
type QuickReplyData struct {
	Title   string `json:"title"`   // Text hiển thị
	Payload string `json:"payload"` // Giá trị gửi lại khi bấm
}

// ButtonData đại diện cho nút bấm trong tin nhắn
type ButtonData struct {
	Type    string `json:"type"`    // postback, web_url, phone_number
	Title   string `json:"title"`   // Text hiển thị
	Payload string `json:"payload"` // Giá trị (URL hoặc payload)
}

// SendResult kết quả gửi tin nhắn
type SendResult struct {
	// Success tin nhắn đã gửi thành công chưa
	Success bool

	// ChannelMessageID là ID tin nhắn được tạo bởi channel
	ChannelMessageID string

	// Error lỗi nếu có
	Error error
}

// ===========================================================================
// Interfaces chính
// ===========================================================================

// Normalizer chuyển đổi webhook payload thành InboundMessage chuẩn
// Mỗi channel type sẽ có implementation riêng
type Normalizer interface {
	// Normalize chuyển đổi raw payload thành InboundMessage
	// channelAccountID là ID của channel account nhận webhook
	Normalize(ctx context.Context, channelAccountID uuid.UUID, payload map[string]interface{}) (*InboundMessage, error)
}

// Sender gửi tin nhắn đi cho khách hàng
// Mỗi channel type sẽ có implementation riêng để gọi API tương ứng
type Sender interface {
	// Send gửi tin nhắn và trả về kết quả
	// credentials là thông tin xác thực để gọi API
	Send(ctx context.Context, msg *OutboundMessage, credentials map[string]string) (*SendResult, error)
}

// SignatureVerifier xác thực chữ ký webhook
// Đảm bảo webhook đến từ đúng nguồn (FB, Zalo) và không bị tamper
type SignatureVerifier interface {
	// Verify kiểm tra chữ ký của request
	// signature là giá trị từ header (X-Hub-Signature, etc.)
	// body là raw body của request
	// secret là secret key để verify
	Verify(signature string, body []byte, secret string) bool
}

// Channel là interface tổng hợp cho một channel adapter
// Mỗi channel type (facebook, zalo, mock) sẽ implement interface này
type Channel interface {
	Normalizer
	Sender
	SignatureVerifier

	// Type trả về loại channel
	Type() string
}
