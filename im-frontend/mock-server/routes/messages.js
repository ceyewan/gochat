const express = require('express')
const router = express.Router()
const { messages, conversations } = require('../data/mockData')
const { authenticateToken } = require('./auth')

// 发送消息（HTTP接口，作为WebSocket的备用）
router.post('/', authenticateToken, (req, res) => {
    const { conversationId, content, type = 'text' } = req.body
    const senderId = req.user.userId

    if (!conversationId || !content) {
        return res.status(400).json({
            success: false,
            message: '会话ID和消息内容不能为空'
        })
    }

    // 检查会话是否存在
    const conversation = conversations.find(c => c.conversationId === conversationId)
    if (!conversation) {
        return res.status(404).json({
            success: false,
            message: '会话不存在'
        })
    }

    // 创建消息
    const messageId = `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
    const message = {
        messageId,
        conversationId,
        senderId,
        senderName: req.user.username,
        content,
        type,
        sendTime: new Date().toISOString(),
        status: 'sent'
    }

    // 添加到消息列表
    if (!messages[conversationId]) {
        messages[conversationId] = []
    }
    messages[conversationId].push(message)

    // 更新会话的最后消息
    conversation.lastMessage = content
    conversation.lastMessageTime = message.sendTime

    res.status(201).json({
        success: true,
        message: '消息发送成功',
        data: message
    })
})

// 获取消息详情
router.get('/:messageId', authenticateToken, (req, res) => {
    const { messageId } = req.params

    // 在所有会话中查找消息
    let foundMessage = null
    for (const conversationId in messages) {
        const message = messages[conversationId].find(m => m.messageId === messageId)
        if (message) {
            foundMessage = message
            break
        }
    }

    if (!foundMessage) {
        return res.status(404).json({
            success: false,
            message: '消息不存在'
        })
    }

    res.json({
        success: true,
        data: foundMessage
    })
})

// 撤回消息
router.delete('/:messageId', authenticateToken, (req, res) => {
    const { messageId } = req.params
    const currentUserId = req.user.userId

    // 查找消息
    let foundMessage = null
    let conversationId = null
    for (const cId in messages) {
        const messageIndex = messages[cId].findIndex(m => m.messageId === messageId)
        if (messageIndex !== -1) {
            foundMessage = messages[cId][messageIndex]
            conversationId = cId
            break
        }
    }

    if (!foundMessage) {
        return res.status(404).json({
            success: false,
            message: '消息不存在'
        })
    }

    // 只能撤回自己的消息
    if (foundMessage.senderId !== currentUserId) {
        return res.status(403).json({
            success: false,
            message: '只能撤回自己的消息'
        })
    }

    // 检查撤回时间限制（2分钟内）
    const messageTime = new Date(foundMessage.sendTime)
    const now = new Date()
    const diffMinutes = (now - messageTime) / (1000 * 60)

    if (diffMinutes > 2) {
        return res.status(400).json({
            success: false,
            message: '超过撤回时间限制（2分钟）'
        })
    }

    // 更新消息为撤回状态
    foundMessage.content = '消息已撤回'
    foundMessage.type = 'recalled'
    foundMessage.recallTime = new Date().toISOString()

    res.json({
        success: true,
        message: '消息撤回成功',
        data: foundMessage
    })
})

// 标记消息为已读
router.put('/:messageId/read', authenticateToken, (req, res) => {
    const { messageId } = req.params

    // 查找消息
    let foundMessage = null
    for (const conversationId in messages) {
        const message = messages[conversationId].find(m => m.messageId === messageId)
        if (message) {
            foundMessage = message
            break
        }
    }

    if (!foundMessage) {
        return res.status(404).json({
            success: false,
            message: '消息不存在'
        })
    }

    // 标记为已读
    foundMessage.readTime = new Date().toISOString()

    res.json({
        success: true,
        message: '标记已读成功'
    })
})

// 搜索消息
router.get('/search/:keyword', authenticateToken, (req, res) => {
    const { keyword } = req.params
    const { conversationId } = req.query

    if (!keyword || keyword.trim().length === 0) {
        return res.status(400).json({
            success: false,
            message: '搜索关键词不能为空'
        })
    }

    let searchResults = []

    if (conversationId) {
        // 在特定会话中搜索
        const conversationMessages = messages[conversationId] || []
        searchResults = conversationMessages.filter(message =>
            message.content.toLowerCase().includes(keyword.toLowerCase()) &&
            message.type !== 'recalled'
        )
    } else {
        // 在所有会话中搜索
        for (const cId in messages) {
            const conversationMessages = messages[cId]
            const matchedMessages = conversationMessages.filter(message =>
                message.content.toLowerCase().includes(keyword.toLowerCase()) &&
                message.type !== 'recalled'
            )
            searchResults.push(...matchedMessages)
        }
    }

    // 按时间倒序排列
    searchResults.sort((a, b) => new Date(b.sendTime) - new Date(a.sendTime))

    res.json({
        success: true,
        data: searchResults,
        total: searchResults.length
    })
})

module.exports = router
