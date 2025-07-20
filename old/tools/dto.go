package tools

// 聊天消息结构
type ChatMessage struct {
	FromUserID   int    `json:"from_user_id"`   // 发送者ID
	ToUserID     int    `json:"to_user_id"`     // 接收者ID
	FromUserName string `json:"from_user_name"` // 发送者用户名
	ToUserName   string `json:"to_user_name"`   // 接收者用户名
	RoomID       int    `json:"room_id"`        // 房间ID
	Message      string `json:"message"`        // 消息内容
	CreateTime   string `json:"create_time"`    // 创建时间
	MessageID    int    `json:"message_id"`     // 消息ID
}

// 房间信息结构
type RoomInfo struct {
	RoomID   int               `json:"room_id"`   // 房间ID
	Count    int               `json:"count"`     // 房间内人数
	UserInfo map[string]string `json:"user_info"` // 用户信息映射，key为用户ID，value为用户名
}
