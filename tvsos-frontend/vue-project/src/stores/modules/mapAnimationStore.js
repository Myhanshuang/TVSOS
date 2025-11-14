// src/stores/moudles/mapAnimationStore.js
import { defineStore } from 'pinia';
import { ref } from 'vue';
import {
    startPollingAndAnimation as serviceStartPolling,
    pausePollingAndAnimation as servicePausePolling,
    stopPolling as serviceStopPolling // 命名为serviceStopPolling避免混淆
} from '@/utils/vehiclePolling'; // 根据你的实际文件路径调整

export const useMapAnimationStore = defineStore('mapAnimation', () => {
    const isPollingActive = ref(false);
    let pollingOptions = null; // 用于存储 mapContainer 传递过来的地图上下文信息

    /**
     * 设置轮询所需的地图上下文和配置
     * 应该在 mapContainer.vue 的地图初始化完成后调用
     * @param {Object} options - 包含 AMapInstance, map, vehiclesMap 等的配置
     */
    const setPollingOptions = (options) => {
        pollingOptions = { ...options, isPollingActiveRef: isPollingActive }; // 将 isPollingActive ref 一同传递
    };

    /**
     * 启动数据轮询和车辆动画
     */
    const startPolling = () => {
        if (!pollingOptions) {
            console.error("Polling options not set. Map not initialized?");
            return;
        }
        serviceStartPolling(pollingOptions);
        console.log("MapAnimationStore: startPolling called.");
    };

    /**
     * 暂停数据轮询和车辆动画
     */
    const pausePolling = () => {
        if (!pollingOptions) {
            console.error("Polling options not set. Map not initialized?");
            return;
        }
        servicePausePolling(pollingOptions);
        console.log("MapAnimationStore: pausePolling called.");
    };

    /**
     * 全局停止轮询（用于组件卸载时）
     */
    const globalStopPolling = () => {
        serviceStopPolling();
        isPollingActive.value = false; // 重置状态
        pollingOptions = null; // 清除选项
        console.log("MapAnimationStore: globalStopPolling called.");
    };

    return {
        isPollingActive,
        setPollingOptions,
        startPolling,
        pausePolling,
        globalStopPolling,
        // 如果需要，也可以暴露 serviceStopPolling，但通常 globalStopPolling 封装了它的用途
    };
});