<!-- 地图容器组件，负责加载高德地图、管理WebGL图层以及处理车辆和POI的交互逻辑 -->
<script setup>
/**
 * 导入工具类、Vue 组合式 API 及 AMap 加载器
 */
import { WebGLLayer } from "@/utils/webGL";
import { onMounted, onUnmounted, ref, shallowRef, watch } from "vue";
import AMapLoader from "@amap/amap-jsapi-loader";
import { useMapAnimationStore, useImformStore, usePoiBoxStore, useMapStore } from "@/stores";
import { getPOIList } from "@/api/poi";
import { storeToRefs } from 'pinia';
import InfoPanel from "./infoPanel.vue";

/** 状态库：面板显示状态 */
const imform = useImformStore();
/** 状态库：POI 数据管理 */
const poiBox = usePoiBoxStore();
/** 状态库：地图基础设置（中心点、层级） */
const mapStore = useMapStore();
const { center, zoom, blinkingPoi } = storeToRefs(mapStore);
/** 从面板状态库解构：当前信息类型及当前选中的车辆对象 */
const { currentInfoType, recentVehicle } = storeToRefs(imform);

/** 地图实例引用，使用 shallowRef 以优化性能（不代理 AMap 内部私有属性） */
let map = shallowRef(null);
/** WebGL 图层管理器，用于海量点渲染 */
let webglManager = null;
/** 存储车辆 Marker 及相关轨迹线的 Map 容器，键为车辆 ID */
const vehiclesMap = shallowRef(new Map());
/** 高德地图运行时命名空间对象 */
let AMapInstance = null;

/** 车辆类型 ID 对应的图标资源路径 */
const VEHICLE_ICONS = {
    1: "/images/货车1.png",
    2: "/images/货车2.png",
    3: "/images/货车3.png",
    4: "/images/货车4.png",
    5: "/images/货车5.png"
};
/** 地图轨迹线颜色配置 */
const VEHICLE_FULL_PATH_COLOR = "#28F"; // 全程路径颜色
const VEHICLE_PASSED_PATH_COLOR = "#AF5"; // 已行驶路径颜色
/** 车辆位置轮询同步频率 */
const updateFrequencyMs = 2000;
/** 状态库：处理地图动画平滑过渡及轮询调度 */
const mapAnimationStore = useMapAnimationStore();

/**
 * 组装传递给动画服务脚本的配置对象
 * @returns {Object} 包含地图资源、配置和状态的回调集合
 */
const getServiceOptions = () => ({
    AMapInstance: AMapInstance,
    map: map.value,
    vehiclesMap: vehiclesMap,
    updateFrequencyMs: updateFrequencyMs,
    VEHICLE_ICONS: VEHICLE_ICONS,
    VEHICLE_FULL_PATH_COLOR: VEHICLE_FULL_PATH_COLOR,
    VEHICLE_PASSED_PATH_COLOR: VEHICLE_PASSED_PATH_COLOR,
    imformStore: imform
});

/** 传给详情面板的实时坐标点（用于维持面板坐标的高频刷新） */
const displayPosition = ref(null);
/** 监听车辆 Marker 坐标变化的定时器 */
let positionInterval = null;

/**
 * 监听最近选中的车辆变更
 * 当选中车辆时，开启高频定时器，从高德 Marker 对象中实时提取当前播放动画中的经纬度，并同步到 UI 面板
 */
watch(recentVehicle, (newVehicle) => {
    if (positionInterval) {
        clearInterval(positionInterval);
        positionInterval = null;
    }

    if (newVehicle && currentInfoType.value === 'vehicle') {
        const vehicle = vehiclesMap.value.get(newVehicle.id);

        if (vehicle && vehicle.marker) {
            const updateDisplayPosition = () => {
                const currentPos = vehicle.marker.getPosition();
                if (currentPos) {
                    displayPosition.value = [currentPos.getLng(), currentPos.getLat()];
                }
            };

            updateDisplayPosition();
            // 每100ms更新一次面板坐标，确保平滑
            positionInterval = setInterval(updateDisplayPosition, 100);
        }
    } else {
        displayPosition.value = null;
    }
}, { deep: true });

/** POI 分类图标映射表 */
const iconMap = {
    1: "/images/加油站.webp",
    2: "/images/加气站.webp",
    3: "/images/其他能源站.webp",
    4: "/images/工厂.webp",
    5: "/images/汽修厂.webp",
    6: "/images/物流园.webp",
    7: "/images/火车站.webp",
    8: "/images/机场.webp",
    9: "/images/购物中心.webp",
    10: "/images/家具建材市场.webp",
};

/**
 * POI 类型编码转换逻辑
 * 将后端返回的细分类型 ID 映射到前端展示的 10 大类图标中
 * @params {number} originalType - 原始分类 ID
 */
const mapPoiType = (originalType) => {
    if (originalType >= 1 && originalType <= 3) {
        return originalType;
    } else if (originalType >= 4 && originalType <= 9) {
        return 4; // 工厂
    } else if (originalType >= 10 && originalType <= 15) {
        return 5; // 汽修厂
    } else if (originalType >= 16 && originalType <= 18) {
        return 6; // 物流园
    } else if (originalType === 19) {
        return 7; // 火车站
    } else if (originalType === 20) {
        return 8; // 机场
    } else if (originalType >= 21 && originalType <= 22) {
        return 9; // 购物中心
    } else if (originalType >= 23 && originalType <= 25) {
        return 10; // 家具建材市场
    }
    return 1; // 兜底类型
};

onMounted(() => {
    /** 配置高德地图安全密钥（JSAPI 2.0 必须） */
    window._AMapSecurityConfig = { securityJsCode: "09582d73da9c81d93b134caf4e6f173a" };

    /** 初始化加载 AMap JSAPI */
    AMapLoader.load({
        key: "84a1985a18fcdb13254b2d85d69885ee",
        version: "2.0",
        plugins: ["AMap.ToolBar", "AMap.Scale", "AMap.MoveAnimation"],
    }).then((AMap) => {
        AMapInstance = AMap;
        /** 创建地图实例 */
        map.value = new AMap.Map("mapContainer", {
            viewMode: "3D", // 开启 3D 模式
            center: center.value,
            zoom: zoom.value,
            mapStyle: "amap://styles/whitesmoke", // 浅色素雅风格
        });

        /** 初始化 WebGL 图层，用于通过 Canvas 绘制海量 POI */
        webglManager = new WebGLLayer(AMap, map.value, {
            // POI 点击回调：更新 Store 状态、定位地图中心并弹出详情面板
            onPoiClick: (poi) => {
                poiBox.recentPoiChange(poi);
                map.value.setZoomAndCenter(15, [poi.lon, poi.lat]);
                imform.imformShow('poi');
            },
            // 点击空白处回调：隐藏面板
            onEmptyClick: () => {
                imform.imformHide();
            }
        });

        /** 初始加载所有 POI 点位数据 */
        getPOIList().then((res) => {
            if (res.data?.code === 1 && res.data.data?.length) {
                webglManager.updateData(res.data.data);
            }
        });

        // 添加地图控件
        map.value.addControl(new AMap.ToolBar()); // 工具条
        map.value.addControl(new AMap.Scale());   // 比例尺

        /**
         * 启动指定 POI 的高亮闪烁效果（通常用于侧边栏搜索定位）
         * @params {Object} poi - 目标 POI 信息
         */
        const startBlinkAnimation = (poi) => {
            if (!map.value) return

            // 清理旧的闪烁标记
            if (window.blinkMarker) {
                map.value.remove(window.blinkMarker)
                clearInterval(window.blinkInterval)
            }

            // 创建一个新的 Marker 用于模拟闪烁
            const marker = new AMap.Marker({
                position: [poi.lon, poi.lat],
                content: createBlinkContent(poi.type),
                zIndex: 9999,
                offset: new AMap.Pixel(-12, -12),

            })
            window.blinkMarker = marker
            map.value.add(marker)

            let blinkTimes = 0
            let visible = true

            // 通过定时器控制 Marker 显隐实现闪烁，达到次数后自动销毁
            window.blinkInterval = setInterval(() => {
                visible = !visible
                if (visible) {
                    marker.show()
                } else {
                    marker.hide()
                }
                blinkTimes++

                if (blinkTimes >= 6 * 2) {
                    clearInterval(window.blinkInterval)
                    map.value.remove(marker)
                    window.blinkMarker = null
                    mapStore.setBlinkingPoi(null) // 重置状态
                }
            }, 300)
        }

        /**
         * 动态创建闪烁标记的 DOM 结构
         */
        const createBlinkContent = (type) => {
            const iconUrl = iconMap[type] || iconMap[1]
            return `
                <div style="
                    width: 24px;
                    height: 24px;
                    background-image: url('${iconUrl}');
                    background-size: cover;
                    border: 2px solid #ff0000;
                    border-radius: 50%;
                    box-shadow: 0 0 10px rgba(255,0,0,0.8);
                "></div>
            `
        }

        /** 监听闪烁请求信号 */
        watch(blinkingPoi, (newPoi) => {
            if (newPoi) {
                startBlinkAnimation(newPoi)
            }
        }, { deep: true })

        /** 输出日志：记录格式化后的 POI 数量 */
        getPOIList().then((res) => {
            if (res.data?.code === 1 && res.data.data?.length) {
                const formattedPoiList = res.data.data.map(poi => ({
                    ...poi,
                    type: mapPoiType(poi.tybe)
                }));
                console.log("初始POI:", formattedPoiList.length);
            }
        });

        /** 配置并启动全局车辆位置轮询服务 */
        mapAnimationStore.setPollingOptions(getServiceOptions());
    });
});

/**
 * 销毁组件时的清理逻辑，防止内存泄漏
 */
onUnmounted(() => {

    if (positionInterval) clearInterval(positionInterval);

    /** 停止状态库中的轮询逻辑 */
    mapAnimationStore.globalStopPolling();

    // 清理 WebGL 相关资源
    if (webglLayerObj?.cleanup) webglLayerObj.cleanup();

    // 显式清理地图上的所有动态 Marker 和 Polyline
    vehiclesMap.value.forEach(car => {
        if (car.marker) {
            car.marker.stopMove();
            car.marker.setMap(null);
        }
        if (car.fullPolyline) car.fullPolyline.setMap(null);
        if (car.realTimeTrackPolyline) car.realTimeTrackPolyline.setMap(null);
        if (car.passedAnimationPolyline) car.passedAnimationPolyline.setMap(null);
    });
    vehiclesMap.value.clear();

    // 彻底销毁高德地图实例
    if (map.value) map.value.destroy();
});

/** 响应式同步：当 Store 中中心点变化时，平移地图 */
watch(center, (newCenter) => {
    if (map.value && newCenter) {
        map.value.setCenter(newCenter)
    }
}, { deep: true })

/** 响应式同步：当 Store 中缩放级别变化时，更新地图层级 */
watch(zoom, (newZoom) => {
    if (map.value && newZoom !== undefined) {
        map.value.setZoom(newZoom)
    }
})
</script>

<template>
    <!-- 全局容器 -->
    <div id="firBorder">
        <!-- 地图区域容器，根据侧边面板显隐动态切换尺寸样式 -->
        <div id="mapBox" :class="{ wideMap: !imform.imformIf, shrotMap: imform.imformIf }">
            <!-- 挂载高德地图的 DOM 节点 -->
            <div id="mapContainer"></div>
        </div>

        <!-- 右侧信息面板子组件：传入实时位置数据 -->
        <InfoPanel :display-position="displayPosition" />
    </div>
</template>

<style scoped>
/* 布局容器样式 */
#firBorder {
    margin: 0;
    padding: 0px 0px;
    width: 100vw;
    height: 92vh;
    display: inline-block;
    text-align: left;
    z-index: 1;
}

/* 地图盒子，包含动态尺寸切换的过渡配合 */
#mapBox {
    margin: 0px;
    padding: 0px;
    width: 100%;
    height: 100%;
    display: inline-block;
    position: relative;
    vertical-align: top;
    z-index: 2;
}

/* 高德地图实际渲染区域 */
#mapContainer {
    width: 100%;
    height: 100%;
}
</style>