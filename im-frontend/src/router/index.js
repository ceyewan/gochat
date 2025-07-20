import { createRouter, createWebHistory } from 'vue-router'
import store from '@/store'

// 路由组件
const Login = () => import('@/views/Login.vue')
const Register = () => import('@/views/Register.vue')
const ChatLayout = () => import('@/views/ChatLayout.vue')

const routes = [
    {
        path: '/',
        redirect: '/login'
    },
    {
        path: '/login',
        name: 'Login',
        component: Login,
        meta: { requiresGuest: true }
    },
    {
        path: '/register',
        name: 'Register',
        component: Register,
        meta: { requiresGuest: true }
    },
    {
        path: '/chat',
        name: 'Chat',
        component: ChatLayout,
        meta: { requiresAuth: true }
    }
]

const router = createRouter({
    history: createWebHistory(),
    routes
})

// 路由守卫
router.beforeEach((to, from, next) => {
    const token = store.state.user.token
    const isAuthenticated = !!token

    if (to.meta.requiresAuth && !isAuthenticated) {
        // 需要登录但未登录，跳转到登录页
        next('/login')
    } else if (to.meta.requiresGuest && isAuthenticated) {
        // 需要访客状态但已登录，跳转到聊天页
        next('/chat')
    } else {
        next()
    }
})

export default router
