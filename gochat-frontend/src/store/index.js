import { createStore } from 'vuex'
import user from './modules/user'
import conversations from './modules/conversations'
import currentChat from './modules/currentChat'
import onlineStatus from './modules/onlineStatus'

export default createStore({
    modules: {
        user,
        conversations,
        currentChat,
        onlineStatus,
    },
    strict: import.meta.env.NODE_ENV !== 'production'
})
