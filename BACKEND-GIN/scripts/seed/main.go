//go:build ignore

// ===========================================================================
// Script tạo seed data cho development/testing
// Chạy: go run scripts/seed/main.go
// ===========================================================================

package main

import (
	"fmt"
	"log"
	"os"

	"chatbox-gin/internal/config"
	"chatbox-gin/internal/database"
	"chatbox-gin/internal/models"
	"chatbox-gin/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func main() {
	fmt.Println("🌱 Bắt đầu seed data...")

	// Load config
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Không thể load config: %v", err)
	}

	// Khởi tạo logger
	zapLog, err := logger.NewLogger(cfg.Logging.Level, cfg.Logging.Format)
	if err != nil {
		log.Fatalf("Không thể tạo logger: %v", err)
	}

	// Kết nối database
	db, err := database.NewConnection(&cfg.Database, zapLog)
	if err != nil {
		log.Fatalf("Không thể kết nối database: %v", err)
	}

	fmt.Println("✅ Đã kết nối database")

	// =========================================================================
	// 1. Tạo Workspace
	// =========================================================================
	workspace := &models.Workspace{
		Name: "Demo Shop",
		Slug: "demo-shop",
		Settings: models.WorkspaceSettings{
			Timezone:   "Asia/Ho_Chi_Minh",
			BotEnabled: true,
			Language:   "vi",
			WorkingHours: &models.WorkingHours{
				Start: "09:00",
				End:   "18:00",
				Days:  []int{1, 2, 3, 4, 5}, // Thứ 2-6
			},
		},
		IsActive: true,
	}

	// Kiểm tra đã tồn tại chưa
	var existingWorkspace models.Workspace
	if err := db.Where("slug = ?", workspace.Slug).First(&existingWorkspace).Error; err == nil {
		fmt.Println("⚠️  Workspace 'demo-shop' đã tồn tại, sử dụng ID hiện có")
		workspace = &existingWorkspace
	} else {
		if err := db.Create(workspace).Error; err != nil {
			log.Fatalf("Không thể tạo workspace: %v", err)
		}
		fmt.Printf("✅ Đã tạo Workspace: %s (ID: %s)\n", workspace.Name, workspace.ID)
	}

	// =========================================================================
	// 2. Tạo Users
	// =========================================================================
	users := []*models.User{
		{
			WorkspaceID: workspace.ID,
			Email:       "admin@demo.com",
			Name:        "Admin Demo",
			Role:        models.RoleOwner,
			IsActive:    true,
		},
		{
			WorkspaceID: workspace.ID,
			Email:       "agent1@demo.com",
			Name:        "Agent Một",
			Role:        models.RoleAgent,
			IsActive:    true,
		},
		{
			WorkspaceID: workspace.ID,
			Email:       "agent2@demo.com",
			Name:        "Agent Hai",
			Role:        models.RoleAgent,
			IsActive:    true,
		},
	}

	for _, user := range users {
		// Set password
		if err := user.SetPassword("Password123!"); err != nil {
			zapLog.Warn("Không thể set password", zap.Error(err))
		}

		// Kiểm tra email đã tồn tại chưa
		var existing models.User
		if err := db.Where("workspace_id = ? AND email = ?", workspace.ID, user.Email).First(&existing).Error; err == nil {
			fmt.Printf("⚠️  User '%s' đã tồn tại\n", user.Email)
			continue
		}

		if err := db.Create(user).Error; err != nil {
			zapLog.Warn("Không thể tạo user", zap.String("email", user.Email), zap.Error(err))
		} else {
			fmt.Printf("✅ Đã tạo User: %s (%s)\n", user.Name, user.Role)
		}
	}

	// =========================================================================
	// 3. Tạo Mock Channel Account
	// =========================================================================
	mockChannel := &models.ChannelAccount{
		WorkspaceID: workspace.ID,
		ChannelType: models.ChannelMock,
		Name:        "Mock Testing Channel",
		Settings: models.ChannelSettings{
			AutoReply:  true,
			BotEnabled: true,
			WelcomeMsg: "Xin chào! Tôi là bot hỗ trợ. Bạn cần giúp gì ạ?",
			OfflineMsg: "Hiện tại không có nhân viên trực. Chúng tôi sẽ phản hồi sớm nhất có thể!",
		},
		IsActive: true,
	}

	var existingChannel models.ChannelAccount
	if err := db.Where("workspace_id = ? AND channel_type = ?", workspace.ID, models.ChannelMock).First(&existingChannel).Error; err == nil {
		fmt.Println("⚠️  Mock channel đã tồn tại")
		mockChannel = &existingChannel
	} else {
		if err := db.Create(mockChannel).Error; err != nil {
			log.Fatalf("Không thể tạo mock channel: %v", err)
		}
		fmt.Printf("✅ Đã tạo Mock Channel (ID: %s)\n", mockChannel.ID)
	}

	// =========================================================================
	// 4. Tạo Tags
	// =========================================================================
	tags := []*models.Tag{
		{WorkspaceID: workspace.ID, Name: "VIP", Color: "#f59e0b"},
		{WorkspaceID: workspace.ID, Name: "Urgent", Color: "#ef4444"},
		{WorkspaceID: workspace.ID, Name: "New Customer", Color: "#10b981"},
		{WorkspaceID: workspace.ID, Name: "Feedback", Color: "#6366f1"},
		{WorkspaceID: workspace.ID, Name: "Support", Color: "#8b5cf6"},
	}

	for _, tag := range tags {
		var existing models.Tag
		if err := db.Where("workspace_id = ? AND name = ?", workspace.ID, tag.Name).First(&existing).Error; err == nil {
			continue
		}
		if err := db.Create(tag).Error; err != nil {
			zapLog.Warn("Không thể tạo tag", zap.String("name", tag.Name), zap.Error(err))
		} else {
			fmt.Printf("✅ Đã tạo Tag: %s\n", tag.Name)
		}
	}

	// =========================================================================
	// 5. Tạo Bot Rules
	// =========================================================================
	rules := []*models.Rule{
		{
			WorkspaceID: workspace.ID,
			Name:        "Chào hỏi",
			Description: strPtr("Trả lời khi khách chào"),
			TriggerType: models.TriggerKeyword,
			TriggerConfig: models.TriggerConfig{
				Keywords:  []string{"hello", "hi", "xin chào", "chào", "alo"},
				MatchType: "contains",
			},
			ResponseType: models.ResponseText,
			ResponseConfig: models.ResponseConfig{
				Text: "Xin chào! Cảm ơn bạn đã liên hệ. Tôi có thể giúp gì cho bạn?",
			},
			Priority: 100,
			IsActive: true,
		},
		{
			WorkspaceID: workspace.ID,
			Name:        "Hỏi giá",
			Description: strPtr("Trả lời khi khách hỏi về giá"),
			TriggerType: models.TriggerKeyword,
			TriggerConfig: models.TriggerConfig{
				Keywords:  []string{"giá", "bao nhiêu", "price", "cost", "phí"},
				MatchType: "contains",
			},
			ResponseType: models.ResponseText,
			ResponseConfig: models.ResponseConfig{
				Text: "Để biết thông tin chi tiết về giá, bạn vui lòng cho mình biết bạn quan tâm đến sản phẩm nào nhé!",
			},
			Priority: 90,
			IsActive: true,
		},
		{
			WorkspaceID: workspace.ID,
			Name:        "Ngoài giờ làm việc",
			Description: strPtr("Thông báo khi ngoài giờ làm việc"),
			TriggerType: models.TriggerTimeWindow,
			TriggerConfig: models.TriggerConfig{
				StartTime: "18:01",
				EndTime:   "08:59",
				Timezone:  "Asia/Ho_Chi_Minh",
				Days:      []int{0, 1, 2, 3, 4, 5, 6}, // Tất cả các ngày
			},
			ResponseType: models.ResponseText,
			ResponseConfig: models.ResponseConfig{
				Text: "Hiện tại đang ngoài giờ làm việc (9:00 - 18:00). Chúng tôi sẽ phản hồi bạn trong thời gian sớm nhất!",
			},
			Priority: 50,
			IsActive: true,
		},
		{
			WorkspaceID: workspace.ID,
			Name:        "Chuyển nhân viên",
			Description: strPtr("Khi khách yêu cầu nói chuyện với người thật"),
			TriggerType: models.TriggerKeyword,
			TriggerConfig: models.TriggerConfig{
				Keywords:  []string{"nói chuyện với người", "nhân viên", "agent", "staff", "tư vấn viên"},
				MatchType: "contains",
			},
			ResponseType: models.ResponseHandoff,
			ResponseConfig: models.ResponseConfig{
				Message:  "Tôi sẽ chuyển bạn đến nhân viên hỗ trợ. Vui lòng đợi trong giây lát!",
				Priority: "high",
			},
			Priority: 80,
			IsActive: true,
		},
		{
			WorkspaceID: workspace.ID,
			Name:        "Fallback - Không hiểu",
			Description: strPtr("Trả lời mặc định khi không match rule nào"),
			TriggerType: models.TriggerFallback,
			ResponseType: models.ResponseText,
			ResponseConfig: models.ResponseConfig{
				Text: "Cảm ơn bạn đã nhắn tin! Nhân viên của chúng tôi sẽ phản hồi bạn sớm nhất có thể.",
			},
			Priority: 0,
			IsActive: true,
		},
	}

	for _, rule := range rules {
		var existing models.Rule
		if err := db.Where("workspace_id = ? AND name = ?", workspace.ID, rule.Name).First(&existing).Error; err == nil {
			fmt.Printf("⚠️  Rule '%s' đã tồn tại\n", rule.Name)
			continue
		}
		if err := db.Create(rule).Error; err != nil {
			zapLog.Warn("Không thể tạo rule", zap.String("name", rule.Name), zap.Error(err))
		} else {
			fmt.Printf("✅ Đã tạo Rule: %s (%s)\n", rule.Name, rule.TriggerType)
		}
	}

	// =========================================================================
	// Summary
	// =========================================================================
	fmt.Println("")
	fmt.Println("========================================")
	fmt.Println("🎉 Seed data hoàn tất!")
	fmt.Println("========================================")
	fmt.Println("")
	fmt.Println("📝 Thông tin đăng nhập:")
	fmt.Println("   Email:    admin@demo.com")
	fmt.Println("   Password: Password123!")
	fmt.Println("")
	fmt.Printf("🔗 Workspace ID: %s\n", workspace.ID)
	fmt.Printf("🔗 Mock Channel ID: %s\n", mockChannel.ID)
	fmt.Println("")
	fmt.Println("💡 Test mock inbound:")
	fmt.Println(`   curl -X POST http://localhost:8080/api/v1/mock/inbound \`)
	fmt.Println(`     -H "Content-Type: application/json" \`)
	fmt.Printf(`     -d '{"workspace_id":"%s","channel_account_id":"%s","sender_id":"user123","message":"Xin chào"}'`, workspace.ID, mockChannel.ID)
	fmt.Println("")

	os.Exit(0)
}

// strPtr helper để tạo pointer từ string
func strPtr(s string) *string {
	return &s
}

// uuidPtr helper để tạo pointer từ UUID
func uuidPtr(id uuid.UUID) *uuid.UUID {
	return &id
}
