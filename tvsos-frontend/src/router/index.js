import { createRouter, createWebHistory } from 'vue-router'
import login from '@/views/login.vue'
import header from '@/views/header.vue'
import index from '@/views/index.vue'
import ForgotPassword from '@/views/ForgotPassword.vue'
import landing from '@/views/landing.vue'
import { useUserStore } from '@/stores'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: '/landing',
      children: [
        { path: 'landing', component: landing, },
        { path: 'login', component: login, },
        { path: 'forgotpassword', component: ForgotPassword }
      ],
    }, {
      path: '/admin',
      component: header,
      redirect: '/admin/home',
      meta: { requiresAuth: true },
      children: [
        { path: '/admin/home', component: index, },
      ],
    },
  ],
})

// 登录守卫：受保护页面无 token 一律回登录页（登录真实化，非演示）
router.beforeEach((to) => {
  const publicPaths = ['/landing', '/login', '/forgotpassword']
  if (publicPaths.includes(to.path)) return true
  const userStore = useUserStore()
  if (!userStore.token) return { path: '/login' }
  return true
})

export default router