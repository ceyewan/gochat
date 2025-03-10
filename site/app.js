new Vue({
    el: '#app',
    data: {
        isLoggedIn: false,
        isLoginMode: true,
        username: '',
        password: '',
        messages: [],
        newMessage: '',
        ws: null,
        token: '',
        userId: null,
        roomId: 0, // 默认房间
    },

    methods: {
        // 认证相关方法
        async login() {
            try {
                const response = await fetch('/api/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        username: this.username,
                        password: this.password
                    })
                });

                const data = await response.json();
                if (data.code === 200) {
                    this.token = data.data.token;
                    await this.checkAuth();
                    this.isLoggedIn = true;
                    this.initWebSocket();
                } else {
                    alert('登录失败：' + (data.error || '未知错误'));
                }
            } catch (error) {
                alert('登录失败：' + error.message);
            }
        },

        async register() {
            try {
                const response = await fetch('/api/register', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        username: this.username,
                        password: this.password
                    })
                });

                const data = await response.json();
                if (data.code === 200) {
                    alert('注册成功！请登录');
                    this.isLoginMode = true;
                } else {
                    alert('注册失败：' + (data.error || '未知错误'));
                }
            } catch (error) {
                alert('注册失败：' + error.message);
            }
        },

        async logout() {
            try {
                const response = await fetch('/api/logout', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        token: this.token
                    })
                });

                const data = await response.json();
                if (data.code === 200) {
                    this.isLoggedIn = false;
                    this.token = '';
                    this.userId = null;
                    if (this.ws) {
                        this.ws.close();
                        this.ws = null;
                    }
                    this.messages = [];
                } else {
                    alert('登出失败：' + (data.error || '未知错误'));
                }
            } catch (error) {
                alert('登出失败：' + error.message);
            }
        },

        async checkAuth() {
            try {
                const response = await fetch('/api/checkAuth', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        token: this.token
                    })
                });

                const data = await response.json();
                if (data.code === 200) {
                    this.userId = data.data.userid;
                } else {
                    throw new Error(data.error || '认证失败');
                }
            } catch (error) {
                alert('认证检查失败：' + error.message);
                this.isLoggedIn = false;
            }
        },

        // WebSocket相关方法
        initWebSocket() {
            this.ws = new WebSocket('ws://' + window.location.host + '/ws');

            this.ws.onopen = () => {
                // 发送认证消息
                this.ws.send(JSON.stringify({
                    user_id: this.userId,
                    room_id: this.roomId,
                    token: this.token,
                    message: 'connect'
                }));
            };

            this.ws.onmessage = (event) => {
                const data = JSON.parse(event.data);
                // 处理接收到的消息
                if (data.msg) {
                    this.messages.push({
                        content: data.msg,
                        isSent: false
                    });
                    this.scrollToBottom();
                }
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket错误:', error);
            };

            this.ws.onclose = () => {
                console.log('WebSocket连接已关闭');
            };
        },

        // 发送消息相关方法
        async sendMessage() {
            if (!this.newMessage.trim()) return;

            // 发送到服务器
            try {
                const response = await fetch('/api/pushRoom', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        msg: this.newMessage,
                        roomId: this.roomId,
                        authToken: this.token
                    })
                });

                const data = await response.json();
                if (data.code === 200) {
                    // 添加到本地消息列表
                    this.messages.push({
                        content: this.newMessage,
                        isSent: true
                    });
                    this.newMessage = '';
                    this.scrollToBottom();
                } else {
                    alert('发送失败：' + (data.error || '未知错误'));
                }
            } catch (error) {
                alert('发送失败：' + error.message);
            }
        },

        // UI相关方法
        toggleAuthMode() {
            this.isLoginMode = !this.isLoginMode;
            this.username = '';
            this.password = '';
        },

        scrollToBottom() {
            this.$nextTick(() => {
                const container = this.$refs.messageContainer;
                container.scrollTop = container.scrollHeight;
            });
        }
    }
});