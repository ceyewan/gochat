# 分布式微服务即时通讯系统前端开发计划


## **一、计划概述**
本开发计划基于之前的**前端设计方案**，针对后端开发人员的视角，聚焦**功能实现、接口对接、状态管理**三大核心，按**准备→架构→组件→功能→联调→上线**的流程推进，确保前端与后端微服务的高效协同。

**总周期**：约15个工作日（可根据团队 velocity 调整）  
**核心目标**：实现设计方案中的所有功能，确保界面简洁、交互流畅、与后端接口正确对接。


## **二、阶段划分与详细任务**


### **阶段1：准备阶段（Day 1）**
**目标**：确认技术选型，搭建项目基础环境，规范开发流程。

#### **1.1 技术选型确认**
- 与后端团队对齐：确认**接口文档**（HTTP/WS）、**数据格式**（JSON）、**认证方式**（JWT Token）。
- 最终确认前端技术栈（适配后端开发人员的学习成本）：
  - 框架：Vue 3（Options API，降低学习成本）；
  - 状态管理：Vuex 4（集中管理用户、会话、消息状态）；
  - HTTP：Axios（拦截器处理Token）；
  - WebSocket：原生API（封装工具类，逻辑清晰）；
  - 构建：Vite（快速开发调试）；
  - 样式：Tailwind CSS（可选，快速实现响应式布局，减少CSS编写量）。

#### **1.2 环境搭建**
- 用Vite创建项目：`npm create vite@latest im-frontend --template vue`；
- 安装依赖：`npm install vuex@next axios`；
- 配置开发工具：
  - ESLint + Prettier（代码规范，避免格式问题）；
  - Vite配置（如`proxy`代理，解决开发环境跨域：`/api`→后端HTTP接口，`/ws`→后端WS接口）。

#### **1.3 输出**
- 项目脚手架（包含Vite配置、依赖、代码规范）；
- 技术选型文档（与后端对齐的接口、数据格式）。


### **阶段2：基础架构搭建（Day 2-3）**
**目标**：搭建前端核心架构，包括路由、状态管理、HTTP/WS工具类，为后续功能开发铺路。

#### **2.1 路由配置**
- 定义路由表（使用Vue Router 4）：
  - 公开路由：`/login`（登录）、`/register`（注册）；
  - 私有路由：`/chat`（主界面，需登录验证）；
- 路由守卫：实现`beforeEach`拦截，未登录用户强制跳转到`/login`（通过`localStorage`中的`auth_token`判断）。

#### **2.2 状态管理（Vuex）**
- 设计Store结构（模块化）：
  ```javascript
  // store/index.js
  import { createStore } from 'vuex';
  import user from './modules/user'; // 用户信息（token、username、avatar）
  import conversations from './modules/conversations'; // 会话列表（单聊+群聊）
  import currentChat from './modules/currentChat'; // 当前聊天（选中的会话、历史消息）
  import onlineStatus from './modules/onlineStatus'; // 在线状态（好友/群成员）

  export default createStore({
    modules: {
      user,
      conversations,
      currentChat,
      onlineStatus,
    },
  });
  ```
- 每个模块定义**state**（初始状态）、**mutations**（同步修改状态）、**actions**（异步操作，如调用接口）：
  - 示例（`user`模块）：
    ```javascript
    // store/modules/user.js
    const state = {
      token: localStorage.getItem('auth_token') || '',
      userInfo: null, // { userId, username, avatar }
    };

    const mutations = {
      setToken(state, token) {
        state.token = token;
        localStorage.setItem('auth_token', token);
      },
      setUserInfo(state, userInfo) {
        state.userInfo = userInfo;
      },
      clearUser(state) {
        state.token = '';
        state.userInfo = null;
        localStorage.removeItem('auth_token');
      },
    };

    const actions = {
      // 登录动作（调用后端接口）
      login({ commit }, { username, password }) {
        return axios.post('/api/auth/login', { username, password })
          .then(res => {
            commit('setToken', res.data.token);
            commit('setUserInfo', res.data.user);
            return res;
          });
      },
    };
    ```

#### **2.3 HTTP工具类（Axios封装）**
- 创建`src/utils/request.js`，封装Axios实例：
  - 请求拦截器：添加`Authorization` header（`Bearer ${token}`）；
  - 响应拦截器：处理错误（如`401` Token过期，触发登出）；
  - 示例：
    ```javascript
    import axios from 'axios';
    import store from '@/store';

    const request = axios.create({
      baseURL: import.meta.env.VITE_API_BASE_URL, // 从.env文件读取后端接口地址
      timeout: 5000,
    });

    // 请求拦截器
    request.interceptors.request.use(config => {
      const token = store.state.user.token;
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
      return config;
    }, error => Promise.reject(error));

    // 响应拦截器
    request.interceptors.response.use(response => response.data, error => {
      if (error.response?.status === 401) {
        store.dispatch('user/logout'); // 触发登出动作
      }
      return Promise.reject(error);
    });

    export default request;
    ```

#### **2.4 WebSocket工具类封装**
- 创建`src/utils/websocket.js`，封装WS连接、重连、消息发送/接收逻辑：
  - 功能：
    1. 初始化连接（携带Token）；
    2. 自动重连（间隔1秒，最多3次）；
    3. 发送消息（JSON格式）；
    4. 监听消息（分发到对应的处理函数）；
  - 示例：
    ```javascript
    export class IMWebSocket {
      constructor(token) {
        this.token = token;
        this.url = `${import.meta.env.VITE_WS_BASE_URL}?token=${token}`;
        this.socket = null;
        this.reconnectCount = 0;
        this.maxReconnect = 3;
        this.init();
      }

      init() {
        this.socket = new WebSocket(this.url);
        this.socket.onopen = () => {
          console.log('WS连接成功');
          this.reconnectCount = 0; // 重置重连次数
        };
        this.socket.onmessage = (event) => {
          const data = JSON.parse(event.data);
          this.handleMessage(data); // 处理后端推送的消息
        };
        this.socket.onclose = () => {
          console.log('WS连接关闭');
          this.reconnect(); // 重连
        };
        this.socket.onerror = (error) => {
          console.error('WS错误:', error);
          this.socket.close();
        };
      }

      reconnect() {
        if (this.reconnectCount < this.maxReconnect) {
          this.reconnectCount++;
          setTimeout(() => {
            console.log(`正在重连（第${this.reconnectCount}次）`);
            this.init();
          }, 1000);
        } else {
          console.error('重连失败，请刷新页面');
        }
      }

      send(message) {
        if (this.socket.readyState === WebSocket.OPEN) {
          this.socket.send(JSON.stringify(message));
        } else {
          console.error('WS未连接，无法发送消息');
        }
      }

      handleMessage(data) {
        // 根据消息类型分发（如new-message、friend-online）
        switch (data.type) {
          case 'new-message':
            store.dispatch('currentChat/addMessage', data.data.message);
            break;
          case 'friend-online':
            store.dispatch('onlineStatus/updateFriendStatus', data.data);
            break;
          // 其他类型...
        }
      }
    }
    ```

#### **2.5 输出**
- 路由配置（包含守卫）；
- Vuex Store（模块化，包含用户、会话、当前聊天、在线状态）；
- Axios封装（请求/响应拦截器）；
- WebSocket工具类（连接、重连、消息处理）。


### **阶段3：界面组件开发（Day 4-6）**
**目标**：实现所有界面组件的UI结构，不包含交互逻辑（后续阶段添加）。

#### **3.1 组件划分**
根据设计方案的布局，拆分以下组件：
- **基础组件**：`Header`（顶部导航栏）、`ConversationItem`（会话列表项）、`MessageBubble`（消息气泡）；
- **页面组件**：`Login`（登录页）、`Register`（注册页）、`ChatLayout`（主界面布局，包含左侧会话列表、右侧聊天区域）；
- **弹窗组件**：`AddFriendModal`（添加好友弹窗）、`CreateGroupModal`（创建群聊弹窗）。

#### **3.2 组件实现（以Vue SFC为例）**
- **Header组件**（`src/components/Header.vue`）：
  ```vue
  <template>
    <div class="header flex justify-between items-center h-12 px-4 bg-white shadow">
      <div class="user-info flex items-center">
        <img :src="userInfo.avatar" class="w-8 h-8 rounded-full mr-2" alt="头像">
        <span class="font-medium">{{ userInfo.username }}</span>
      </div>
      <button class="text-blue-500" @click="logout">登出</button>
    </div>
  </template>

  <script>
  import { mapState } from 'vuex';
  export default {
    computed: {
      ...mapState('user', ['userInfo']),
    },
    methods: {
      logout() {
        this.$store.dispatch('user/logout');
      },
    },
  };
  </script>
  ```
- **ConversationItem组件**（`src/components/ConversationItem.vue`）：
  ```vue
  <template>
    <div class="conversation-item flex items-center px-3 py-2 cursor-pointer hover:bg-gray-100" @click="selectConversation">
      <img :src="conversation.target.avatar" class="w-10 h-10 rounded-full mr-3" alt="头像">
      <div class="flex-1 overflow-hidden">
        <div class="flex justify-between items-center mb-1">
          <span class="font-medium">{{ conversation.target.username || conversation.target.groupName }}</span>
          <span class="text-xs text-gray-500">{{ conversation.lastMessageTime }}</span>
        </div>
        <p class="text-xs text-gray-500 truncate">{{ conversation.lastMessage }}</p>
      </div>
      <span v-if="conversation.unreadCount > 0" class="w-5 h-5 flex items-center justify-center text-xs text-white bg-red-500 rounded-full">{{ conversation.unreadCount }}</span>
    </div>
  </template>

  <script>
  export default {
    props: {
      conversation: {
        type: Object,
        required: true,
      },
    },
    methods: {
      selectConversation() {
        this.$emit('select', this.conversation);
      },
    },
  };
  </script>
  ```
- **MessageBubble组件**（`src/components/MessageBubble.vue`）：
  ```vue
  <template>
    <div class="message-item mb-4" :class="{ 'self': isSelf }">
      <div class="bubble px-3 py-2 rounded-lg max-w-[60%]" :class="{ 'self-bubble': isSelf, 'other-bubble': !isSelf }">
        {{ message.content }}
      </div>
      <span class="text-xs text-gray-500 mt-1">{{ message.sendTime }}</span>
    </div>
  </template>

  <script>
  import { mapState } from 'vuex';
  export default {
    props: {
      message: {
        type: Object,
        required: true,
      },
    },
    computed: {
      ...mapState('user', ['userInfo']),
      isSelf() {
        return this.message.senderId === this.userInfo.userId;
      },
    },
    styles: {
      '.message-item': {
        display: 'flex',
        flexDirection: 'column',
      },
      '.self': {
        alignItems: 'flex-end',
      },
      '.self-bubble': {
        backgroundColor: '#0078ff',
        color: 'white',
      },
      '.other-bubble': {
        backgroundColor: '#fff',
        border: '1px solid #e5e5e5',
      },
    },
  };
  </script>
  ```

#### **3.3 输出**
- 所有界面组件（UI结构完整，样式符合设计方案）；
- 组件间的props/emit传递逻辑（如`ConversationItem`的`select`事件）。


### **阶段4：核心功能实现（Day 7-11）**
**目标**：为组件添加交互逻辑，实现所有核心功能（用户认证、会话管理、聊天交互等）。

#### **4.1 用户认证模块（Day 7）**
- **登录功能**：
  - 实现`Login`组件的表单校验（用户名/密码不为空）；
  - 调用`user`模块的`login` action（通过Axios调用`POST /api/auth/login`）；
  - 成功后跳转到`/chat`页面；
  - 失败时提示错误信息（如“用户名或密码错误”）。
- **注册功能**：
  - 实现`Register`组件的表单校验（用户名/密码不为空，密码一致）；
  - 调用`POST /api/auth/register`接口；
  - 成功后跳转到`/login`页面。
- **登出功能**：
  - 实现`Header`组件的“登出”按钮逻辑：调用`user`模块的`logout` action（调用`POST /api/auth/logout`接口，清除`localStorage`和store中的用户信息，跳转到`/login`）。

#### **4.2 会话管理模块（Day 8）**
- **加载会话列表**：
  - 在`ChatLayout`组件的`mounted`钩子中，调用`conversations`模块的`fetchConversations` action（通过Axios调用`GET /api/conversations`）；
  - 将返回的会话列表存入`conversations`模块的`state`，渲染到`ConversationList`组件。
- **切换会话**：
  - 点击`ConversationItem`组件，触发`select`事件，传递当前会话对象；
  - 更新`currentChat`模块的`currentConversation` state；
  - 调用`currentChat`模块的`fetchHistoryMessages` action（通过Axios调用`GET /api/conversations/{conversationId}/messages`）；
  - 将历史消息存入`currentChat`模块的`currentMessages` state，渲染到`ChatMain`组件的聊天记录区域，并自动滚动到底部（使用`element.scrollIntoView({ behavior: 'smooth' })`）。

#### **4.3 聊天交互模块（Day 9-10）**
- **发送文字消息**：
  - 实现`ChatMain`组件的输入框逻辑（`textarea`，支持回车键发送）；
  - 点击“发送”按钮或按回车键时，校验输入不为空；
  - 构造消息对象（`conversationId`、`type: 'text'`、`content`）；
  - 调用`WebSocket`工具类的`send`方法发送消息；
  - 前端先将消息添加到`currentChat`模块的`currentMessages` state（标记为“发送中”状态，如`status: 'sending'`）；
  - 收到后端推送的`message-ack`事件（消息确认）后，更新消息状态为“已发送”（`status: 'sent'`）。
- **接收实时消息**：
  - 在`WebSocket`工具类的`handleMessage`方法中，处理`new-message`事件；
  - 如果`conversationId`等于当前选中的会话ID，将消息添加到`currentChat`模块的`currentMessages` state，并滚动到底部；
  - 否则，更新`conversations`模块中对应会话的`lastMessage`、`lastMessageTime`和`unreadCount`（+1），显示未读小红点。

#### **4.4 联系人与群管理模块（Day 11）**
- **添加好友**：
  - 实现`AddFriendModal`组件（输入框、搜索按钮、好友信息展示）；
  - 点击“搜索”按钮，调用`GET /api/users/{username}`接口，获取好友信息；
  - 点击“添加好友”按钮，调用`POST /api/friends`接口（参数：`friendId`）；
  - 成功后，后端自动创建单聊会话，前端调用`conversations`模块的`fetchConversations` action，更新会话列表。
- **创建群聊**：
  - 实现`CreateGroupModal`组件（群名称输入框、好友下拉框、创建按钮）；
  - 点击“创建”按钮，调用`POST /api/groups`接口（参数：`groupName`、`members`）；
  - 成功后，后端返回群信息，前端将群聊会话添加到`conversations`模块的`state`，更新会话列表。

#### **4.5 在线状态模块（Day 11）**
- **好友在线状态**：
  - 登录成功后，`WebSocket`连接建立，后端推送`friend-online`事件（好友在线状态列表）；
  - 在`WebSocket`工具类的`handleMessage`方法中，处理`friend-online`事件，更新`onlineStatus`模块的`friendStatus` state（`key: userId`，`value: boolean`）；
  - 渲染`ConversationItem`组件时，根据`friendStatus`显示头像旁的状态指示灯（绿色=在线，灰色=离线）。
- **群成员在线状态**（简化版）：
  - 进入群聊会话时，调用`GET /api/groups/{groupId}/members/online`接口，获取群成员在线状态；
  - 将状态存入`onlineStatus`模块的`groupMemberStatus` state；
  - 在`ChatMain`组件的右侧显示群成员列表（可选），标注在线状态。

#### **4.6 输出**
- 所有核心功能实现（登录/注册/登出、会话管理、消息发送/接收、添加好友/创建群聊、在线状态显示）；
- 功能与后端接口正确对接（HTTP/WS）；
- 状态管理正确（store中的数据与后端一致）。


### **阶段5：联调测试（Day 12-14）**
**目标**：与后端微服务联调，解决接口问题，确保功能正常运行。

#### **5.1 接口联调**
- 与后端团队一起验证每个接口的正确性：
  - 用户认证接口（`/api/auth/login`、`/api/auth/register`、`/api/auth/logout`）；
  - 会话管理接口（`/api/conversations`、`/api/conversations/{id}/messages`）；
  - 联系人与群管理接口（`/api/friends`、`/api/groups`）；
  - WebSocket事件（`new-message`、`friend-online`、`message-ack`）。
- 解决跨域问题（通过Vite的`proxy`配置，或后端设置`CORS`）；
- 解决Token验证问题（确保`Authorization` header正确传递，后端能正确解析）。

#### **5.2 功能测试**
- **手动测试**：
  - 测试所有功能流程（如登录→查看会话列表→切换会话→发送消息→接收消息→添加好友→创建群聊→登出）；
  - 测试边界情况（如输入为空、消息过长、WebSocket断开重连）；
  - 测试未读消息计数（切换会话后重置为0，刷新页面不丢失）。
- **自动化测试**（可选）：
  - 使用Cypress进行端到端测试（测试登录、消息发送等核心流程）；
  - 使用Jest进行单元测试（测试Vuex的mutations、actions，Axios拦截器等）。

#### **5.3 性能优化**
- 优化消息滚动（使用`requestAnimationFrame`避免频繁重绘）；
- 优化输入框（`textarea`自动高度调整，避免滚动条）；
- 优化WebSocket重连（增加重连间隔，避免频繁请求）。

#### **5.4 输出**
- 所有接口联调通过；
- 功能测试报告（无严重BUG）；
- 性能优化后的代码。


### **阶段6：上线部署（Day 15）**
**目标**：将前端应用部署到生产环境，确保用户能访问。

#### **6.1 构建生产包**
- 运行`vite build`命令，生成生产环境包（`dist`目录）；
- 检查构建结果（是否包含所有静态资源，是否压缩正确）。

#### **6.2 部署到服务器**
- **选择部署方式**：
  - 静态托管（如Vercel、Netlify）：适合快速部署，无需配置服务器；
  - 自建服务器（如阿里云ECS）：使用Nginx反向代理，配置`dist`目录为根目录，处理跨域（将`/api`和`/ws`代理到后端接口地址）。
- **Nginx配置示例**：
  ```nginx
  server {
    listen 80;
    server_name im.example.com;

    location / {
      root /path/to/dist;
      index index.html;
      try_files $uri $uri/ /index.html; # 处理单页应用的路由问题
    }

    location /api {
      proxy_pass http://backend-api.example.com; # 后端HTTP接口地址
      proxy_set_header Host $host;
      proxy_set_header X-Real-IP $remote_addr;
    }

    location /ws {
      proxy_pass http://backend-ws.example.com; # 后端WS接口地址
      proxy_http_version 1.1;
      proxy_set_header Upgrade $http_upgrade;
      proxy_set_header Connection "upgrade";
    }
  }
  ```

#### **6.3 监控与运维**
- 配置前端监控（如Sentry），监控错误（如JS错误、接口错误）；
- 配置访问日志（Nginx的`access.log`），分析用户访问情况；
- 上线后，收集用户反馈，及时修复BUG。

#### **6.4 输出**
- 生产环境部署完成（应用可访问）；
- 监控配置完成；
- 上线报告（包含部署地址、监控链接）。


## **三、风险与应对**
| 风险                | 应对措施                                  |
|---------------------|-------------------------------------------|
| 后端接口延迟        | 提前与后端确认接口开发进度，预留联调时间  |
| WebSocket连接问题   | 实现自动重连机制，增加错误日志打印        |
| 状态管理混乱        | 提前设计Store结构，避免数据冗余            |
| 跨域问题            | 使用Vite的`proxy`配置（开发环境），Nginx代理（生产环境） |


## **四、总结**
本开发计划**聚焦后端开发人员的需求**，明确了每个阶段的任务、输出和依赖，确保前端与后端微服务的高效协同。通过**模块化开发**（状态管理、组件、工具类）和**清晰的接口对接**（HTTP/WS），降低了后端开发人员理解前端逻辑的成本。最终实现一个**简洁、直观、易操作**的即时通讯前端应用，满足就业面试的要求。