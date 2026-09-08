import request from '@/utils/request'

// 登录：返回 { token, userId, username, displayName }
export const userLogin = ({ username, password }) =>
    request.post('/auth/login', { username, password })

// 注册（简单演示级）
export const userRegister = ({ username, password, displayName }) =>
    request.post('/auth/register', { username, password, displayName })

// 当前登录用户信息
export const userProfile = () => request.get('/auth/profile')

// 退出登录
export const userLogout = () => request.post('/auth/logout')
