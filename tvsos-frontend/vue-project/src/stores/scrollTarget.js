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
      watchLissoner.value = false
    }
  return { watchLissoner, targetId, targetChange }
})
