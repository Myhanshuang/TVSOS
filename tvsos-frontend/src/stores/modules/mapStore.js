import { defineStore } from 'pinia'

/**
 * 核心地图基础参数管理 Store
 * 用于跨组件维护和调度高德地图的视口状态（中心坐标、缩放层级）及交互提示信号
 */
export const useMapStore = defineStore('map', {
    state: () => ({
        /** 地图视野中心点坐标 [经度, 纬度]，初始默认坐标为成都 */
        center: [104.065861, 30.6574013],

        /** 地图当前的缩放层级 */
        zoom: 13,

        /** 当前触发高亮闪烁效果的 POI 原始数据对象，当地图组件监听到此值变化时会触发 Marker 加点闪烁动画 */
        blinkingPoi: null
    }),
    actions: {
        /**
         * 动态更新地图中心点
         * @param {Array} center [lng, lat] 坐标数组
         */
        setCenter(center) {
            this.center = center
        },

        /**
         * 动态更新地图缩放层级
         * @param {number} zoom 缩放数值
         */
        setZoom(zoom) {
            this.zoom = zoom
        },

        /**
         * 一次性同步更新地图中心点及层级
         */
        setCenterAndZoom(center, zoom) {
            this.center = center
            this.zoom = zoom
        },

        /**
         * 设置具体的 POI 点位使其在地图上产生闪烁提示
         * 常用于侧边栏搜索结果的定位联动
         */
        setBlinkingPoi(poi) {
            this.blinkingPoi = poi
        }
    }
})