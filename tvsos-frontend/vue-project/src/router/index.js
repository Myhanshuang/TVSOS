import { createRouter, createWebHistory } from 'vue-router'

import login from '@/views/login.vue'

import header from '@/views/header.vue'
import index from '@/views/index.vue'
import ForgotPassword from '@/views/ForgotPassword.vue'
import landing from '@/views/landing.vue'


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
      children: [
        { path: '/admin/home', component: index, },
      ],

    },
  ],
})

export default router
