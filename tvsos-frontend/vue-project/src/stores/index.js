import { createPinia } from 'pinia'
import persist from 'pinia-plugin-persistedstate'
const pinia = createPinia()
pinia.use(persist)
export default pinia

//  Pinia Stores 集成
export * from './modules/user'
export * from './modules/imform'
export * from './modules/isVisibleComponents'
export * from './modules/scrollTarget'
export * from './modules/poiBox'
export * from './modules/mapStore'
export * from './modules/mapAnimationStore'
export * from './modules/vehicle';
export * from './modules/modVehicle';