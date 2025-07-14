const express = require('express')
const router = express.Router()
const { groups, conversations, messages, users } = require('../data/mockData')
const { authenticateToken } = require('./auth')

// 创建群聊
router.post('/', authenticateToken, (req, res) => {
    const { groupName, members } = req.body
    const currentUserId = req.user.userId

    console.log(`用户 ${currentUserId} 创建群聊: ${groupName}`)

    if (!groupName || !groupName.trim()) {
        return res.status(400).json({
            success: false,
            message: '群名称不能为空'
        })
    }

    if (!members || !Array.isArray(members) || members.length === 0) {
        return res.status(400).json({
            success: false,
            message: '至少需要选择一个成员'
        })
    }

    // 验证所有成员是否存在
    for (const memberId of members) {
        const user = users.find(u => u.userId === memberId)
        if (!user) {
            return res.status(400).json({
                success: false,
                message: `用户 ${memberId} 不存在`
            })
        }
    }

    // 创建群聊
    const groupId = `group_${Date.now()}`
    const newGroup = {
        groupId,
        groupName: groupName.trim(),
        avatar: '',
        description: '',
        members: [
            { userId: currentUserId, role: 'admin', joinTime: new Date().toISOString() },
            ...members.map(memberId => ({
                userId: memberId,
                role: 'member',
                joinTime: new Date().toISOString()
            }))
        ],
        createTime: new Date().toISOString()
    }

    groups.push(newGroup)

    // 创建群聊会话
    const newConversation = {
        conversationId: `conv_${Date.now()}`,
        type: 'group',
        target: {
            groupId: groupId,
            groupName: groupName.trim(),
            avatar: '',
            memberCount: newGroup.members.length
        },
        lastMessage: '群聊已创建',
        lastMessageTime: new Date().toISOString(),
        unreadCount: 0
    }

    conversations.push(newConversation)
    messages[newConversation.conversationId] = []

    res.status(201).json({
        success: true,
        message: '群聊创建成功',
        data: {
            group: newGroup,
            conversation: newConversation
        }
    })
})

// 获取群聊信息
router.get('/:groupId', authenticateToken, (req, res) => {
    const { groupId } = req.params

    const group = groups.find(g => g.groupId === groupId)

    if (!group) {
        return res.status(404).json({
            success: false,
            message: '群聊不存在'
        })
    }

    // 获取成员详细信息
    const membersWithInfo = group.members.map(member => {
        const user = users.find(u => u.userId === member.userId)
        return {
            ...member,
            username: user ? user.username : '未知用户',
            avatar: user ? user.avatar : ''
        }
    })

    res.json({
        success: true,
        data: {
            ...group,
            members: membersWithInfo
        }
    })
})

// 获取用户加入的群聊列表
router.get('/', authenticateToken, (req, res) => {
    const currentUserId = req.user.userId

    // 查找用户加入的群聊
    const userGroups = groups.filter(group =>
        group.members.some(member => member.userId === currentUserId)
    )

    const groupsWithInfo = userGroups.map(group => ({
        groupId: group.groupId,
        groupName: group.groupName,
        avatar: group.avatar,
        description: group.description,
        memberCount: group.members.length,
        createTime: group.createTime,
        role: group.members.find(m => m.userId === currentUserId)?.role
    }))

    res.json({
        success: true,
        data: groupsWithInfo
    })
})

// 添加群成员
router.post('/:groupId/members', authenticateToken, (req, res) => {
    const { groupId } = req.params
    const { memberIds } = req.body
    const currentUserId = req.user.userId

    const group = groups.find(g => g.groupId === groupId)

    if (!group) {
        return res.status(404).json({
            success: false,
            message: '群聊不存在'
        })
    }

    // 检查权限（只有管理员可以添加成员）
    const currentMember = group.members.find(m => m.userId === currentUserId)
    if (!currentMember || currentMember.role !== 'admin') {
        return res.status(403).json({
            success: false,
            message: '没有权限添加成员'
        })
    }

    if (!memberIds || !Array.isArray(memberIds)) {
        return res.status(400).json({
            success: false,
            message: '成员ID列表格式错误'
        })
    }

    // 添加新成员
    const newMembers = []
    for (const memberId of memberIds) {
        // 检查用户是否存在
        const user = users.find(u => u.userId === memberId)
        if (!user) {
            continue
        }

        // 检查是否已是群成员
        const existingMember = group.members.find(m => m.userId === memberId)
        if (existingMember) {
            continue
        }

        const newMember = {
            userId: memberId,
            role: 'member',
            joinTime: new Date().toISOString()
        }

        group.members.push(newMember)
        newMembers.push(newMember)
    }

    res.json({
        success: true,
        message: `成功添加 ${newMembers.length} 名成员`,
        data: newMembers
    })
})

// 退出群聊
router.delete('/:groupId/members/:userId', authenticateToken, (req, res) => {
    const { groupId, userId } = req.params
    const currentUserId = req.user.userId

    const group = groups.find(g => g.groupId === groupId)

    if (!group) {
        return res.status(404).json({
            success: false,
            message: '群聊不存在'
        })
    }

    // 只能退出自己或管理员可以踢出他人
    const currentMember = group.members.find(m => m.userId === currentUserId)
    if (userId !== currentUserId && (!currentMember || currentMember.role !== 'admin')) {
        return res.status(403).json({
            success: false,
            message: '没有权限执行此操作'
        })
    }

    const memberIndex = group.members.findIndex(m => m.userId === userId)
    if (memberIndex === -1) {
        return res.status(404).json({
            success: false,
            message: '用户不在群聊中'
        })
    }

    // 移除成员
    group.members.splice(memberIndex, 1)

    res.json({
        success: true,
        message: userId === currentUserId ? '退出群聊成功' : '移除成员成功'
    })
})

module.exports = router
