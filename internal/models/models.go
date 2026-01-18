package models

// ===========================================================================
// Models Index
// Cung cấp danh sách tất cả models cho GORM AutoMigrate
// ===========================================================================

// AllModels trả về danh sách tất cả models
// Dùng cho database.AutoMigrate() để tự động tạo/update tables
func AllModels() []interface{} {
	return []interface{}{
		&Workspace{},       // Không gian làm việc
		&User{},            // Người dùng hệ thống
		&ChannelAccount{},  // Tài khoản kênh chat
		&Participant{},     // Khách hàng
		&Conversation{},    // Cuộc hội thoại
		&Message{},         // Tin nhắn
		&Rule{},            // Quy tắc bot
		&WebhookEvent{},    // Sự kiện webhook
		&Tag{},             // Nhãn
		&ConversationTag{}, // Liên kết conversation-tag
		&Note{},            // Ghi chú nội bộ
	}
}
