import { createRouter, createWebHistory } from 'vue-router'

import login from '@/views/login.vue'

import header from '@/views/header.vue'
import index from '@/views/index.vue'



const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: '/login',
      children:[
        {path: '/login', component: login,},
      ],
    },{
      path: '/admin',
      component: header,
      redirect: '/admin/home',
      children: [
        {path: '/admin/home', component: index,},
      ],

    },
  ],
})

export default router
