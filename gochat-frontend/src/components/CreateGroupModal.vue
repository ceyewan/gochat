<template>
    <div class="modal-overlay" @click="handleOverlayClick">
        <div class="modal-content" @click.stop>
            <div class="modal-header">
                <h3>创建群聊</h3>
                <button class="close-btn" @click="closeModal">×</button>
            </div>
            
            <div class="modal-body">
                <!-- 群名称输入 -->
                <div class="form-section">
                    <label for="groupName">群名称</label>
                    <input
                        id="groupName"
                        v-model="groupName"
                        type="text"
                        placeholder="请输入群名称"
                        :disabled="creating"
                        maxlength="50"
                    />
                </div>
                
                <!-- 选择成员 -->
                <div class="form-section">
                    <label>选择群成员</label>
                    <div class="member-selection">
                        <!-- 搜索框 -->
                        <div class="search-box">
                            <input
                                v-model="searchKeyword"
                                type="text"
                                placeholder="搜索好友..."
                                :disabled="creating"
                            />
                        </div>
                        
                        <!-- 好友列表 -->
                        <div class="friend-list">
                            <div 
                                v-for="friend in filteredFriends"
                                :key="friend.userId"
                                class="friend-item"
                                :class="{ selected: isSelected(friend.userId) }"
                                @click="toggleFriend(friend)"
                            >
                                <img 
                                    :src="friend.avatar || generateAvatar(friend.username)" 
                                    :alt="friend.username"
                                    class="friend-avatar"
                                    @error="(e) => handleAvatarError(e, friend.username)"
                                />
                                <div class="friend-info">
                                    <span class="friend-name">{{ friend.username }}</span>
                                    <span class="friend-status" :class="{ online: friend.online }">
                                        {{ friend.online ? '在线' : '离线' }}
                                    </span>
                                </div>
                                <div class="select-indicator">
                                    <span v-if="isSelected(friend.userId)" class="selected-mark">✓</span>
                                </div>
                            </div>
                            
                            <div v-if="filteredFriends.length === 0" class="no-friends">
                                {{ searchKeyword ? '没有找到匹配的好友' : '暂无好友' }}
                            </div>
                        </div>
                        
                        <!-- 已选择的成员 -->
                        <div v-if="selectedMembers.length > 0" class="selected-members">
                            <h4>已选择成员 ({{ selectedMembers.length }})</h4>
                            <div class="selected-list">
                                <div 
                                    v-for="member in selectedMembers"
                                    :key="member.userId"
                                    class="selected-member"
                                >
                                    <img 
                                        :src="member.avatar || generateAvatar(member.username)" 
                                        :alt="member.username"
                                        class="member-avatar"
                                    />
                                    <span class="member-name">{{ member.username }}</span>
                                    <button 
                                        class="remove-btn"
                                        @click="removeMember(member)"
                                        :disabled="creating"
                                    >
                                        ×
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                
                <!-- 错误提示 -->
                <div v-if="errorMessage" class="error-message">
                    {{ errorMessage }}
                </div>
                
                <!-- 成功提示 -->
                <div v-if="successMessage" class="success-message">
                    {{ successMessage }}
                </div>
            </div>
            
            <div class="modal-footer">
                <button class="cancel-btn" @click="closeModal" :disabled="creating">
                    取消
                </button>
                <button 
                    class="create-btn" 
                    @click="handleCreateGroup"
                    :disabled="!canCreate || creating"
                >
                    {{ creating ? '创建中...' : '创建群聊' }}
                </button>
            </div>
        </div>
    </div>
</template>

<script>
import { mapActions, mapState } from 'vuex'

export default {
    name: 'CreateGroupModal',
    data() {
        return {
            groupName: '',
            searchKeyword: '',
            selectedMembers: [],
            errorMessage: '',
            successMessage: '',
            creating: false,
            // 模拟好友列表数据，实际项目中应该从store获取
            mockFriends: [
                { userId: 'user1', username: '张三', avatar: '', online: true },
                { userId: 'user2', username: '李四', avatar: '', online: false },
                { userId: 'user3', username: '王五', avatar: '', online: true },
                { userId: 'user4', username: '赵六', avatar: '', online: false },
            ]
        }
    },
    computed: {
        ...mapState('user', ['userInfo']),
        
        // 过滤后的好友列表
        filteredFriends() {
            if (!this.searchKeyword.trim()) {
                return this.mockFriends
            }
            
            const keyword = this.searchKeyword.trim().toLowerCase()
            return this.mockFriends.filter(friend => 
                friend.username.toLowerCase().includes(keyword)
            )
        },
        
        // 是否可以创建群聊
        canCreate() {
            return this.groupName.trim().length > 0 && this.selectedMembers.length > 0
        }
    },
    methods: {
        ...mapActions('conversations', ['createGroup']),
        
        // 切换好友选择状态
        toggleFriend(friend) {
            if (this.creating) return
            
            if (this.isSelected(friend.userId)) {
                this.removeMember(friend)
            } else {
                this.selectedMembers.push(friend)
            }
        },
        
        // 检查好友是否已选择
        isSelected(userId) {
            return this.selectedMembers.some(member => member.userId === userId)
        },
        
        // 移除成员
        removeMember(member) {
            if (this.creating) return
            
            const index = this.selectedMembers.findIndex(m => m.userId === member.userId)
            if (index >= 0) {
                this.selectedMembers.splice(index, 1)
            }
        },
        
        // 创建群聊
        async handleCreateGroup() {
            if (!this.canCreate || this.creating) return
            
            this.creating = true
            this.errorMessage = ''
            this.successMessage = ''
            
            try {
                const memberIds = this.selectedMembers.map(member => member.userId)
                
                await this.createGroup({
                    groupName: this.groupName.trim(),
                    memberIds
                })
                
                this.successMessage = '群聊创建成功！'
                
                // 延迟关闭弹窗
                setTimeout(() => {
                    this.closeModal()
                }, 2000)
                
            } catch (error) {
                console.error('创建群聊失败:', error)
                this.errorMessage = error.response?.data?.message || '创建群聊失败，请重试'
            } finally {
                this.creating = false
            }
        },
        
        // 关闭弹窗
        closeModal() {
            this.$emit('close')
        },
        
        // 点击遮罩关闭
        handleOverlayClick() {
            this.closeModal()
        },
        
        // 生成默认头像
        generateAvatar(username) {
            const firstLetter = username ? username.charAt(0).toUpperCase() : '?'
            const canvas = document.createElement('canvas')
            canvas.width = 40
            canvas.height = 40
            const ctx = canvas.getContext('2d')
            
            const colors = ['#f56a00', '#7265e6', '#ffbf00', '#00a2ae', '#fa541c', '#eb2f96', '#722ed1', '#13c2c2']
            const colorIndex = username ? username.charCodeAt(0) % colors.length : 0
            
            ctx.fillStyle = colors[colorIndex]
            ctx.fillRect(0, 0, 40, 40)
            
            ctx.fillStyle = '#fff'
            ctx.font = 'bold 16px Arial'
            ctx.textAlign = 'center'
            ctx.textBaseline = 'middle'
            ctx.fillText(firstLetter, 20, 20)
            
            return canvas.toDataURL()
        },
        
        // 头像加载错误处理
        handleAvatarError(event, username) {
            event.target.src = this.generateAvatar(username)
        },
        
        // 键盘事件处理
        handleKeyDown(event) {
            if (event.key === 'Escape') {
                this.closeModal()
            }
        }
    },
    
    mounted() {
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
    max-width: 600px;
    max-height: 80vh;
    overflow: hidden;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
    display: flex;
    flex-direction: column;
}

.modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 20px;
    border-bottom: 1px solid #e5e5e5;
    flex-shrink: 0;
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
    flex: 1;
    overflow-y: auto;
}

.form-section {
    margin-bottom: 20px;
}

.form-section label {
    display: block;
    margin-bottom: 8px;
    font-size: 14px;
    font-weight: 500;
    color: #333;
}

.form-section input {
    width: 100%;
    padding: 10px 12px;
    border: 1px solid #d9d9d9;
    border-radius: 6px;
    font-size: 14px;
    outline: none;
    transition: border-color 0.2s;
}

.form-section input:focus {
    border-color: #0078ff;
}

.form-section input:disabled {
    background-color: #f5f5f5;
    cursor: not-allowed;
}

.member-selection {
    border: 1px solid #e5e5e5;
    border-radius: 6px;
    overflow: hidden;
}

.search-box {
    padding: 12px;
    border-bottom: 1px solid #e5e5e5;
}

.search-box input {
    width: 100%;
    padding: 8px 12px;
    border: 1px solid #d9d9d9;
    border-radius: 4px;
    font-size: 13px;
}

.friend-list {
    max-height: 200px;
    overflow-y: auto;
}

.friend-item {
    display: flex;
    align-items: center;
    padding: 12px;
    cursor: pointer;
    transition: background-color 0.2s;
    border-bottom: 1px solid #f5f5f5;
}

.friend-item:hover {
    background-color: #f8f9fa;
}

.friend-item.selected {
    background-color: #e6f7ff;
}

.friend-avatar {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    margin-right: 12px;
    object-fit: cover;
}

.friend-info {
    flex: 1;
}

.friend-name {
    display: block;
    font-size: 14px;
    font-weight: 500;
    color: #333;
    margin-bottom: 2px;
}

.friend-status {
    font-size: 12px;
    color: #999;
}

.friend-status.online {
    color: #52c41a;
}

.select-indicator {
    width: 20px;
    text-align: center;
}

.selected-mark {
    color: #0078ff;
    font-weight: bold;
    font-size: 16px;
}

.no-friends {
    padding: 20px;
    text-align: center;
    color: #999;
    font-size: 14px;
}

.selected-members {
    margin-top: 16px;
    padding: 12px;
    background-color: #f8f9fa;
    border-radius: 4px;
}

.selected-members h4 {
    margin: 0 0 8px 0;
    font-size: 14px;
    color: #333;
}

.selected-list {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
}

.selected-member {
    display: flex;
    align-items: center;
    padding: 4px 8px;
    background-color: white;
    border: 1px solid #d9d9d9;
    border-radius: 16px;
    font-size: 12px;
}

.member-avatar {
    width: 20px;
    height: 20px;
    border-radius: 50%;
    margin-right: 6px;
}

.member-name {
    margin-right: 6px;
}

.remove-btn {
    background: none;
    border: none;
    color: #999;
    cursor: pointer;
    font-size: 14px;
    padding: 0;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    display: flex;
    justify-content: center;
    align-items: center;
}

.remove-btn:hover {
    background-color: #f5f5f5;
    color: #666;
}

.error-message {
    color: #ff4d4f;
    font-size: 14px;
    margin-top: 8px;
    text-align: center;
}

.success-message {
    color: #52c41a;
    font-size: 14px;
    margin-top: 8px;
    text-align: center;
    padding: 10px;
    background-color: #f6ffed;
    border: 1px solid #b7eb8f;
    border-radius: 4px;
}

.modal-footer {
    padding: 16px 20px;
    border-top: 1px solid #e5e5e5;
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    flex-shrink: 0;
}

.cancel-btn {
    padding: 8px 16px;
    background-color: transparent;
    color: #666;
    border: 1px solid #d9d9d9;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
}

.cancel-btn:hover:not(:disabled) {
    border-color: #999;
    color: #333;
}

.create-btn {
    padding: 8px 16px;
    background-color: #0078ff;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
    font-weight: 500;
}

.create-btn:hover:not(:disabled) {
    background-color: #0056cc;
}

.create-btn:disabled {
    background-color: #ccc;
    cursor: not-allowed;
}

/* 滚动条样式 */
.friend-list::-webkit-scrollbar,
.modal-body::-webkit-scrollbar {
    width: 4px;
}

.friend-list::-webkit-scrollbar-track,
.modal-body::-webkit-scrollbar-track {
    background: #f1f1f1;
}

.friend-list::-webkit-scrollbar-thumb,
.modal-body::-webkit-scrollbar-thumb {
    background: #c1c1c1;
    border-radius: 2px;
}

/* 响应式设计 */
@media (max-width: 768px) {
    .modal-content {
        width: 95%;
        max-width: none;
    }
    
    .modal-header,
    .modal-body,
    .modal-footer {
        padding: 15px;
    }
    
    .friend-list {
        max-height: 150px;
    }
    
    .selected-list {
        justify-content: center;
    }
    
    .modal-footer {
        flex-direction: column;
    }
    
    .cancel-btn,
    .create-btn {
        width: 100%;
    }
}
</style>
