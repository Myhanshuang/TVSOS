// 这一个 Pinia Store 是为了监测主件是否在用户的视窗内

import { ref, computed } from 'vue'
import { defineStore } from 'pinia'

export const useVisibleStore = defineStore('visible', () => {
    const isFirstVisible = ref(false)

    const isSecondVisible = ref(false)

    const isThirdVisible = ref(false)

  return {
    isFirstVisible,
    isSecondVisible,
    isThirdVisible
  }
})
