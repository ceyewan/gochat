<template>
    <div 
        class="conversation-item"
        :class="{ active: isActive }"
        @click="handleSelect"
    >
        <!-- 头像区域 -->
        <div class="avatar-container">
            <img 
                :src="avatarUrl" 
                :alt="displayName"
                class="avatar"
                @error="handleAvatarError"
            />
            <!-- 在线状态指示器 -->
            <div 
                class="online-indicator"
                :class="{ online: isOnline }"
                v-if="conversation.type === 'single'"
            ></div>
        </div>
        
        <!-- 内容区域 -->
        <div class="content">
            <div class="header">
                <span class="name">{{ displayName }}</span>
                <span class="time">{{ formattedTime }}</span>
            </div>
            <div class="message-preview">
                <span class="last-message">{{ lastMessagePreview }}</span>
                <div class="unread-badge" v-if="conversation.unreadCount > 0">
                    {{ conversation.unreadCount > 99 ? '99+' : conversation.unreadCount }}
                </div>
            </div>
        </div>
    </div>
</template>

<script>
import { mapGetters } from 'vuex'

export default {
    name: 'ConversationItem',
    props: {
        conversation: {
            type: Object,
            required: true
        },
        isActive: {
            type: Boolean,
            default: false
        }
    },
    computed: {
        ...mapGetters('onlineStatus', ['getFriendStatus']),
        
        displayName() {
            if (this.conversation.type === 'single') {
                return this.conversation.target?.username || '未知用户'
            } else {
                return this.conversation.target?.groupName || '未知群聊'
            }
        },
        
        avatarUrl() {
            const avatar = this.conversation.target?.avatar
            if (avatar) {
                return avatar
            }
            return this.generateDefaultAvatar()
        },
        
        lastMessagePreview() {
            const message = this.conversation.lastMessage
            if (!message) {
                return this.conversation.type === 'single' ? '开始聊天吧~' : '群聊已创建'
            }
            
            // 限制消息预览长度
            const maxLength = 30
            return message.length > maxLength ? 
                message.substring(0, maxLength) + '...' : 
                message
        },
        
        formattedTime() {
            if (!this.conversation.lastMessageTime) {
                return ''
            }
            
            const messageTime = new Date(this.conversation.lastMessageTime)
            const now = new Date()
            const diffMs = now - messageTime
            const diffMinutes = Math.floor(diffMs / 60000)
            const diffHours = Math.floor(diffMs / 3600000)
            const diffDays = Math.floor(diffMs / 86400000)
            
            if (diffMinutes < 1) {
                return '刚刚'
            } else if (diffMinutes < 60) {
                return `${diffMinutes}分钟前`
            } else if (diffHours < 24) {
                return `${diffHours}小时前`
            } else if (diffDays < 7) {
                return `${diffDays}天前`
            } else {
                return messageTime.toLocaleDateString()
            }
        },
        
        isOnline() {
            if (this.conversation.type === 'single') {
                const userId = this.conversation.target?.userId
                return userId ? this.getFriendStatus(userId) : false
            }
            return false
        }
    },
    methods: {
        handleSelect() {
            this.$emit('select', this.conversation)
        },
        
        generateDefaultAvatar() {
            // 生成默认头像
            const name = this.displayName
            const firstLetter = name ? name.charAt(0).toUpperCase() : '?'
            const canvas = document.createElement('canvas')
            canvas.width = 40
            canvas.height = 40
            const ctx = canvas.getContext('2d')
            
            // 根据会话类型选择背景色
            const bgColor = this.conversation.type === 'single' ? '#52c41a' : '#1890ff'
            ctx.fillStyle = bgColor
            ctx.fillRect(0, 0, 40, 40)
            
            // 文字
            ctx.fillStyle = '#fff'
            ctx.font = 'bold 18px Arial'
            ctx.textAlign = 'center'
            ctx.textBaseline = 'middle'
            ctx.fillText(firstLetter, 20, 20)
            
            return canvas.toDataURL()
        },
        
        handleAvatarError(event) {
            event.target.src = this.generateDefaultAvatar()
        }
    }
}
</script>

<style scoped>
.conversation-item {
    display: flex;
    align-items: center;
    padding: 12px 16px;
    cursor: pointer;
    transition: background-color 0.2s;
    border-bottom: 1px solid #f5f5f5;
}

.conversation-item:hover {
    background-color: #f8f9fa;
}

.conversation-item.active {
    background-color: #e6f7ff;
    border-right: 3px solid #1890ff;
}

.avatar-container {
    position: relative;
    margin-right: 12px;
}

.avatar {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    object-fit: cover;
}

.online-indicator {
    position: absolute;
    bottom: 0;
    right: 0;
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background-color: #ccc;
    border: 2px solid #fff;
}

.online-indicator.online {
    background-color: #52c41a;
}

.content {
    flex: 1;
    min-width: 0;
}

.header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 4px;
}

.name {
    font-size: 14px;
    font-weight: 500;
    color: #333;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    margin-right: 8px;
}

.time {
    font-size: 11px;
    color: #999;
    white-space: nowrap;
}

.message-preview {
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.last-message {
    font-size: 12px;
    color: #666;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    margin-right: 8px;
}

.unread-badge {
    background-color: #ff4d4f;
    color: white;
    font-size: 10px;
    font-weight: 500;
    padding: 1px 5px;
    border-radius: 8px;
    min-width: 16px;
    text-align: center;
    white-space: nowrap;
}

/* 响应式设计 */
@media (max-width: 768px) {
    .conversation-item {
        padding: 10px 12px;
    }
    
    .avatar {
        width: 36px;
        height: 36px;
    }
    
    .online-indicator {
        width: 10px;
        height: 10px;
    }
    
    .name {
        font-size: 13px;
    }
    
    .last-message {
        font-size: 11px;
    }
    
    .time {
        font-size: 10px;
    }
}
</style>
