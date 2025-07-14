// 模拟用户数据
const users = [
    {
        userId: 'user1',
        username: '张三',
        avatar: '',
        email: 'zhangsan@example.com',
        createTime: '2024-01-01T00:00:00.000Z'
    },
    {
        userId: 'user2',
        username: '李四',
        avatar: '',
        email: 'lisi@example.com',
        createTime: '2024-01-02T00:00:00.000Z'
    },
    {
        userId: 'user3',
        username: '王五',
        avatar: '',
        email: 'wangwu@example.com',
        createTime: '2024-01-03T00:00:00.000Z'
    },
    {
        userId: 'user4',
        username: '赵六',
        avatar: '',
        email: 'zhaoliu@example.com',
        createTime: '2024-01-04T00:00:00.000Z'
    }
]

// 模拟会话数据
const conversations = [
    {
        conversationId: 'conv1',
        type: 'single',
        target: {
            userId: 'user2',
            username: '李四',
            avatar: ''
        },
        lastMessage: '你好，最近怎么样？',
        lastMessageTime: '2024-01-13T14:30:00.000Z',
        unreadCount: 2
    },
    {
        conversationId: 'conv2',
        type: 'group',
        target: {
            groupId: 'group1',
            groupName: '技术交流群',
            avatar: '',
            memberCount: 5
        },
        lastMessage: '大家好，有个问题想请教一下',
        lastMessageTime: '2024-01-13T13:15:00.000Z',
        unreadCount: 1
    },
    {
        conversationId: 'conv3',
        type: 'single',
        target: {
            userId: 'user3',
            username: '王五',
            avatar: ''
        },
        lastMessage: '明天的会议准备好了吗？',
        lastMessageTime: '2024-01-13T12:00:00.000Z',
        unreadCount: 0
    }
]

// 模拟消息数据
const messages = {
    conv1: [
        {
            messageId: 'msg1',
            conversationId: 'conv1',
            senderId: 'user2',
            senderName: '李四',
            content: '你好！',
            type: 'text',
            sendTime: '2024-01-13T14:25:00.000Z',
            status: 'sent'
        },
        {
            messageId: 'msg2',
            conversationId: 'conv1',
            senderId: 'user2',
            senderName: '李四',
            content: '最近在忙什么呢？',
            type: 'text',
            sendTime: '2024-01-13T14:26:00.000Z',
            status: 'sent'
        },
        {
            messageId: 'msg3',
            conversationId: 'conv1',
            senderId: 'testuser',
            senderName: '我',
            content: '在学习Vue 3，你呢？',
            type: 'text',
            sendTime: '2024-01-13T14:28:00.000Z',
            status: 'sent'
        },
        {
            messageId: 'msg4',
            conversationId: 'conv1',
            senderId: 'user2',
            senderName: '李四',
            content: '你好，最近怎么样？',
            type: 'text',
            sendTime: '2024-01-13T14:30:00.000Z',
            status: 'sent'
        }
    ],
    conv2: [
        {
            messageId: 'msg5',
            conversationId: 'conv2',
            senderId: 'user3',
            senderName: '王五',
            content: '大家好！',
            type: 'text',
            sendTime: '2024-01-13T13:10:00.000Z',
            status: 'sent'
        },
        {
            messageId: 'msg6',
            conversationId: 'conv2',
            senderId: 'user4',
            senderName: '赵六',
            content: '你好，王五！',
            type: 'text',
            sendTime: '2024-01-13T13:12:00.000Z',
            status: 'sent'
        },
        {
            messageId: 'msg7',
            conversationId: 'conv2',
            senderId: 'user3',
            senderName: '王五',
            content: '大家好，有个问题想请教一下',
            type: 'text',
            sendTime: '2024-01-13T13:15:00.000Z',
            status: 'sent'
        }
    ],
    conv3: [
        {
            messageId: 'msg8',
            conversationId: 'conv3',
            senderId: 'user3',
            senderName: '王五',
            content: '明天的会议准备好了吗？',
            type: 'text',
            sendTime: '2024-01-13T12:00:00.000Z',
            status: 'sent'
        }
    ]
}

// 模拟好友关系
const friendships = [
    { userId: 'testuser', friendId: 'user2', status: 'accepted' },
    { userId: 'testuser', friendId: 'user3', status: 'accepted' },
    { userId: 'testuser', friendId: 'user4', status: 'accepted' }
]

// 模拟群聊数据
const groups = [
    {
        groupId: 'group1',
        groupName: '技术交流群',
        avatar: '',
        description: '技术讨论和交流的地方',
        members: [
            { userId: 'testuser', role: 'admin', joinTime: '2024-01-01T00:00:00.000Z' },
            { userId: 'user2', role: 'member', joinTime: '2024-01-02T00:00:00.000Z' },
            { userId: 'user3', role: 'member', joinTime: '2024-01-03T00:00:00.000Z' },
            { userId: 'user4', role: 'member', joinTime: '2024-01-04T00:00:00.000Z' }
        ],
        createTime: '2024-01-01T00:00:00.000Z'
    }
]

// 在线状态
const onlineStatus = {
    user2: true,
    user3: false,
    user4: true
}

module.exports = {
    users,
    conversations,
    messages,
    friendships,
    groups,
    onlineStatus
}
