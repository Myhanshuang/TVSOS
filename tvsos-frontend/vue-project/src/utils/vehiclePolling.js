// src/utils/vehiclePolling.js
import { getVehiclesData } from '@/api/vehicle'; // <--- 导入车辆API
import { useVehicleStore } from '@/stores';
let pollingIntervalId = null;

const DEFAULT_STATIC_VEHICLE_ANGLE = 90;

const fetchVehicleData = async () => {
    try {
        const response = await getVehiclesData();
        if (response.data.code === 1 && response.data.data) {
             const cleanedVehicleList = response.data.data
              .filter(backendCar => {
                  const lon = parseFloat(backendCar.lon)
                  const lat = parseFloat(backendCar.lat);
                  const isValidCoord = !isNaN(lon) && !isNaN(lat) &&
                                       lon >= 70 && lon <= 140 &&
                                       lat >= 3 && lat <= 55;
                  if (!isValidCoord) {
                      console.warn(`车辆 ${backendCar.license || backendCar.id} 坐标无效 (${lon}, ${lat})，已过滤。`);
                  }
                  return isValidCoord;
              })
              .map(backendCar => ({
                  id: backendCar.id,
                  license: backendCar.license,
                  currentPosition: [parseFloat(backendCar.lon), parseFloat(backendCar.lat)],
                  path: Array.isArray(backendCar.polyline) ? backendCar.polyline.map(p => [parseFloat(p[0]), parseFloat(p[1])]) : [], 
                  status: backendCar.status,
                  speed: backendCar.speed,
                  createTime: backendCar.createTime,
                  updateTime: backendCar.updateTime,
                  categoryId: backendCar.categoryId,
                  cargoSize: backendCar.cargoSize,
                  distance: backendCar.distance,
                  duration: backendCar.duration
              }));
            return cleanedVehicleList;
            
        } else {
            console.error("后端返回错误或数据为空:", response.data.message);
            return [];
        }
    } catch (error) {
        console.error("从后端获取车辆数据失败:", error);
        return [];
    }
};

const updateVehiclesOnMapLogic = async ({
    AMapInstance,
    map,
    vehiclesMap,
    VEHICLE_ICONS,
    VEHICLE_FULL_PATH_COLOR,
    VEHICLE_PASSED_PATH_COLOR,
    imformStore
}) => {
    if (!AMapInstance || !map) return;

    const newVehicleDataList = await fetchVehicleData();
    const vehicleStore = useVehicleStore();
    vehicleStore.setVehicles(newVehicleDataList);

    const currentCarIds = new Set();
    
    for (const carData of newVehicleDataList) {
        currentCarIds.add(carData.id);
        let vehicle = vehiclesMap.value.get(carData.id);

        const newPosition = carData.currentPosition;
        if (!newPosition) continue;

        const iconUrl = VEHICLE_ICONS[carData.categoryId] || VEHICLE_ICONS.default;
        const hasNewPath = carData.path && carData.path.length > 1; 

        if (!vehicle) {
            vehicle = {
                id: carData.id,
                marker: null,
                fullPolyline: null,
                passedPolyline: null, 
                currentPath: null,    
            };

            const markerIcon = new AMapInstance.Icon({
                size: new AMapInstance.Size(40, 64), // 根据您的SVG图标实际大小调整
                image: iconUrl,                     // 使用动态选择的图标 URL
                imageSize: new AMapInstance.Size(40, 64) // 确保这里也与实际图标大小匹配
            });

            vehicle.marker = new AMapInstance.Marker({
                map: map,
                position: newPosition,
                icon: markerIcon,
                offset: new AMapInstance.Pixel(-20, -32), // 根据图标大小调整偏移
                angle: hasNewPath ? undefined : DEFAULT_STATIC_VEHICLE_ANGLE
            });
            
            // === 核心修正点 ===
            // 定义点击事件处理器
            const handleVehicleMarkerClick = () => {
                // 注意：这里 imformStore.imformShow 使用的 carData 是闭包中的，可能不是最新的。
                // 如果需要显示最新的车辆信息，应从 store 中根据 vehicle.id 或 carData.license 重新获取。
                imformStore.imformShow('vehicle', carData);

                // 从 marker 对象实时获取当前位置来设置地图中心
                const currentMarkerPosition = vehicle.marker.getPosition(); 
                if (map && currentMarkerPosition) {
                    map.setCenter(currentMarkerPosition);
                }
            };
            // 绑定点击事件
            vehicle.marker.on('click', handleVehicleMarkerClick);

            // 绑定移动事件
            vehicle.marker.on('moving', (e) => {
                if (vehicle.passedPolyline) vehicle.passedPolyline.setPath(e.passedPath);
            });

            vehiclesMap.value.set(carData.id, vehicle);
        }
        
        
        const isPathChanged = hasNewPath && (JSON.stringify(vehicle.currentPath) !== JSON.stringify(carData.path));
        
        if (isPathChanged) {
            if (vehicle.fullPolyline) {
                vehicle.fullPolyline.setPath(carData.path);
            } else {
                vehicle.fullPolyline = new AMapInstance.Polyline({ map, path: carData.path, showDir: true, strokeColor: VEHICLE_FULL_PATH_COLOR, strokeWeight: 6, zIndex: 90 });
            }

            if (vehicle.passedPolyline) {
                vehicle.passedPolyline.setPath([]); 
            } else {
                vehicle.passedPolyline = new AMapInstance.Polyline({ map, strokeColor: VEHICLE_PASSED_PATH_COLOR, strokeWeight: 4, strokeStyle: "solid", lineJoin: "round", zIndex: 100 });
            }
           
            vehicle.currentPath = carData.path; 
            vehicle.marker.stopMove(); 
            vehicle.marker.moveAlong(carData.path, {
                duration: carData.duration ? carData.duration * 1000 : (AMapInstance.GeometryUtil.distanceOfLine(carData.path) / (carData.speed || 50) * 3.6 * 1000),
                autoRotation: true,
            });

        } else if (!hasNewPath) {
             vehicle.marker.stopMove();
             vehicle.marker.setPosition(newPosition);
             vehicle.marker.setAngle(DEFAULT_STATIC_VEHICLE_ANGLE);
             
             if (vehicle.fullPolyline) {
                 vehicle.fullPolyline.setMap(null);
                 vehicle.fullPolyline = null;
             }
             if (vehicle.passedPolyline) {
                 vehicle.passedPolyline.setMap(null);
                 vehicle.passedPolyline = null;
             }
             vehicle.currentPath = null;
        }
    }

    // 移除不再存在的车辆
    for (const [id, vehicle] of vehiclesMap.value.entries()) {
        if (!currentCarIds.has(id)) {
            if (vehicle.marker) {
                vehicle.marker.stopMove();
                vehicle.marker.off('click');
                vehicle.marker.off('moving');
                vehicle.marker.setMap(null);
            }
            if (vehicle.fullPolyline) vehicle.fullPolyline.setMap(null);
            if (vehicle.passedPolyline) vehicle.passedPolyline.setMap(null);
            vehiclesMap.value.delete(id);
        }
    }
};

// 启动轮询和动画
export const startPollingAndAnimation = (options) => {
    const {
        AMapInstance, map, vehiclesMap, updateFrequencyMs,
        isPollingActiveRef
    } = options;

    if (!AMapInstance || !map) {
        console.warn("AMap instance or map not ready, cannot start polling.");
        return;
    }
    if (pollingIntervalId) {
        clearInterval(pollingIntervalId); 
    }

    const boundUpdateVehicles = () => updateVehiclesOnMapLogic(options);

    pollingIntervalId = setInterval(boundUpdateVehicles, updateFrequencyMs);
    isPollingActiveRef.value = true; 
    boundUpdateVehicles(); 

    // 恢复所有现有小车的 moveAlong 动画
    vehiclesMap.value.forEach(car => {
        if (car.marker) {
            car.marker.resumeMove();
        }
    });

    console.log("Polling and animation started.");
};

// 暂停轮询和动画
export const pausePollingAndAnimation = (options) => {
    const { isPollingActiveRef, vehiclesMap } = options;

    if (pollingIntervalId) {
        clearInterval(pollingIntervalId);
        pollingIntervalId = null;
    }
    isPollingActiveRef.value = false;

    // 暂停所有现有小车的 moveAlong 动画
    vehiclesMap.value.forEach(car => {
        if (car.marker) {
            car.marker.pauseMove();
        }
    });
    console.log("Polling and animation paused.");
};

// 用于组件卸载时彻底停止轮询
export const stopPolling = () => {
    if (pollingIntervalId) {
        clearInterval(pollingIntervalId);
        pollingIntervalId = null;
    }
};