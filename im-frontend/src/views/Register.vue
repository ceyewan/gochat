<template>
    <div class="register-container">
        <div class="register-card">
            <h2 class="register-title">注册</h2>
            <form @submit.prevent="handleRegister" class="register-form">
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
                <div class="form-group">
                    <label for="confirmPassword">确认密码</label>
                    <input
                        id="confirmPassword"
                        v-model="form.confirmPassword"
                        type="password"
                        placeholder="请确认密码"
                        required
                        :disabled="loading"
                    />
                </div>
                <button type="submit" class="register-btn" :disabled="loading">
                    {{ loading ? '注册中...' : '注册' }}
                </button>
                <p class="login-link">
                    已有账号？<router-link to="/login">登录</router-link>
                </p>
            </form>
            <div v-if="errorMessage" class="error-message">
                {{ errorMessage }}
            </div>
            <div v-if="successMessage" class="success-message">
                {{ successMessage }}
            </div>
        </div>
    </div>
</template>

<script>
import { mapActions } from 'vuex'

export default {
    name: 'Register',
    data() {
        return {
            form: {
                username: '',
                password: '',
                confirmPassword: ''
            },
            loading: false,
            errorMessage: '',
            successMessage: ''
        }
    },
    methods: {
        ...mapActions('user', ['register']),
        
        async handleRegister() {
            // 表单验证
            if (!this.form.username.trim()) {
                this.errorMessage = '请输入用户名'
                return
            }
            
            if (this.form.username.trim().length < 3) {
                this.errorMessage = '用户名至少3个字符'
                return
            }
            
            if (!this.form.password) {
                this.errorMessage = '请输入密码'
                return
            }
            
            if (this.form.password.length < 6) {
                this.errorMessage = '密码至少6个字符'
                return
            }
            
            if (this.form.password !== this.form.confirmPassword) {
                this.errorMessage = '两次输入的密码不一致'
                return
            }
            
            this.loading = true
            this.errorMessage = ''
            this.successMessage = ''
            
            try {
                await this.register({
                    username: this.form.username.trim(),
                    password: this.form.password
                })
                
                this.successMessage = '注册成功！正在跳转到登录页...'
                
                // 延迟跳转到登录页
                setTimeout(() => {
                    this.$router.push('/login')
                }, 2000)
                
            } catch (error) {
                console.error('注册失败:', error)
                this.errorMessage = error.response?.data?.message || '注册失败，请重试'
            } finally {
                this.loading = false
            }
        }
    }
}
</script>

<style scoped>
.register-container {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 100vh;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    padding: 20px;
}

.register-card {
    background: white;
    border-radius: 10px;
    padding: 40px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
    width: 100%;
    max-width: 400px;
}

.register-title {
    text-align: center;
    margin-bottom: 30px;
    color: #333;
    font-size: 28px;
    font-weight: 600;
}

.register-form {
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

.register-btn {
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

.register-btn:hover:not(:disabled) {
    background-color: #0056cc;
}

.register-btn:disabled {
    background-color: #ccc;
    cursor: not-allowed;
}

.login-link {
    text-align: center;
    color: #666;
}

.login-link a {
    color: #0078ff;
    text-decoration: none;
}

.login-link a:hover {
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

.success-message {
    margin-top: 15px;
    padding: 10px;
    background-color: #efe;
    border: 1px solid #cfc;
    border-radius: 5px;
    color: #383;
    text-align: center;
    font-size: 14px;
}
</style>
