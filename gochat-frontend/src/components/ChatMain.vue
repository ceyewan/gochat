<template>
    <div class="chat-main">
        <!-- 聊天头部 -->
        <div class="chat-header" v-if="currentConversation">
            <div class="conversation-info">
                <img 
                    :src="conversationAvatar" 
                    :alt="conversationName"
                    class="conversation-avatar"
                    @error="handleAvatarError"
                />
                <div class="conversation-details">
                    <h3 class="conversation-name">{{ conversationName }}</h3>
                    <span class="conversation-status">{{ conversationStatus }}</span>
                </div>
            </div>
        </div>
        
        <!-- 消息显示区域 -->
        <div class="message-area" ref="messageArea">
            <div v-if="!currentConversation" class="no-conversation">
                <div class="welcome-message">
                    <h2>欢迎使用即时通讯</h2>
                    <p>选择一个会话开始聊天吧</p>
                </div>
            </div>
            <div v-else-if="loading" class="loading">
                加载消息中...
            </div>
            <div v-else class="message-list" @scroll="handleScroll">
                <!-- 加载更多按钮 -->
                <div v-if="hasMoreMessages" class="load-more">
                    <button @click="loadMoreMessages" :disabled="loadingMore">
                        {{ loadingMore ? '加载中...' : '加载更多消息' }}
                    </button>
                </div>
                
                <!-- 消息列表 -->
                <MessageBubble
                    v-for="message in messages"
                    :key="message.messageId"
                    :message="message"
                    :isOwnMessage="message.senderId === currentUserId"
                />
                
                <!-- 消息为空时的提示 -->
                <div v-if="messages.length === 0" class="empty-messages">
                    <p>还没有消息，开始聊天吧~</p>
                </div>
            </div>
        </div>
        
        <!-- 消息输入区域 -->
        <div class="input-area" v-if="currentConversation">
            <div class="input-container">
                <textarea
                    ref="messageInput"
                    v-model="messageContent"
                    placeholder="输入消息..."
                    rows="1"
                    :disabled="sending"
                    @keydown="handleKeyDown"
                    @input="handleInput"
                ></textarea>
                <button 
                    class="send-button"
                    @click="handleSendMessage"
                    :disabled="!canSend"
                >
                    {{ sending ? '发送中...' : '发送' }}
                </button>
            </div>
        </div>
    </div>
</template>

<script>
import { mapState, mapGetters, mapActions } from 'vuex'
import MessageBubble from './MessageBubble.vue'

export default {
    name: 'ChatMain',
    components: {
        MessageBubble
    },
    data() {
        return {
            messageContent: '',
            sending: false,
            loadingMore: false
        }
    },
    computed: {
        ...mapState('currentChat', ['currentConversation', 'messages', 'loading']),
        ...mapState('user', ['userInfo']),
        ...mapGetters('currentChat', ['hasMoreMessages']),
        ...mapGetters('onlineStatus', ['getFriendStatus']),
        
        currentUserId() {
            return this.userInfo?.userId
        },
        
        conversationName() {
            if (!this.currentConversation) return ''
            return this.currentConversation.type === 'single' 
                ? this.currentConversation.target?.username || '未知用户'
                : this.currentConversation.target?.groupName || '未知群聊'
        },
        
        conversationAvatar() {
            if (!this.currentConversation) return ''
            return this.currentConversation.target?.avatar || this.generateDefaultAvatar()
        },
        
        conversationStatus() {
            if (!this.currentConversation) return ''
            
            if (this.currentConversation.type === 'single') {
                const userId = this.currentConversation.target?.userId
                const isOnline = userId ? this.getFriendStatus(userId) : false
                return isOnline ? '在线' : '离线'
            } else {
                // 群聊显示成员数量
                const memberCount = this.currentConversation.target?.memberCount || 0
                return `${memberCount}人`
            }
        },
        
        canSend() {
            return !this.sending && this.messageContent.trim().length > 0
        }
    },
    watch: {
        currentConversation() {
            // 切换会话时清空输入框
            this.messageContent = ''
            // 滚动到底部
            this.$nextTick(() => {
                this.scrollToBottom()
            })
        },
        
        messages() {
            // 新消息时滚动到底部
            this.$nextTick(() => {
                this.scrollToBottom()
            })
        }
    },
    methods: {
        ...mapActions('currentChat', ['sendMessage', 'loadMoreMessages']),
        
        async handleSendMessage() {
            if (!this.canSend) return
            
            const content = this.messageContent.trim()
            this.messageContent = ''
            this.sending = true
            
            try {
                await this.sendMessage({ content, type: 'text' })
            } catch (error) {
                console.error('发送消息失败:', error)
                // 发送失败时恢复消息内容
                this.messageContent = content
            } finally {
                this.sending = false
                this.$refs.messageInput?.focus()
            }
        },
        
        handleKeyDown(event) {
            if (event.key === 'Enter' && !event.shiftKey) {
                event.preventDefault()
                this.handleSendMessage()
            }
        },
        
        handleInput(event) {
            // 自动调整textarea高度
            const textarea = event.target
            textarea.style.height = 'auto'
            const maxHeight = 120 // 最大高度
            const newHeight = Math.min(textarea.scrollHeight, maxHeight)
            textarea.style.height = newHeight + 'px'
        },
        
        async handleLoadMore() {
            if (this.loadingMore || !this.hasMoreMessages) return
            
            this.loadingMore = true
            try {
                await this.loadMoreMessages()
            } catch (error) {
                console.error('加载更多消息失败:', error)
            } finally {
                this.loadingMore = false
            }
        },
        
        handleScroll(event) {
            const { scrollTop } = event.target
            if (scrollTop === 0 && this.hasMoreMessages && !this.loadingMore) {
                this.handleLoadMore()
            }
        },
        
        scrollToBottom() {
            const messageArea = this.$refs.messageArea
            if (messageArea) {
                messageArea.scrollTop = messageArea.scrollHeight
            }
        },
        
        generateDefaultAvatar() {
            const name = this.conversationName
            const firstLetter = name ? name.charAt(0).toUpperCase() : '?'
            const canvas = document.createElement('canvas')
            canvas.width = 40
            canvas.height = 40
            const ctx = canvas.getContext('2d')
            
            const bgColor = this.currentConversation?.type === 'single' ? '#52c41a' : '#1890ff'
            ctx.fillStyle = bgColor
            ctx.fillRect(0, 0, 40, 40)
            
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
.chat-main {
    display: flex;
    flex-direction: column;
    height: 100%;
}

.chat-header {
    padding: 15px 20px;
    border-bottom: 1px solid #e5e5e5;
    background-color: #fafafa;
}

.conversation-info {
    display: flex;
    align-items: center;
}

.conversation-avatar {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    margin-right: 12px;
    object-fit: cover;
}

.conversation-details {
    flex: 1;
}

.conversation-name {
    font-size: 16px;
    font-weight: 600;
    color: #333;
    margin: 0 0 4px 0;
}

.conversation-status {
    font-size: 12px;
    color: #999;
}

.message-area {
    flex: 1;
    overflow-y: auto;
    background-color: #f8f9fa;
}

.no-conversation {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 100%;
}

.welcome-message {
    text-align: center;
    color: #999;
}

.welcome-message h2 {
    font-size: 24px;
    margin-bottom: 10px;
}

.welcome-message p {
    font-size: 14px;
}

.loading {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 200px;
    color: #999;
}

.message-list {
    padding: 20px;
    min-height: 100%;
}

.load-more {
    text-align: center;
    margin-bottom: 20px;
}

.load-more button {
    padding: 8px 16px;
    background-color: #f0f0f0;
    border: 1px solid #d9d9d9;
    border-radius: 4px;
    cursor: pointer;
    color: #666;
    font-size: 12px;
}

.load-more button:hover:not(:disabled) {
    background-color: #e6e6e6;
}

.load-more button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
}

.empty-messages {
    text-align: center;
    color: #999;
    font-size: 14px;
    margin-top: 50px;
}

.input-area {
    padding: 15px 20px;
    border-top: 1px solid #e5e5e5;
    background-color: #fff;
}

.input-container {
    display: flex;
    gap: 10px;
    align-items: flex-end;
}

.input-container textarea {
    flex: 1;
    padding: 10px 12px;
    border: 1px solid #d9d9d9;
    border-radius: 6px;
    resize: none;
    font-size: 14px;
    line-height: 1.4;
    min-height: 40px;
    max-height: 120px;
    outline: none;
    transition: border-color 0.2s;
}

.input-container textarea:focus {
    border-color: #0078ff;
}

.input-container textarea:disabled {
    background-color: #f5f5f5;
    cursor: not-allowed;
}

.send-button {
    padding: 10px 20px;
    background-color: #0078ff;
    color: white;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    font-size: 14px;
    font-weight: 500;
    transition: background-color 0.2s;
    white-space: nowrap;
}

.send-button:hover:not(:disabled) {
    background-color: #0056cc;
}

.send-button:disabled {
    background-color: #ccc;
    cursor: not-allowed;
}

/* 滚动条样式 */
.message-area::-webkit-scrollbar {
    width: 6px;
}

.message-area::-webkit-scrollbar-track {
    background: #f1f1f1;
}

.message-area::-webkit-scrollbar-thumb {
    background: #c1c1c1;
    border-radius: 3px;
}

.message-area::-webkit-scrollbar-thumb:hover {
    background: #a8a8a8;
}

/* 响应式设计 */
@media (max-width: 768px) {
    .chat-header {
        padding: 10px 15px;
    }
    
    .message-list {
        padding: 15px;
    }
    
    .input-area {
        padding: 10px 15px;
    }
    
    .send-button {
        padding: 8px 16px;
        font-size: 13px;
    }
}
</style>
