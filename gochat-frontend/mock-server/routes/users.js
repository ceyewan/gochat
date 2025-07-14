const express = require('express')
const router = express.Router()
const { users } = require('../data/mockData')
const { authenticateToken } = require('./auth')

// 获取当前用户信息
router.get('/info', authenticateToken, (req, res) => {
    res.json({
        success: true,
        data: {
            userId: req.user.userId,
            username: req.user.username,
            avatar: req.user.avatar,
            email: req.user.email,
            createTime: req.user.createTime
        }
    })
})

// 根据用户名搜索用户
router.get('/:username', authenticateToken, (req, res) => {
    const { username } = req.params

    console.log('搜索用户:', username)

    const user = users.find(u => u.username === username)

    if (!user) {
        return res.status(404).json({
            success: false,
            message: '用户不存在'
        })
    }

    res.json({
        success: true,
        data: {
            userId: user.userId,
            username: user.username,
            avatar: user.avatar,
            email: user.email
        }
    })
})

// 获取用户列表（可选，用于开发调试）
router.get('/', authenticateToken, (req, res) => {
    const userList = users.map(user => ({
        userId: user.userId,
        username: user.username,
        avatar: user.avatar,
        email: user.email
    }))

    res.json({
        success: true,
        data: userList
    })
})

// 更新用户信息
router.put('/profile', authenticateToken, (req, res) => {
    const { username, avatar } = req.body
    const userId = req.user.userId

    // 查找用户
    const userIndex = users.findIndex(u => u.userId === userId)
    if (userIndex === -1) {
        return res.status(404).json({
            success: false,
            message: '用户不存在'
        })
    }

    // 如果要更新用户名，检查是否已存在
    if (username && username !== users[userIndex].username) {
        const existingUser = users.find(u => u.username === username && u.userId !== userId)
        if (existingUser) {
            return res.status(409).json({
                success: false,
                message: '用户名已存在'
            })
        }
        users[userIndex].username = username
    }

    // 更新头像
    if (avatar !== undefined) {
        users[userIndex].avatar = avatar
    }

    res.json({
        success: true,
        message: '更新成功',
        data: {
            userId: users[userIndex].userId,
            username: users[userIndex].username,
            avatar: users[userIndex].avatar,
            email: users[userIndex].email
        }
    })
})

module.exports = router
