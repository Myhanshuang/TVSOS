import { ref, computed } from 'vue'
import { defineStore } from 'pinia'

export const useCounterStore = defineStore('counter', () => {
  const imformIf = ref(false)

  function imformShow(){
    imformIf.value = true
  }

  function imformHide(){
    imformIf.value = false
  }

  function imformChange(){
    imformIf.value = !imformIf.value
    console.log(imformIf.value)
  }
  return { imformIf, imformShow, imformHide, imformChange }
})
