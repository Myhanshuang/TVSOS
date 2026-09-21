import { ref } from 'vue'
import { defineStore } from 'pinia'

/**
 * 地图数据覆盖层显隐状态 Store
 * 由导航栏“显示数据”按钮控制：开启后凹字形数据面板从地图四周渐入，
 * 覆盖在地图图层之上（不改变地图实际尺寸）；
 * 车辆/POI 信息面板打开期间覆盖层自动退出，关闭后自动回落（联动逻辑在 mapDataOverlay.vue）。
 */
export const useDataPanelStore = defineStore('dataPanel', () => {
  /** 数据面板总开关（true: 显示, false: 隐藏） */
  const showDataPanels = ref(false)

  /** 切换数据面板显隐 */
  function toggleDataPanels() {
    showDataPanels.value = !showDataPanels.value
  }

  /** 显式设置数据面板显隐（如进入实验页时复位为隐藏） */
  function setDataPanels(visible) {
    showDataPanels.value = !!visible
  }

  return {
    showDataPanels,
    toggleDataPanels,
    setDataPanels
  }
})
