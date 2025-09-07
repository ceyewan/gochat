package model

import (
	"time"
)

// User 用户数据模型
type User struct {
	// 用户 ID（主键）
	ID uint64 `gorm:"primaryKey;column:id" json:"id"`

	// 用户名（唯一）
	Username string `gorm:"uniqueIndex;size:50;not null;column:username" json:"username"`

	// 密码哈希
	PasswordHash string `gorm:"size:255;not null;column:password_hash" json:"-"`

	// 昵称
	Nickname string `gorm:"size:50;column:nickname" json:"nickname"`

	// 头像 URL
	AvatarURL string `gorm:"size:255;column:avatar_url" json:"avatar_url"`

	// 创建时间
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`

	// 更新时间
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`

	// 是否为游客
	IsGuest bool `gorm:"not null;default:false;column:is_guest" json:"is_guest"`
}

// TableName 返回表名
func (User) TableName() string {
	return "users"
}

// Group 群组数据模型
type Group struct {
	// 群组 ID（主键）
	ID uint64 `gorm:"primaryKey;column:id" json:"id"`

	// 群组名称
	Name string `gorm:"size:50;not null;column:name" json:"name"`

	// 群主 ID
	OwnerID uint64 `gorm:"index;not null;column:owner_id" json:"owner_id"`

	// 成员数量（冗余字段，用于快速查询）
	MemberCount int `gorm:"not null;default:0;column:member_count" json:"member_count"`

	// 群组头像 URL
	AvatarURL string `gorm:"size:255;column:avatar_url" json:"avatar_url"`

	// 群组描述
	Description string `gorm:"type:text;column:description" json:"description"`

	// 创建时间
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`

	// 更新时间
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName 返回表名
func (Group) TableName() string {
	return "groups"
}

// GroupMember 群组成员数据模型
type GroupMember struct {
	// 记录 ID（主键）
	ID uint64 `gorm:"primaryKey;column:id" json:"id"`

	// 群组 ID
	GroupID uint64 `gorm:"uniqueIndex:idx_group_user;column:group_id" json:"group_id"`

	// 用户 ID
	UserID uint64 `gorm:"uniqueIndex:idx_group_user;index;column:user_id" json:"user_id"`

	// 成员角色 (1:成员, 2:管理员, 3:群主)
	Role int `gorm:"not null;default:1;column:role" json:"role"`

	// 加入时间
	JoinedAt time.Time `gorm:"column:joined_at" json:"joined_at"`
}

// TableName 返回表名
func (GroupMember) TableName() string {
	return "group_members"
}

// Message 消息数据模型
type Message struct {
	// 消息 ID（主键）
	ID uint64 `gorm:"primaryKey;column:id" json:"id"`

	// 会话 ID
	ConversationID string `gorm:"uniqueIndex:idx_conv_seq;index:idx_conv_time;size:64;column:conversation_id" json:"conversation_id"`

	// 发送者 ID
	SenderID uint64 `gorm:"not null;column:sender_id" json:"sender_id"`

	// 消息类型 (1:文本, 2:图片, 3:文件, 4:系统消息)
	MessageType int `gorm:"not null;default:1;column:message_type" json:"message_type"`

	// 消息内容
	Content string `gorm:"type:text;not null;column:content" json:"content"`

	// 会话内序列号
	SeqID uint64 `gorm:"uniqueIndex:idx_conv_seq;not null;column:seq_id" json:"seq_id"`

	// 是否已删除
	Deleted bool `gorm:"not null;default:false;column:deleted" json:"deleted"`

	// 消息扩展信息（JSON 格式）
	Extra string `gorm:"type:text;column:extra" json:"extra"`

	// 创建时间
	CreatedAt time.Time `gorm:"index:idx_conv_time;column:created_at" json:"created_at"`

	// 更新时间
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName 返回表名
func (Message) TableName() string {
	return "messages"
}

// UserReadPointer 用户已读位置数据模型
type UserReadPointer struct {
	// 记录 ID（主键）
	ID uint64 `gorm:"primaryKey;column:id" json:"id"`

	// 用户 ID
	UserID uint64 `gorm:"uniqueIndex:idx_user_conv;column:user_id" json:"user_id"`

	// 会话 ID
	ConversationID string `gorm:"uniqueIndex:idx_user_conv;size:64;column:conversation_id" json:"conversation_id"`

	// 最后已读消息的序列号
	LastReadSeqID uint64 `gorm:"not null;column:last_read_seq_id" json:"last_read_seq_id"`

	// 更新时间
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName 返回表名
func (UserReadPointer) TableName() string {
	return "user_read_pointers"
}

// Conversation 会话数据模型
type Conversation struct {
	// 会话 ID（主键）
	ID string `gorm:"primaryKey;size:64;column:id" json:"id"`

	// 会话类型 (1:单聊, 2:群聊)
	Type int `gorm:"not null;column:type" json:"type"`

	// 最后一条消息的 ID
	LastMessageID uint64 `gorm:"column:last_message_id" json:"last_message_id"`

	// 创建时间
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`

	// 更新时间
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName 返回表名
func (Conversation) TableName() string {
	return "conversations"
}
