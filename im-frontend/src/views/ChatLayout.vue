<template>
    <div class="chat-layout">
        <!-- 顶部导航栏 -->
        <Header />
        
        <!-- 主要内容区域 -->
        <div class="main-content">
            <!-- 左侧会话列表 -->
            <div class="conversation-sidebar">
                <ConversationList />
            </div>
            
            <!-- 右侧聊天区域 -->
            <div class="chat-main">
                <ChatMain />
            </div>
        </div>
        
        <!-- 弹窗组件 -->
        <AddFriendModal 
            v-if="showAddFriendModal" 
            @close="showAddFriendModal = false"
        />
        <CreateGroupModal 
            v-if="showCreateGroupModal" 
            @close="showCreateGroupModal = false"
        />
    </div>
</template>

<script>
import { mapState, mapActions } from 'vuex'
import Header from '@/components/common/Header.vue'
import ConversationList from '@/components/ConversationList.vue'
import ChatMain from '@/components/ChatMain.vue'
import AddFriendModal from '@/components/AddFriendModal.vue'
import CreateGroupModal from '@/components/CreateGroupModal.vue'

export default {
    name: 'ChatLayout',
    components: {
        Header,
        ConversationList,
        ChatMain,
        AddFriendModal,
        CreateGroupModal
    },
    data() {
        return {
            showAddFriendModal: false,
            showCreateGroupModal: false
        }
    },
    computed: {
        ...mapState('user', ['userInfo', 'token']),
        ...mapState('conversations', ['conversations']),
    },
    async mounted() {
        // 组件挂载时初始化数据
        try {
            // 如果没有用户信息，先获取用户信息
            if (!this.userInfo && this.token) {
                await this.fetchUserInfo()
            }

            // 加载会话列表
            await this.fetchConversations()

            // 初始化在线状态
            await this.initializeOnlineStatus()

            // 监听全局事件
            this.setupEventListeners()

        } catch (error) {
            console.error('初始化聊天界面失败:', error)
            // 如果初始化失败，可能是token无效，跳转到登录页
            this.$router.push('/login')
        }
    },
    beforeUnmount() {
        // 清理事件监听
        this.cleanupEventListeners()
    },
    methods: {
        ...mapActions('user', ['fetchUserInfo']),
        ...mapActions('conversations', ['fetchConversations']),
        ...mapActions('onlineStatus', ['initializeOnlineStatus']),
        
        setupEventListeners() {
            // 监听自定义事件
            window.addEventListener('show-add-friend-modal', this.handleShowAddFriendModal)
            window.addEventListener('show-create-group-modal', this.handleShowCreateGroupModal)
        },
        
        cleanupEventListeners() {
            window.removeEventListener('show-add-friend-modal', this.handleShowAddFriendModal)
            window.removeEventListener('show-create-group-modal', this.handleShowCreateGroupModal)
        },
        
        handleShowAddFriendModal() {
            this.showAddFriendModal = true
        },
        
        handleShowCreateGroupModal() {
            this.showCreateGroupModal = true
        },

        async initializeOnlineStatus() {
            try {
                // 获取在线状态数据
                await this.$store.dispatch('onlineStatus/fetchOnlineStatus')
                console.log('在线状态初始化完成')
            } catch (error) {
                console.error('获取在线状态失败:', error)
            }
        }
    }
}
</script>

<style scoped>
.chat-layout {
    display: flex;
    flex-direction: column;
    height: 100vh;
    width: 100vw;
    background-color: #f5f5f5;
    overflow: hidden;
}

.main-content {
    display: flex;
    flex: 1;
    overflow: hidden;
    min-height: 0; /* 确保flex子元素可以收缩 */
}

.conversation-sidebar {
    width: 250px;
    background-color: #fff;
    border-right: 1px solid #e5e5e5;
    display: flex;
    flex-direction: column;
}

.chat-main {
    flex: 1;
    background-color: #fff;
    display: flex;
    flex-direction: column;
}

/* 响应式设计 */
@media (max-width: 768px) {
    .chat-layout {
        height: 100vh;
        height: 100dvh; /* 动态视口高度，适配移动端 */
    }

    .main-content {
        flex-direction: column;
        height: calc(100vh - 50px); /* 减去header高度 */
        height: calc(100dvh - 50px);
    }

    .conversation-sidebar {
        width: 100%;
        height: 200px;
        border-right: none;
        border-bottom: 1px solid #e5e5e5;
        flex-shrink: 0;
    }

    .chat-main {
        flex: 1;
        min-height: 0;
    }
}
</style>
