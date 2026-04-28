<script setup>
import request from '@/utils/request.js'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useMapAnimationStore, useVisibleStore, useTargetStore } from '@/stores'
import { ref, onMounted, computed } from 'vue'
import { storeToRefs } from 'pinia'
import { setSimulationSpeed, getSimulationSpeed } from '@/api/simulation'
import PoiList from '@/components/poiList.vue'
import VehicleList from '@/components/vehicleList.vue'
// 导入图标
import {
    MapLocation, DataAnalysis, List,
    Van, Location, VideoPlay, VideoPause,
    Box, SwitchButton, Odometer, ArrowDown
} from '@element-plus/icons-vue'
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()
const target = useTargetStore()
const visible = useVisibleStore()
const drawerVisible = ref(false)
const vehicleDrawerVisible = ref(false)

const openDrawer = () => drawerVisible.value = true
const openVehicleDrawer = () => vehicleDrawerVisible.value = true

const mapAnimationStore = useMapAnimationStore()
const { isPollingActive } = storeToRefs(mapAnimationStore)
const mockCount = ref(5)

// ... (逻辑部分保持不变，仅优化 HTML/CSS) ...

const handleMockShipments = async () => {
    if (mockCount.value <= 0 || !Number.isInteger(mockCount.value)) {
        ElMessage.warning('请输入一个有效的正整数')
        return
    }
    try {
        await ElMessageBox.confirm(`确定要生成 ${mockCount.value} 条模拟订单数据吗？`, '确认操作', {
            confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning',
        })
        const res = await request.post(`/shipments/mock/${mockCount.value}`)
        if (res.data.code === 1) ElMessage.success(`成功生成数据`)
    } catch (err) { /* handle error */ }
}

const currentSpeed = ref(1.0)
const speedOptions = [1.0, 2.0, 5.0, 10.0, 20.0, 50.0]

const handleSetSpeed = async (speed) => {
    try {
        const res = await setSimulationSpeed(speed)
        if (res.data.code === 1) {
            currentSpeed.value = speed
            ElMessage.success(`仿真倍速: ${speed}x`)
        }
    } catch (error) { ElMessage.error('设置失败') }
}

const initSpeed = async () => {
    try {
        const res = await getSimulationSpeed()
        if (res.data.code === 1) currentSpeed.value = res.data.data
    } catch (error) { }
}

onMounted(() => initSpeed())
</script>

<template>
  <div class="layout-container">
    <header class="nav-header">
        <!-- 1. Logo 区域 -->
        <div class="logo-section">
            <el-icon class="logo-icon">
                <MapLocation />
            </el-icon>
            <span class="logo-text">TVSOS</span>
        </div>

        <!-- 2. 主导航区域 -->
        <div class="nav-main">
            <div class="nav-item" :class="{ active: route.path === '/admin/home' && visible.isFirstVisible }" @click="() => { router.push('/admin/home'); target.targetChange('first') }">
                <el-icon>
                    <MapLocation />
                </el-icon>
                <span class="nav-text">地图视图</span>
            </div>
            <div class="nav-item" :class="{ active: route.path === '/admin/home' && visible.isSecondVisible }" @click="() => { router.push('/admin/home'); target.targetChange('second') }">
                <el-icon>
                    <DataAnalysis />
                </el-icon>
                <span class="nav-text">统计分析</span>
            </div>
            <div class="nav-item" :class="{ active: route.path === '/admin/home' && visible.isThirdVisible }" @click="() => { router.push('/admin/home'); target.targetChange('third') }">
                <el-icon>
                    <List />
                </el-icon>
                <span class="nav-text">任务管理</span>
            </div>
            <div class="nav-item" :class="{ active: route.path === '/admin/home' && visible.isFourthVisible }" @click="() => { router.push('/admin/home'); target.targetChange('fourth') }">
                <el-icon>
                    <DataAnalysis />
                </el-icon>
                <span class="nav-text">算法演示</span>
            </div>
        </div>

        <!-- 3. 数据列表入口 (小按钮组) -->
        <div class="data-group">
            <el-button-group>
                <el-button @click="openVehicleDrawer" :icon="Van">
                    <span class="btn-text">车辆</span>
                </el-button>
                <el-button @click="openDrawer" :icon="Location">
                    <span class="btn-text">POI</span>
                </el-button>
            </el-button-group>
        </div>

        <!-- 4. 仿真控制区 (带背景的集成区域) -->
        <div class="simulation-panel">
            <div class="speed-control">
                <el-icon>
                    <Odometer />
                </el-icon>
                <!-- 宽屏显示的 Radio Group -->
                <el-radio-group v-model="currentSpeed" size="small" @change="handleSetSpeed" class="speed-radio-group">
                    <el-radio-button v-for="s in speedOptions" :key="s" :label="s">{{ s }}x</el-radio-button>
                </el-radio-group>

                <!-- 窄屏显示的 Dropdown -->
                <el-dropdown trigger="click" @command="handleSetSpeed" class="speed-dropdown">
                    <el-button size="small" type="primary" link>
                        {{ currentSpeed }}x <el-icon class="el-icon--right"><arrow-down /></el-icon>
                    </el-button>
                    <template #dropdown>
                        <el-dropdown-menu>
                            <el-dropdown-item v-for="s in speedOptions" :key="s" :command="s">
                                {{ s }}x
                            </el-dropdown-item>
                        </el-dropdown-menu>
                    </template>
                </el-dropdown>
            </div>

            <div class="action-btns">
                <el-button :type="isPollingActive ? 'info' : 'success'" :icon="VideoPlay" circle
                    @click="mapAnimationStore.startPolling" :disabled="isPollingActive" />
                <el-button :type="!isPollingActive ? 'info' : 'warning'" :icon="VideoPause" circle
                    @click="mapAnimationStore.pausePolling" :disabled="!isPollingActive" />
            </div>

            <div class="mock-group">
                <el-input-number v-model="mockCount" :min="1" size="small" controls-position="right"
                    class="mock-input" />
                <el-button type="primary" plain size="small" :icon="Box" @click="handleMockShipments">
                    <span class="btn-text">Mock</span>
                </el-button>
            </div>
        </div>

        <!-- 5. 用户退出 -->
        <div class="user-section">
            <router-link to="/login" class="exit-btn">
                <el-icon>
                    <SwitchButton />
                </el-icon>
                <span>退出登录</span>
            </router-link>
        </div>
    </header>

    <poi-list :visible="drawerVisible" @update:visible="drawerVisible = $event" />
    <vehicle-list :visible="vehicleDrawerVisible" @update:visible="vehicleDrawerVisible = $event" />
    <RouterView></RouterView>
  </div>
</template>

<style scoped>
.nav-header {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 64px;
    display: flex;
    align-items: center;
    padding: 0 24px;
    background: rgba(255, 255, 255, 0.8);
    backdrop-filter: blur(12px);
    box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.05);
    border-bottom: 1px solid rgba(235, 238, 245, 0.8);
    z-index: 2000;
    box-sizing: border-box;
}

/* Logo 样式 */
.logo-section {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-right: 40px;
}

.logo-icon {
    font-size: 24px;
    color: #409eff;
}

.logo-text {
    font-size: 22px;
    font-weight: 800;
    font-family: 'Inter', sans-serif;
    background: linear-gradient(120deg, #409eff, #3622ab);
    background-clip: text;
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    letter-spacing: 1px;
}

/* 主导航项 */
.nav-main {
    display: flex;
    gap: 8px;
    height: 100%;
}

.nav-item {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 0 5px;
    height: 100%;
    cursor: pointer;
    color: #606266;
    font-size: 15px;
    font-weight: 500;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    position: relative;
    transition: color 0.3s, background-color 0.3s;
}

.nav-item:hover {
    color: #409eff;
    background: rgba(64, 158, 255, 0.05);
}

.nav-item.active {
    color: #409eff;
}

.nav-item.active::after {
    content: "";
    position: absolute;
    bottom: 0;
    left: 15%;
    width: 70%;
    height: 3px;
    background: #409eff;
    border-radius: 3px 3px 0 0;
    transition: all 0.3s ease;
}

.data-group {
    margin-left: 20px;
}

/* 仿真面板集成区 */
.simulation-panel {
    margin-left:10px; /* 推到右侧 */
    display: flex;
    align-items: center;
    gap: 10px;
    background: #f5f7fa;
    padding: 6px 16px;
    border-radius: 32px;
    border: 1px solid #e4e7ed;
}

.speed-control {
    display: flex;
    align-items: center;
    gap: 8px;
    color: #909399;
}

.action-btns {
    display: flex;
    gap: 8px;
    padding: 0 12px;
    border-left: 1px solid #dcdfe6;
    border-right: 1px solid #dcdfe6;
}

.mock-group {
    display: flex;
    align-items: center;
    gap: 8px;
}

/* 退出登录 */
.user-section {
    margin-left: 20px;
}

.exit-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    text-decoration: none;
    color: #909399;
    font-size: 14px;
    padding: 8px 12px;
    border-radius: 8px;
    transition: all 0.3s;
}

.exit-btn:hover {
    color: #f56c6c;
    background: rgba(245, 108, 108, 0.1);
}

/* 兼容部分按钮样式 */
:deep(.el-radio-button__inner) {
    border-radius: 12px !important;
    margin: 0 2px;
    border: none !important;
}
.nav-item,
.exit-btn,
.logo-text {
    white-space: nowrap;
    flex-shrink: 0; 
}

/* 2. 修复“车辆”和“POI”按钮组变成上下排列的问题 */
.data-group {
    display: flex;
    flex-shrink: 0;
}
:deep(.el-button-group) {
    display: flex;
    flex-wrap: nowrap;
    white-space: nowrap;
}

/* 3. 修复倍速控制（50x）掉到下一行的问题 */
.speed-control {
    white-space: nowrap;
    flex-shrink: 0;
}
:deep(.el-radio-group) {
    display: flex !important;
    flex-wrap: nowrap !important;
}

/* 4. 防止几个主要大区块被过度压缩 */
.logo-section,
.nav-main,
.simulation-panel,
.user-section {
    flex-shrink: 0;
}
.speed-dropdown {
    display: none;
}

/* 1. 中等屏幕 (1200px 以下)：隐藏导航文字和 Logo 文字 */
@media screen and (max-width: 1200px) {

    .logo-text,
    .nav-text {
        display: none;
    }

    .logo-section {
        margin-right: 20px;
    }

    .nav-item {
        padding: 0 12px;
    }
}

/* 2. 窄屏幕 (1000px 以下)：速度切换变为下拉菜单，隐藏按钮文字 */
@media screen and (max-width: 1000px) {

    /* 隐藏所有次要文字 */
    .btn-text,
    .exit-btn span {
        display: none;
    }

    /* 缩小间距 */
    .nav-header {
        padding: 0 12px;
    }

    .simulation-panel {
        gap: 8px;
        padding: 6px 10px;
        margin-left: 10px;
    }

    .data-group,
    .user-section {
        margin-left: 10px;
    }

    /* 切换速度控制器：隐藏 Radio，显示 Dropdown */
    .speed-radio-group {
        display: none;
    }

    .speed-dropdown {
        display: inline-flex;
    }
}

/* 3. 极窄屏幕 (768px 以下)：进一步压缩 */
@media screen and (max-width: 768px) {
    .mock-input {
        width: 60px !important;
        /* 缩小输入框 */
    }

    .action-btns {
        padding: 0 4px;
        gap: 4px;
    }

    .simulation-panel {
        background: transparent;
        /* 节省空间，去掉背景 */
        border: none;
        padding: 0;
    }
}
</style>