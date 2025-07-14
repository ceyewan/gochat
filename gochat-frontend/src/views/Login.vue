<template>
    <div class="login-container">
        <div class="login-card">
            <h2 class="login-title">登录</h2>
            <form @submit.prevent="handleLogin" class="login-form">
                <div class="form-group">
                    <label for="username">用户名</label>
                    <input
                        id="username"
                        v-model="form.username"
                        type="text"
                        placeholder="请输入用户名"
                        required
                        :disabled="loading"
                    />
                </div>
                <div class="form-group">
                    <label for="password">密码</label>
                    <input
                        id="password"
                        v-model="form.password"
                        type="password"
                        placeholder="请输入密码"
                        required
                        :disabled="loading"
                    />
                </div>
                <button type="submit" class="login-btn" :disabled="loading">
                    {{ loading ? '登录中...' : '登录' }}
                </button>
                <p class="register-link">
                    还没有账号？<router-link to="/register">注册</router-link>
                </p>
            </form>
            <div v-if="errorMessage" class="error-message">
                {{ errorMessage }}
            </div>
        </div>
    </div>
</template>

<script>
import { mapActions } from 'vuex'

export default {
    name: 'Login',
    data() {
        return {
            form: {
                username: '',
                password: ''
            },
            loading: false,
            errorMessage: ''
        }
    },
    methods: {
        ...mapActions('user', ['login']),
        
        async handleLogin() {
            if (!this.form.username.trim() || !this.form.password.trim()) {
                this.errorMessage = '请填写用户名和密码'
                return
            }
            
            this.loading = true
            this.errorMessage = ''
            
            try {
                await this.login({
                    username: this.form.username.trim(),
                    password: this.form.password
                })
                
                // 登录成功，路由守卫会自动跳转到聊天页面
                this.$router.push('/chat')
            } catch (error) {
                console.error('登录失败:', error)
                this.errorMessage = error.response?.data?.message || '登录失败，请检查用户名和密码'
            } finally {
                this.loading = false
            }
        }
    }
}
</script>

<style scoped>
.login-container {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 100vh;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    padding: 20px;
}

.login-card {
    background: white;
    border-radius: 10px;
    padding: 40px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
    width: 100%;
    max-width: 400px;
}

.login-title {
    text-align: center;
    margin-bottom: 30px;
    color: #333;
    font-size: 28px;
    font-weight: 600;
}

.login-form {
    display: flex;
    flex-direction: column;
}

.form-group {
    margin-bottom: 20px;
}

.form-group label {
    display: block;
    margin-bottom: 5px;
    color: #555;
    font-weight: 500;
}

.form-group input {
    width: 100%;
    padding: 12px;
    border: 1px solid #ddd;
    border-radius: 5px;
    font-size: 16px;
    transition: border-color 0.3s;
}

.form-group input:focus {
    outline: none;
    border-color: #0078ff;
    box-shadow: 0 0 0 2px rgba(0, 120, 255, 0.2);
}

.form-group input:disabled {
    background-color: #f5f5f5;
    cursor: not-allowed;
}

.login-btn {
    width: 100%;
    padding: 14px;
    background-color: #0078ff;
    color: white;
    border: none;
    border-radius: 5px;
    font-size: 16px;
    font-weight: 500;
    cursor: pointer;
    transition: background-color 0.3s;
    margin-bottom: 20px;
}

.login-btn:hover:not(:disabled) {
    background-color: #0056cc;
}

.login-btn:disabled {
    background-color: #ccc;
    cursor: not-allowed;
}

.register-link {
    text-align: center;
    color: #666;
}

.register-link a {
    color: #0078ff;
    text-decoration: none;
}

.register-link a:hover {
    text-decoration: underline;
}

.error-message {
    margin-top: 15px;
    padding: 10px;
    background-color: #fee;
    border: 1px solid #fcc;
    border-radius: 5px;
    color: #c33;
    text-align: center;
    font-size: 14px;
}
</style>
