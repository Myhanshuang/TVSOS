import { ref } from 'vue'
import { defineStore } from 'pinia'

/**
 * POI 数据选定状态管理 Store
 * 用于存储当前在地图或列表中选中的 POI 详情，并处理其状态码、类型码到中文文本的转换逻辑
 */
export const usePoiBoxStore = defineStore('poiBox', () => {

  /** 当前选中的 POI 原始数据对象 */
  const recentPoi = ref(false)

  /** 当前选中 POI 的状态中文描述（如：正常、异常） */
  const recentPoiStatus = ref(false)

  /** 当前选中 POI 的类型中文描述（如：加油站、工厂等） */
  const recentPoiKind = ref(false)

  /**
   * 变更当前选中的 POI 并同步更新对应的中文翻译文本
   * @param {Object} input - 包含 POI 详情的输入对象，需包含 status 和 type 字段
   */
  function recentPoiChange(input) {
    // 保存原始对象
    recentPoi.value = input

    // 转换状态编码：0 映射为正常，其余映射为异常
    if (recentPoi.value.status === 0) {
      recentPoiStatus.value = "异常"
    } else {
      recentPoiStatus.value = "正常"
    }

    // 获取 POI 类型编码并进行分支判断映射
    var val = recentPoi.value.type;

    if (val === 1) {
      recentPoiKind.value = "加油站"
    } else if (val === 2) {
      recentPoiKind.value = "加气站"
    } else if (val === 3) {
      recentPoiKind.value = "其他能源站"
    } else if (val === 4) {
      recentPoiKind.value = "工厂"
    } else if (val === 5) {
      recentPoiKind.value = "汽修厂"
    } else if (val === 6) {
      recentPoiKind.value = "物流园"
    } else if (val === 7) {
      recentPoiKind.value = "火车站"
    } else if (val === 8) {
      recentPoiKind.value = "机场"
    } else if (val === 9) {
      recentPoiKind.value = "购物中心"
    } else if (val === 10) {
      recentPoiKind.value = "家具建材市场"
    } else {
      // 默认兜底类型
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