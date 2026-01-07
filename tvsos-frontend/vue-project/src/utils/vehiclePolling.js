// src/utils/vehiclePolling.js
import { getVehiclesData, getVehiclePath } from '@/api/vehicle'; // <--- 导入车辆API
import { getSimulationSpeed } from '@/api/simulation';
import { useVehicleStore } from '@/stores';

let pollingTimerId = null;
let currentPollingDelay = 2000;
let isPollingRunning = false; // 内部运行标志

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
                  angle: backendCar.angle, 
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

// 获取仿真倍速并调整轮询间隔
const adjustPollingFrequency = async () => {
    try {
        const res = await getSimulationSpeed();
        if (res.data.code === 1) {
            const multiplier = res.data.data;
            // 基础间隔 2000ms。
            // 1x -> 2000ms
            // 5x -> 400ms
            // 10x -> 200ms
            // 下限 200ms 防止请求过于频繁
            let delay = 2000 / multiplier;
            if (delay < 200) delay = 200;
            currentPollingDelay = delay;
        }
    } catch (e) {
        // ignore
    }
};

// 计算并绘制已行驶路径
const updatePassedPath = (vehicle, carData, AMapInstance) => {
    if (!vehicle.fullPathPoints || vehicle.fullPathPoints.length === 0) return;
    
    // 简单算法：找到完整路径中离当前位置最近的点，截取前面的部分
    const currentPos = new AMapInstance.LngLat(carData.currentPosition[0], carData.currentPosition[1]);
    
    let minDistance = Infinity;
    let closestIndex = 0;
    
    // 优化：不需要每次遍历所有点，可以从上次的 index 开始往后找
    // 这里为了鲁棒性，先遍历
    for (let i = 0; i < vehicle.fullPathPoints.length; i++) {
        const pt = vehicle.fullPathPoints[i];
        const dist = AMapInstance.GeometryUtil.distance(currentPos, [pt[0], pt[1]]);
        if (dist < minDistance) {
            minDistance = dist;
            closestIndex = i;
        }
    }
    
    // 截取 0 到 closestIndex 的点作为已行驶路径
    const passedPath = vehicle.fullPathPoints.slice(0, closestIndex + 1);
    // 加上当前点，保证连接到车身
    passedPath.push(carData.currentPosition);
    
    if (vehicle.passedPolyline) {
        vehicle.passedPolyline.setPath(passedPath);
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
    
    // 顺便更新倍速
    await adjustPollingFrequency();

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

        if (!vehicle) {
            vehicle = {
                id: carData.id,
                marker: null,
                fullPolyline: null,
                passedPolyline: null,
                fullPathPoints: null, // 存储完整路径坐标数组
                hasFetchedPath: false // 标记是否已获取路径
            };

            const markerIcon = new AMapInstance.Icon({
                size: new AMapInstance.Size(40, 64), 
                image: iconUrl,                     
                imageSize: new AMapInstance.Size(40, 64) 
            });

            vehicle.marker = new AMapInstance.Marker({
                map: map,
                position: newPosition,
                icon: markerIcon,
                offset: new AMapInstance.Pixel(-20, -32),
                angle: carData.angle || DEFAULT_STATIC_VEHICLE_ANGLE
            });
            
            const handleVehicleMarkerClick = () => {
                imformStore.imformShow('vehicle', carData);
                const currentMarkerPosition = vehicle.marker.getPosition(); 
                if (map && currentMarkerPosition) {
                    map.setCenter(currentMarkerPosition);
                }
            };
            vehicle.marker.on('click', handleVehicleMarkerClick);

            vehiclesMap.value.set(carData.id, vehicle);
        }
        
        // === 路径获取与绘制逻辑 ===
        const isMovingTask = (carData.status === 2 || carData.status === 4);
        
        // 如果处于任务状态且尚未获取路径，或者路径需要更新（例如换了任务，这里简单判断没有路径就请求）
        if (isMovingTask && !vehicle.hasFetchedPath) {
            try {
                const res = await getVehiclePath(carData.id);
                if (res.data.code === 1 && res.data.data && res.data.data.length > 0) {
                    vehicle.fullPathPoints = res.data.data;
                    vehicle.hasFetchedPath = true;
                    
                    // 绘制完整路径 (灰色底)
                    if (!vehicle.fullPolyline) {
                        vehicle.fullPolyline = new AMapInstance.Polyline({
                            map: map,
                            path: vehicle.fullPathPoints,
                            strokeColor: "#999999", // 灰色
                            strokeOpacity: 0.5,
                            strokeWeight: 6,
                            zIndex: 80
                        });
                    } else {
                        vehicle.fullPolyline.setPath(vehicle.fullPathPoints);
                        vehicle.fullPolyline.setMap(map);
                    }
                    
                    // 初始化已行驶路径线 (高亮)
                    if (!vehicle.passedPolyline) {
                        vehicle.passedPolyline = new AMapInstance.Polyline({
                            map: map,
                            path: [],
                            strokeColor: VEHICLE_PASSED_PATH_COLOR, // 绿色
                            strokeOpacity: 1,
                            strokeWeight: 6,
                            zIndex: 81
                        });
                    } else {
                        vehicle.passedPolyline.setMap(map);
                    }
                }
            } catch (e) {
                console.error("Failed to fetch path for vehicle " + carData.id, e);
            }
        }
        
        // 如果状态变为空闲，清理路径
        if (!isMovingTask) {
             vehicle.hasFetchedPath = false;
             vehicle.fullPathPoints = null;
             if (vehicle.fullPolyline) vehicle.fullPolyline.setMap(null);
             if (vehicle.passedPolyline) vehicle.passedPolyline.setMap(null);
        }

        // === 移动与更新 ===
        
        // 1. 平滑移动
        // duration 使用动态计算的 delay，保证动画速度匹配轮询频率
        vehicle.marker.moveTo(newPosition, {
            duration: currentPollingDelay, 
            autoRotation: false 
        });
        
        // 2. 更新角度
        if (carData.angle !== undefined && carData.angle !== null) {
            vehicle.marker.setAngle(carData.angle);
        }
        
        // 3. 更新已行驶路径
        if (vehicle.passedPolyline && vehicle.fullPathPoints) {
            updatePassedPath(vehicle, carData, AMapInstance);
        }
        
        // [Fix] 实时更新信息面板数据
        if (imformStore.recentVehicle && imformStore.recentVehicle.id === carData.id) {
            // 保留原有引用，更新属性，或者直接替换对象（取决于 store 实现，直接替换通常更安全）
            imformStore.imformShow('vehicle', carData);
        }
    }

    // 移除不再存在的车辆
    for (const [id, vehicle] of vehiclesMap.value.entries()) {
        if (!currentCarIds.has(id)) {
            if (vehicle.marker) {
                vehicle.marker.stopMove();
                vehicle.marker.off('click');
                vehicle.marker.setMap(null);
            }
            if (vehicle.fullPolyline) vehicle.fullPolyline.setMap(null);
            if (vehicle.passedPolyline) vehicle.passedPolyline.setMap(null);
            vehiclesMap.value.delete(id);
        }
    }
};

// 轮询调度器 (递归 setTimeout)
const pollingLoop = async (options) => {
    if (!isPollingRunning) return;
    
    await updateVehiclesOnMapLogic(options);
    
    if (isPollingRunning) {
        pollingTimerId = setTimeout(() => pollingLoop(options), currentPollingDelay);
    }
};

// 启动轮询和动画
export const startPollingAndAnimation = (options) => {
    const {
        AMapInstance, map, vehiclesMap, 
        isPollingActiveRef
    } = options;

    if (!AMapInstance || !map) {
        console.warn("AMap instance or map not ready, cannot start polling.");
        return;
    }
    
    if (isPollingRunning) return; // 已经在运行

    isPollingRunning = true;
    isPollingActiveRef.value = true;
    
    // 立即开始
    pollingLoop(options);

    console.log("Polling and animation started.");
};

// 暂停轮询和动画
export const pausePollingAndAnimation = (options) => {
    const { isPollingActiveRef, vehiclesMap } = options;

    isPollingRunning = false;
    if (pollingTimerId) {
        clearTimeout(pollingTimerId);
        pollingTimerId = null;
    }
    isPollingActiveRef.value = false;

    // 暂停所有现有小车的动画
    vehiclesMap.value.forEach(car => {
        if (car.marker) {
            car.marker.stopMove(); // moveTo 没有 pause，只能 stop
        }
    });
    console.log("Polling and animation paused.");
};

// 用于组件卸载时彻底停止轮询
export const stopPolling = () => {
    isPollingRunning = false;
    if (pollingTimerId) {
        clearTimeout(pollingTimerId);
        pollingTimerId = null;
    }
};