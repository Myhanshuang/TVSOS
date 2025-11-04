// 这一个 Pinia Store 是为了监测当前的 显示框 中最近的一次点击的对象信息


import { ref, computed } from 'vue'
import { defineStore } from 'pinia'

export const usePoiBoxStore = defineStore('poiBox', () => {

  const recentPoi = ref(false)

  const recentPoiStatus = ref(false)

  const recentPoiKind = ref(false)

  function recentPoiChange(input){
    recentPoi.value = input
    
    if (recentPoi.value.status === 0){
      recentPoiStatus.value = "正常"
    } else {
      recentPoiStatus.value = "异常"
    }
    
    if (recentPoi.value.type === 1){
      recentPoiKind.value = "加油站"
    } else if (recentPoi.value.type === 2){
      recentPoiKind.value = "加气站"
    } else if (recentPoi.value.type === 3){
      recentPoiKind.value = "其他能源站"
    } else if (recentPoi.value.type === 4){
      recentPoiKind.value = "工厂"
    } else if (recentPoi.value.type === 5){
      recentPoiKind.value = "公司企业"
    } else if (recentPoi.value.type === 6){
      recentPoiKind.value = "购物中心"
    } else if (recentPoi.value.type === 7){
      recentPoiKind.value = "家具建材市场"
    } else {
      recentPoiKind.value = "加油站"
    }

  }

  return { 
    recentPoi,
    recentPoiKind,
    recentPoiStatus,
    recentPoiChange
  }
})
