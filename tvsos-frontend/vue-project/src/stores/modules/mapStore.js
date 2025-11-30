import { defineStore } from 'pinia'

export const useMapStore = defineStore('map', {
    state: () => ({
        center: [104.065861, 30.6574013], // 默认中心点
        zoom: 13,
        blinkingPoi: null
    }),
    actions: {
        setCenter(center) {
            this.center = center
        },
        setZoom(zoom) {
            this.zoom = zoom
        },
        setCenterAndZoom(center, zoom) {
            this.center = center
            this.zoom = zoom
        },
        setBlinkingPoi(poi) {
            this.blinkingPoi = poi
        }
    }
})