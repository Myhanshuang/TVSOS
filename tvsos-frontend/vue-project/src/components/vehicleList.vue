<!-- 货车列表抽屉组件：提供车辆的多条件查询、分页展示及地图点击定位功能 -->
<template>
  <!-- Element Plus 抽屉：展示在屏幕右侧，占用 50% 宽度 -->
  <el-drawer title="货车列表" direction="rtl" size="50%" :model-value="visible"
    @update:model-value="$emit('update:visible', $event)">

    <!-- 顶部筛选表单 -->
    <el-form inline>
      <!-- 车辆状态下拉筛选 -->
      <el-form-item label="车辆状态">
        <el-select placeholder="请选择" style="width: 120px" v-model="selectedStatus">
          <el-option label="全部" value=""></el-option>
            <el-option label="行驶中" :value="1"></el-option>
            <el-option label="空闲" :value="2"></el-option>
            <el-option label="待发车" :value="3"></el-option>
        </el-select>
      </el-form-item>

      <!-- 车牌号模糊搜索 -->
      <el-form-item label="车牌号">
        <el-input v-model="licenseKeyword" placeholder="请输入车牌号" style="width: 120px" />
      </el-form-item>

      <!-- 操作按钮集 -->
      <el-form-item>
        <el-button type="primary" @click="handleSearch">搜索</el-button>
        <el-button @click="handleReset">重置</el-button>
      </el-form-item>
    </el-form>

    <!-- 车辆数据表格 -->
    <el-table :data="pagedVehicleList" @row-click="handleRowClick" style="margin-top: 20px;">
      <!-- 车牌号列：渲染为链接样式 -->
      <el-table-column label="车牌号" prop="license">
        <template #default="{ row }">
          <el-link type="primary" :underline="false">{{ row.license || ('车辆-' + row.id) }}</el-link>
        </template>
      </el-table-column>

      <!-- 车辆状态列：调用翻译函数显示文本 -->
      <el-table-column label="车辆状态" prop="status">
        <template #default="{ row }">
          {{ getStatusText(row.status) }}
        </template>
      </el-table-column>

      <!-- 速度列：保留 1 位小数 -->
      <el-table-column label="速度 (km/h)" prop="speed" align="center">
        <template #default="{ row }">
          {{ row.speed !== undefined ? row.speed.toFixed(1) : '-' }}
        </template>
      </el-table-column>
    </el-table>

    <!-- 分页器：基于计算后的过滤列表长度进行分页 -->
    <el-pagination :current-page="currentPage" :page-size="pageSize" :total="filteredVehicleList.length"
      :page-sizes="[10, 20, 30, 40]" layout="total, sizes, prev, pager, next, jumper" @size-change="handleSizeChange"
      @current-change="handleCurrentChange" style="margin-top: 20px; justify-content: end;" />
  </el-drawer>
</template>

<script setup>
import { ref, computed, defineProps, defineEmits } from 'vue';
import { useMapStore } from '@/stores';
import { useVehicleStore } from '@/stores';

/**
 * 定义组件属性
 * @property {boolean} visible - 控制抽屉显示的开关
 */
const props = defineProps({
  visible: {
    type: Boolean,
    default: false,
  },
});

/** 定义事件发送器 */
const emit = defineEmits(['update:visible']);

/** 状态库实例化 */
const mapStore = useMapStore();      // 用于控制地图定位
const vehicleStore = useVehicleStore(); // 用于获取实时车辆列表数据

/** 本地交互响应式变量 */
const selectedStatus = ref('');     // 当前选中的状态过滤值
const licenseKeyword = ref('');     // 当前输入的车牌搜索关键字
const currentPage = ref(1);         // 当前页码
const pageSize = ref(10);           // 每页显示条数

/**
 * 计算属性：根据筛选条件对 Store 中的全量车辆数据进行过滤
 */
const filteredVehicleList = computed(() => {
  return vehicleStore.vehicleList.filter(vehicle => {
    // 状态过滤逻辑
    if (selectedStatus.value !== '' && vehicle.status !== Number(selectedStatus.value)) {
      return false;
    }
    // 车牌模糊匹配逻辑
    if (licenseKeyword.value) {
      const displayLicense = vehicle.license || `车辆-${vehicle.id}`;
      if (!displayLicense.includes(licenseKeyword.value)) {
        return false;
      }
    }
    return true;
  });
});

/**
 * 计算属性：对过滤后的结果进行前端分页切片
 */
const pagedVehicleList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value;
  const end = start + pageSize.value;
  return filteredVehicleList.value.slice(start, end);
});

/** 状态码到中文的简易映射（注：此处映射与 script 内逻辑略有差异，实际以业务需求为准） */
const statusMap = {
  1: '行驶中',
  2: '空闲',
  3: '待发车',
};

/**
 * 获取状态描述文本
 * @param {number} status 状态码
 */
const getStatusText = (status) => {
  return statusMap[status] || '未知状态';
};

/** 执行搜索（目前主要用于触发逻辑统计或日志） */
const handleSearch = () => {
  console.log('搜索条件:', { status: selectedStatus.value, keyword: licenseKeyword.value });
};

/** 重置搜索条件并跳回第一页 */
const handleReset = () => {
  selectedStatus.value = '';
  licenseKeyword.value = '';
  currentPage.value = 1;
  console.log('重置筛选条件');
};

/** 处理每页条数变化 */
const handleSizeChange = (newSize) => {
  pageSize.value = newSize;
  currentPage.value = 1;
};

/** 处理页码切换 */
const handleCurrentChange = (newPage) => {
  currentPage.value = newPage;
};

/**
 * 列表行点击事件处理
 * 功能：将地图中心平滑移至选中车辆的当前位置，并关闭列表抽屉
 * @param {Object} row 点击的车辆数据对象
 */
const handleRowClick = (row) => {
  if (row.currentPosition && Array.isArray(row.currentPosition) && row.currentPosition.length === 2) {
    // 同步经纬度到地图 Store
    mapStore.setCenter(row.currentPosition);
    // 设置地图缩放系数至较细颗粒度
    mapStore.setZoom(16);
    // 关闭抽屉
    emit('update:visible', false);
  } else {
    console.warn(`车辆 ${row.license} 的位置信息不完整，无法定位。`);
  }
};
</script>

<style scoped>
/* 调整表单和表格的侧向边距，确保内容不贴边 */
.el-form {
  padding: 0 20px;
}

.el-table {
  padding: 0 20px;
}
</style>