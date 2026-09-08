import { defineStore } from 'pinia'
import { ref } from 'vue'

/**
 * 用户鉴权状态管理 Store
 * 维护登录凭证（Token）与当前用户信息（实验者/研究人员），持久化存储
 */
export const useUserStore = defineStore('big-user', () => {
    /** 用户身份验证令牌，所有受保护的 API 调用均需在请求头携带此值 */
    const token = ref('')
    /** 当前登录用户信息（实验记录 created_by 的展示来源） */
    const userId = ref(0)
    const username = ref('')
    const displayName = ref('')

    const setToken = (newToken) => { token.value = newToken }
    const removeToken = () => { token.value = '' }

    const setUserInfo = ({ id, name, display }) => {
        userId.value = id || 0
        username.value = name || ''
        displayName.value = display || ''
    }
    const clearUserInfo = () => {
        userId.value = 0
        username.value = ''
        displayName.value = ''
    }

    return { token, userId, username, displayName, setToken, removeToken, setUserInfo, clearUserInfo }
},
    {
        /** 启用 Pinia 插件持久化，确保页面刷新后登录状态不丢失 */
        persist: true
    })
