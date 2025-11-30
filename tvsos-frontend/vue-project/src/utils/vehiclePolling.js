// src/utils/vehiclePolling.js
import { getVehiclesData } from '@/api/vehicle'; // <--- 导入车辆API
import { useVehicleStore } from '@/stores';
let pollingIntervalId = null;

// 模拟从JSON文件读取车辆数据的函数
// 此函数也被移到服务中，因为它是updateVehiclesOnMapLogic的依赖
//------------------------后续需要与后端对接-------------------------------------------------------
const fetchVehicleData = async () => {

try {
        const response = await getVehiclesData(); // 直接调用封装的 API 函数
        console.log(response.data);
        if (response.data.code === 1 && response.data.data) {
             const cleanedVehicleList = response.data.data
              .filter(backendCar => {
                  const lon = parseFloat(backendCar.lon);
                  const lat = parseFloat(backendCar.lat);
                  // 过滤掉坐标无效的数据 (可以根据需要调整范围)
                  const isValidCoord = !isNaN(lon) && !isNaN(lat) &&
                                       lon >= 70 && lon <= 140 &&
                                       lat >= 3 && lat <= 55;
                  if (!isValidCoord) {
                      console.warn(`车辆 ${backendCar.license || backendCar.id} 坐标无效 (${lon}, ${lat})，已过滤。`);
                  }
                  return isValidCoord;
              })
              .map(backendCar => ({
                  id: backendCar.license || backendCar.id?.toString(),
                  license: backendCar.license,
                  currentPosition: [parseFloat(backendCar.lon), parseFloat(backendCar.lat)],
                  path: Array.isArray(backendCar.fullPath) ? backendCar.fullPath.map(p => [parseFloat(p[0]), parseFloat(p[1])]) : [],
                  status: backendCar.status,
                  speed: backendCar.speed,
                  createTime: backendCar.createTime,
                  updateTime: backendCar.updateTime
                  // ... 其他你需要的字段
              }));
              console.log(cleanedVehicleList);
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

//-----------------------------------------------------------------------------


// 根据新数据更新地图上的车辆，实现平滑移动的核心逻辑
const updateVehiclesOnMapLogic = async ({
    AMapInstance,
    map,
    vehiclesMap,
    updateFrequencyMs,
    DEFAULT_VEHICLE_ICON,
    VEHICLE_FULL_PATH_COLOR,
    VEHICLE_PASSED_PATH_COLOR,
    imformStore
}) => {
    if (!AMapInstance || !map) return;

    const newVehicleDataList = await fetchVehicleData();

    // 新增：获取 vehicleStore 实例
    const vehicleStore = useVehicleStore();
    // 新增：将数据同步到 Pinia Store
    vehicleStore.setVehicles(newVehicleDataList);

    const currentCarIds = new Set();

    // 更新或添加车辆
    for (const carData of newVehicleDataList) {
        currentCarIds.add(carData.id);
        let vehicle = vehiclesMap.value.get(carData.id); // vehiclesMap 是一个 Ref，需要 .value

        const newPosition = carData.currentPosition;
        if (!newPosition) continue; // 如果没有当前位置，则跳过

        if (!vehicle) {
            // 新增车辆
            vehicle = {
                id: carData.id,
                marker: null,
                fullPolyline: null, // 表示车辆的完整规划路径
                realTimeTrackPolyline: null, // 表示车辆实际实时移动的轨迹
                lastPosition: newPosition, // 记录上一次的位置
                passedAnimationPolyline: null, // 用于moveAlong动画的已走过路径
            };

            vehicle.marker = new AMapInstance.Marker({
                map: map, // 使用传递进来的 map 实例
                position: newPosition, // 初始化位置
                icon: DEFAULT_VEHICLE_ICON, // 使用全局定义图标
                offset: new AMapInstance.Pixel(-13, -26), // 调整图标偏移量
                autoRotation: true // 开启自动旋转
            });

            // 为小车 Marker 添加点击事件，显示详细信息
            const handleVehicleMarkerClick = () => {
                imformStore.imformShow('vehicle', carData); // 传递类型和车辆数据
                // 点击小车时，将地图中心移到小车当前位置
                if (map && newPosition) {
                    map.setCenter(newPosition);
                }
            };
            vehicle.marker.on('click', handleVehicleMarkerClick);


            // 初始化实时轨迹线 (仅当carData.path不为空时才创建)
            if (carData.path && carData.path.length > 0) {
                 vehicle.realTimeTrackPolyline = new AMapInstance.Polyline({
                    map: map,
                    path: [newPosition], // 轨迹线从当前位置开始
                    strokeColor: VEHICLE_PASSED_PATH_COLOR, // 使用全局定义的实时轨迹线颜色
                    strokeWeight: 4,
                    strokeStyle: "solid",
                    lineJoin: "round",
                    zIndex: 100 // 确保轨迹线在marker之下但高于基础地图
                });
            }

            // 如果提供了完整路径，则初始化完整路径线
            if (carData.path && carData.path.length > 0) {
                vehicle.fullPolyline = new AMapInstance.Polyline({
                    map: map,
                    path: carData.path,
                    showDir: true,
                    strokeColor: VEHICLE_FULL_PATH_COLOR, // 使用全局定义的完整路径颜色
                    strokeWeight: 6,
                    zIndex: 90 // 完整路径线在实时轨迹线之下
                });
            }
            vehiclesMap.value.set(carData.id, vehicle);

        } else {
            // 更新现有车辆
            const lastPosition = vehicle.lastPosition;

            // 只有当位置发生变化时才进行动画
            if (lastPosition && (lastPosition[0] !== newPosition[0] || lastPosition[1] !== newPosition[1])) {
                // 使用moveAlong实现从上一个点到当前点的平滑移动
                vehicle.marker.moveAlong([lastPosition, newPosition], {
                    duration: updateFrequencyMs, // 动画时长与更新频率一致=============================================<<
                    autoRotation: true,
                });

                // 更新实时轨迹线
                // 如果carData.path存在且有数据，就更新轨迹线
                if (carData.path && carData.path.length > 0) {
                    if (!vehicle.realTimeTrackPolyline) { // 如果不存在则创建
                        vehicle.realTimeTrackPolyline = new AMapInstance.Polyline({
                            map: map,
                            path: [newPosition], // 轨迹线从当前位置开始
                            strokeColor: VEHICLE_PASSED_PATH_COLOR, // 使用全局定义的实时轨迹线颜色
                            strokeWeight: 4,
                            strokeStyle: "solid",
                            lineJoin: "round",
                            zIndex: 100
                        });
                    } else { // 存在就更新
                        const currentTrackPath = vehicle.realTimeTrackPolyline.getPath();
                        // 避免重复添加同一个点
                        if (currentTrackPath.length === 0 ||
                            currentTrackPath[currentTrackPath.length - 1][0] !== newPosition[0] ||
                            currentTrackPath[currentTrackPath.length - 1][1] !== newPosition[1]) {
                                // 停止当前moveAlong，清空其已走过的路径，然后设置新路径
                                // vehicle.marker.stopMove(true); // 停止当前moveAlong，并定位到停止点
                                currentTrackPath.push(newPosition);
                                vehicle.realTimeTrackPolyline.setPath(currentTrackPath);
                        }
                    }
                } else if (vehicle.realTimeTrackPolyline) { // 如果 carData.path 为空或 null，且轨迹线存在，则移除
                    vehicle.realTimeTrackPolyline.setMap(null);
                    vehicle.realTimeTrackPolyline = null;
                }

            } else if (!lastPosition) { // 如果lastPosition是空的（首次获取到），直接定位
                vehicle.marker.setPosition(newPosition);
                // 此时也检查是否需要初始化或清除实时轨迹线
                if (carData.path && carData.path.length > 0) {
                     if (!vehicle.realTimeTrackPolyline) {
                         vehicle.realTimeTrackPolyline = new AMapInstance.Polyline({
                            map: map,
                            path: [newPosition],
                            strokeColor: VEHICLE_PASSED_PATH_COLOR, // 使用全局定义的实时轨迹线颜色
                            strokeWeight: 4,
                            strokeStyle: "solid",
                            lineJoin: "round",
                            zIndex: 100
                        });
                    } else { // 如果已经存在，更新路径为当前位置
                        vehicle.realTimeTrackPolyline.setPath([newPosition]);
                    }
                } else if (vehicle.realTimeTrackPolyline) {
                    vehicle.realTimeTrackPolyline.setMap(null);
                    vehicle.realTimeTrackPolyline = null;
                }
            }

            // 更新lastPosition以供下一次更新使用
            vehicle.lastPosition = newPosition;

            // 如果fullPolyline (完整路径) 是动态变化的，这里需要更新
            if (carData.path && carData.path.length > 0) {
                const currentFullPath = vehicle.fullPolyline ? vehicle.fullPolyline.getPath() : [];
                // 简单的判断路径是否改变，更严谨的应该深比较
                if (!vehicle.fullPolyline || JSON.stringify(currentFullPath) !== JSON.stringify(carData.path)) {
                    if (vehicle.fullPolyline) {
                        vehicle.fullPolyline.setMap(null); // 移除旧路径
                    }
                    vehicle.fullPolyline = new AMapInstance.Polyline({
                        map: map,
                        path: carData.path,
                        showDir: true,
                        strokeColor: VEHICLE_FULL_PATH_COLOR, // 使用全局定义的完整路径颜色
                        strokeWeight: 6,
                        zIndex: 90
                    });
                }
            } else if (vehicle.fullPolyline) {
                // 如果新数据没有路径，但旧车辆有路径，则移除旧路径
                vehicle.fullPolyline.setMap(null);
                vehicle.fullPolyline = null;
            }
        }
    }

    // 移除不再存在的车辆
    for (const [id, vehicle] of vehiclesMap.value.entries()) {
        if (!currentCarIds.has(id)) {
            if (vehicle.marker) {
                vehicle.marker.stopMove(); // 停止可能的动画
                vehicle.marker.setMap(null); // 移除marker
            }
            if (vehicle.fullPolyline) {
                vehicle.fullPolyline.setMap(null); // 移除完整路径
            }
            if (vehicle.realTimeTrackPolyline) {
                vehicle.realTimeTrackPolyline.setMap(null); // 移除实时轨迹线
            }
            if (vehicle.passedAnimationPolyline) {
                vehicle.passedAnimationPolyline.setMap(null); // 移除动画轨迹线
            }
            vehiclesMap.value.delete(id); // 从Map中删除
        }
    }
};

// 启动轮询和动画
export const startPollingAndAnimation = (options) => {
    const {
        AMapInstance, map, vehiclesMap, updateFrequencyMs,
        DEFAULT_VEHICLE_ICON, VEHICLE_FULL_PATH_COLOR, VEHICLE_PASSED_PATH_COLOR,
        isPollingActiveRef, // Reactive ref from component to update UI state
        imformStore
    } = options;

    if (!AMapInstance || !map) {
        console.warn("AMap instance or map not ready, cannot start polling.");
        return;
    }
    if (pollingIntervalId) {
        clearInterval(pollingIntervalId); // 清除现有定时器，避免重复启动
    }

    // 绑定参数到 updateVehiclesOnMapLogic
    const boundUpdateVehicles = () => updateVehiclesOnMapLogic({
        AMapInstance, map, vehiclesMap, updateFrequencyMs,
        DEFAULT_VEHICLE_ICON, VEHICLE_FULL_PATH_COLOR, VEHICLE_PASSED_PATH_COLOR,
        imformStore
    });

    pollingIntervalId = setInterval(boundUpdateVehicles, updateFrequencyMs);
    isPollingActiveRef.value = true; // 更新组件中的状态
    boundUpdateVehicles(); // 立即获取并更新一次数据以显示初始状态

    // 恢复所有现有小车的 moveAlong 动画（如果它们被暂停了）
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
    isPollingActiveRef.value = false; // 更新组件中的状态

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
    mockTick = 0; // 重置模拟数据计数器，确保下次启动时数据从头开始
};