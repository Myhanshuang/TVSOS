//持久化管理
import { createPinia } from 'pinia'
import persist from 'pinia-plugin-persistedstate'
const pinia= createPinia()
pinia.use(persist)
export default pinia 
// import { useUserStore } from './modules/user'
// export {useUserStore}
// import { useCounterStore } from './modules/counter'
// export{useCounterStore}
export * from './modules/user'
export * from './modules/counter'