import axios from 'axios'
import store from '@/store'
import router from '@/router'

const request = axios.create({
    baseURL: import.meta.env.VITE_API_BASE_URL,
    timeout: 5000,
})

// 请求拦截器
request.interceptors.request.use(
    config => {
        const token = store.state.user.token
        if (token) {
            config.headers.Authorization = `Bearer ${token}`
        }
        return config
    },
    error => {
        return Promise.reject(error)
    }
)

// 响应拦截器
request.interceptors.response.use(
    response => {
        return response.data
    },
    error => {
        if (error.response?.status === 401) {
            // Token过期或无效，清除用户信息并跳转到登录页
            store.dispatch('user/logout')
            router.push('/login')
        }
        return Promise.reject(error)
    }
)

export default request
