package bot

import (
	"context"

	"chatbox-gin/internal/channel"
	"chatbox-gin/internal/models"
	"chatbox-gin/internal/repositories"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ===========================================================================
// Bot Responder
// Xử lý logic bot: nhận message, match rule, tạo response
// Orchestrate RuleEngine và ResponseBuilder
// ===========================================================================

// BotResponse kết quả xử lý của bot
type BotResponse struct {
	// ShouldReply bot có cần trả lời không
	ShouldReply bool

	// Response tin nhắn trả lời (nếu có)
	Response *channel.OutboundMessage

	// ShouldHandoff có cần chuyển cho agent không
	ShouldHandoff bool

	// HandoffReason lý do chuyển cho agent
	HandoffReason string

	// MatchedRule rule đã match
	MatchedRule *models.Rule

	// MatchedKeyword keyword đã match
	MatchedKeyword string

	// Confidence độ tin cậy
	Confidence float64
}

// Responder interface cho bot responder
type Responder interface {
	// Process xử lý inbound message và tạo response
	Process(ctx context.Context, workspaceID uuid.UUID, recipientID string, content string) (*BotResponse, error)
}

// ===========================================================================
// Responder Implementation
// ===========================================================================

// responder triển khai Responder
type responder struct {
	ruleRepo        repositories.RuleRepository
	ruleEngine      RuleEngine
	responseBuilder ResponseBuilder
	logger          *zap.Logger
}

// NewResponder tạo instance mới của Responder
func NewResponder(
	ruleRepo repositories.RuleRepository,
	ruleEngine RuleEngine,
	responseBuilder ResponseBuilder,
	logger *zap.Logger,
) Responder {
	return &responder{
		ruleRepo:        ruleRepo,
		ruleEngine:      ruleEngine,
		responseBuilder: responseBuilder,
		logger:          logger,
	}
}

// Process xử lý inbound message
func (r *responder) Process(ctx context.Context, workspaceID uuid.UUID, recipientID string, content string) (*BotResponse, error) {
	// 1. Lấy tất cả active rules của workspace
	rules, err := r.ruleRepo.FindActiveByWorkspace(ctx, workspaceID)
	if err != nil {
		r.logger.Error("failed to get rules",
			zap.String("workspace_id", workspaceID.String()),
			zap.Error(err),
		)
		return nil, err
	}

	if len(rules) == 0 {
		r.logger.Debug("no active rules found",
			zap.String("workspace_id", workspaceID.String()),
		)
		return &BotResponse{ShouldReply: false}, nil
	}

	// 2. Match message với rules
	matchResult := r.ruleEngine.Match(ctx, rules, content)

	if !matchResult.Matched {
		r.logger.Debug("no rule matched",
			zap.String("workspace_id", workspaceID.String()),
			zap.String("content", truncateForLog(content, 50)),
		)
		return &BotResponse{ShouldReply: false}, nil
	}

	rule := matchResult.Rule

	// 3. Tăng hit count cho rule
	if err := r.ruleRepo.IncrementHitCount(ctx, rule.ID); err != nil {
		r.logger.Warn("failed to increment hit count",
			zap.String("rule_id", rule.ID.String()),
			zap.Error(err),
		)
		// Không fail vì lỗi này, tiếp tục xử lý
	}

	// 4. Xử lý theo response type
	response := &BotResponse{
		MatchedRule:    rule,
		MatchedKeyword: matchResult.MatchedKeyword,
		Confidence:     matchResult.Confidence,
	}

	if rule.IsHandoffResponse() {
		// Handoff to agent
		response.ShouldHandoff = true
		response.HandoffReason = rule.ResponseConfig.Message
		if response.HandoffReason == "" {
			response.HandoffReason = "Customer requested agent"
		}

		// Vẫn gửi message thông báo
		if rule.ResponseConfig.Message != "" {
			response.ShouldReply = true
			response.Response = r.responseBuilder.BuildFromRule(rule, recipientID)
		}
	} else {
		// Regular response
		response.ShouldReply = true
		response.Response = r.responseBuilder.BuildFromRule(rule, recipientID)
	}

	r.logger.Info("bot response generated",
		zap.String("rule_name", rule.Name),
		zap.Bool("should_reply", response.ShouldReply),
		zap.Bool("should_handoff", response.ShouldHandoff),
	)

	return response, nil
}

// truncateForLog cắt string cho logging
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
