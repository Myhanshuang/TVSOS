<script setup>
import request from '@/utils/request.js'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useMapAnimationStore, useVisibleStore, useTargetStore} from '@/stores'
import { ref } from 'vue'
import PoiList from '@/components/poiList.vue'
import VehicleList from '@/components/vehicleList.vue';
import { storeToRefs } from 'pinia';
import { setSimulationSpeed, getSimulationSpeed } from '@/api/simulation';

const target = useTargetStore()
const visible = useVisibleStore()
const drawerVisible = ref(false)
const vehicleDrawerVisible = ref(false)
const openDrawer = () => {
    drawerVisible.value = true
}

const openVehicleDrawer = () => {
    vehicleDrawerVisible.value = true;
};

const mapAnimationStore = useMapAnimationStore();
const { isPollingActive } = storeToRefs(mapAnimationStore);
const mockCount = ref(5)

const handleMockShipments = async () => {
    if (mockCount.value <= 0 || !Number.isInteger(mockCount.value)) {
        ElMessage.warning('请输入一个有效的正整数')
        return
    }

    try {
        await ElMessageBox.confirm(
            `确定要生成 ${mockCount.value} 条模拟订单数据吗？`,
            '确认操作',
            {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'warning',
            }
        )
        const res = await request.post(`/shipments/mock/${mockCount.value}`)
        if (res.data.code === 1) {
            ElMessage.success(`成功生成 ${res.data.data?.length || mockCount.value} 条订单数据`)
            console.log('生成的订单数据：', res.data.data)
        } else {
            ElMessage.error(res.data.message || 'Mock 订单失败')
        }
    } catch (err) {
        if (err && typeof err === 'object' && err.message) {
            ElMessage.error(err.message)
        } else if (err === undefined) {
        } else {
            ElMessage.error('请求失败')
        }
    }
}

const currentSpeed = ref(1.0);
const speedOptions = [1.0, 2.0, 5.0, 10.0, 20.0, 50.0];

const handleSetSpeed = async (speed) => {
    try {
        const res = await setSimulationSpeed(speed);
        if (res.data.code === 1) {
            currentSpeed.value = speed;
            ElMessage.success(`仿真倍速已设置为 ${speed}x`);
        } else {
            ElMessage.error(res.data.message || '设置失败');
        }
    } catch (error) {
        ElMessage.error('网络错误');
    }
};

const initSpeed = async () => {
    try {
        const res = await getSimulationSpeed();
        if (res.data.code === 1) {
            currentSpeed.value = res.data.data;
        }
    } catch (error) {
        console.warn('获取仿真倍速失败');
    }
};

import { onMounted } from 'vue';
onMounted(() => {
    initSpeed();
});
</script>

<template>
    <nav>
        <div class="title">TVSOS</div>
        <div class="space"></div>
        <el-button @click="target.targetChange('first')"
            :class="{ scollButton: 1, scollButtonActive: visible.isFirstVisible }">地图</el-button>
        <div class="space"></div>
        <el-button @click="target.targetChange('second')"
            :class="{ scollButton: 1, scollButtonActive: visible.isSecondVisible }">统计</el-button>
        <div class="space"></div>
        <el-button @click="target.targetChange('third')"
            :class="{ scollButton: 1, scollButtonActive: visible.isThirdVisible }">任务管理</el-button>
        <el-button @click="openVehicleDrawer">货车列表</el-button>
        <el-button type="primary" @click="openDrawer"> poi列表</el-button>
        <div class="longSpace"></div>

        <div class="speed-control" style="margin-right: 20px;">
            <span style="margin-right: 8px; font-size: 14px; color: #666;">倍速:</span>
            <el-radio-group v-model="currentSpeed" size="small" @change="handleSetSpeed">
                <el-radio-button v-for="speed in speedOptions" :key="speed" :label="speed">{{ speed
                    }}x</el-radio-button>
            </el-radio-group>
        </div>

        <el-button @click="mapAnimationStore.startPolling" :disabled="isPollingActive">开始动画</el-button>
        <el-button @click="mapAnimationStore.pausePolling" :disabled="!isPollingActive">暂停动画</el-button>
        <div class="mock-section">
            <el-input-number v-model="mockCount" :min="1" size="small"
                style="width: 100px; margin-right: 10px;"></el-input-number>
            <el-button type="warning" @click="handleMockShipments">Mock 订单</el-button>
        </div>
        <div class="loginOutBox">
            <router-link to="/login" class="loginOut">退出登录</router-link>
        </div>
    </nav>
    <poi-list :visible="drawerVisible" @update:visible="drawerVisible = $event" />
    <vehicle-list :visible="vehicleDrawerVisible" @update:visible="vehicleDrawerVisible = $event" />
    <RouterView></RouterView>
</template>

<style scoped>
.mock-section {
    display: flex;
    align-items: center;
    margin: 0 20px;
}

nav {
    position: fixed;
    top: 0;
    margin: 0px;
    padding: 0px;
    height: 60px;
    width: 100%;
    z-index: 1000;
    background: #ffffff;
    display: flex;
    align-items: center;
    justify-items: center;
    box-shadow: 5px 5px 6px #a8a8a8,
        -5px -5px 6px #ffffff;
}

.title {
    display: flex;
    align-items: center;
    justify-items: center;
    height: 100%;
    width: auto;
    padding: 0px 20px;
    font-size: 20px;
    font-family: 'Courier New', Courier, monospace;
}

.scollButton {
    height: 100%;
    width: auto;
    border: 0;
    padding: 0px 20px;
    margin: 0px 0px 0px 20px;
    position: relative;
    font-size: 18px;
}

.scollButton:hover {
    background-color: #ffffff;
    color: rgb(126, 173, 249);
}

.scollButton::after {
    content: " ";
    position: absolute;
    display: inline-block;
    width: 0px;
    height: 3px;
    background-color: rgb(126, 173, 249);
    bottom: 13px;
    transition: all 0.3s;
}

.scollButton:hover::after {
    width: 55%;
}

.scollButtonActive {
    background-color: #ffffff;
    color: rgb(126, 173, 249);
}


.scollButtonActive::after {
    width: 50%;
}

.space {
    display: inline-block;
    height: 100%;
    flex: 1;
}

.longSpace {
    display: inline-block;
    height: 100%;
    flex: 100;
}

.loginOutBox {
    display: flex;
    align-items: center;
    justify-items: center;
    height: 100%;
    width: auto;
    padding: 0px 20px;
}

.loginOut {
    color: rgb(97, 98, 102);
    font-size: 18px;
    font-family: 'Franklin Gothic Medium', 'Arial Narrow', Arial, sans-serif;
    text-decoration: none;
    transition: all 0.5s;
}

.loginOut:hover {
    color: rgb(255, 0, 0);
}
</style>