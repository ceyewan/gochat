const state = {
    friendStatus: {}, // { userId: boolean }
    groupMemberStatus: {}, // { groupId: { userId: boolean } }
}

const mutations = {
    setFriendStatus(state, { userId, online }) {
        state.friendStatus = {
            ...state.friendStatus,
            [userId]: online
        }
    },
    updateFriendStatusBatch(state, statusList) {
        const newStatus = {}
        statusList.forEach(({ userId, online }) => {
            newStatus[userId] = online
        })
        state.friendStatus = {
            ...state.friendStatus,
            ...newStatus
        }
    },
    setGroupMemberStatus(state, { groupId, members }) {
        state.groupMemberStatus = {
            ...state.groupMemberStatus,
            [groupId]: members.reduce((acc, { userId, online }) => {
                acc[userId] = online
                return acc
            }, {})
        }
    },
    updateGroupMemberStatus(state, { groupId, userId, online }) {
        if (!state.groupMemberStatus[groupId]) {
            state.groupMemberStatus[groupId] = {}
        }
        state.groupMemberStatus[groupId] = {
            ...state.groupMemberStatus[groupId],
            [userId]: online
        }
    },
    clearOnlineStatus(state) {
        state.friendStatus = {}
        state.groupMemberStatus = {}
    },
}

const actions = {
    // 更新好友在线状态（WebSocket接收）
    updateFriendStatus({ commit }, statusData) {
        if (Array.isArray(statusData)) {
            // 批量更新
            commit('updateFriendStatusBatch', statusData)
        } else {
            // 单个更新
            commit('setFriendStatus', statusData)
        }
    },

    // 更新群成员在线状态（WebSocket接收）
    updateGroupMemberStatus({ commit }, statusData) {
        const { groupId, members, userId, online } = statusData

        if (members) {
            // 批量更新群成员状态
            commit('setGroupMemberStatus', { groupId, members })
        } else if (userId !== undefined) {
            // 单个成员状态更新
            commit('updateGroupMemberStatus', { groupId, userId, online })
        }
    },

    // 初始化在线状态
    initializeOnlineStatus({ commit }, { friends = [], groups = [] }) {
        // 初始化好友在线状态
        if (friends.length > 0) {
            commit('updateFriendStatusBatch', friends)
        }

        // 初始化群成员在线状态
        groups.forEach(group => {
            if (group.members && group.members.length > 0) {
                commit('setGroupMemberStatus', {
                    groupId: group.groupId,
                    members: group.members
                })
            }
        })
    },
}

const getters = {
    // 获取好友在线状态
    getFriendStatus: state => userId => {
        return state.friendStatus[userId] || false
    },

    // 获取群成员在线状态
    getGroupMemberStatus: state => (groupId, userId) => {
        return state.groupMemberStatus[groupId]?.[userId] || false
    },

    // 获取群的所有在线成员
    getGroupOnlineMembers: state => groupId => {
        const groupStatus = state.groupMemberStatus[groupId] || {}
        return Object.entries(groupStatus)
            .filter(([_, online]) => online)
            .map(([userId]) => userId)
    },

    // 获取在线好友数量
    onlineFriendsCount: state => {
        return Object.values(state.friendStatus).filter(status => status).length
    },

    // 获取群在线成员数量
    getGroupOnlineCount: state => groupId => {
        const groupStatus = state.groupMemberStatus[groupId] || {}
        return Object.values(groupStatus).filter(status => status).length
    },

    // 检查用户是否在线（先检查好友状态，再检查群成员状态）
    isUserOnline: state => userId => {
        // 首先检查好友状态
        if (state.friendStatus[userId] !== undefined) {
            return state.friendStatus[userId]
        }

        // 然后检查是否在任何群中在线
        for (const groupId in state.groupMemberStatus) {
            if (state.groupMemberStatus[groupId][userId] !== undefined) {
                return state.groupMemberStatus[groupId][userId]
            }
        }

        return false
    },
}

export default {
    namespaced: true,
    state,
    mutations,
    actions,
    getters,
}
