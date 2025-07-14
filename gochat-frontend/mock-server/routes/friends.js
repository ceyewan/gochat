const express = require('express')
const router = express.Router()
const { friendships, users, conversations, messages } = require('../data/mockData')
const { authenticateToken } = require('./auth')

// 添加好友
router.post('/', authenticateToken, (req, res) => {
    const { friendId } = req.body
    const currentUserId = req.user.userId

    console.log(`用户 ${currentUserId} 尝试添加好友 ${friendId}`)

    if (!friendId) {
        return res.status(400).json({
            success: false,
            message: '好友ID不能为空'
        })
    }

    if (friendId === currentUserId) {
        return res.status(400).json({
            success: false,
            message: '不能添加自己为好友'
        })
    }

    // 检查用户是否存在
    const friend = users.find(u => u.userId === friendId)
    if (!friend) {
        return res.status(404).json({
            success: false,
            message: '用户不存在'
        })
    }

    // 检查是否已经是好友
    const existingFriendship = friendships.find(f =>
        (f.userId === currentUserId && f.friendId === friendId) ||
        (f.userId === friendId && f.friendId === currentUserId)
    )

    if (existingFriendship) {
        return res.status(409).json({
            success: false,
            message: '已经是好友关系'
        })
    }

    // 添加好友关系
    friendships.push({
        userId: currentUserId,
        friendId: friendId,
        status: 'accepted' // 简化流程，直接接受
    })

    // 创建会话
    const newConversation = {
        conversationId: `conv_${Date.now()}`,
        type: 'single',
        target: {
            userId: friendId,
            username: friend.username,
            avatar: friend.avatar
        },
        lastMessage: '',
        lastMessageTime: new Date().toISOString(),
        unreadCount: 0
    }

    conversations.push(newConversation)
    messages[newConversation.conversationId] = []

    res.status(201).json({
        success: true,
        message: '好友添加成功',
        data: {
            friendship: {
                userId: currentUserId,
                friendId: friendId,
                status: 'accepted'
            },
            conversation: newConversation
        }
    })
})

// 获取好友列表
router.get('/', authenticateToken, (req, res) => {
    const currentUserId = req.user.userId

    // 获取用户的好友关系
    const userFriendships = friendships.filter(f =>
        (f.userId === currentUserId || f.friendId === currentUserId) &&
        f.status === 'accepted'
    )

    // 获取好友信息
    const friends = userFriendships.map(friendship => {
        const friendId = friendship.userId === currentUserId ?
            friendship.friendId : friendship.userId

        const friend = users.find(u => u.userId === friendId)

        return {
            userId: friend.userId,
            username: friend.username,
            avatar: friend.avatar,
            online: Math.random() > 0.5 // 随机在线状态
        }
    })

    res.json({
        success: true,
        data: friends
    })
})

// 删除好友
router.delete('/:friendId', authenticateToken, (req, res) => {
    const { friendId } = req.params
    const currentUserId = req.user.userId

    // 查找好友关系
    const friendshipIndex = friendships.findIndex(f =>
        (f.userId === currentUserId && f.friendId === friendId) ||
        (f.userId === friendId && f.friendId === currentUserId)
    )

    if (friendshipIndex === -1) {
        return res.status(404).json({
            success: false,
            message: '好友关系不存在'
        })
    }

    // 删除好友关系
    friendships.splice(friendshipIndex, 1)

    // 删除相关会话
    const conversationIndex = conversations.findIndex(c =>
        c.type === 'single' && c.target.userId === friendId
    )

    if (conversationIndex !== -1) {
        const conversationId = conversations[conversationIndex].conversationId
        conversations.splice(conversationIndex, 1)
        delete messages[conversationId]
    }

    res.json({
        success: true,
        message: '好友删除成功'
    })
})

module.exports = router
