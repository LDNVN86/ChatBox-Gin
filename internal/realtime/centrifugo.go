package realtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ===========================================================================
// Centrifugo Client
// Publish realtime events to Centrifugo server
// ===========================================================================

// Publisher interface for realtime events
type Publisher interface {
	// PublishNewMessage publishes new message event
	PublishNewMessage(workspaceID uuid.UUID, event *MessageEvent) error

	// PublishConversationUpdate publishes conversation update event
	PublishConversationUpdate(workspaceID uuid.UUID, event *ConversationEvent) error
}

// MessageEvent event khi có tin nhắn mới
type MessageEvent struct {
	Type           string    `json:"type"`
	MessageID      uuid.UUID `json:"message_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	Direction      string    `json:"direction"`
	SenderType     string    `json:"sender_type"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
	// Thông tin thêm
	ParticipantName string `json:"participant_name,omitempty"`
	ChannelType     string `json:"channel_type,omitempty"`
}

// ConversationEvent event khi conversation thay đổi
type ConversationEvent struct {
	Type           string    `json:"type"`
	ConversationID uuid.UUID `json:"conversation_id"`
	Status         string    `json:"status,omitempty"`
	AssignedTo     string    `json:"assigned_to,omitempty"`
}

// CentrifugoClient implements Publisher
type CentrifugoClient struct {
	url    string
	apiKey string
	client *http.Client
	log    *zap.Logger
}

// NewCentrifugoClient creates a new Centrifugo client
func NewCentrifugoClient(url, apiKey string, log *zap.Logger) *CentrifugoClient {
	return &CentrifugoClient{
		url:    url,
		apiKey: apiKey,
		client: &http.Client{Timeout: 5 * time.Second},
		log:    log,
	}
}

// publishRequest sends a request to Centrifugo API
type publishRequest struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

type publishParams struct {
	Channel string      `json:"channel"`
	Data    interface{} `json:"data"`
}

func (c *CentrifugoClient) publish(channel string, data interface{}) error {
	req := publishRequest{
		Method: "publish",
		Params: publishParams{
			Channel: channel,
			Data:    data,
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.url+"/api", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "apikey "+c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		c.log.Warn("centrifugo publish failed", zap.Error(err))
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.log.Warn("centrifugo publish bad status",
			zap.Int("status", resp.StatusCode),
			zap.String("channel", channel),
		)
		return fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	c.log.Debug("published to centrifugo",
		zap.String("channel", channel),
	)

	return nil
}

// PublishNewMessage publishes new message event to workspace channel
func (c *CentrifugoClient) PublishNewMessage(workspaceID uuid.UUID, event *MessageEvent) error {
	event.Type = "new_message"
	channel := fmt.Sprintf("chat:workspace_%s", workspaceID.String())
	return c.publish(channel, event)
}

// PublishConversationUpdate publishes conversation update event
func (c *CentrifugoClient) PublishConversationUpdate(workspaceID uuid.UUID, event *ConversationEvent) error {
	event.Type = "conversation_update"
	channel := fmt.Sprintf("chat:workspace_%s", workspaceID.String())
	return c.publish(channel, event)
}

// ===========================================================================
// Noop Publisher (for when Centrifugo is not configured)
// ===========================================================================

// NoopPublisher does nothing (used when realtime is disabled)
type NoopPublisher struct{}

func NewNoopPublisher() *NoopPublisher {
	return &NoopPublisher{}
}

func (n *NoopPublisher) PublishNewMessage(workspaceID uuid.UUID, event *MessageEvent) error {
	return nil
}

func (n *NoopPublisher) PublishConversationUpdate(workspaceID uuid.UUID, event *ConversationEvent) error {
	return nil
}
