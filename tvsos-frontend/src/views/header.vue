<script setup>
import { ElMessage, ElMessageBox } from 'element-plus'
import { useMapAnimationStore, useVisibleStore, useTargetStore, useUserStore, useDataPanelStore, useImformStore } from '@/stores'
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { setSimulationSpeed, getSimulationSpeed } from '@/api/simulation'
import { getExperimentStatus, stopExperiment } from '@/api/sandbox'
import { userLogout } from '@/api/user'
import PoiList from '@/components/poiList.vue'
import VehicleList from '@/components/vehicleList.vue'
// 导入图标
import {
    MapLocation, DataAnalysis, List,
    Van, Location, VideoPlay, VideoPause,
    SwitchButton, Odometer, ArrowDown, Collection, Clock, CloseBold
} from '@element-plus/icons-vue'
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()
const target = useTargetStore()
const visible = useVisibleStore()
const userStore = useUserStore()
const dataPanel = useDataPanelStore()
const imform = useImformStore()
const drawerVisible = ref(false)
const vehicleDrawerVisible = ref(false)

// ---- “显示数据”开关：信息面板打开时先收回它，再拉入数据面板 ----
// 互斥方向一：数据面板已开 → 点击车辆/POI 时 mapDataOverlay 自动渐出、infoPanel 延迟进入；
// 互斥方向二：infoPanel 已开 → 点“显示数据”先收回 infoPanel，稍候再渐进拉入数据整板。
let pendingShowPanels = null
const handleToggleData = () => {
    // 300ms 内连点：取消尚未落地的“待显示”，视为关闭
    if (pendingShowPanels) {
        clearTimeout(pendingShowPanels)
        pendingShowPanels = null
        return
    }
    if (dataPanel.showDataPanels) {
        dataPanel.setDataPanels(false) // 关闭：整板直接渐出
        return
    }
    if (imform.imformIf) {
        imform.imformHide() // 先收回车辆/POI 信息面板
        pendingShowPanels = setTimeout(() => {
            pendingShowPanels = null
            dataPanel.setDataPanels(true) // 信息面板退出后再拉入数据面板
        }, 300)
    } else {
        dataPanel.setDataPanels(true)
    }
}
onUnmounted(() => { if (pendingShowPanels) { clearTimeout(pendingShowPanels); pendingShowPanels = null } })

const openDrawer = () => drawerVisible.value = true
const openVehicleDrawer = () => vehicleDrawerVisible.value = true

const mapAnimationStore = useMapAnimationStore()
const { isPollingActive } = storeToRefs(mapAnimationStore)

// 当前是否处于实验视图（三屏页）
const experimentId = computed(() => {
    const m = route.path.match(/^\/admin\/experiment\/(\d+)$/)
    return m ? Number(m[1]) : 0
})
const inExperiment = computed(() => experimentId.value > 0)

// ---- 实验状态条：虚拟时钟/进度（每5s轮询实验状态） ----
const expStatus = ref({ virtualTime: '', progress: 0, status: '' })
let statusTimer = null
const pollExperimentStatus = async () => {
    if (!experimentId.value) return
    const requestedId = experimentId.value
    try {
        const res = await getExperimentStatus(requestedId)
        // 丢弃过期响应：请求期间路由已离开实验页（如人工退出），避免旧结果触发二次跳转
        if (!experimentId.value || experimentId.value !== requestedId) return
        if (res.data.code === 1) expStatus.value = res.data.data
        // 实验已结束（自然完成/中止）→ 提示并跳结果页或返回沙箱
        if (['finished', 'aborted', 'failed'].includes(res.data.data.status)) {
            stopStatusTimer()
            if (res.data.data.status === 'finished') {
                ElMessage.success('模拟周期已完成，正在跳转结果页')
                router.push(`/admin/experiment/${requestedId}/result`)
            } else {
                ElMessage.warning('实验已结束（中止/失败）')
                goSandboxPage()
            }
        }
    } catch (e) { /* 忽略瞬时错误 */ }
}
const stopStatusTimer = () => { if (statusTimer) { clearInterval(statusTimer); statusTimer = null } }
watch(experimentId, (id) => {
    stopStatusTimer()
    if (id > 0) {
        pollExperimentStatus()
        statusTimer = setInterval(pollExperimentStatus, 5000)
    }
}, { immediate: true })
onUnmounted(stopStatusTimer)

// 结束并退出实验（人工中止 → 回沙箱详情页）
const handleExitExperiment = async () => {
    try {
        await ElMessageBox.confirm('确定要中止当前实验并退出吗？（数据将标记为中止，不生成终态结果）', '结束实验', {
            confirmButtonText: '中止并退出', cancelButtonText: '继续模拟', type: 'warning',
        })
    } catch { return }
    // 关键：先停状态轮询，再做中止——否则中止后在途的轮询响应会触发额外跳转，
    // 与本次 push 形成乒乓（实验页重挂载再弹回沙箱页），曾导致沙箱页请求高峰期瞬时失败而整页空白
    stopStatusTimer()
    try {
        await stopExperiment(experimentId.value)
        ElMessage.info('实验已中止')
    } catch (e) { /* 可能已自然完成 */ }
    goSandboxPage()
}
// 从实验详情里拿不到 sandboxId 时回沙箱首页兜底
const routeSandboxId = ref('')
// 确定性回沙箱：优先沙箱详情页，缺失时回沙箱列表页（不用 router.back，避免历史栈不确定）
const goSandboxPage = () => {
    if (routeSandboxId.value) router.push(`/admin/sandbox/${routeSandboxId.value}`)
    else router.push('/admin/sandboxes')
}
import { getExperiment } from '@/api/sandbox'
watch(experimentId, async (id) => {
    if (id > 0) {
        try {
            const res = await getExperiment(id)
            if (res.data.code === 1) routeSandboxId.value = res.data.data.experiment.sandboxId
        } catch (e) { }
    }
}, { immediate: true })

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

// 真实退出登录：调后端注销 + 清本地 token + 回登录页
const handleLogout = async () => {
    try { await userLogout() } catch (e) { }
    userStore.removeToken()
    userStore.clearUserInfo()
    router.push('/login')
}
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

        <!-- 2. 主导航区域（按场景切换：沙箱管理侧 / 实验运行侧） -->
        <div class="nav-main" v-if="!inExperiment">
            <div class="nav-item" :class="{ active: route.path.startsWith('/admin/sandbox') }" @click="router.push('/admin/sandboxes')">
                <el-icon>
                    <Collection />
                </el-icon>
                <span class="nav-text">沙箱管理</span>
            </div>
        </div>
        <div class="nav-main" v-else>
            <div class="nav-item" :class="{ active: visible.isFirstVisible }" @click="target.targetChange('first')">
                <el-icon>
                    <MapLocation />
                </el-icon>
                <span class="nav-text">地图视图</span>
            </div>
            <div class="nav-item" :class="{ active: visible.isSecondVisible }" @click="target.targetChange('second')">
                <el-icon>
                    <List />
                </el-icon>
                <span class="nav-text">任务管理</span>
            </div>
            <!-- 数据面板开关：凹字形面板从地图四周渐入/渐出（与车辆/POI信息面板互斥） -->
            <div class="nav-item" :class="{ active: dataPanel.showDataPanels }" @click="handleToggleData()">
                <el-icon>
                    <DataAnalysis />
                </el-icon>
                <span class="nav-text">{{ dataPanel.showDataPanels ? '隐藏数据' : '显示数据' }}</span>
            </div>
            <!-- 实验状态条：虚拟时钟 + 进度 -->
            <div class="exp-status" v-if="expStatus.virtualTime">
                <el-icon><Clock /></el-icon>
                <span class="exp-vt">{{ expStatus.virtualTime }}</span>
                <el-progress :percentage="Math.round((expStatus.progress || 0) * 100)" :stroke-width="8" style="width: 120px" />
            </div>
        </div>

        <!-- 3. 数据列表入口 (小按钮组；仅实验运行视图显示) -->
        <div class="data-group" v-if="inExperiment">
            <el-button-group>
                <el-button @click="openVehicleDrawer" :icon="Van">
                    <span class="btn-text">车辆</span>
                </el-button>
                <el-button @click="openDrawer" :icon="Location">
                    <span class="btn-text">POI</span>
                </el-button>
            </el-button-group>
        </div>

        <!-- 4. 仿真控制区 (带背景的集成区域；仅实验运行视图显示) -->
        <div class="simulation-panel" v-if="inExperiment">
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

            <!-- 实验视图：结束并退出 -->
            <div class="exit-exp" v-if="inExperiment">
                <el-button type="danger" plain size="small" :icon="CloseBold" @click="handleExitExperiment">
                    <span class="btn-text">结束实验</span>
                </el-button>
            </div>
        </div>

        <!-- 5. 用户退出（真实登录：显示当前用户，点击注销） -->
        <div class="user-section">
            <span class="user-name">{{ userStore.displayName || userStore.username }}</span>
            <a class="exit-btn" @click="handleLogout">
                <el-icon>
                    <SwitchButton />
                </el-icon>
                <span>退出登录</span>
            </a>
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

.exp-status {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-left: 24px;
    color: #606266;
    font-size: 13px;
}
.exp-vt { font-family: monospace; }
.exit-exp { margin-left: 16px; }
.user-section {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-left: auto;
}
.user-name { color: #303133; font-size: 14px; font-weight: 500; }
.exit-btn {
    display: flex;
    align-items: center;
    gap: 4px;
    color: #606266;
    cursor: pointer;
    text-decoration: none;
}
.exit-btn:hover { color: #f56c6c; }
</style>