/**
 * 车辆实时轮询与地图动画核心逻辑
 * 负责：定时拉取车辆数据、维护地图 Marker、绘制平滑移动动画、展示行驶轨迹轨迹以及同步全局 Store
 */
import { getVehiclesData } from '@/api/vehicle';
import { useVehicleStore } from '@/stores';
import { useWebSocket } from '@vueuse/core'; //
/** 轮询配置 */
let pollingTimerId = null;
const POLLING_DELAY = 2000;    // 刷新频率：2秒一次，与移动动画时长匹配以实现平滑过渡
let isPollingRunning = false;  // 内部运行锁

const DEFAULT_STATIC_VEHICLE_ANGLE = 90; // 默认车辆车头朝向

/**
 * 根据 ID 生成固定的随机颜色
 * 确保同一辆车在地图上的轨迹颜色始终保持一致
 */
const getVehicleColor = (id) => {
    const colors = [
        '#FF5733', '#33FF57', '#3357FF', '#FF33A1', '#33FFF5',
        '#F5FF33', '#FF8C33', '#8C33FF', '#33FF8C', '#FF3333'
    ];
    return colors[id % colors.length];
};

/**
 * 获取并清洗车辆数据
 * 过滤掉坐标超范围或非法的脏数据
 */
const fetchVehicleData = async () => {
    try {
        const response = await getVehiclesData();
        if (response.data.code === 1 && response.data.data) {
            return response.data.data
                .filter(backendCar => {
                    const lon = parseFloat(backendCar.lon)
                    const lat = parseFloat(backendCar.lat);
                    // 中国陆地大致经纬度范围校验
                    const isValidCoord = !isNaN(lon) && !isNaN(lat) &&
                        lon >= 70 && lon <= 140 &&
                        lat >= 3 && lat <= 55;
                    return isValidCoord;
                })
                .map(backendCar => ({
                    ...backendCar,
                    currentPosition: [parseFloat(backendCar.lon), parseFloat(backendCar.lat)]
                }));
        }
        return [];
    } catch (error) {
        console.error("从后端获取车辆数据失败:", error);
        return [];
    }
};

/**
 * 实时更新“已行驶”部分的路径
 * 算法逻辑：在完整路径点集中找到离当前位置最近的点，截取该点之前的轨迹进行高亮显示
 */
const updatePassedPath = (vehicle, carData, AMapInstance) => {
    if (!vehicle.fullPathPoints || vehicle.fullPathPoints.length === 0) return;

    const currentPos = new AMapInstance.LngLat(carData.currentPosition[0], carData.currentPosition[1]);
    let minDistance = Infinity;
    let closestIndex = 0;

    // 寻找最近路径索引点
    for (let i = 0; i < vehicle.fullPathPoints.length; i++) {
        const pt = vehicle.fullPathPoints[i];
        const dist = AMapInstance.GeometryUtil.distance(currentPos, [pt[0], pt[1]]);
        if (dist < minDistance) {
            minDistance = dist;
            closestIndex = i;
        }
    }

    // 截取 0 到 closestIndex 的点作为蓝色/有色高亮路径
    const passedPath = vehicle.fullPathPoints.slice(0, closestIndex + 1);
    passedPath.push(carData.currentPosition); // 末尾连接到车身当前真实点

    if (vehicle.passedPolyline) {
        vehicle.passedPolyline.setPath(passedPath);
    }
};

/**
 * 核心引擎：地图车辆及路径更新主逻辑
 */
const updateVehiclesOnMapLogic = async ({
    AMapInstance,
    map,
    vehiclesMap,
    VEHICLE_ICONS,
    imformStore
}) => {
    if (!AMapInstance || !map) return;

    // 1. 获取新数据并同步到 Pinia 全局状态
    const newVehicleDataList = await fetchVehicleData();
    const vehicleStore = useVehicleStore();
    vehicleStore.setVehicles(newVehicleDataList);

    const currentCarIds = new Set();

    for (const carData of newVehicleDataList) {
        currentCarIds.add(carData.id);
        let vehicle = vehiclesMap.value.get(carData.id);

        const newPosition = carData.currentPosition;
        if (!newPosition) continue;

        // 2. 如果是新车辆，初始化 Marker 和事件
        if (!vehicle) {
            const iconUrl = VEHICLE_ICONS[carData.categoryId] || VEHICLE_ICONS.default;
            vehicle = {
                id: carData.id,
                marker: new AMapInstance.Marker({
                    map: map,
                    position: newPosition,
                    icon: new AMapInstance.Icon({
                        size: new AMapInstance.Size(40, 64),
                        image: iconUrl,
                        imageSize: new AMapInstance.Size(40, 64)
                    }),
                    offset: new AMapInstance.Pixel(-20, -32),
                    angle: carData.angle || DEFAULT_STATIC_VEHICLE_ANGLE
                }),
                fullPolyline: null,     // 灰色底线
                passedPolyline: null,   // 彩色行驶线
                fullPathPoints: null,
                fetchedPathForStatus: null
            };

            // 点击车辆 Marker 展示右侧详情面板
            vehicle.marker.on('click', () => {
                imformStore.imformShow('vehicle', carData);
                map.setCenter(vehicle.marker.getPosition());
            });

            vehiclesMap.value.set(carData.id, vehicle);
        }

        // 3. 路径维护逻辑 (仅在状态 2:接单 或 4:运货中 显示路径)
        const isMovingTask = (carData.status === 2 || carData.status === 4);
        const needsPathUpdate = isMovingTask && (vehicle.fetchedPathForStatus !== carData.status);

        // if (needsPathUpdate && carData.polyline) {
        //     vehicle.fullPathPoints = carData.polyline;
        //     vehicle.fetchedPathForStatus = carData.status;

        //     // 初始化或更新灰色底线轨迹 (全长)
        //     if (!vehicle.fullPolyline) {
        //         vehicle.fullPolyline = new AMapInstance.Polyline({
        //             map: map, path: vehicle.fullPathPoints, strokeColor: "#999999",
        //             strokeOpacity: 0.3, strokeWeight: 6, zIndex: 80
        //         });
        //     } else {
        //         vehicle.fullPolyline.setPath(vehicle.fullPathPoints);
        //     }

        //     // 初始化或更新高亮进度轨迹 (已行驶)
        //     const dynamicColor = getVehicleColor(carData.id);
        //     if (!vehicle.passedPolyline) {
        //         vehicle.passedPolyline = new AMapInstance.Polyline({
        //             map: map, path: [], strokeColor: dynamicColor,
        //             strokeOpacity: 0.9, strokeWeight: 6, zIndex: 81
        //         });
        //     } else {
        //         vehicle.passedPolyline.setOptions({ strokeColor: dynamicColor });
        //         vehicle.passedPolyline.setPath([]);
        //     }
        // }
        if (needsPathUpdate) {
            vehicle.fetchedPathForStatus = carData.status;

            // 如果已有旧连接，先关闭
            if (vehicle.wsClient) vehicle.wsClient.close();

            // 使用 WebSocket 获取对应小车路径（请把 ws://... 换成你的真实后端地址）
            vehicle.wsClient = useWebSocket(`ws://你的后端真实地址/ws/vehicles/path/${carData.id}`, {
                autoReconnect: true,
                onMessage: (ws, event) => {
                    const pathData = JSON.parse(event.data);
                    if (pathData && pathData.length > 0) {
                        vehicle.fullPathPoints = pathData;

                        // 初始化或更新灰色底线轨迹 (全长)
                        if (!vehicle.fullPolyline) {
                            vehicle.fullPolyline = new AMapInstance.Polyline({
                                map: map, path: vehicle.fullPathPoints, strokeColor: "#999999",
                                strokeOpacity: 0.3, strokeWeight: 6, zIndex: 80
                            });
                        } else {
                            vehicle.fullPolyline.setPath(vehicle.fullPathPoints);
                        }

                        // 初始化或更新高亮进度轨迹 (已行驶)
                        const dynamicColor = getVehicleColor(carData.id);
                        if (!vehicle.passedPolyline) {
                            vehicle.passedPolyline = new AMapInstance.Polyline({
                                map: map, path: [], strokeColor: dynamicColor,
                                strokeOpacity: 0.9, strokeWeight: 6, zIndex: 81
                            });
                        } else {
                            vehicle.passedPolyline.setOptions({ strokeColor: dynamicColor });
                            vehicle.passedPolyline.setPath([]);
                        }
                    }
                }
            });
        }
        // 4. 根据移动状态控制轨迹可见性
        if (!isMovingTask) {
            vehicle.fullPolyline?.hide();
            vehicle.passedPolyline?.hide();
            if (carData.status === 1) { // 任务彻底结束，重置状态位以待下次拉取任务路径
                vehicle.fetchedPathForStatus = null;
                vehicle.fullPathPoints = null;
                if (vehicle.wsClient) {
                    vehicle.wsClient.close();
                    vehicle.wsClient = null;
                }
            }
        } else {
            vehicle.fullPolyline?.show();
            vehicle.passedPolyline?.show();
        }

        // 5. 执行平滑移动动画
        // duration 设为与轮询频率一致的 2000ms，确保 Marker 在视觉上持续运动而不停顿
        vehicle.marker.moveTo(newPosition, {
            duration: POLLING_DELAY,
            autoRotation: false
        });

        // 6. 更新 Marker 车头角度
        if (carData.angle !== undefined) vehicle.marker.setAngle(carData.angle);

        // 7. 更新已行驶轨迹的显示
        if (vehicle.passedPolyline && vehicle.fullPathPoints) {
            updatePassedPath(vehicle, carData, AMapInstance);
        }

        // 8. 若当前右侧面板正展示此车辆，通过 Store 同步最新动态
        if (imformStore.recentVehicle?.id === carData.id) {
            imformStore.imformShow('vehicle', carData);
        }
    }

    // 9. 清理逻辑：移除数据库中已删除或不再返回的车辆
    for (const [id, vehicle] of vehiclesMap.value.entries()) {
        if (!currentCarIds.has(id)) {
            vehicle.marker.setMap(null);
            vehicle.fullPolyline?.setMap(null);
            vehicle.passedPolyline?.setMap(null);
            if (vehicle.wsClient) vehicle.wsClient.close();
            vehiclesMap.value.delete(id);
        }
    }
};

/**
 * 轮询递归调度器
 */
const pollingLoop = async (options) => {
    if (!isPollingRunning) return;
    await updateVehiclesOnMapLogic(options);
    if (isPollingRunning) {
        pollingTimerId = setTimeout(() => pollingLoop(options), POLLING_DELAY);
    }
};

/**
 * [导出接口] 启动轮询与动画
 */
export const startPollingAndAnimation = (options) => {
    if (!options.AMapInstance || !options.map || isPollingRunning) return;
    isPollingRunning = true;
    options.isPollingActiveRef.value = true;
    pollingLoop(options);
};

/**
 * [导出接口] 暂停轮询与动画
 * 停止刷新数据，并让现有小车停止移动
 */
export const pausePollingAndAnimation = (options) => {
    isPollingRunning = false;
    if (pollingTimerId) {
        clearTimeout(pollingTimerId);
        pollingTimerId = null;
    }
    options.isPollingActiveRef.value = false;
    options.vehiclesMap.value.forEach(car => car.marker?.stopMove());
};

/**
 * [导出接口] 彻底停止（用于销毁组件）
 */
export const stopPolling = () => {
    isPollingRunning = false;
    if (pollingTimerId) {
        clearTimeout(pollingTimerId);
        pollingTimerId = null;
    }
};