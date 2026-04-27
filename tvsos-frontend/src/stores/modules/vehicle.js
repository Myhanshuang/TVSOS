import { defineStore } from 'pinia';
import { ref, computed } from 'vue';

/**
 * 实时车辆数据状态管理 Store
 * 使用 Map 结构存储全量车辆信息，以支持高性能的 key-value 检索（按车牌号）
 */
export const useVehicleStore = defineStore('vehicle', () => {
  /** 
   * 车辆原始数据的 Map 容器 
   * Key: 车辆 License (车牌号), Value: 完整的车辆属性对象 
   */
  const vehicles = ref(new Map());

  /**
   * 更新或新增单辆车的数据
   * @param {Object} vehicleData - 车辆信息对象，必须包含 license 属性
   */
  const setVehicle = (vehicleData) => {
    if (!vehicleData) return;
    const key = vehicleData.license || vehicleData.id;
    if (key !== undefined && key !== null) {
      vehicles.value.set(key, vehicleData);
    }
  };

  /**
   * 批量更新车辆数据（全量覆盖）
   * 通常用于从后端获取全量实时位置列表后的同步操作
   * @param {Array} vehiclesArray - 车辆数据数组
   */
  const setVehicles = (vehiclesArray) => {
    if (Array.isArray(vehiclesArray)) {
      const newMap = new Map();
      vehiclesArray.forEach(vehicle => {
        const key = vehicle?.license || vehicle?.id;
        if (key !== undefined && key !== null) {
          newMap.set(key, vehicle);
        }
      });
      // 整体替换以保持响应式更新
      vehicles.value = newMap;
    }
  };

  /**
   * 根据车牌号移除指定车辆
   * @param {string} license - 车牌号
   */
  const removeVehicle = (license) => {
    vehicles.value.delete(license);
  };

  /**
   * 清空所有车辆数据
   */
  const clearVehicles = () => {
    vehicles.value.clear();
  };

  /**
   * 精确查找：根据车牌号获取车辆对象
   * @param {string} license - 车牌号
   * @returns {Object|undefined} 车辆详情或 undefined
   */
  const getVehicleByLicense = (license) => {
    return vehicles.value.get(license);
  };

  /** 
   * 计算属性：将 Map 中的所有车辆值转换为数组格式 
   * 方便在 Vue 模板中通过 v-for 进行列表遍历展示 
   */
  const vehicleList = computed(() => {
    return Array.from(vehicles.value.values());
  });

  /** 
   * 计算属性：返回当前监控中的车辆总数 
   */
  const vehicleCount = computed(() => {
    return vehicles.value.size;
  });

  return {
    vehicles,
    vehicleList,
    vehicleCount,
    setVehicle,
    setVehicles,
    removeVehicle,
    clearVehicles,
    getVehicleByLicense,
  };
});