import { ref } from 'vue'
import { defineStore } from 'pinia'

/**
 * 滚动目标定位状态管理 Store
 * 用于处理跨组件的元素定位逻辑，通过触发一个临时的“监听信号”来通知目标组件执行滚动或高亮操作
 */
export const useTargetStore = defineStore('target', () => {
  /** 
   * 监听触发信号开关 
   * 当为 true 时，表示当前有定位请求正在发生，通常由业务组件通过 watch 监听此值的变化 
   */
  const watchLissoner = ref(false)

  /** 目标项的唯一标识符（如订单 ID 或车辆 ID） */
  const targetId = ref(null)

  /** 内部工具函数：异步延迟延迟 */
  function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms))
  }

  /**
   * 触发一次目标定位变更流程
   * 流程：设置目标 ID -> 开启监听信号 -> 等待动画/处理时间 -> 重置状态
   * @param {string|number} id - 需要定位的目标 ID
   */
  async function targetChange(id) {
    targetId.value = id
    watchLissoner.value = true

    // 持续 400ms 的信号周期，通常配合 CSS 过渡或滚动动画的时间
    await sleep(400)

    // 自动重置，为下一次触发做准备
    watchLissoner.value = false
    targetId.value = null
  }

  return {
    watchLissoner,
    targetId,
    targetChange,
  }
})