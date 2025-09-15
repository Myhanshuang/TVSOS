// 这一个 Pinia Store 是为了监测翻页按钮状态，完成翻页效果

import { ref, computed } from 'vue'
import { defineStore } from 'pinia'

export const useTargetStore = defineStore('target', () => {
    const watchLissoner = ref(false)
    const targetId = ref(null)
      // sleep 函数
    function sleep(ms) {
        return new Promise(resolve => setTimeout(resolve, ms))
    }

    async function targetChange(id){
      targetId.value = id
      
      watchLissoner.value = true
      // 异步请求等待一秒让滑动动画完成
      await sleep(1000)   // 
      // 关闭翻页状态，停止翻页
      watchLissoner.value = false
      // 重置翻页目标，准备二次翻页
      targetId.value = null
    }
    
  return {
    watchLissoner,    // 监听翻页函数，指示翻页动画
    targetId,         // 翻页目标
    targetChange,      // 修改翻页目标，翻页函数
  }
})
