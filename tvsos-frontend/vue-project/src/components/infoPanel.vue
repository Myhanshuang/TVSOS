<!-- 车辆/POI信息展示面板组件 -->

<script setup>
/**
 * 导入依赖和状态管理
 */
import { useImformStore, usePoiBoxStore } from "@/stores";
import { storeToRefs } from 'pinia';

/**
 * 定义组件 Props
 */
const props = defineProps({
    // 从父组件实时获取的车辆坐标 [经度, 纬度]，用于显示当前位置
    displayPosition: {
        type: Array,
        default: () => null
    }
});

/**
 * 状态库实例化与响应式解构
 */
const imform = useImformStore(); // 面板显示状态及选中的信息详情
const poiBox = usePoiBoxStore(); // POI 相关数据
const { recentPoi } = storeToRefs(poiBox); // 当前查看的 POI 对象
const { currentInfoType, recentVehicle, imformIf } = storeToRefs(imform); // 当前类型(poi/vehicle)、车辆详情、面板显隐开关

/**
 * 将车辆状态编码转换为中文描述
 * @param {number} status 状态码
 * @returns {string} 状态文本
 */
const getVehicleStatusText = (status) => {
    const statusMap = {
        1: '空闲', 2: '接单行驶', 3: '装货', 4: '运货行驶',
        5: '卸货中', 6: '停留等待', 7: '加油', 8: '维修'
    };
    return statusMap[status] || '未知状态';
};

/**
 * 将车辆分类编码转换为中文描述
 * @param {number} category 分类 ID
 * @returns {string} 分类文本
 */
const getVehicleCategoryText = (category) => {
    const statusMap = {
        1: '平板货车', 2: '高护栏货车', 3: '厢式货车',
        4: '冷链运输车', 5: '危化品运输车',
    };
    return statusMap[category] || '未知类型';
};
</script>

<template>
    <!-- 外层容器：控制整个面板的位移和显示/隐藏动画 -->
    <div id="carImfromBox" :class="{ imformShow: imformIf, imformHide: !imformIf }">
        <!-- 内层包装容器：控制内部内容的淡入淡出动画 -->
        <div id="imfromBox" :class="{ show: imformIf, hide: !imformIf }">
            <!-- 关闭按钮：调用 Store 中的隐藏方法 -->
            <el-button class="imformOut" @click="imform.imformHide">×</el-button>

            <!-- POI 详情区域：当 currentInfoType 为 'poi' 且数据存在时显示 -->
            <div v-if="currentInfoType === 'poi' && recentPoi">
                <h3 class="imfromTitle">POI 详细信息</h3>
                <div class="detailedInformation">编号：{{ recentPoi.id }}</div><br>
                <div class="detailedInformation">名称：{{ recentPoi.name }}</div><br>
                <div class="detailedInformation">类型：{{ poiBox.recentPoiKind }}</div><br>
                <div class="detailedInformation">状态：{{ poiBox.recentPoiStatus }}</div>
            </div>

            <!-- 车辆详情区域：当 currentInfoType 为 'vehicle' 且数据存在时显示 -->
            <div v-else-if="currentInfoType === 'vehicle' && recentVehicle">
                <h3 class="imfromTitle">车辆详细信息</h3>
                <div class="detailedInformation">ID：{{ recentVehicle.id }}</div><br>
                <div class="detailedInformation">车牌号：{{ recentVehicle.license }}</div><br>
                <!-- 转换车辆分类为文字 -->
                <div class="detailedInformation">车辆类型：{{ getVehicleCategoryText(recentVehicle.categoryId) }}</div>
                <br>
                <!-- 坐标保留 5 位小数 -->
                <div class="detailedInformation">
                    位置：{{ props.displayPosition?.[0]?.toFixed(5) }}, {{ props.displayPosition?.[1]?.toFixed(5) }}
                </div><br>
                <!-- 速度保留 2 位小数 -->
                <div class="detailedInformation">速度：{{ recentVehicle.speed.toFixed(2) }} km/h</div><br>
                <!-- 转换车辆状态为文字 -->
                <div class="detailedInformation">当前状态：{{ getVehicleStatusText(recentVehicle.status) }}</div><br>
                <!-- 运输距离保留 2 位小数，若为空显示 NaN -->
                <div class="detailedInformation">
                    运输距离：{{ recentVehicle.distance == null ? "NaN" : recentVehicle.distance.toFixed(2) }}（km）
                </div><br>
                <!-- 预计时间保留 2 位小数，若为空显示 NaN -->
                <div class="detailedInformation">
                    预计到达时间：{{ recentVehicle.duration == null ? "NaN" : recentVehicle.duration.toFixed(2) }}（小时）
                </div><br>
                <!-- 最后更新时间 -->
                <div class="detailedInformation">最后更新时间：{{ recentVehicle.updateTime }}</div><br>
            </div>
        </div>
    </div>
</template>

<style scoped>
/* 主面板基础样式及悬浮阴影 */
#carImfromBox {
    display: inline-block;
    position: absolute;
    right: 2vw;
    top: 13vh;
    vertical-align: top;
    width: 20vw;
    height: 80vh;
    border-radius: 20px;
    background: white;
    box-shadow: 14px 14px 30px #bebebe, -14px -14px 30px #ffffff;
    z-index: 3;
}

/* 内部溢出处理及内边距 */
#imfromBox {
    display: inline-block;
    background-color: rgb(255, 255, 255);
    height: calc(100% - 20px);
    width: calc(100% - 10px);
    margin: 10px 10px 10px 10px;
    position: relative;
}

/* 标题样式及底部装饰线条 */
.imfromTitle {
    font-size: 27px;
    font-family: "PingFang SC", "Microsoft YaHei UI";
    display: block;
    position: relative;
    margin: 15px 10px;
}

.imfromTitle::after {
    content: "";
    display: inline-block;
    position: absolute;
    bottom: -6px;
    left: 0px;
    width: calc(100% - 30px);
    height: 4px;
    background-color: rgb(94, 150, 200);
    border-radius: 2px;
}

/* 面板隐藏时的动画状态 */
.imformHide {
    margin: 0px;
    padding: 0px;
    width: 0px;
    opacity: 0;
    transform: translateX(200px);
    transition: all 0.4s cubic-bezier(.35, .74, .33, .75) 0.4s;
}

/* 面板显示时的动画状态 */
.imformShow {
    margin: 0px 0px 0px 100px;
    border-radius: 25px;
    width: 300px;
    opacity: 1;
    transform: translateX(0px);
    transition: all 0.4s cubic-bezier(.35, .74, .33, .75);
}

/* 内部内容显示过渡 */
.show {
    transform: translateX(0px);
    opacity: 1;
    transition: all 0.4s cubic-bezier(.35, .74, .33, .75) 0.4s;
}

/* 内部内容隐藏过渡 */
.hide {
    transform: translateX(50px);
    opacity: 0;
    transition: all 0.4s cubic-bezier(.35, .74, .33, .75);
}

/* 每一行详细信息的标签样式 */
.detailedInformation {
    font-size: 18px;
    font-family: monospace;
    display: inline-block;
    margin: 8px 5px 8px 18px;
    padding: 2px 5px 2px 10px;
    border-left: rgb(94, 150, 200) 4px solid;
    border-radius: 4px;
    background-color: rgb(211, 227, 242);
}

/* 关闭按钮样式 */
.imformOut {
    display: inline-block;
    margin: 0px;
    padding: 0px;
    width: 30px;
    height: 30px;
    position: absolute;
    top: 10px;
    right: 10px;
    font-size: 30px;
    border-radius: 50%;
    border: 0px;
    transition: all 0.5s;
    z-index: 1;
}

/* 关闭按钮悬停效果 */
.imformOut:hover {
    color: rgb(255, 14, 14);
    background-color: white;
}
</style>