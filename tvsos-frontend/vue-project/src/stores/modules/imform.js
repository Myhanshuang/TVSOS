// 这一个 Pinia Store 是为了监测小车详细信息栏是否显现的


import { ref, computed } from 'vue'
import { defineStore } from 'pinia'

export const useImformStore = defineStore('imform', () => {
  const imformIf = ref(false)


  function imformShow(){
    imformIf.value = true
  }

  function imformHide(){
    imformIf.value = false
  }

  function imformChange(){
    imformIf.value = !imformIf.value
  }
  return { 
    imformIf,       // 小车状态栏是否显现
    imformShow,     // 显现函数
    imformHide,     // 隐藏函数
    imformChange    // 状态修改函数
  }
})
