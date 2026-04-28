import { ref } from 'vue'
import { defineStore } from 'pinia'

/**
 * 全局组件/视图可见性状态管理 Store
 * 用于控制页面中不同业务模块（如地图主页、统计分析、任务管理等）的显示与隐藏
 */
export const useVisibleStore = defineStore('visible', () => {

  /** 
   * 第一组件（地图界面/大屏主页）的可见性开关 
   */
  const isFirstVisible = ref(false)

  /** 
   * 第二组件（统计分析/图表界面）的可见性开关 
   */
  const isSecondVisible = ref(false)

  /** 
   * 第三组件（订单管理/任务列表界面）的可见性开关 
   */
  const isThirdVisible = ref(false)

  /** 
   * 第四组件（答辩演示页面）的可见性开关 
   */
  const isFourthVisible = ref(false)

  return {
    isFirstVisible,
    isSecondVisible,
    isThirdVisible,
    isFourthVisible
  }
})