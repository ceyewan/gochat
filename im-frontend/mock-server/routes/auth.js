const express = require('express')
const router = express.Router()
const { users } = require('../data/mockData')

// 模拟token生成
function generateToken(userId) {
    return `mock_token_${userId}_${Date.now()}`
}

// 登录
router.post('/login', (req, res) => {
    const { username, password } = req.body

    console.log('登录请求:', { username, password })

    // 简单验证
    if (!username || !password) {
        return res.status(400).json({
            success: false,
            message: '用户名和密码不能为空'
        })
    }

    // 模拟用户验证（任何密码都可以登录）
    let user = users.find(u => u.username === username)

    if (!user) {
        return res.status(401).json({
            success: false,
            message: '用户名不存在'
        })
    }

    // 生成token
    const token = generateToken(user.userId)

    res.json({
        success: true,
        message: '登录成功',
        data: {
            token,
            user: {
                userId: user.userId,
                username: user.username,
                avatar: user.avatar,
                email: user.email
            }
        }
    })
})

// 注册
router.post('/register', (req, res) => {
    const { username, password } = req.body

    console.log('注册请求:', { username, password })

    // 验证必填字段
    if (!username || !password) {
        return res.status(400).json({
            success: false,
            message: '用户名和密码不能为空'
        })
    }

    // 验证用户名长度
    if (username.length < 3) {
        return res.status(400).json({
            success: false,
            message: '用户名至少3个字符'
        })
    }

    // 验证密码长度
    if (password.length < 6) {
        return res.status(400).json({
            success: false,
            message: '密码至少6个字符'
        })
    }

    // 检查用户名是否已存在
    const existingUser = users.find(u => u.username === username)
    if (existingUser) {
        return res.status(409).json({
            success: false,
            message: '用户名已存在'
        })
    }

    // 创建新用户
    const newUser = {
        userId: `user_${Date.now()}`,
        username,
        avatar: '',
        email: `${username}@example.com`,
        createTime: new Date().toISOString(),
        isGuest: false
    }

    // 添加到用户列表（实际项目中应该保存到数据库）
    users.push(newUser)

    res.status(201).json({
        success: true,
        message: '注册成功',
        data: {
            user: {
                userId: newUser.userId,
                username: newUser.username,
                avatar: newUser.avatar,
                email: newUser.email,
                isGuest: false
            }
        }
    })
})

// 游客登录
router.post('/guest', (req, res) => {
    const { guestName } = req.body

    console.log('游客登录请求:', { guestName })

    // 生成游客用户名
    const timestamp = Date.now()
    const randomNum = Math.floor(Math.random() * 1000)
    const username = guestName || `游客${randomNum}`

    // 创建游客用户
    const guestUser = {
        userId: `guest_${timestamp}`,
        username,
        avatar: '',
        email: '',
        createTime: new Date().toISOString(),
        isGuest: true
    }

    // 添加到用户列表（游客用户也需要在内存中保存，用于WebSocket连接）
    users.push(guestUser)

    // 生成token
    const token = generateToken(guestUser.userId)

    res.json({
        success: true,
        message: '游客登录成功',
        data: {
            token,
            user: {
                userId: guestUser.userId,
                username: guestUser.username,
                avatar: guestUser.avatar,
                isGuest: true
            }
        }
    })
})

// 登出
router.post('/logout', (req, res) => {
    // 模拟登出逻辑（实际项目中可能需要使token失效）
    res.json({
        success: true,
        message: '登出成功'
    })
})

// 验证token（中间件）
function authenticateToken(req, res, next) {
    const authHeader = req.headers['authorization']
    const token = authHeader && authHeader.split(' ')[1]

    if (!token) {
        return res.status(401).json({
            success: false,
            message: '缺少访问令牌'
        })
    }

    // 模拟token验证
    if (!token.startsWith('mock_token_')) {
        return res.status(403).json({
            success: false,
            message: '无效的访问令牌'
        })
    }

    // 从token中提取用户ID
    // token格式: mock_token_{userId}_{timestamp}
    const parts = token.split('_')
    // 对于游客用户，userId是guest_{timestamp}，所以需要重新组合
    let userId
    if (parts[2] === 'guest') {
        userId = `guest_${parts[3]}`
    } else {
        userId = parts[2]
    }

    const user = users.find(u => u.userId === userId)
    if (!user) {
        return res.status(403).json({
            success: false,
            message: '用户不存在'
        })
    }

    req.user = user
    next()
}

module.exports = {
    router,
    authenticateToken
}
