import { defineStore } from 'pinia';
import { ref } from 'vue';
import {
    startPollingAndAnimation as serviceStartPolling,
    pausePollingAndAnimation as servicePausePolling,
    stopPolling as serviceStopPolling
} from '@/utils/vehiclePolling';

/**
 * 地图动画及轮询调度状态 Store
 * 负责协调底层车辆位置轮询服务与 UI 层的控制逻辑（启动、暂停、全局停止）
 */
export const useMapAnimationStore = defineStore('mapAnimation', () => {
    /** 原子状态：标识当前轮询是否处于激活运行中 */
    const isPollingActive = ref(false);

    /** 内部存储地图相关的运行上下文配置（包括地图实例、车辆映射表、刷新频率等） */
    let pollingOptions = null;

    /**
     * 初始化/更新轮询服务所需的配置对象
     * @param {Object} options - 包含业务所需的关键地图和车辆资源对象
     */
    const setPollingOptions = (options) => {
        // 将 isActive 的响应式引用注入配置中，以便底层服务能直接修改状态
        pollingOptions = { ...options, isPollingActiveRef: isPollingActive };
    };

    /**
     * 启动地图车辆位置的轮询与轨迹移动动画
     * 该动作通常在进入地图主页或从其他视图切回地图时调用
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
     * 暂停轮询与动画
     * 停止发送后端请求，Marker 会保持在当前动画停止的位置，通常用于临时遮盖或切换标签
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
     * 全局彻底停止轮询服务
     * 清理所有定时器，重置所有内部引用，通常在退出地图模块或组件销毁时调用
     */
    const globalStopPolling = () => {
        serviceStopPolling();
        isPollingActive.value = false;
        pollingOptions = null;
        console.log("MapAnimationStore: globalStopPolling called.");
    };

    return {
        isPollingActive,
        setPollingOptions,
        startPolling,
        pausePolling,
        globalStopPolling,
    };
});