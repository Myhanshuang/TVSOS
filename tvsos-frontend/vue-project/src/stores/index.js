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

//  Pinia Stores 集成
export * from './modules/user'
// export * from './modules/counter'
export * from './modules/imform'
export * from './modules/isVisibleComponents'
export * from './modules/scrollTarget'
export * from './modules/poiBox'
export * from './modules/mapStore'