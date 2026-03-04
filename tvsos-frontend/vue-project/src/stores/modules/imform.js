import { ref } from 'vue'
import { defineStore } from 'pinia'

/**
 * 信息面板状态管理 Store
 * 用于控制右侧信息展示面板的显隐、展示类型以及选中的车辆数据
 */
export const useImformStore = defineStore('imform', () => {

  /** 面板显隐控制开关 (true: 显示, false: 隐藏) */
  const imformIf = ref(false)

  /** 当前展示的信息类型，例如 'vehicle' (车辆) 或 'poi' (地点) */
  const currentInfoType = ref(null);

  /** 当前选中的车辆详细数据对象 */
  const recentVehicle = ref(null);

  /**
   * 显示信息面板
   * @param {string} type - 信息类型 ('vehicle' | 'poi')
   * @param {Object} data - 要展示的数据详情（主要用于车辆）
   */
  function imformShow(type = null, data = null) {
    currentInfoType.value = type;
    // 如果是车辆类型，则保存车辆对象；否则（如 POI）由 poiBoxStore 处理具体数据
    if (type === 'vehicle') {
      recentVehicle.value = data;
    } else {
      recentVehicle.value = null;
    }
    imformIf.value = true
  }

  /**
   * 隐藏信息面板
   * 清除所有状态位和引用的数据对象
   */
  function imformHide() {
    imformIf.value = false
    currentInfoType.value = null;
    recentVehicle.value = null;
  }

  /**
   * 切换信息面板的显隐状态
   * 如果切换为隐藏，则同步清空当前选中的信息
   */
  function imformChange() {
    imformIf.value = !imformIf.value
    if (!imformIf.value) {
      currentInfoType.value = null;
      recentVehicle.value = null;
    }
  }

  return {
    imformIf,
    currentInfoType,
    recentVehicle,
    imformShow,
    imformHide,
    imformChange
  }
})