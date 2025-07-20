// 全局变量
let ws = null;
let authToken = null;
let userId = null;

// DOM元素
const loginForm = document.getElementById('login-form');
const registerForm = document.getElementById('register-form');
const showRegister = document.getElementById('show-register');
const showLogin = document.getElementById('show-login');
const chatSection = document.getElementById('chat');
const authSection = document.getElementById('auth');
const messagesContainer = document.getElementById('messages');
const messageInput = document.getElementById('message-input');
const sendButton = document.getElementById('send-button');
const userList = document.getElementById('user-list');

// 显示注册表单
showRegister.addEventListener('click', (e) => {
    e.preventDefault();
    loginForm.classList.add('hidden');
    registerForm.classList.remove('hidden');
});

// 显示登录表单
showLogin.addEventListener('click', (e) => {
    e.preventDefault();
    registerForm.classList.add('hidden');
    loginForm.classList.remove('hidden');
});

// 用户注册
document.getElementById('register-button').addEventListener('click', async (e) => {
    e.preventDefault();
    const username = document.getElementById('register-username').value;
    const password = document.getElementById('register-password').value;

    try {
        const response = await fetch('http://localhost:8080/user/register', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ username, password })
        });

        if (response.ok) { // 检查 HTTP 状态码是否为 200-299 范围
            alert('注册成功，请登录');
            registerForm.classList.add('hidden');
            loginForm.classList.remove('hidden');
        } else {
            const data = await response.json();
            console.log('注册失败:', data);
            console.log('HTTP状态码:', response.status);
            alert('注册失败：' + (data.error || '未知错误'));
        }
    } catch (error) {
        console.error('注册错误:', error);
        alert('注册过程中发生错误');
    }
});

// 用户登录
document.getElementById('login-button').addEventListener('click', async (e) => {
    e.preventDefault();
    const username = document.getElementById('login-username').value;
    const password = document.getElementById('login-password').value;

    try {
        const response = await fetch('http://localhost:8080/user/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ username, password })
        });

        if (response.ok) { // 检查 HTTP 状态码是否为 200-299 范围
            const data = await response.json();
            authToken = data.data.token;
            await checkAuth();
            connectWebSocket();
            authSection.classList.add('hidden');
            chatSection.classList.remove('hidden');
        } else {
            const data = await response.json();
            console.log('登录失败:', data);
            console.log('HTTP状态码:', response.status);
            alert('登录失败：' + (data.error || '未知错误'));
        }
    } catch (error) {
        console.error('登录错误:', error);
        alert('登录过程中发生错误');
    }
});

// 检查认证状态
async function checkAuth() {
    try {
        const response = await fetch('http://localhost:8080/user/checkAuth', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ authToken })
        });

        const data = await response.json();
        if (data.code === 200) {
            userId = data.data.userId;
        } else {
            throw new Error('认证失败');
        }
    } catch (error) {
        console.error('认证检查错误:', error);
        alert('认证检查失败');
    }
}

// 建立WebSocket连接
function connectWebSocket() {
    ws = new WebSocket('ws://connect:8081/ws');

    ws.onopen = () => {
        // 发送认证消息
        ws.send(JSON.stringify({
            user_id: userId,
            room_id: 0,
            token: authToken,
            message: 'connect'
        }));
    };

    ws.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        handleMessage(msg);
    };

    ws.onerror = (error) => {
        console.error('WebSocket错误:', error);
        alert('WebSocket连接错误');
    };

    ws.onclose = () => {
        console.log('WebSocket连接关闭');
        alert('连接已断开，请刷新页面重新连接');
    };
}

// 处理接收到的消息
function handleMessage(msg) {
    if (msg.count !== -1) {
        updateUserCount(msg.count);
    }
    if (msg.msg) {
        displayMessage(msg.msg);
    }
    if (msg.room_user_info) {
        updateUserList(msg.room_user_info);
    }
}

// 更新在线用户数量
function updateUserCount(count) {
    const header = document.querySelector('header h1');
    header.textContent = `GoChat (在线用户: ${count})`;
}

// 显示消息
function displayMessage(message) {
    const messageElement = document.createElement('div');
    messageElement.classList.add('message');
    messageElement.textContent = message;
    messagesContainer.appendChild(messageElement);
    messagesContainer.scrollTop = messagesContainer.scrollHeight;
}

// 更新在线用户列表
function updateUserList(users) {
    userList.innerHTML = '';
    for (const [userId, userInfo] of Object.entries(users)) {
        const li = document.createElement('li');
        li.textContent = userInfo;
        userList.appendChild(li);
    }
}

// 发送消息
sendButton.addEventListener('click', () => {
    const message = messageInput.value.trim();
    if (message) {
        ws.send(JSON.stringify({
            user_id: userId,
            room_id: 0,
            token: authToken,
            message: message
        }));
        messageInput.value = '';
    }
});

// 回车键发送消息
messageInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
        sendButton.click();
    }
});