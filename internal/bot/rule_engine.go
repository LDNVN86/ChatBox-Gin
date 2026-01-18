package bot

import (
	"context"
	"time"

	"chatbox-gin/internal/models"

	"go.uber.org/zap"
)

// ===========================================================================
// Rule Engine
// Match tin nhắn với các rules đã định nghĩa
// Trả về rule phù hợp nhất theo priority
// ===========================================================================

// MatchResult kết quả match rule
type MatchResult struct {
	// Matched có match rule nào không
	Matched bool

	// Rule rule đã match (nếu có)
	Rule *models.Rule

	// MatchedKeyword keyword đã match (cho keyword trigger)
	MatchedKeyword string

	// Confidence độ tin cậy của match (0-1)
	Confidence float64
}

// RuleEngine interface cho rule matching
type RuleEngine interface {
	// Match tìm rule phù hợp với message content
	// Rules được sắp xếp theo priority, trả về rule đầu tiên match
	Match(ctx context.Context, rules []models.Rule, content string) MatchResult
}

// ===========================================================================
// Rule Engine Implementation
// ===========================================================================

// ruleEngine triển khai RuleEngine
type ruleEngine struct {
	logger *zap.Logger
}

// NewRuleEngine tạo instance mới của RuleEngine
func NewRuleEngine(logger *zap.Logger) RuleEngine {
	return &ruleEngine{logger: logger}
}

// Match tìm rule phù hợp với message content
func (e *ruleEngine) Match(ctx context.Context, rules []models.Rule, content string) MatchResult {
	now := time.Now()

	// Duyệt qua các rules theo thứ tự priority (đã được sort từ DB)
	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}

		switch rule.TriggerType {
		case models.TriggerKeyword:
			if matched, keyword := e.matchKeyword(&rule, content); matched {
				e.logger.Debug("rule matched",
					zap.String("rule_name", rule.Name),
					zap.String("trigger_type", string(rule.TriggerType)),
					zap.String("keyword", keyword),
				)
				return MatchResult{
					Matched:        true,
					Rule:           &rule,
					MatchedKeyword: keyword,
					Confidence:     1.0,
				}
			}

		case models.TriggerTimeWindow:
			if rule.MatchesTimeWindow(now) {
				e.logger.Debug("time window rule matched",
					zap.String("rule_name", rule.Name),
				)
				return MatchResult{
					Matched:    true,
					Rule:       &rule,
					Confidence: 1.0,
				}
			}

		case models.TriggerFallback:
			// Fallback sẽ được xử lý cuối cùng
			continue
		}
	}

	// Không match rule nào, tìm fallback
	for _, rule := range rules {
		if rule.TriggerType == models.TriggerFallback && rule.IsActive {
			e.logger.Debug("fallback rule used",
				zap.String("rule_name", rule.Name),
			)
			return MatchResult{
				Matched:    true,
				Rule:       &rule,
				Confidence: 0.5, // Fallback có confidence thấp hơn
			}
		}
	}

	// Không có rule nào match
	return MatchResult{Matched: false}
}

// matchKeyword kiểm tra content có match keywords không
func (e *ruleEngine) matchKeyword(rule *models.Rule, content string) (bool, string) {
	if rule.MatchesKeyword(content) {
		// Tìm keyword đã match
		for _, keyword := range rule.TriggerConfig.Keywords {
			if rule.MatchesKeyword(keyword) {
				return true, keyword
			}
		}
		return true, ""
	}
	return false, ""
}
