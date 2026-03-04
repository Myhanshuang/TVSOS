import { defineStore } from 'pinia'
import { ref } from 'vue'

/**
 * 用户鉴权状态管理 Store
 * 负责维护用户的登录凭证（Token），并支持持久化存储（localStorage）
 */
export const useUserStore = defineStore('big-user', () => {
    /** 用户身份验证令牌，所有受保护的 API 调用均需在请求头携带此值 */
    const token = ref('')

    /**
     * 更新用户 Token（通常在登录成功后调用）
     * @param {string} newToken - 服务器返回的新 Token 字符串
     */
    const setToken = (newToken) => {
        token.value = newToken
    }

    /**
     * 清除用户 Token（通常在退出登录或 Token 过期时调用）
     */
    const removeToken = () => {
        token.value = ''
    }

    return {
        token,
        setToken,
        removeToken
    }
},
    {
        /** 启用 Pinia 插件持久化，确保页面刷新后登录状态不丢失 */
        persist: true
    })