import request from '@/utils/request'
import { initWebSocket, closeWebSocket } from '@/utils/websocket'
import router from '@/router'

const state = {
    token: localStorage.getItem('auth_token') || '',
    userInfo: null, // { userId, username, avatar }
}

const mutations = {
    setToken(state, token) {
        state.token = token
        if (token) {
            localStorage.setItem('auth_token', token)
        } else {
            localStorage.removeItem('auth_token')
        }
    },
    setUserInfo(state, userInfo) {
        state.userInfo = userInfo
    },
    clearUser(state) {
        state.token = ''
        state.userInfo = null
        localStorage.removeItem('auth_token')
    },
}

const actions = {
    // 登录
    async login({ commit, dispatch }, { username, password }) {
        try {
            const response = await request.post('/auth/login', {
                username,
                password
            })

            const { token, user } = response.data

            // 保存用户信息
            commit('setToken', token)
            commit('setUserInfo', user)

            // 初始化WebSocket连接
            initWebSocket(token)

            // 加载会话列表
            await dispatch('conversations/fetchConversations', null, { root: true })

            return response
        } catch (error) {
            console.error('登录失败:', error)
            throw error
        }
    },

    // 注册
    async register({ commit }, { username, password }) {
        try {
            const response = await request.post('/auth/register', {
                username,
                password
            })
            return response
        } catch (error) {
            console.error('注册失败:', error)
            throw error
        }
    },

    // 游客登录
    async guestLogin({ commit, dispatch }) {
        try {
            const response = await request.post('/auth/guest', {
                guestName: '' // 让后端自动生成游客昵称
            })

            const { token, user } = response.data

            // 保存用户信息
            commit('setToken', token)
            commit('setUserInfo', user)

            // 初始化WebSocket连接
            initWebSocket(token)

            // 加载会话列表（包含世界聊天室）
            await dispatch('conversations/fetchConversations', null, { root: true })

            return response
        } catch (error) {
            console.error('游客登录失败:', error)
            throw error
        }
    },

    // 登出
    async logout({ commit }) {
        try {
            // 调用后端登出接口
            await request.post('/auth/logout')
        } catch (error) {
            console.error('登出请求失败:', error)
        } finally {
            // 无论请求是否成功，都清除本地状态
            commit('clearUser')

            // 关闭WebSocket连接
            closeWebSocket()

            // 清除其他模块的状态
            commit('conversations/clearConversations', null, { root: true })
            commit('currentChat/clearCurrentChat', null, { root: true })
            commit('onlineStatus/clearOnlineStatus', null, { root: true })

            // 跳转到登录页
            router.push('/login')
        }
    },

    // 获取用户信息
    async fetchUserInfo({ commit, state }) {
        if (!state.token) {
            throw new Error('未登录')
        }

        try {
            const response = await request.get('/user/info')
            commit('setUserInfo', response.data)
            return response
        } catch (error) {
            console.error('获取用户信息失败:', error)
            throw error
        }
    },
}

const getters = {
    isAuthenticated: state => !!state.token,
    currentUser: state => state.userInfo,
    userId: state => state.userInfo?.userId,
    username: state => state.userInfo?.username,
    avatar: state => state.userInfo?.avatar,
    isGuest: state => state.userInfo?.isGuest || false,
}

export default {
    namespaced: true,
    state,
    mutations,
    actions,
    getters,
}
