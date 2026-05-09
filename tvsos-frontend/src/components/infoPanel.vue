<script setup>
import { useImformStore, usePoiBoxStore } from "@/stores";
import { storeToRefs } from 'pinia';
import { Close } from '@element-plus/icons-vue'; // 引入图标

const props = defineProps({
    displayPosition: {
        type: Array,
        default: () => null
    }
});

const imform = useImformStore();
const poiBox = usePoiBoxStore();
const { recentPoi } = storeToRefs(poiBox);
const { currentInfoType, recentVehicle, imformIf } = storeToRefs(imform);

const getVehicleStatusText = (status) => {
    const statusMap = {
        1: '行驶中', 2: '空闲', 3: '待发车',
        4: '运货行驶', 5: '卸货中', 6: '停留等待', 7: '加油', 8: '维修'
    };
    return statusMap[status] || '未知状态';
};

const getVehicleCategoryText = (category) => {
    const categoryMap = {
        1: '普通厢式货车', 2: '冷藏车', 3: '平板车',
        4: '危化品罐车', 5: '高栏车', 6: '微型面包车'
    };
    return categoryMap[category] || '未知类型';
};
</script>

<template>
    <!-- 使用 Vue Transition 统一管理动画 -->
    <transition name="panel-slide">
        <div v-if="imformIf" id="carImfromBox">
            <div id="imfromBox">
                <!-- 头部工具栏 -->
                <div class="panel-header">
                    <h3 class="imfromTitle">
                        {{ currentInfoType === 'poi' ? 'POI 详细信息' : '车辆详细信息' }}
                    </h3>
                    <el-button class="close-btn" :icon="Close" circle @click="imform.imformHide" />
                </div>

                <div class="panel-content">
                    <!-- POI 详情 -->
                    <div v-if="currentInfoType === 'poi' && recentPoi" class="info-list">
                        <div class="info-item"><span class="label">编号:</span><span class="value">{{ recentPoi.id
                                }}</span></div>
                        <div class="info-item"><span class="label">名称:</span><span class="value">{{ recentPoi.name
                                }}</span></div>
                        <div class="info-item"><span class="label">类型:</span><span class="value">{{ poiBox.recentPoiKind
                                }}</span></div>
                        <div class="info-item"><span class="label">状态:</span><span class="value status-tag">{{
                                poiBox.recentPoiStatus }}</span></div>
                    </div>

                    <!-- 车辆详情 -->
                    <div v-else-if="currentInfoType === 'vehicle' && recentVehicle" class="info-list">
                        <div class="info-item"><span class="label">车牌号:</span><span class="value highlight">{{
                                recentVehicle.license || ('车辆-' + recentVehicle.id) }}</span></div>
                        <div class="info-item"><span class="label">车辆类型:</span><span class="value">{{
                            getVehicleCategoryText(recentVehicle.categoryId || recentVehicle.tybe) }}</span></div>
                        <div class="info-item">
                            <span class="label">当前位置:</span>
                            <span class="value coords">
                                {{ props.displayPosition?.[0]?.toFixed(5) }}, {{ props.displayPosition?.[1]?.toFixed(5)
                                }}
                            </span>
                        </div>
                        <div class="info-item">
                            <span class="label">行驶速度:</span>
                            <span class="value">{{ Number(recentVehicle.speed || 0).toFixed(2) }} <small>km/h</small></span>
                        </div>
                        <div class="info-item">
                            <span class="label">当前状态:</span>
                            <span class="value status-tag active">{{ getVehicleStatusText(recentVehicle.status)
                                }}</span>
                        </div>
                        <div class="info-divider"></div>
                        <div class="info-item">
                            <span class="label">运输距离:</span>
                            <span class="value">{{ recentVehicle.distance?.toFixed(2) || '0.00' }}
                                <small>km</small></span>
                        </div>
                        <div class="info-item">
                            <span class="label">预计耗时:</span>
                            <span class="value">{{ recentVehicle.duration?.toFixed(2) || '0.00' }}
                                <small>h</small></span>
                        </div>
                        <div class="info-item">
                            <span class="label">车辆载重:</span>
                            <span class="value">{{ recentVehicle.capacity || '未知' }} <small>kg</small></span>
                        </div>
                        <div class="info-item">
                            <span class="label">车辆容积:</span>
                            <span class="value">长:{{
                                recentVehicle.length?.toFixed(2) || '未知'
                                }}</span>
                            <span class="value">宽:{{
                                recentVehicle.width?.toFixed(2) || '未知'
                                }}</span>
                            <span class="value">高:{{
                                recentVehicle.height?.toFixed(2) || '未知'
                                }}</span>
                        </div>
                        <div class="info-item">
                            <span class="label">等待时间（当前/总计）:</span>
                            <span class="value">{{ recentVehicle.waitTime?.toFixed(2) || '0.00' }} / {{ recentVehicle.totalWaitTime?.toFixed(2) || '0.00' }}
                                <small>min</small></span>
                        </div>
                        <div class="info-item">
                            <span class="label">空驶里程:</span>
                            <span class="value">{{ recentVehicle.emptyMileage?.toFixed(2) || '0.00' }} <small>km</small></span>
                        </div>
                        <div class="info-item footer-time">
                            <span class="label">最后更新:</span>
                            <span class="value">{{ recentVehicle.updateTime }}</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </transition>
</template>

<style scoped>
/* 面板主体：引入毛玻璃效果和更现代的阴影 */
#carImfromBox {
    position: absolute;
    right: 20px;
    top: 100px;
    width: 320px;
    max-height: 75vh;
    background: rgba(255, 255, 255, 0.85);
    backdrop-filter: blur(10px);
    border-radius: 16px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
    border: 1px solid rgba(255, 255, 255, 0.3);
    z-index: 1000;
    overflow: hidden;
}

#imfromBox {
    padding: 20px;
    display: flex;
    flex-direction: column;
}

/* 头部样式 */
.panel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
}

.imfromTitle {
    font-size: 18px;
    font-weight: 600;
    color: #2c3e50;
    margin: 0;
    position: relative;
}

.imfromTitle::after {
    content: "";
    position: absolute;
    bottom: -6px;
    left: 0;
    width: 30px;
    height: 3px;
    background: #409eff;
    border-radius: 2px;
}

.close-btn {
    border: none;
    background: transparent;
    transition: all 0.3s;
}

.close-btn:hover {
    background: rgba(255, 73, 73, 0.1);
    color: #ff4949;
    transform: rotate(90deg);
}

/* 列表样式优化 */
.info-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.info-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 14px;
    line-height: 1.5;
}

.label {
    color: #909399;
    font-weight: 500;
}

.value {
    color: #303133;
    font-weight: 600;
}

.value.highlight {
    color: #409eff;
}

.value.coords {
    font-family: 'Courier New', Courier, monospace;
    font-size: 13px;
    background: #f4f4f5;
    padding: 2px 6px;
    border-radius: 4px;
}

/* 状态标签样式 */
.status-tag {
    background: #e1f3d8;
    color: #67c23a;
    padding: 2px 10px;
    border-radius: 12px;
    font-size: 12px;
}

.status-tag.active {
    background: #ecf5ff;
    color: #409eff;
}

.info-divider {
    height: 1px;
    background: linear-gradient(90deg, transparent, #ebeef5, transparent);
    margin: 8px 0;
}

.footer-time {
    margin-top: 10px;
    font-size: 12px;
}

.footer-time .value {
    font-weight: normal;
    color: #c0c4cc;
}

/* 过渡动画：平滑右侧划入 */
.panel-slide-enter-active,
.panel-slide-leave-active {
    transition: all 0.4s cubic-bezier(0.165, 0.84, 0.44, 1);
}

.panel-slide-enter-from,
.panel-slide-leave-to {
    opacity: 0;
    transform: translateX(30px);
}
</style>