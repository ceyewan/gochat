<template>
    <div class="modal-overlay" @click="handleOverlayClick">
        <div class="modal-content" @click.stop>
            <div class="modal-header">
                <h3>添加好友</h3>
                <button class="close-btn" @click="closeModal">×</button>
            </div>
            
            <div class="modal-body">
                <!-- 搜索输入框 -->
                <div class="search-section">
                    <div class="input-group">
                        <input
                            v-model="searchUsername"
                            type="text"
                            placeholder="请输入好友用户名"
                            @keydown.enter="searchUser"
                            :disabled="searching"
                        />
                        <button 
                            class="search-btn"
                            @click="searchUser"
                            :disabled="!searchUsername.trim() || searching"
                        >
                            {{ searching ? '搜索中...' : '搜索' }}
                        </button>
                    </div>
                </div>
                
                <!-- 搜索结果 -->
                <div class="search-result" v-if="searchResult || searchError">
                    <div v-if="searchError" class="error-message">
                        {{ searchError }}
                    </div>
                    <div v-else-if="searchResult" class="user-info">
                        <div class="user-card">
                            <img 
                                :src="userAvatar" 
                                :alt="searchResult.username"
                                class="user-avatar"
                                @error="handleAvatarError"
                            />
                            <div class="user-details">
                                <h4 class="username">{{ searchResult.username }}</h4>
                                <p class="user-id">ID: {{ searchResult.userId }}</p>
                            </div>
                            <button 
                                class="add-btn"
                                @click="addFriend"
                                :disabled="adding"
                            >
                                {{ adding ? '添加中...' : '添加好友' }}
                            </button>
                        </div>
                    </div>
                </div>
                
                <!-- 成功提示 -->
                <div v-if="successMessage" class="success-message">
                    {{ successMessage }}
                </div>
            </div>
        </div>
    </div>
</template>

<script>
import { mapActions } from 'vuex'

export default {
    name: 'AddFriendModal',
    data() {
        return {
            searchUsername: '',
            searchResult: null,
            searchError: '',
            successMessage: '',
            searching: false,
            adding: false
        }
    },
    computed: {
        userAvatar() {
            if (!this.searchResult) return ''
            return this.searchResult.avatar || this.generateAvatar(this.searchResult.username)
        }
    },
    methods: {
        ...mapActions('conversations', ['addFriend']),
        
        async searchUser() {
            if (!this.searchUsername.trim() || this.searching) return
            
            this.searching = true
            this.searchResult = null
            this.searchError = ''
            this.successMessage = ''
            
            try {
                // 模拟搜索用户的API调用
                // 实际项目中应该调用真实的搜索接口
                const response = await this.$store.dispatch('conversations/searchUser', this.searchUsername.trim())
                this.searchResult = response.data
            } catch (error) {
                console.error('搜索用户失败:', error)
                if (error.response?.status === 404) {
                    this.searchError = '用户不存在'
                } else {
                    this.searchError = error.response?.data?.message || '搜索失败，请重试'
                }
            } finally {
                this.searching = false
            }
        },
        
        async addFriend() {
            if (!this.searchResult || this.adding) return
            
            this.adding = true
            this.searchError = ''
            
            try {
                await this.addFriend({ username: this.searchResult.username })
                this.successMessage = '好友添加成功！'
                
                // 延迟关闭弹窗
                setTimeout(() => {
                    this.closeModal()
                }, 2000)
                
            } catch (error) {
                console.error('添加好友失败:', error)
                this.searchError = error.response?.data?.message || '添加好友失败，请重试'
            } finally {
                this.adding = false
            }
        },
        
        closeModal() {
            this.$emit('close')
        },
        
        handleOverlayClick() {
            this.closeModal()
        },
        
        generateAvatar(username) {
            const firstLetter = username ? username.charAt(0).toUpperCase() : '?'
            const canvas = document.createElement('canvas')
            canvas.width = 60
            canvas.height = 60
            const ctx = canvas.getContext('2d')
            
            // 生成基于用户名的颜色
            const colors = ['#f56a00', '#7265e6', '#ffbf00', '#00a2ae', '#fa541c', '#eb2f96', '#722ed1', '#13c2c2']
            const colorIndex = username ? username.charCodeAt(0) % colors.length : 0
            
            ctx.fillStyle = colors[colorIndex]
            ctx.fillRect(0, 0, 60, 60)
            
            ctx.fillStyle = '#fff'
            ctx.font = 'bold 24px Arial'
            ctx.textAlign = 'center'
            ctx.textBaseline = 'middle'
            ctx.fillText(firstLetter, 30, 30)
            
            return canvas.toDataURL()
        },
        
        handleAvatarError(event) {
            event.target.src = this.generateAvatar(this.searchResult?.username || '')
        },
        
        handleKeyDown(event) {
            if (event.key === 'Escape') {
                this.closeModal()
            }
        }
    },
    
    mounted() {
        // 监听ESC键关闭弹窗
        document.addEventListener('keydown', this.handleKeyDown)
    },
    
    beforeUnmount() {
        document.removeEventListener('keydown', this.handleKeyDown)
    }
}
</script>

<style scoped>
.modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background-color: rgba(0, 0, 0, 0.5);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
}

.modal-content {
    background-color: white;
    border-radius: 8px;
    width: 90%;
    max-width: 500px;
    max-height: 80vh;
    overflow: hidden;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
}

.modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 20px;
    border-bottom: 1px solid #e5e5e5;
}

.modal-header h3 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: #333;
}

.close-btn {
    background: none;
    border: none;
    font-size: 24px;
    color: #999;
    cursor: pointer;
    padding: 0;
    width: 30px;
    height: 30px;
    display: flex;
    justify-content: center;
    align-items: center;
    border-radius: 50%;
    transition: background-color 0.2s;
}

.close-btn:hover {
    background-color: #f5f5f5;
    color: #666;
}

.modal-body {
    padding: 20px;
}

.search-section {
    margin-bottom: 20px;
}

.input-group {
    display: flex;
    gap: 10px;
}

.input-group input {
    flex: 1;
    padding: 10px 12px;
    border: 1px solid #d9d9d9;
    border-radius: 6px;
    font-size: 14px;
    outline: none;
    transition: border-color 0.2s;
}

.input-group input:focus {
    border-color: #0078ff;
}

.input-group input:disabled {
    background-color: #f5f5f5;
    cursor: not-allowed;
}

.search-btn {
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

.search-btn:hover:not(:disabled) {
    background-color: #0056cc;
}

.search-btn:disabled {
    background-color: #ccc;
    cursor: not-allowed;
}

.search-result {
    border: 1px solid #e5e5e5;
    border-radius: 6px;
    padding: 16px;
}

.error-message {
    color: #ff4d4f;
    text-align: center;
    font-size: 14px;
}

.success-message {
    color: #52c41a;
    text-align: center;
    font-size: 14px;
    margin-top: 16px;
    padding: 10px;
    background-color: #f6ffed;
    border: 1px solid #b7eb8f;
    border-radius: 4px;
}

.user-card {
    display: flex;
    align-items: center;
    gap: 12px;
}

.user-avatar {
    width: 60px;
    height: 60px;
    border-radius: 50%;
    object-fit: cover;
}

.user-details {
    flex: 1;
}

.username {
    margin: 0 0 4px 0;
    font-size: 16px;
    font-weight: 600;
    color: #333;
}

.user-id {
    margin: 0;
    font-size: 12px;
    color: #999;
}

.add-btn {
    padding: 8px 16px;
    background-color: #52c41a;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
    font-weight: 500;
    transition: background-color 0.2s;
}

.add-btn:hover:not(:disabled) {
    background-color: #389e0d;
}

.add-btn:disabled {
    background-color: #ccc;
    cursor: not-allowed;
}

/* 响应式设计 */
@media (max-width: 768px) {
    .modal-content {
        width: 95%;
        max-width: none;
    }
    
    .modal-header,
    .modal-body {
        padding: 15px;
    }
    
    .input-group {
        flex-direction: column;
    }
    
    .search-btn {
        width: 100%;
    }
    
    .user-card {
        flex-direction: column;
        text-align: center;
    }
    
    .add-btn {
        width: 100%;
    }
}
</style>
