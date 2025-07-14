const express = require('express')
const router = express.Router()
const { conversations, messages } = require('../data/mockData')
const { authenticateToken } = require('./auth')

// 获取会话列表
router.get('/', authenticateToken, (req, res) => {
    console.log('获取会话列表，用户:', req.user.username)

    // 返回用户的会话列表
    res.json({
        success: true,
        data: conversations
    })
})

// 获取特定会话信息
router.get('/:conversationId', authenticateToken, (req, res) => {
    const { conversationId } = req.params

    const conversation = conversations.find(c => c.conversationId === conversationId)

    if (!conversation) {
        return res.status(404).json({
            success: false,
            message: '会话不存在'
        })
    }

    res.json({
        success: true,
        data: conversation
    })
})

// 获取会话消息
router.get('/:conversationId/messages', authenticateToken, (req, res) => {
    const { conversationId } = req.params
    const { page = 1, size = 50 } = req.query

    console.log(`获取会话 ${conversationId} 的消息，页码: ${page}, 大小: ${size}`)

    const conversationMessages = messages[conversationId] || []

    // 模拟分页
    const pageNum = parseInt(page)
    const pageSize = parseInt(size)
    const startIndex = (pageNum - 1) * pageSize
    const endIndex = startIndex + pageSize

    const paginatedMessages = conversationMessages.slice(startIndex, endIndex)

    res.json({
        success: true,
        data: paginatedMessages,
        pagination: {
            page: pageNum,
            size: pageSize,
            total: conversationMessages.length,
            hasMore: endIndex < conversationMessages.length
        }
    })
})

// 标记会话为已读
router.put('/:conversationId/read', authenticateToken, (req, res) => {
    const { conversationId } = req.params

    console.log(`标记会话 ${conversationId} 为已读`)

    // 查找会话
    const conversation = conversations.find(c => c.conversationId === conversationId)

    if (!conversation) {
        return res.status(404).json({
            success: false,
            message: '会话不存在'
        })
    }

    // 清除未读计数
    conversation.unreadCount = 0

    res.json({
        success: true,
        message: '标记已读成功'
    })
})

// 创建新会话（私聊）
router.post('/', authenticateToken, (req, res) => {
    const { targetUserId } = req.body
    const currentUserId = req.user.userId

    if (!targetUserId) {
        return res.status(400).json({
            success: false,
            message: '目标用户ID不能为空'
        })
    }

    if (targetUserId === currentUserId) {
        return res.status(400).json({
            success: false,
            message: '不能与自己创建会话'
        })
    }

    // 检查会话是否已存在
    const existingConversation = conversations.find(c =>
        c.type === 'single' &&
        c.target.userId === targetUserId
    )

    if (existingConversation) {
        return res.json({
            success: true,
            message: '会话已存在',
            data: existingConversation
        })
    }

    // 创建新会话
    const newConversation = {
        conversationId: `conv_${Date.now()}`,
        type: 'single',
        target: {
            userId: targetUserId,
            username: '新朋友', // 实际应该从用户表获取
            avatar: ''
        },
        lastMessage: '',
        lastMessageTime: new Date().toISOString(),
        unreadCount: 0
    }

    conversations.push(newConversation)
    messages[newConversation.conversationId] = []

    res.status(201).json({
        success: true,
        message: '会话创建成功',
        data: newConversation
    })
})

// 删除会话
router.delete('/:conversationId', authenticateToken, (req, res) => {
    const { conversationId } = req.params

    const conversationIndex = conversations.findIndex(c => c.conversationId === conversationId)

    if (conversationIndex === -1) {
        return res.status(404).json({
            success: false,
            message: '会话不存在'
        })
    }

    // 删除会话和消息
    conversations.splice(conversationIndex, 1)
    delete messages[conversationId]

    res.json({
        success: true,
        message: '会话删除成功'
    })
})

module.exports = router
