import store from '@/store'

export class IMWebSocket {
    constructor(token) {
        this.token = token
        this.url = `${import.meta.env.VITE_WS_BASE_URL}?token=${token}`
        this.socket = null
        this.reconnectCount = 0
        this.maxReconnect = 3
        this.reconnectTimer = null
        this.isManualClose = false
        this.init()
    }

    init() {
        try {
            this.socket = new WebSocket(this.url)
            this.bindEvents()
        } catch (error) {
            console.error('WebSocket连接失败:', error)
            this.reconnect()
        }
    }

    bindEvents() {
        this.socket.onopen = () => {
            console.log('WebSocket连接成功')
            this.reconnectCount = 0
            this.isManualClose = false
        }

        this.socket.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data)
                this.handleMessage(data)
            } catch (error) {
                console.error('解析WebSocket消息失败:', error)
            }
        }

        this.socket.onclose = (event) => {
            console.log('WebSocket连接关闭:', event.code, event.reason)
            if (!this.isManualClose) {
                this.reconnect()
            }
        }

        this.socket.onerror = (error) => {
            console.error('WebSocket错误:', error)
        }
    }

    reconnect() {
        if (this.reconnectCount < this.maxReconnect) {
            this.reconnectCount++
            console.log(`正在重连WebSocket（第${this.reconnectCount}次）`)

            this.reconnectTimer = setTimeout(() => {
                this.init()
            }, 1000 * this.reconnectCount) // 递增延迟重连
        } else {
            console.error('WebSocket重连失败，请刷新页面')
            // 可以在这里触发一个全局事件，通知用户网络问题
        }
    }

    send(message) {
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            try {
                this.socket.send(JSON.stringify(message))
                return true
            } catch (error) {
                console.error('发送WebSocket消息失败:', error)
                return false
            }
        } else {
            console.error('WebSocket未连接，无法发送消息')
            return false
        }
    }

    handleMessage(data) {
        console.log('收到WebSocket消息:', data)

        switch (data.type) {
            case 'connected':
                // WebSocket连接确认
                console.log('WebSocket连接确认:', data.data)
                break
            case 'new-message':
                // 新消息
                store.dispatch('currentChat/receiveMessage', data.data)
                break
            case 'message-ack':
                // 消息确认
                store.dispatch('currentChat/updateMessageStatus', {
                    messageId: data.data.messageId,
                    status: 'sent'
                })
                break
            case 'friend-online':
                // 好友在线状态
                store.dispatch('onlineStatus/updateFriendStatus', data.data)
                break
            case 'group-member-online':
                // 群成员在线状态
                store.dispatch('onlineStatus/updateGroupMemberStatus', data.data)
                break
            case 'conversation-update':
                // 会话更新（如新的会话、最后消息更新等）
                store.dispatch('conversations/updateConversation', data.data)
                break
            default:
                console.warn('未知的WebSocket消息类型:', data.type)
        }
    }

    close() {
        this.isManualClose = true
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer)
            this.reconnectTimer = null
        }
        if (this.socket) {
            this.socket.close()
            this.socket = null
        }
    }

    // 获取连接状态
    getReadyState() {
        return this.socket ? this.socket.readyState : WebSocket.CLOSED
    }

    // 检查是否连接
    isConnected() {
        return this.socket && this.socket.readyState === WebSocket.OPEN
    }
}

// 全局WebSocket实例
let wsInstance = null

export const initWebSocket = (token) => {
    if (wsInstance) {
        wsInstance.close()
    }
    wsInstance = new IMWebSocket(token)
    return wsInstance
}

export const getWebSocket = () => {
    return wsInstance
}

export const closeWebSocket = () => {
    if (wsInstance) {
        wsInstance.close()
        wsInstance = null
    }
}
