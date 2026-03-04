import { defineStore } from 'pinia';
import { ref, computed } from 'vue';

/**
 * 选中/修改中的车辆状态管理 Store
 * 用于处理非主地图场景（如管理后台或修改弹窗）下的车辆详情数据及文本翻译
 */
export const useModVehicleStore = defineStore('modvehicle', () => {
    /** 当前正在操作或查看的车辆详细数据对象 */
    const recentModVehicle = ref(null);

    /**
     * 更新当前选中的车辆数据
     * @param {Object} vehicleData - 完整的车辆信息对象
     */
    function recentModVehicleChange(vehicleData) {
        recentModVehicle.value = vehicleData;
    }

    /**
     * 计算属性：根据车辆分类 ID 返回对应的中文分类名称
     * @returns {string} 映射后的车辆分类文本
     */
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

    /**
     * 计算属性：根据车辆状态码返回对应的中文状态描述
     * @returns {string} 映射后的业务状态文本
     */
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
        recentModVehicle,
        recentModVehicleChange,
        modVehicleCategoryText,
        modVehicleStatusText
    };
});