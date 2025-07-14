import request from '@/utils/request'

const state = {
    conversations: [], // 会话列表
    loading: false,
}

const mutations = {
    setConversations(state, conversations) {
        state.conversations = conversations
    },
    addConversation(state, conversation) {
        const existingIndex = state.conversations.findIndex(
            conv => conv.conversationId === conversation.conversationId
        )
        if (existingIndex >= 0) {
            // 更新现有会话
            state.conversations.splice(existingIndex, 1, conversation)
        } else {
            // 添加新会话
            state.conversations.unshift(conversation)
        }
    },
    updateConversation(state, { conversationId, updates }) {
        const conversation = state.conversations.find(
            conv => conv.conversationId === conversationId
        )
        if (conversation) {
            Object.assign(conversation, updates)
        }
    },
    updateLastMessage(state, { conversationId, lastMessage, lastMessageTime }) {
        const conversation = state.conversations.find(
            conv => conv.conversationId === conversationId
        )
        if (conversation) {
            conversation.lastMessage = lastMessage
            conversation.lastMessageTime = lastMessageTime

            // 将会话移到最前面
            const index = state.conversations.indexOf(conversation)
            if (index > 0) {
                state.conversations.splice(index, 1)
                state.conversations.unshift(conversation)
            }
        }
    },
    incrementUnreadCount(state, conversationId) {
        const conversation = state.conversations.find(
            conv => conv.conversationId === conversationId
        )
        if (conversation) {
            conversation.unreadCount = (conversation.unreadCount || 0) + 1
        }
    },
    clearUnreadCount(state, conversationId) {
        const conversation = state.conversations.find(
            conv => conv.conversationId === conversationId
        )
        if (conversation) {
            conversation.unreadCount = 0
        }
    },
    setLoading(state, loading) {
        state.loading = loading
    },
    clearConversations(state) {
        state.conversations = []
        state.loading = false
    },
}

const actions = {
    // 获取会话列表
    async fetchConversations({ commit }) {
        commit('setLoading', true)
        try {
            const response = await request.get('/conversations')
            commit('setConversations', response.data || [])
            return response
        } catch (error) {
            console.error('获取会话列表失败:', error)
            throw error
        } finally {
            commit('setLoading', false)
        }
    },

    // 搜索用户
    async searchUser({ commit }, username) {
        try {
            const response = await request.get(`/users/${username}`)
            return response
        } catch (error) {
            console.error('搜索用户失败:', error)
            throw error
        }
    },

    // 添加好友（创建单聊会话）
    async addFriend({ commit }, { username }) {
        try {
            // 先搜索用户
            const searchResponse = await request.get(`/users/${username}`)
            const friendInfo = searchResponse.data

            // 添加好友
            const addResponse = await request.post('/friends', {
                friendId: friendInfo.userId
            })

            // 如果返回了会话信息，添加到会话列表
            if (addResponse.data.conversation) {
                commit('addConversation', addResponse.data.conversation)
            }

            return addResponse
        } catch (error) {
            console.error('添加好友失败:', error)
            throw error
        }
    },

    // 创建群聊
    async createGroup({ commit }, { groupName, memberIds }) {
        try {
            const response = await request.post('/groups', {
                groupName,
                members: memberIds
            })

            // 添加群聊会话到列表
            if (response.data.conversation) {
                commit('addConversation', response.data.conversation)
            }

            return response
        } catch (error) {
            console.error('创建群聊失败:', error)
            throw error
        }
    },

    // 标记会话为已读
    async markAsRead({ commit }, conversationId) {
        try {
            await request.put(`/conversations/${conversationId}/read`)
            commit('clearUnreadCount', conversationId)
        } catch (error) {
            console.error('标记已读失败:', error)
            throw error
        }
    },

    // 更新会话信息（通过WebSocket接收）
    updateConversation({ commit }, conversationData) {
        if (conversationData.type === 'new-message') {
            const { conversationId, message } = conversationData
            commit('updateLastMessage', {
                conversationId,
                lastMessage: message.content,
                lastMessageTime: message.sendTime
            })

            // 如果不是当前会话，增加未读计数
            const currentConversationId = this.state.currentChat.currentConversation?.conversationId
            if (conversationId !== currentConversationId) {
                commit('incrementUnreadCount', conversationId)
            }
        } else if (conversationData.type === 'conversation-update') {
            commit('updateConversation', {
                conversationId: conversationData.conversationId,
                updates: conversationData.updates
            })
        }
    },
}

const getters = {
    conversationList: state => state.conversations,
    conversationById: state => id => {
        return state.conversations.find(conv => conv.conversationId === id)
    },
    totalUnreadCount: state => {
        return state.conversations.reduce((total, conv) => total + (conv.unreadCount || 0), 0)
    },
    isLoading: state => state.loading,
}

export default {
    namespaced: true,
    state,
    mutations,
    actions,
    getters,
}
