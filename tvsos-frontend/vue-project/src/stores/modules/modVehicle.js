import { defineStore } from 'pinia';
import { ref, computed } from 'vue';

export const useModVehicleStore = defineStore('modvehicle', () => {
    // 1. State: 存储当前被选中的特殊车辆的详细信息
    const recentModVehicle = ref(null);

    // 2. Actions: 定义修改 state 的方法
    function recentModVehicleChange(vehicleData) {
        recentModVehicle.value = vehicleData;
    }

    // 3. Getters: 定义派生状态（计算属性），用于格式化显示
    const modVehicleCategoryText = computed(() => {
        if (!recentModVehicle.value) return '未知类型';
        const categoryMap = {
            1: '平板货车',
            2: '高护栏货车',
            3: '厢式货车',
            4: '冷链运输车',
            5: '危化品运输车',
        };
        return categoryMap[recentModVehicle.value.categoryId] || '未知类型';
    });

    const modVehicleStatusText = computed(() => {
        if (!recentModVehicle.value) return '未知状态';
        const statusMap = {
            1: '空闲',
            2: '接单行驶',
            3: '装货',
            4: '运货行驶',
            5: '卸货中',
            6: '停留等待',
            7: '加油',
            8: '维修'
        };
        return statusMap[recentModVehicle.value.status] || '未知状态';
    });

    return {
        // State
        recentModVehicle,
        // Actions
        recentModVehicleChange,
        // Getters
        modVehicleCategoryText,
        modVehicleStatusText
    };
});