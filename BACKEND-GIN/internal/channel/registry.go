package channel

import (
	"fmt"
	"sync"
)

// ===========================================================================
// Registry quản lý và lưu trữ các channel adapters
// Cho phép đăng ký và lấy channel adapter theo type
// ===========================================================================

// Registry là container chứa tất cả channel adapters đã đăng ký
type Registry struct {
	// mu bảo vệ channels map khỏi concurrent access
	mu sync.RWMutex

	// channels map từ channel type (facebook/zalo/mock) -> Channel implementation
	channels map[string]Channel
}

// NewRegistry tạo một Registry mới
func NewRegistry() *Registry {
	return &Registry{
		channels: make(map[string]Channel),
	}
}

// Register đăng ký một channel adapter vào registry
// Nếu channel type đã tồn tại, nó sẽ bị ghi đè
func (r *Registry) Register(channel Channel) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.channels[channel.Type()] = channel
}

// Get lấy channel adapter theo type
// Trả về error nếu channel type chưa được đăng ký
func (r *Registry) Get(channelType string) (Channel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	channel, exists := r.channels[channelType]
	if !exists {
		return nil, fmt.Errorf("channel type '%s' chưa được đăng ký", channelType)
	}

	return channel, nil
}

// GetAll trả về danh sách tất cả channel types đã đăng ký
func (r *Registry) GetAll() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.channels))
	for t := range r.channels {
		types = append(types, t)
	}

	return types
}

// Has kiểm tra xem channel type đã được đăng ký chưa
func (r *Registry) Has(channelType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.channels[channelType]
	return exists
}

// Count trả về số lượng channels đã đăng ký
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.channels)
}
