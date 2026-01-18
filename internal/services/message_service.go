package services

import (
	"context"

	"chatbox-gin/internal/channel"

	"github.com/google/uuid"
)

// ===========================================================================
// Message Service Interface
// Xử lý luồng chính: nhận message -> lưu DB -> match rule -> gửi response
// ===========================================================================

// ProcessResult kết quả xử lý message
type ProcessResult struct {
	// ParticipantID ID participant (mới tạo hoặc có sẵn)
	ParticipantID uuid.UUID

	// ParticipantCreated participant mới được tạo
	ParticipantCreated bool

	// ConversationID ID conversation (mới tạo hoặc có sẵn)
	ConversationID uuid.UUID

	// ConversationCreated conversation mới được tạo
	ConversationCreated bool

	// MessageID ID message đã lưu
	MessageID uuid.UUID

	// BotReplied bot đã trả lời hay chưa
	BotReplied bool

	// BotHandoff bot đã chuyển cho agent
	BotHandoff bool

	// ResponseSent tin nhắn response đã gửi (nếu có)
	ResponseSent *channel.OutboundMessage
}

// MessageService interface cho message processing
type MessageService interface {
	// ProcessInbound xử lý inbound message từ channel
	// Flow: normalize -> find/create participant -> find/create conversation -> save message -> match rule -> send response
	ProcessInbound(ctx context.Context, workspaceID, channelAccountID uuid.UUID, inbound *channel.InboundMessage) (*ProcessResult, error)
}
