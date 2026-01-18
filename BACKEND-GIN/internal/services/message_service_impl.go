package services

import (
	"context"
	"time"

	"chatbox-gin/internal/bot"
	"chatbox-gin/internal/channel"
	"chatbox-gin/internal/models"
	"chatbox-gin/internal/realtime"
	"chatbox-gin/internal/repositories"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ===========================================================================
// Message Service Implementation
// Xử lý toàn bộ luồng nhận và trả lời tin nhắn
// ===========================================================================

// messageService triển khai MessageService
type messageService struct {
	participantRepo    repositories.ParticipantRepository
	conversationRepo   repositories.ConversationRepository
	messageRepo        repositories.MessageRepository
	channelAccountRepo repositories.ChannelAccountRepository
	channelRegistry    *channel.Registry
	botResponder       bot.Responder
	publisher          realtime.Publisher
	logger             *zap.Logger
}

// NewMessageService tạo instance mới của MessageService
func NewMessageService(
	participantRepo repositories.ParticipantRepository,
	conversationRepo repositories.ConversationRepository,
	messageRepo repositories.MessageRepository,
	channelAccountRepo repositories.ChannelAccountRepository,
	channelRegistry *channel.Registry,
	botResponder bot.Responder,
	publisher realtime.Publisher,
	logger *zap.Logger,
) MessageService {
	return &messageService{
		participantRepo:    participantRepo,
		conversationRepo:   conversationRepo,
		messageRepo:        messageRepo,
		channelAccountRepo: channelAccountRepo,
		channelRegistry:    channelRegistry,
		botResponder:       botResponder,
		publisher:          publisher,
		logger:             logger,
	}
}

// ProcessInbound xử lý inbound message
func (s *messageService) ProcessInbound(ctx context.Context, workspaceID, channelAccountID uuid.UUID, inbound *channel.InboundMessage) (*ProcessResult, error) {
	result := &ProcessResult{}

	// 1. Tìm hoặc tạo Participant
	participant, participantCreated, err := s.findOrCreateParticipant(ctx, workspaceID, channelAccountID, inbound)
	if err != nil {
		return nil, err
	}
	result.ParticipantID = participant.ID
	result.ParticipantCreated = participantCreated

	// 2. Tìm hoặc tạo Conversation
	conversation, conversationCreated, err := s.findOrCreateConversation(ctx, workspaceID, channelAccountID, participant.ID)
	if err != nil {
		return nil, err
	}
	result.ConversationID = conversation.ID
	result.ConversationCreated = conversationCreated

	// 3. Lưu Message
	message, err := s.saveMessage(ctx, conversation.ID, inbound)
	if err != nil {
		return nil, err
	}
	result.MessageID = message.ID

	// 4. Cập nhật conversation last message
	if err := s.updateConversationLastMessage(ctx, conversation, inbound); err != nil {
		s.logger.Warn("failed to update conversation last message", zap.Error(err))
	}

	// 5. Publish realtime event cho FE
	if s.publisher != nil {
		go func() {
			event := &realtime.MessageEvent{
				MessageID:       message.ID,
				ConversationID:  conversation.ID,
				Direction:       string(message.Direction),
				SenderType:      string(message.SenderType),
				Content:         inbound.Content,
				CreatedAt:       message.CreatedAt,
				ParticipantName: inbound.SenderName,
			}
			if err := s.publisher.PublishNewMessage(workspaceID, event); err != nil {
				s.logger.Warn("failed to publish new message event", zap.Error(err))
			}
		}()
	}

	// 5. Xử lý bot response (nếu conversation không bị pause)
	if !conversation.IsBotPaused() {
		botResponse, err := s.processBotResponse(ctx, workspaceID, channelAccountID, conversation, inbound, message)
		if err != nil {
			s.logger.Warn("bot response failed", zap.Error(err))
		} else if botResponse != nil {
			result.BotReplied = botResponse.ShouldReply
			result.BotHandoff = botResponse.ShouldHandoff
			result.ResponseSent = botResponse.ResponseSent
		}
	}

	s.logger.Info("inbound message processed",
		zap.String("message_id", result.MessageID.String()),
		zap.Bool("participant_created", result.ParticipantCreated),
		zap.Bool("conversation_created", result.ConversationCreated),
		zap.Bool("bot_replied", result.BotReplied),
	)

	return result, nil
}

// findOrCreateParticipant tìm hoặc tạo participant
func (s *messageService) findOrCreateParticipant(ctx context.Context, workspaceID, channelAccountID uuid.UUID, inbound *channel.InboundMessage) (*models.Participant, bool, error) {
	participant := &models.Participant{
		WorkspaceID:      workspaceID,
		ChannelAccountID: channelAccountID,
		ChannelUserID:    inbound.SenderID,
		FirstSeenAt:      time.Now(),
	}

	// Set optional fields từ inbound nếu có
	if inbound.SenderName != "" {
		participant.Name = &inbound.SenderName
	}
	if inbound.SenderAvatar != "" {
		participant.AvatarURL = &inbound.SenderAvatar
	}

	existing, created, err := s.participantRepo.FindOrCreate(ctx, participant)
	if err != nil {
		return nil, false, err
	}

	// Fetch profile từ Facebook nếu chưa có name (async)
	if existing.Name == nil && inbound.ChannelType == "facebook" {
		go s.fetchAndUpdateFBProfile(existing, channelAccountID)
	}

	return existing, created, nil
}

// fetchAndUpdateFBProfile gọi Facebook Graph API để lấy profile và update participant
func (s *messageService) fetchAndUpdateFBProfile(participant *models.Participant, channelAccountID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Lấy channel account để có access token
	channelAccount, err := s.channelAccountRepo.FindByID(ctx, channelAccountID)
	if err != nil {
		s.logger.Warn("fetch fb profile: failed to get channel account", zap.Error(err))
		return
	}

	accessToken := channelAccount.Credentials.PageAccessToken
	if accessToken == "" {
		s.logger.Warn("fetch fb profile: missing page_access_token")
		return
	}

	// Lấy Facebook channel từ registry
	ch, err := s.channelRegistry.Get("facebook")
	if err != nil {
		s.logger.Warn("fetch fb profile: channel not found", zap.Error(err))
		return
	}

	fbChannel, ok := ch.(*channel.FacebookChannel)
	if !ok {
		s.logger.Warn("fetch fb profile: invalid channel type")
		return
	}

	// Gọi Graph API lấy profile
	profile, err := fbChannel.GetUserProfile(ctx, participant.ChannelUserID, accessToken)
	if err != nil {
		s.logger.Warn("fetch fb profile: api error", zap.Error(err))
		return
	}

	// Update participant với name và avatar
	participant.Name = &profile.Name
	participant.AvatarURL = &profile.ProfilePic
	if err := s.participantRepo.Update(ctx, participant); err != nil {
		s.logger.Warn("fetch fb profile: failed to update participant", zap.Error(err))
		return
	}

	s.logger.Info("fb profile updated",
		zap.String("participant_id", participant.ID.String()),
		zap.String("name", profile.Name),
	)
}

// findOrCreateConversation tìm hoặc tạo conversation
func (s *messageService) findOrCreateConversation(ctx context.Context, workspaceID, channelAccountID, participantID uuid.UUID) (*models.Conversation, bool, error) {
	conversation := &models.Conversation{
		WorkspaceID:      workspaceID,
		ChannelAccountID: channelAccountID,
		ParticipantID:    participantID,
		Status:           models.StatusOpen,
		Priority:         models.PriorityNormal,
	}

	return s.conversationRepo.FindOrCreate(ctx, conversation)
}

// saveMessage lưu message vào database
func (s *messageService) saveMessage(ctx context.Context, conversationID uuid.UUID, inbound *channel.InboundMessage) (*models.Message, error) {
	message := &models.Message{
		ConversationID: conversationID,
		Direction:      models.DirectionIn,
		SenderType:     models.SenderCustomer,
		ContentType:    models.ContentType(inbound.ContentType),
	}

	// Set content
	if inbound.Content != "" {
		message.Content = &inbound.Content
	}

	// Set channel message ID
	if inbound.ChannelMessageID != "" {
		message.ChannelMessageID = &inbound.ChannelMessageID
	}

	// Convert attachments
	if len(inbound.Attachments) > 0 {
		attachments := make(models.Attachments, len(inbound.Attachments))
		for i, att := range inbound.Attachments {
			attachments[i] = models.Attachment{
				Type:     att.Type,
				URL:      att.URL,
				Name:     att.Name,
				Size:     att.Size,
				MimeType: att.MimeType,
			}
		}
		message.Attachments = attachments
	}

	if err := s.messageRepo.Create(ctx, message); err != nil {
		return nil, err
	}

	return message, nil
}

// updateConversationLastMessage cập nhật last message của conversation
func (s *messageService) updateConversationLastMessage(ctx context.Context, conv *models.Conversation, inbound *channel.InboundMessage) error {
	conv.UpdateLastMessage(inbound.Content, inbound.Timestamp)
	return s.conversationRepo.Update(ctx, conv)
}

// processBotResponse xử lý bot response
func (s *messageService) processBotResponse(ctx context.Context, workspaceID, channelAccountID uuid.UUID, conv *models.Conversation, inbound *channel.InboundMessage, inboundMsg *models.Message) (*botProcessResult, error) {
	// Gọi bot responder
	botResponse, err := s.botResponder.Process(ctx, workspaceID, inbound.SenderID, inbound.Content)
	if err != nil {
		return nil, err
	}

	result := &botProcessResult{}

	// Xử lý handoff
	if botResponse.ShouldHandoff {
		conv.PauseBot(botResponse.HandoffReason)
		if err := s.conversationRepo.Update(ctx, conv); err != nil {
			s.logger.Warn("failed to pause bot on conversation", zap.Error(err))
		}
		result.ShouldHandoff = true
	}

	// Gửi response nếu có
	if botResponse.ShouldReply && botResponse.Response != nil {
		// Lưu outbound message
		outboundMsg := &models.Message{
			ConversationID: conv.ID,
			Direction:      models.DirectionOut,
			SenderType:     models.SenderBot,
			Content:        &botResponse.Response.Content,
			ContentType:    models.ContentText,
			Metadata: models.MessageMetadata{
				MatchedRuleID:  &botResponse.MatchedRule.ID,
				MatchedKeyword: botResponse.MatchedKeyword,
				Confidence:     botResponse.Confidence,
			},
		}

		if err := s.messageRepo.Create(ctx, outboundMsg); err != nil {
			s.logger.Warn("failed to save bot message", zap.Error(err))
		} else {
			// Publish bot message to Centrifugo for realtime updates
			if s.publisher != nil {
				go func() {
					content := ""
					if outboundMsg.Content != nil {
						content = *outboundMsg.Content
					}
					event := &realtime.MessageEvent{
						MessageID:      outboundMsg.ID,
						ConversationID: conv.ID,
						Direction:      string(outboundMsg.Direction),
						SenderType:     string(outboundMsg.SenderType),
						Content:        content,
						CreatedAt:      outboundMsg.CreatedAt,
					}
					if err := s.publisher.PublishNewMessage(workspaceID, event); err != nil {
						s.logger.Warn("failed to publish bot message event", zap.Error(err))
					}
				}()
			}
		}

		// Gửi message qua channel
		ch, err := s.channelRegistry.Get(inbound.ChannelType)
		if err != nil {
			return nil, err
		}

		// Lấy channel account để có credentials
		channelAcct, err := s.channelAccountRepo.FindByID(ctx, channelAccountID)
		if err != nil {
			s.logger.Warn("failed to get channel account", zap.Error(err))
			return result, nil
		}

		// Tạo credentials map từ channel account
		credentials := map[string]string{
			"page_access_token": channelAcct.Credentials.PageAccessToken,
			"app_secret":        channelAcct.Credentials.AppSecret,
		}

		sendResult, err := ch.Send(ctx, botResponse.Response, credentials)
		if err != nil {
			s.logger.Warn("failed to send bot response", zap.Error(err))
		} else if sendResult.Success {
			result.ShouldReply = true
			result.ResponseSent = botResponse.Response

			// Cập nhật channel message ID
			if sendResult.ChannelMessageID != "" {
				outboundMsg.ChannelMessageID = &sendResult.ChannelMessageID
				s.messageRepo.Update(ctx, outboundMsg)
			}

			s.logger.Info("bot response sent to channel",
				zap.String("channel_type", inbound.ChannelType),
				zap.String("channel_message_id", sendResult.ChannelMessageID),
			)
		} else if sendResult.Error != nil {
			s.logger.Warn("channel send failed",
				zap.String("channel_type", inbound.ChannelType),
				zap.Error(sendResult.Error),
			)
		}
	}

	return result, nil
}

// botProcessResult kết quả xử lý bot
type botProcessResult struct {
	ShouldReply   bool
	ShouldHandoff bool
	ResponseSent  *channel.OutboundMessage
}
