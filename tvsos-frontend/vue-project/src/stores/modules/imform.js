// 这一个 Pinia Store 是为了监测小车详细信息栏是否显现的


import { ref, computed } from 'vue'
import { defineStore } from 'pinia'

export const useImformStore = defineStore('imform', () => {
  const imformIf = ref(false)
  const currentInfoType = ref(null); // 'poi' or 'vehicle'
  const recentVehicle = ref(null); // To store clicked vehicle data

  function imformShow(type = null, data = null){
    currentInfoType.value = type;
    if (type === 'vehicle') {
      recentVehicle.value = data;
    } else {
      recentVehicle.value = null; // 清除车辆信息，防止混淆
    }
     imformIf.value = true
   }

  function imformHide(){
    imformIf.value = false
    currentInfoType.value = null; // 隐藏时清除类型
    recentVehicle.value = null;   // 隐藏时清除车辆信息
  }

  function imformChange(){
    imformIf.value = !imformIf.value
    if (!imformIf.value) { // 如果切换到隐藏状态，则清除内容
      currentInfoType.value = null;
      recentVehicle.value = null;
    }
  }
  return { 
    imformIf,           // 信息栏是否显现
    currentInfoType,    // 当前显示的信息类型 ('poi', 'vehicle' 或 null)
    recentVehicle,      // 最近点击的小车信息
    imformShow,         // 显现函数 (通用，现在带参数)
    imformHide,         // 隐藏函数
    imformChange        // 状态修改函数
  }
})
