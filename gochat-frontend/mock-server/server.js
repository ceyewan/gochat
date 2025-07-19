const express = require('express')
const cors = require('cors')
const WebSocket = require('ws')
const http = require('http')

// 导入路由模块
const { router: authRoutes } = require('./routes/auth')
const userRoutes = require('./routes/users')
const conversationRoutes = require('./routes/conversations')
const messageRoutes = require('./routes/messages')
const friendRoutes = require('./routes/friends')
const groupRoutes = require('./routes/groups')

// 导入mock数据
const { users, onlineStatus } = require('./data/mockData')

const app = express()
const server = http.createServer(app)

// 中间件
app.use(cors())
app.use(express.json())

// 添加请求日志
app.use((req, res, next) => {
    console.log(`${new Date().toISOString()} ${req.method} ${req.path}`)
    next()
})

// 路由
app.use('/api/auth', authRoutes)
app.use('/api/users', userRoutes)
app.use('/api/conversations', conversationRoutes)
app.use('/api/messages', messageRoutes)
app.use('/api/friends', friendRoutes)
app.use('/api/groups', groupRoutes)

// WebSocket服务器
const wss = new WebSocket.Server({ server })

// 存储连接的客户端
const clients = new Map()

wss.on('connection', (ws, req) => {
    const url = new URL(req.url, `http://${req.headers.host}`)
    const token = url.searchParams.get('token')

    console.log('WebSocket连接建立，token:', token)

    if (token) {
        // 模拟token验证
        const userId = token.includes('user1') ? 'user1' :
            token.includes('user2') ? 'user2' : 'testuser'
        clients.set(ws, { userId, token })

        // 更新在线状态
        onlineStatus[userId] = true

        // 发送连接成功消息
        ws.send(JSON.stringify({
            type: 'connected',
            data: { message: 'WebSocket连接成功' }
        }))

        // 广播用户上线状态给其他用户
        setTimeout(() => {
            broadcastOnlineStatus(userId, true)
        }, 1000)
    }

    ws.on('message', (message) => {
        try {
            const data = JSON.parse(message)
            console.log('收到WebSocket消息:', data)

            handleWebSocketMessage(ws, data)
        } catch (error) {
            console.error('解析WebSocket消息失败:', error)
        }
    })

    ws.on('close', () => {
        const clientInfo = clients.get(ws)
        if (clientInfo) {
            console.log('WebSocket连接关闭，用户:', clientInfo.userId)

            // 更新离线状态
            onlineStatus[clientInfo.userId] = false

            // 广播用户离线状态
            broadcastOnlineStatus(clientInfo.userId, false)

            clients.delete(ws)
        }
    })

    ws.on('error', (error) => {
        console.error('WebSocket错误:', error)
    })
})

// 处理WebSocket消息
function handleWebSocketMessage(ws, data) {
    const clientInfo = clients.get(ws)
    if (!clientInfo) return

    switch (data.type) {
        case 'send-message':
            // 模拟消息发送
            const messageId = `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`

            // 根据用户ID查找真实用户名
            const sender = users.find(u => u.userId === clientInfo.userId)
            const senderName = sender ? sender.username : '未知用户'

            const message = {
                messageId,
                conversationId: data.data.conversationId,
                senderId: clientInfo.userId,
                senderName: senderName,
                content: data.data.content,
                type: data.data.messageType || 'text',
                sendTime: new Date().toISOString(),
                status: 'sent'
            }

            // 发送消息确认
            ws.send(JSON.stringify({
                type: 'message-ack',
                data: {
                    messageId: messageId,
                    tempMessageId: data.data.tempMessageId
                }
            }))

            // 模拟向其他用户发送消息
            setTimeout(() => {
                clients.forEach((client, clientWs) => {
                    if (clientWs !== ws) {
                        clientWs.send(JSON.stringify({
                            type: 'new-message',
                            data: {
                                conversationId: data.data.conversationId,
                                message: message
                            }
                        }))
                    }
                })
            }, 100)
            break

        case 'ping':
            ws.send(JSON.stringify({ type: 'pong' }))
            break
    }
}

// 广播在线状态
function broadcastOnlineStatus(userId, isOnline) {
    const statusMessage = {
        type: 'friend-online',
        data: {
            userId: userId,
            online: isOnline
        }
    }

    // 向所有连接的客户端广播状态更新
    clients.forEach((clientInfo, clientWs) => {
        if (clientInfo.userId !== userId) { // 不向自己发送
            try {
                clientWs.send(JSON.stringify(statusMessage))
            } catch (error) {
                console.error('发送在线状态失败:', error)
            }
        }
    })

    console.log(`广播用户 ${userId} ${isOnline ? '上线' : '离线'} 状态`)
}

// 启动服务器
const PORT = process.env.PORT || 8080

server.listen(PORT, () => {
    console.log(`Mock服务器启动成功！`)
    console.log(`HTTP API: http://localhost:${PORT}/api`)
    console.log(`WebSocket: ws://localhost:${PORT}/ws`)
})

// 优雅关闭
process.on('SIGINT', () => {
    console.log('正在关闭服务器...')
    server.close(() => {
        console.log('服务器已关闭')
        process.exit(0)
    })
})
