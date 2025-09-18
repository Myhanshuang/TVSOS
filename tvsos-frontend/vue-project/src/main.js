import '@/assets/global.css'


import { createApp } from 'vue'
import ElementPlus from 'element-plus';
import 'element-plus/dist/index.css'
import App from './App.vue'
import router from './router'
//  可持久化 Pinia 配置
import pinia from '@/stores/index'


const app = createApp(App)

app.use(pinia)
app.use(router)
app.use(ElementPlus)

app.mount('#app')
