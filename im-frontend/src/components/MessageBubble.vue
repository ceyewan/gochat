<template>
    <div class="message-bubble" :class="{ 'own-message': isOwnMessage }">
        <div class="message-content">
            <!-- 发送者头像（对方消息显示，自己的消息不显示） -->
            <div class="avatar-container" v-if="!isOwnMessage">
                <img 
                    :src="senderAvatar" 
                    :alt="message.senderName"
                    class="sender-avatar"
                    @error="handleAvatarError"
                />
            </div>
            
            <!-- 消息气泡 -->
            <div class="bubble-container">
                <!-- 发送者名称（群聊中对方消息显示） -->
                <div class="sender-name" v-if="!isOwnMessage && showSenderName">
                    {{ message.senderName }}
                </div>
                
                <!-- 消息气泡 -->
                <div class="bubble" :class="bubbleClass">
                    <div class="message-text">{{ message.content }}</div>
                    
                    <!-- 消息状态和时间 -->
                    <div class="message-meta">
                        <span class="message-time">{{ formattedTime }}</span>
                        <span class="message-status" v-if="isOwnMessage">
                            {{ statusText }}
                        </span>
                    </div>
                </div>
            </div>
            
            <!-- 自己的头像（自己的消息显示） -->
            <div class="avatar-container" v-if="isOwnMessage">
                <img 
                    :src="ownAvatar" 
                    :alt="ownName"
                    class="sender-avatar"
                    @error="handleOwnAvatarError"
                />
            </div>
        </div>
    </div>
</template>

<script>
import { mapState, mapGetters } from 'vuex'

export default {
    name: 'MessageBubble',
    props: {
        message: {
            type: Object,
            required: true
        },
        isOwnMessage: {
            type: Boolean,
            default: false
        },
        showSenderName: {
            type: Boolean,
            default: true // 在群聊中显示发送者名称
        }
    },
    computed: {
        ...mapState('user', ['userInfo']),
        ...mapState('currentChat', ['currentConversation']),
        
        formattedTime() {
            if (!this.message.sendTime) return ''
            
            const messageTime = new Date(this.message.sendTime)
            const now = new Date()
            const diffMs = now - messageTime
            const diffMinutes = Math.floor(diffMs / 60000)
            const diffHours = Math.floor(diffMs / 3600000)
            const diffDays = Math.floor(diffMs / 86400000)
            
            if (diffMinutes < 1) {
                return messageTime.toLocaleTimeString('zh-CN', { 
                    hour: '2-digit', 
                    minute: '2-digit' 
                })
            } else if (diffMinutes < 60) {
                return `${diffMinutes}分钟前`
            } else if (diffHours < 24) {
                return messageTime.toLocaleTimeString('zh-CN', { 
                    hour: '2-digit', 
                    minute: '2-digit' 
                })
            } else if (diffDays < 7) {
                return messageTime.toLocaleDateString('zh-CN', { 
                    month: 'short', 
                    day: 'numeric',
                    hour: '2-digit', 
                    minute: '2-digit' 
                })
            } else {
                return messageTime.toLocaleDateString('zh-CN', { 
                    year: 'numeric',
                    month: 'short', 
                    day: 'numeric'
                })
            }
        },
        
        statusText() {
            if (!this.isOwnMessage) return ''
            
            switch (this.message.status) {
                case 'sending':
                    return '发送中'
                case 'sent':
                    return '已发送'
                case 'failed':
                    return '发送失败'
                default:
                    return ''
            }
        },
        
        bubbleClass() {
            const classes = []
            
            if (this.isOwnMessage) {
                classes.push('own-bubble')
            } else {
                classes.push('other-bubble')
            }
            
            if (this.message.status === 'failed') {
                classes.push('failed-bubble')
            }
            
            return classes
        },
        
        senderAvatar() {
            // 对方的头像，可以从消息中获取或者生成默认头像
            return this.generateAvatar(this.message.senderName)
        },
        
        ownAvatar() {
            return this.userInfo?.avatar || this.generateAvatar(this.userInfo?.username || 'Me')
        },
        
        ownName() {
            return this.userInfo?.username || 'Me'
        }
    },
    methods: {
        generateAvatar(name) {
            const firstLetter = name ? name.charAt(0).toUpperCase() : '?'
            const canvas = document.createElement('canvas')
            canvas.width = 32
            canvas.height = 32
            const ctx = canvas.getContext('2d')
            
            // 根据名字生成颜色
            const colors = ['#f56a00', '#7265e6', '#ffbf00', '#00a2ae', '#fa541c', '#eb2f96', '#722ed1', '#13c2c2']
            const colorIndex = name ? name.charCodeAt(0) % colors.length : 0
            
            ctx.fillStyle = colors[colorIndex]
            ctx.fillRect(0, 0, 32, 32)
            
            ctx.fillStyle = '#fff'
            ctx.font = 'bold 14px Arial'
            ctx.textAlign = 'center'
            ctx.textBaseline = 'middle'
            ctx.fillText(firstLetter, 16, 16)
            
            return canvas.toDataURL()
        },
        
        handleAvatarError(event) {
            event.target.src = this.generateAvatar(this.message.senderName)
        },
        
        handleOwnAvatarError(event) {
            event.target.src = this.generateAvatar(this.ownName)
        }
    }
}
</script>

<style scoped>
.message-bubble {
    margin-bottom: 16px;
}

.message-bubble.own-message {
    display: flex;
    justify-content: flex-end;
}

.message-content {
    display: flex;
    align-items: flex-end;
    max-width: 70%;
    gap: 8px;
}

.own-message .message-content {
    flex-direction: row-reverse;
}

.avatar-container {
    flex-shrink: 0;
}

.sender-avatar {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    object-fit: cover;
}

.bubble-container {
    flex: 1;
    min-width: 0;
}

.sender-name {
    font-size: 12px;
    color: #666;
    margin-bottom: 4px;
    padding-left: 8px;
}

.own-message .sender-name {
    text-align: right;
    padding-left: 0;
    padding-right: 8px;
}

.bubble {
    position: relative;
    padding: 10px 12px;
    border-radius: 12px;
    word-wrap: break-word;
    word-break: break-word;
}

.bubble.other-bubble {
    background-color: #fff;
    border: 1px solid #e5e5e5;
    border-bottom-left-radius: 4px;
}

.bubble.own-bubble {
    background-color: #0078ff;
    color: white;
    border-bottom-right-radius: 4px;
}

.bubble.failed-bubble {
    background-color: #ff4d4f;
    color: white;
}

.message-text {
    font-size: 14px;
    line-height: 1.4;
    margin-bottom: 4px;
}

.message-meta {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 6px;
    font-size: 11px;
    opacity: 0.7;
}

.other-bubble .message-meta {
    color: #999;
}

.own-bubble .message-meta {
    color: rgba(255, 255, 255, 0.8);
}

.message-time {
    white-space: nowrap;
}

.message-status {
    white-space: nowrap;
}

.message-status {
    font-size: 10px;
}

/* 消息状态颜色 */
.bubble.failed-bubble .message-status {
    color: rgba(255, 255, 255, 0.9);
}

/* 响应式设计 */
@media (max-width: 768px) {
    .message-content {
        max-width: 85%;
    }
    
    .sender-avatar {
        width: 28px;
        height: 28px;
    }
    
    .bubble {
        padding: 8px 10px;
    }
    
    .message-text {
        font-size: 13px;
    }
    
    .message-meta {
        font-size: 10px;
    }
}
</style>
