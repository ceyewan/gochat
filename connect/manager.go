package connect

import (
	"sync"
)

// ConnectionManager 管理所有的连接和房间
// 负责用户连接的生命周期管理和房间的维护
type ConnectionManager struct {
	userChannels map[int]*Channel // 按用户ID索引的所有连接
	rooms        map[int]*Room    // 按房间ID索引的所有房间
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
// userID: 用户唯一标识
// roomID: 用户加入的房间ID
// ch: 用户的通信通道
func (cm *ConnectionManager) AddUser(userID int, roomID int, ch *Channel) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 添加到用户映射
	cm.userChannels[userID] = ch

	// 确保房间存在
	room, exists := cm.rooms[roomID]
	if !exists {
		room = NewRoom(roomID)
		cm.rooms[roomID] = room
	}

	// 添加用户到房间
	room.AddChannel(userID, ch)
}

// RemoveUser 从系统移除用户
// userID: 要移除的用户ID
// roomID: 用户所在的房间ID
func (cm *ConnectionManager) RemoveUser(userID int, roomID int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 从用户映射删除
	delete(cm.userChannels, userID)

	// 从房间移除用户
	if room, exists := cm.rooms[roomID]; exists {
		room.RemoveChannel(userID)

		// 如果房间为空，删除房间
		if room.GetUserCount() == 0 {
			delete(cm.rooms, roomID)
		}
	}
}

// GetUser 获取用户连接
// 返回用户的通信通道和用户是否存在
func (cm *ConnectionManager) GetUser(userID int) (*Channel, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	ch, exists := cm.userChannels[userID]
	return ch, exists
}

// GetRoom 获取房间
// 返回房间对象和房间是否存在
func (cm *ConnectionManager) GetRoom(roomID int) (*Room, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	room, exists := cm.rooms[roomID]
	return room, exists
}

// BroadcastToRoom 向指定房间广播消息
// roomID: 目标房间ID
// message: 要广播的消息
// 返回是否成功
func (cm *ConnectionManager) BroadcastToRoom(roomID int, message []byte) bool {
	room, exists := cm.GetRoom(roomID)
	if !exists {
		return false
	}

	room.Broadcast(message)
	return true
}
