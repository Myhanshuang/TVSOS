// src/stores/modules/vehicle.js
import { defineStore } from 'pinia';
import { ref, computed } from 'vue';

export const useVehicleStore = defineStore('vehicle', () => {
  // 使用 ref 存储车辆数据，key 为 license，value 为车辆对象
  // 结构示例: { '粤B12345': { id: '粤B12345', license: '粤B12345', currentPosition: [lng, lat], status: 2, speed: 60, ... }, ... }
  const vehicles = ref(new Map());

  // 设置单个车辆数据
  const setVehicle = (vehicleData) => {
    if (vehicleData && vehicleData.license) {
      vehicles.value.set(vehicleData.license, vehicleData);
    }
  };

  // 批量设置车辆数据（通常用于轮询更新）
  const setVehicles = (vehiclesArray) => {
    if (Array.isArray(vehiclesArray)) {
      const newMap = new Map();
      vehiclesArray.forEach(vehicle => {
        if (vehicle.license) {
          newMap.set(vehicle.license, vehicle);
        }
      });
      vehicles.value = newMap; // 替换整个 Map 以触发响应式更新
    }
  };

  // 删除单个车辆
  const removeVehicle = (license) => {
    vehicles.value.delete(license);
  };

  // 清空所有车辆
  const clearVehicles = () => {
    vehicles.value.clear();
  };

  // 获取单个车辆信息
  const getVehicleByLicense = (license) => {
    return vehicles.value.get(license);
  };

  // 计算属性：将 Map 转换为数组，方便在列表中使用
  const vehicleList = computed(() => {
    return Array.from(vehicles.value.values());
  });

  // 计算属性：获取车辆总数
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