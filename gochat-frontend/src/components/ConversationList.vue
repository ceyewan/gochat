<template>
    <div class="conversation-list">
        <!-- 标题栏 -->
        <div class="list-header">
            <h3 class="list-title">会话列表</h3>
            <div class="unread-count" v-if="totalUnreadCount > 0">
                {{ totalUnreadCount }}
            </div>
        </div>
        
        <!-- 会话列表 -->
        <div class="list-content">
            <div v-if="loading" class="loading">
                加载中...
            </div>
            <div v-else-if="conversationList.length === 0" class="empty">
                暂无会话
            </div>
            <div v-else class="conversation-items">
                <ConversationItem
                    v-for="conversation in conversationList"
                    :key="conversation.conversationId"
                    :conversation="conversation"
                    :isActive="currentConversation?.conversationId === conversation.conversationId"
                    @select="handleSelectConversation"
                />
            </div>
        </div>
        
        <!-- 底部操作按钮 -->
        <div class="list-footer">
            <button class="action-btn" @click="showAddFriend">
                <span class="btn-icon">+</span>
                <span class="btn-text">添加好友</span>
            </button>
            <button class="action-btn" @click="showCreateGroup">
                <span class="btn-icon">👥</span>
                <span class="btn-text">创建群聊</span>
            </button>
        </div>
    </div>
</template>

<script>
import { mapState, mapGetters, mapActions } from 'vuex'
import ConversationItem from './ConversationItem.vue'

export default {
    name: 'ConversationList',
    components: {
        ConversationItem
    },
    computed: {
        ...mapState('conversations', ['loading']),
        ...mapState('currentChat', ['currentConversation']),
        ...mapGetters('conversations', ['conversationList', 'totalUnreadCount']),
    },
    methods: {
        ...mapActions('currentChat', ['selectConversation']),
        
        async handleSelectConversation(conversation) {
            try {
                await this.selectConversation(conversation)
            } catch (error) {
                console.error('选择会话失败:', error)
            }
        },
        
        showAddFriend() {
            // 触发全局事件显示添加好友弹窗
            window.dispatchEvent(new Event('show-add-friend-modal'))
        },
        
        showCreateGroup() {
            // 触发全局事件显示创建群聊弹窗
            window.dispatchEvent(new Event('show-create-group-modal'))
        }
    }
}
</script>

<style scoped>
.conversation-list {
    display: flex;
    flex-direction: column;
    height: 100%;
}

.list-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 15px 20px;
    border-bottom: 1px solid #e5e5e5;
}

.list-title {
    font-size: 16px;
    font-weight: 600;
    color: #333;
    margin: 0;
}

.unread-count {
    background-color: #ff4d4f;
    color: white;
    font-size: 12px;
    font-weight: 500;
    padding: 2px 6px;
    border-radius: 10px;
    min-width: 18px;
    text-align: center;
}

.list-content {
    flex: 1;
    overflow-y: auto;
}

.loading, .empty {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 200px;
    color: #999;
    font-size: 14px;
}

.conversation-items {
    padding: 0;
}

.list-footer {
    padding: 15px 20px;
    border-top: 1px solid #e5e5e5;
    display: flex;
    gap: 10px;
}

.action-btn {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 5px;
    padding: 8px 12px;
    background-color: #f8f9fa;
    border: 1px solid #e9ecef;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s;
    font-size: 12px;
}

.action-btn:hover {
    background-color: #e9ecef;
    border-color: #dee2e6;
}

.btn-icon {
    font-size: 14px;
}

.btn-text {
    color: #495057;
    font-weight: 500;
}

/* 滚动条样式 */
.list-content::-webkit-scrollbar {
    width: 4px;
}

.list-content::-webkit-scrollbar-track {
    background: #f1f1f1;
}

.list-content::-webkit-scrollbar-thumb {
    background: #c1c1c1;
    border-radius: 2px;
}

.list-content::-webkit-scrollbar-thumb:hover {
    background: #a8a8a8;
}

/* 响应式设计 */
@media (max-width: 768px) {
    .list-header {
        padding: 10px 15px;
    }
    
    .list-footer {
        padding: 10px 15px;
    }
    
    .action-btn {
        padding: 6px 8px;
        font-size: 11px;
    }
    
    .btn-text {
        display: none;
    }
}
</style>
