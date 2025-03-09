package connect

import (
	"sync"
)

// Room 表示一个聊天室，维护该房间内的所有连接
type Room struct {
	ID       int              // 房间ID
	Channels map[int]*Channel // 用户ID -> Channel映射
	mu       sync.RWMutex     // 保护并发访问
}

// NewRoom 创建新房间实例
func NewRoom(id int) *Room {
	return &Room{
		ID:       id,
		Channels: make(map[int]*Channel),
	}
}

// AddChannel 添加用户通道到房间
func (r *Room) AddChannel(userID int, ch *Channel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Channels[userID] = ch
}

// RemoveChannel 从房间中移除用户通道
func (r *Room) RemoveChannel(userID int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Channels, userID)
}

// Broadcast 向房间内所有用户广播消息
func (r *Room) Broadcast(message []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, ch := range r.Channels {
		select {
		case ch.send <- message:
			// 消息已发送
		default:
			// 通道已满或关闭，这里可以考虑其他处理方式
		}
	}
}

// GetUserList 获取房间内所有用户ID
func (r *Room) GetUserList() []int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]int, 0, len(r.Channels))
	for userID := range r.Channels {
		users = append(users, userID)
	}
	return users
}

// GetUserCount 获取房间内用户数量
func (r *Room) GetUserCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Channels)
}
