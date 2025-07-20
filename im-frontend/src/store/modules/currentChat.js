import request from '@/utils/request'
import { getWebSocket } from '@/utils/websocket'

const state = {
    currentConversation: null, // 当前选中的会话
    messages: [], // 当前会话的消息列表
    loading: false,
    hasMore: true, // 是否还有更多历史消息
    page: 1,
    pageSize: 50,
}

const mutations = {
    setCurrentConversation(state, conversation) {
        state.currentConversation = conversation
        state.messages = []
        state.page = 1
        state.hasMore = true
    },
    setMessages(state, messages) {
        state.messages = messages
    },
    addMessage(state, message) {
        // 检查消息是否已存在，避免重复
        const existingIndex = state.messages.findIndex(msg => msg.messageId === message.messageId)
        if (existingIndex >= 0) {
            // 更新现有消息
            state.messages.splice(existingIndex, 1, message)
        } else {
            // 添加新消息
            state.messages.push(message)
        }
    },
    prependMessages(state, messages) {
        // 在消息列表前面添加历史消息
        state.messages.unshift(...messages)
    },
    updateMessage(state, { messageId, updates }) {
        const message = state.messages.find(msg => msg.messageId === messageId)
        if (message) {
            Object.assign(message, updates)
        }
    },
    updateMessageStatus(state, { messageId, status, realMessageId }) {
        const message = state.messages.find(msg => msg.messageId === messageId)
        if (message) {
            message.status = status
            // 如果有真实的消息ID，更新消息ID
            if (realMessageId && realMessageId !== messageId) {
                message.messageId = realMessageId
            }
        }
    },
    setLoading(state, loading) {
        state.loading = loading
    },
    setHasMore(state, hasMore) {
        state.hasMore = hasMore
    },
    incrementPage(state) {
        state.page++
    },
    clearCurrentChat(state) {
        state.currentConversation = null
        state.messages = []
        state.loading = false
        state.hasMore = true
        state.page = 1
    },
}

const actions = {
    // 选择会话
    async selectConversation({ commit, dispatch }, conversation) {
        commit('setCurrentConversation', conversation)

        // 标记会话为已读
        if (conversation.unreadCount > 0) {
            await dispatch('conversations/markAsRead', conversation.conversationId, { root: true })
        }

        // 加载历史消息
        await dispatch('fetchMessages')
    },

    // 获取消息历史
    async fetchMessages({ commit, state }, { loadMore = false } = {}) {
        if (!state.currentConversation) return

        if (!loadMore) {
            commit('setLoading', true)
        }

        try {
            const response = await request.get(`/conversations/${state.currentConversation.conversationId}/messages`, {
                params: {
                    page: loadMore ? state.page : 1,
                    size: state.pageSize
                }
            })

            const messages = response.data || []

            if (loadMore) {
                commit('prependMessages', messages)
                commit('incrementPage')
            } else {
                commit('setMessages', messages)
                state.page = 1
            }

            // 检查是否还有更多消息
            if (messages.length < state.pageSize) {
                commit('setHasMore', false)
            }

            return response
        } catch (error) {
            console.error('获取消息历史失败:', error)
            throw error
        } finally {
            if (!loadMore) {
                commit('setLoading', false)
            }
        }
    },

    // 发送消息
    async sendMessage({ commit, state, rootState }, { content, type = 'text' }) {
        if (!state.currentConversation) {
            throw new Error('未选择会话')
        }

        const ws = getWebSocket()
        if (!ws || !ws.isConnected()) {
            throw new Error('WebSocket未连接')
        }

        // 生成临时消息ID
        const tempMessageId = `temp_${Date.now()}_${Math.random()}`

        // 构造消息对象
        const message = {
            messageId: tempMessageId,
            conversationId: state.currentConversation.conversationId,
            senderId: rootState.user.userInfo.userId,
            senderName: rootState.user.userInfo.username,
            content,
            type,
            sendTime: new Date().toISOString(),
            status: 'sending' // 发送中状态
        }

        // 先添加到本地消息列表
        commit('addMessage', message)

        // 通过WebSocket发送消息
        const success = ws.send({
            type: 'send-message',
            data: {
                conversationId: state.currentConversation.conversationId,
                content,
                messageType: type,
                tempMessageId: tempMessageId // 传递临时消息ID用于确认
            }
        })

        if (!success) {
            // 发送失败，更新消息状态
            commit('updateMessageStatus', {
                messageId: tempMessageId,
                status: 'failed'
            })
            throw new Error('消息发送失败')
        }

        // 设置超时处理，如果5秒内没有收到确认，标记为失败
        setTimeout(() => {
            const currentMessage = state.messages.find(msg => msg.messageId === tempMessageId)
            if (currentMessage && currentMessage.status === 'sending') {
                commit('updateMessageStatus', {
                    messageId: tempMessageId,
                    status: 'failed'
                })
            }
        }, 5000)

        return message
    },

    // 接收消息（WebSocket）
    receiveMessage({ commit, state }, messageData) {
        const { conversationId, message } = messageData

        // 如果是当前会话的消息，添加到消息列表
        if (state.currentConversation && state.currentConversation.conversationId === conversationId) {
            commit('addMessage', message)
        }

        // 更新会话列表中的最后消息
        this.dispatch('conversations/updateConversation', {
            type: 'new-message',
            conversationId,
            message
        })
    },

    // 更新消息状态
    updateMessageStatus({ commit }, { messageId, status }) {
        commit('updateMessageStatus', { messageId, status })
    },

    // 加载更多历史消息
    async loadMoreMessages({ dispatch, state }) {
        if (!state.hasMore || state.loading) return

        return await dispatch('fetchMessages', { loadMore: true })
    },
}

const getters = {
    currentConversation: state => state.currentConversation,
    messages: state => state.messages,
    isLoading: state => state.loading,
    hasMoreMessages: state => state.hasMore,
    messageById: state => id => {
        return state.messages.find(msg => msg.messageId === id)
    },
    // 按发送者分组的消息（用于显示连续消息）
    groupedMessages: state => {
        const grouped = []
        let currentGroup = null

        state.messages.forEach(message => {
            if (!currentGroup || currentGroup.senderId !== message.senderId) {
                currentGroup = {
                    senderId: message.senderId,
                    senderName: message.senderName,
                    messages: [message]
                }
                grouped.push(currentGroup)
            } else {
                currentGroup.messages.push(message)
            }
        })

        return grouped
    },
}

export default {
    namespaced: true,
    state,
    mutations,
    actions,
    getters,
}
