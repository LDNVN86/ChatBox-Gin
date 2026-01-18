package channel

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ===========================================================================
// Facebook Channel
// Adapter để nhận và gửi tin nhắn qua Facebook Messenger
// ===========================================================================

// FacebookChannel implements Channel interface cho Facebook Messenger
type FacebookChannel struct {
	logger *zap.Logger
}

// NewFacebookChannel tạo Facebook channel mới
func NewFacebookChannel(logger *zap.Logger) *FacebookChannel {
	return &FacebookChannel{
		logger: logger,
	}
}

// Type trả về loại channel
func (c *FacebookChannel) Type() string {
	return "facebook"
}

// ===========================================================================
// Webhook Payload Structures
// ===========================================================================

// FBWebhookPayload cấu trúc webhook từ Facebook
type FBWebhookPayload struct {
	Object string          `json:"object"`
	Entry  []FBWebhookEntry `json:"entry"`
}

// FBWebhookEntry một entry trong webhook
type FBWebhookEntry struct {
	ID        string             `json:"id"`
	Time      int64              `json:"time"`
	Messaging []FBMessagingEvent `json:"messaging"`
}

// FBMessagingEvent một sự kiện messaging
type FBMessagingEvent struct {
	Sender    FBUser       `json:"sender"`
	Recipient FBUser       `json:"recipient"`
	Timestamp int64        `json:"timestamp"`
	Message   *FBMessage   `json:"message,omitempty"`
	Postback  *FBPostback  `json:"postback,omitempty"`
}

// FBUser thông tin user
type FBUser struct {
	ID string `json:"id"`
}

// FBMessage tin nhắn từ user
type FBMessage struct {
	MID         string          `json:"mid"`
	Text        string          `json:"text"`
	Attachments []FBAttachment  `json:"attachments,omitempty"`
	QuickReply  *FBQuickReply   `json:"quick_reply,omitempty"`
}

// FBAttachment file đính kèm  
type FBAttachment struct {
	Type    string          `json:"type"` // image, video, audio, file
	Payload FBAttachPayload `json:"payload"`
}

// FBAttachPayload payload của attachment
type FBAttachPayload struct {
	URL      string `json:"url"`
	Title    string `json:"title,omitempty"`
	Sticker  int64  `json:"sticker_id,omitempty"`
}

// FBQuickReply quick reply được chọn
type FBQuickReply struct {
	Payload string `json:"payload"`
}

// FBPostback postback button được bấm
type FBPostback struct {
	Title   string `json:"title"`
	Payload string `json:"payload"`
}

// FBUserProfile thông tin profile user từ Graph API
type FBUserProfile struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ProfilePic string `json:"profile_pic"`
}

// ===========================================================================
// GetUserProfile - Lấy thông tin user từ Facebook Graph API
// ===========================================================================

// GetUserProfile gọi Facebook Graph API để lấy name và avatar của user
func (c *FacebookChannel) GetUserProfile(ctx context.Context, userID, accessToken string) (*FBUserProfile, error) {
	url := fmt.Sprintf(
		"https://graph.facebook.com/v18.0/%s?fields=name,profile_pic&access_token=%s",
		userID, accessToken,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("fb profile api error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)),
		)
		return nil, fmt.Errorf("fb api error: status %d", resp.StatusCode)
	}

	var profile FBUserProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("unmarshal profile: %w", err)
	}

	c.logger.Debug("fb profile fetched",
		zap.String("user_id", userID),
		zap.String("name", profile.Name),
	)

	return &profile, nil
}

// ===========================================================================
// Normalize - Parse webhook payload
// ===========================================================================

// Normalize chuyển đổi FB webhook payload thành InboundMessage chuẩn
func (c *FacebookChannel) Normalize(ctx context.Context, channelAccountID uuid.UUID, payload map[string]interface{}) (*InboundMessage, error) {
	// Parse payload
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	var fbPayload FBWebhookPayload
	if err := json.Unmarshal(jsonBytes, &fbPayload); err != nil {
		return nil, fmt.Errorf("unmarshal fb payload: %w", err)
	}

	// Validate
	if fbPayload.Object != "page" {
		return nil, fmt.Errorf("invalid object type: %s", fbPayload.Object)
	}

	if len(fbPayload.Entry) == 0 || len(fbPayload.Entry[0].Messaging) == 0 {
		return nil, fmt.Errorf("no messaging events")
	}

	// Get first messaging event
	event := fbPayload.Entry[0].Messaging[0]
	
	// Build InboundMessage
	inbound := &InboundMessage{
		ChannelType:      "facebook",
		SenderID:         event.Sender.ID,
		RecipientID:      event.Recipient.ID,
		Timestamp:        time.UnixMilli(event.Timestamp),
		RawPayload:       payload,
	}

	// Handle message
	if event.Message != nil {
		inbound.ChannelMessageID = event.Message.MID
		inbound.Content = event.Message.Text
		inbound.ContentType = "text"

		// Handle attachments
		for _, att := range event.Message.Attachments {
			attData := AttachmentData{
				Type: att.Type,
				URL:  att.Payload.URL,
			}
			inbound.Attachments = append(inbound.Attachments, attData)
			
			// Update content type
			if inbound.ContentType == "text" && att.Type != "" {
				inbound.ContentType = att.Type
			}
		}

		// Handle quick reply
		if event.Message.QuickReply != nil {
			inbound.Content = event.Message.QuickReply.Payload
		}
	}

	// Handle postback
	if event.Postback != nil {
		inbound.Content = event.Postback.Payload
		inbound.ContentType = "postback"
		inbound.ChannelMessageID = fmt.Sprintf("postback_%d", event.Timestamp)
	}

	c.logger.Info("normalized fb message",
		zap.String("sender_id", inbound.SenderID),
		zap.String("content_type", inbound.ContentType),
	)

	return inbound, nil
}

// ===========================================================================
// Send - Gửi tin nhắn qua FB Graph API
// ===========================================================================

// FBSendRequest request gửi tin nhắn
type FBSendRequest struct {
	Recipient    FBUser         `json:"recipient"`
	Message      FBSendMessage  `json:"message"`
	MessagingType string        `json:"messaging_type"`
}

// FBSendMessage tin nhắn gửi đi
type FBSendMessage struct {
	Text         string           `json:"text,omitempty"`
	Attachment   *FBSendAttachment `json:"attachment,omitempty"`
	QuickReplies []FBSendQR       `json:"quick_replies,omitempty"`
}

// FBSendAttachment attachment gửi đi
type FBSendAttachment struct {
	Type    string `json:"type"`
	Payload struct {
		URL string `json:"url"`
	} `json:"payload"`
}

// FBSendQR quick reply gửi đi
type FBSendQR struct {
	ContentType string `json:"content_type"`
	Title       string `json:"title"`
	Payload     string `json:"payload"`
}

// Send gửi tin nhắn qua Facebook Messenger
func (c *FacebookChannel) Send(ctx context.Context, msg *OutboundMessage, credentials map[string]string) (*SendResult, error) {
	accessToken := credentials["page_access_token"]
	if accessToken == "" {
		return &SendResult{Success: false, Error: fmt.Errorf("missing page_access_token")}, nil
	}

	// Build request
	fbReq := FBSendRequest{
		Recipient:     FBUser{ID: msg.RecipientID},
		MessagingType: "RESPONSE",
	}

	// Set message content
	if msg.Content != "" {
		fbReq.Message.Text = msg.Content
	}

	// Add quick replies
	for _, qr := range msg.QuickReplies {
		fbReq.Message.QuickReplies = append(fbReq.Message.QuickReplies, FBSendQR{
			ContentType: "text",
			Title:       qr.Title,
			Payload:     qr.Payload,
		})
	}

	// Send to Graph API
	url := fmt.Sprintf("https://graph.facebook.com/v18.0/me/messages?access_token=%s", accessToken)
	
	jsonBody, _ := json.Marshal(fbReq)
	resp, err := http.Post(url, "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		return &SendResult{Success: false, Error: err}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("fb send failed",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)),
		)
		return &SendResult{Success: false, Error: fmt.Errorf("fb api error: %s", string(body))}, nil
	}

	// Parse response
	var fbResp struct {
		MessageID string `json:"message_id"`
	}
	json.Unmarshal(body, &fbResp)

	c.logger.Info("fb message sent",
		zap.String("recipient", msg.RecipientID),
		zap.String("message_id", fbResp.MessageID),
	)

	return &SendResult{
		Success:          true,
		ChannelMessageID: fbResp.MessageID,
	}, nil
}

// ===========================================================================
// Verify - Xác thực webhook signature
// ===========================================================================

// Verify kiểm tra X-Hub-Signature-256 header
func (c *FacebookChannel) Verify(signature string, body []byte, secret string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	
	expectedSig := signature[7:]
	
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	actualSig := hex.EncodeToString(mac.Sum(nil))
	
	return hmac.Equal([]byte(expectedSig), []byte(actualSig))
}
