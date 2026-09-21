import { createRouter, createWebHistory } from 'vue-router'
import login from '@/views/login.vue'
import header from '@/views/header.vue'
import index from '@/views/index.vue'
import ForgotPassword from '@/views/ForgotPassword.vue'
import landing from '@/views/landing.vue'
import sandboxList from '@/views/sandboxList.vue'
import sandboxDetail from '@/views/sandboxDetail.vue'
import experimentResult from '@/views/experimentResult.vue'
import experimentCompare from '@/views/experimentCompare.vue'
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
      redirect: '/admin/sandboxes',
      meta: { requiresAuth: true },
      children: [
        // 沙箱管理首页（默认主页，替代原地图主页）
        { path: '/admin/sandboxes', component: sandboxList, },
        // 沙箱详情：方案列表 + 实验记录 + 新建实验 + 种子导入导出
        { path: '/admin/sandbox/:id', component: sandboxDetail, },
        // 实验对比视图（同沙箱/跨沙箱）
        { path: '/admin/sandbox/:id/compare', component: experimentCompare, },
        // 实验视图：原三屏（地图/统计/任务），仅实验运行期进入
        { path: '/admin/experiment/:id', component: index, },
        // 实验终态结果集查看
        { path: '/admin/experiment/:id/result', component: experimentResult, },
        // 旧路径兼容
        { path: '/admin/home', redirect: '/admin/sandboxes' },
      ],
    },
    // 兜底：任何未匹配路径（如 /admin/sandbox/ 空参）回沙箱列表，避免导航被静默丢弃"卡死"在当前页
    { path: '/:pathMatch(.*)*', redirect: '/admin/sandboxes' },
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
