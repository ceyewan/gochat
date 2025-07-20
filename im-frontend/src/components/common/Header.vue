<template>
    <div class="header">
        <!-- 左侧用户信息 -->
        <div class="user-info">
            <img 
                :src="userAvatar" 
                class="user-avatar" 
                :alt="username"
                @error="handleAvatarError"
            />
            <span class="username">{{ username }}</span>
        </div>
        
        <!-- 右侧操作按钮 -->
        <div class="header-actions">
            <!-- 在线状态显示 -->
            <div class="online-status">
                <span class="status-dot online"></span>
                <span class="status-text">在线</span>
            </div>
            
            <!-- 登出按钮 -->
            <button 
                class="logout-btn" 
                @click="handleLogout"
                :disabled="logoutLoading"
            >
                {{ logoutLoading ? '登出中...' : '登出' }}
            </button>
        </div>
    </div>
</template>

<script>
import { mapState, mapActions, mapGetters } from 'vuex'

export default {
    name: 'Header',
    data() {
        return {
            logoutLoading: false,
            defaultAvatar: '/src/assets/default-avatar.png'
        }
    },
    computed: {
        ...mapState('user', ['userInfo']),
        ...mapGetters('user', ['username', 'avatar']),
        
        userAvatar() {
            return this.avatar || this.generateDefaultAvatar()
        }
    },
    methods: {
        ...mapActions('user', ['logout']),
        
        async handleLogout() {
            if (this.logoutLoading) return
            
            this.logoutLoading = true
            
            try {
                await this.logout()
                // logout action 会自动跳转到登录页
            } catch (error) {
                console.error('登出失败:', error)
                // 即使登出失败，也清除本地状态
                this.$router.push('/login')
            } finally {
                this.logoutLoading = false
            }
        },
        
        generateDefaultAvatar() {
            // 生成简单的默认头像（基于用户名首字母）
            const firstLetter = this.username ? this.username.charAt(0).toUpperCase() : '?'
            const canvas = document.createElement('canvas')
            canvas.width = 64
            canvas.height = 64
            const ctx = canvas.getContext('2d')
            
            // 背景色
            ctx.fillStyle = '#0078ff'
            ctx.fillRect(0, 0, 64, 64)
            
            // 文字
            ctx.fillStyle = '#fff'
            ctx.font = 'bold 32px Arial'
            ctx.textAlign = 'center'
            ctx.textBaseline = 'middle'
            ctx.fillText(firstLetter, 32, 32)
            
            return canvas.toDataURL()
        },
        
        handleAvatarError(event) {
            // 头像加载失败时使用默认头像
            event.target.src = this.generateDefaultAvatar()
        }
    }
}
</script>

<style scoped>
.header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    height: 50px;
    padding: 0 20px;
    background-color: #fff;
    border-bottom: 1px solid #e5e5e5;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.user-info {
    display: flex;
    align-items: center;
}

.user-avatar {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    margin-right: 10px;
    object-fit: cover;
    border: 2px solid #f0f0f0;
}

.username {
    font-size: 16px;
    font-weight: 500;
    color: #333;
}

.header-actions {
    display: flex;
    align-items: center;
    gap: 15px;
}

.online-status {
    display: flex;
    align-items: center;
    gap: 5px;
}

.status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background-color: #ccc;
}

.status-dot.online {
    background-color: #52c41a;
}

.status-dot.offline {
    background-color: #ccc;
}

.status-text {
    font-size: 12px;
    color: #666;
}

.logout-btn {
    padding: 6px 12px;
    background-color: transparent;
    color: #0078ff;
    border: 1px solid #0078ff;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
    transition: all 0.3s;
}

.logout-btn:hover:not(:disabled) {
    background-color: #0078ff;
    color: white;
}

.logout-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
}

/* 响应式设计 */
@media (max-width: 768px) {
    .header {
        padding: 0 15px;
    }
    
    .username {
        display: none;
    }
    
    .online-status .status-text {
        display: none;
    }
    
    .logout-btn {
        padding: 4px 8px;
        font-size: 12px;
    }
}
</style>
