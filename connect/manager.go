package connect

import (
	"gochat/clog"
	"sync"
)

// ConnectionManager 管理所有连接和房间
type ConnectionManager struct {
	userChannels map[int]*Channel // 用户ID索引的所有连接
	rooms        map[int]*Room    // 房间ID索引的所有房间
	mu           sync.RWMutex     // 保护并发访问
}

// NewConnectionManager 创建连接管理器实例
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		userChannels: make(map[int]*Channel),
		rooms:        make(map[int]*Room),
	}
}

// AddUser 添加用户到系统
func (cm *ConnectionManager) AddUser(userID int, roomID int, ch *Channel) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.userChannels[userID] = ch
	room, exists := cm.rooms[roomID]
	if !exists {
		room = NewRoom(roomID)
		cm.rooms[roomID] = room
		clog.Debug("Created new room %d", roomID)
	}

	room.AddChannel(userID, ch)
	clog.Info("User %d added to room %d", userID, roomID)
}

// RemoveUser 从系统移除用户
func (cm *ConnectionManager) RemoveUser(userID int, roomID int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.userChannels, userID)
	if room, exists := cm.rooms[roomID]; exists {
		room.RemoveChannel(userID)
		if room.GetUserCount() == 0 {
			delete(cm.rooms, roomID)
			clog.Debug("Deleted empty room %d", roomID)
		}
	}
	clog.Info("User %d removed from room %d", userID, roomID)
}

// GetUser 获取用户连接
func (cm *ConnectionManager) GetUser(userID int) (*Channel, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	ch, exists := cm.userChannels[userID]
	return ch, exists
}

// GetRoom 获取房间
func (cm *ConnectionManager) GetRoom(roomID int) (*Room, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	room, exists := cm.rooms[roomID]
	return room, exists
}

// BroadcastToRoom 向指定房间广播消息
func (cm *ConnectionManager) BroadcastToRoom(roomID int, message []byte) bool {
	room, exists := cm.GetRoom(roomID)
	if !exists {
		clog.Warning("Room %d does not exist", roomID)
		return false
	}

	room.Broadcast(message)
	clog.Debug("Broadcasted message to room %d", roomID)
	return true
}
