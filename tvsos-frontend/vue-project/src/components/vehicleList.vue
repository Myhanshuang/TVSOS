<!-- src/components/VehicleList.vue -->
<template>
  <el-drawer
    title="货车列表"
    direction="rtl"
    size="50%"
    :model-value="visible"
    @update:model-value="$emit('update:visible', $event)"
  >
    <!-- 查询栏 -->
    <el-form inline>
      <el-form-item label="车辆状态">
        <el-select
          placeholder="请选择"
          style="width: 120px"
          v-model="selectedStatus"
        >
          <el-option label="全部" value=""></el-option>
          <el-option label="停运" :value="0"></el-option>
          <el-option label="待命" :value="1"></el-option>
          <el-option label="行驶中" :value="2"></el-option>
        </el-select>
      </el-form-item>
      <el-form-item label="车牌号">
        <el-input
          v-model="licenseKeyword"
          placeholder="请输入车牌号"
          style="width: 120px"
        />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="handleSearch">搜索</el-button>
        <el-button @click="handleReset">重置</el-button>
      </el-form-item>
    </el-form>

    <!-- 表格 -->
    <el-table
      :data="pagedVehicleList"
      @row-click="handleRowClick"
      style="margin-top: 20px;"
    >
      <el-table-column label="车牌号" prop="license">
        <template #default="{ row }">
          <el-link type="primary" :underline="false">{{ row.license }}</el-link>
        </template>
      </el-table-column>
      <el-table-column label="车辆状态" prop="status">
        <template #default="{ row }">
          {{ getStatusText(row.status) }}
        </template>
      </el-table-column>
      <!-- 可以根据需要添加更多列，例如速度 -->
      <el-table-column label="速度 (km/h)" prop="speed" align="center">
         <template #default="{ row }">
          {{ row.speed !== undefined ? row.speed.toFixed(1) : '-' }}
        </template>
      </el-table-column>
    </el-table>

    <!-- 分页区域 -->
    <el-pagination
      v-model:current-page="currentPage"
      v-model:page-size="pageSize"
      :total="filteredVehicleList.length"
      :page-sizes="[10, 20, 30, 40]"
      layout="total, sizes, prev, pager, next, jumper"
      @size-change="handleSizeChange"
      @current-change="handleCurrentChange"
      style="margin-top: 20px; justify-content: end;"
    />
  </el-drawer>
</template>

<script setup>
import { ref, computed, defineProps, defineEmits } from 'vue';
import { useMapStore } from '@/stores'; // 用于地图定位
import { useVehicleStore } from '@/stores'; // 导入 useVehicleStore

// 接收父组件传入的 visible 值
const props = defineProps({
  visible: {
    type: Boolean,
    default: false,
  },
});

// 定义 emit 事件，用于控制抽屉关闭
const emit = defineEmits(['update:visible']);

// 获取 mapStore，用于控制地图中心点
const mapStore = useMapStore();
// 获取 vehicleStore，用于获取车辆数据
const vehicleStore = useVehicleStore();

// 筛选条件
const selectedStatus = ref(''); // 状态筛选
const licenseKeyword = ref(''); // 车牌号搜索关键词

// 当前页码和每页显示条数
const currentPage = ref(1);
const pageSize = ref(10);

// 计算属性：根据筛选条件过滤车辆列表
const filteredVehicleList = computed(() => {
  return vehicleStore.vehicleList.filter(vehicle => {
    if (selectedStatus.value !== '' && vehicle.status !== Number(selectedStatus.value)) {
      return false;
    }
    if (licenseKeyword.value && !vehicle.license.includes(licenseKeyword.value)) {
      return false;
    }
    return true;
  });
});

// 分页后的车辆列表
const pagedVehicleList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value;
  const end = start + pageSize.value;
  return filteredVehicleList.value.slice(start, end);
});

// 状态映射
const statusMap = {
  0: '停运',
  1: '待命',
  2: '行驶中',
};

// 获取状态文本
const getStatusText = (status) => {
  return statusMap[status] || '未知状态';
};

// 搜索处理 (过滤逻辑在 computed 中)
const handleSearch = () => {
  console.log('搜索条件:', { status: selectedStatus.value, keyword: licenseKeyword.value });
};

// 重置处理
const handleReset = () => {
  selectedStatus.value = '';
  licenseKeyword.value = '';
  currentPage.value = 1; // 重置当前页码
  console.log('重置筛选条件');
};

// 处理页面大小变化
const handleSizeChange = (newSize) => {
  pageSize.value = newSize;
  currentPage.value = 1; // 改变页面大小时重置当前页码
};

// 处理当前页码变化
const handleCurrentChange = (newPage) => {
  currentPage.value = newPage;
};

// 表格行点击事件
const handleRowClick = (row) => {
  if (row.currentPosition && Array.isArray(row.currentPosition) && row.currentPosition.length === 2) {
    // 设置地图中心点到该车辆的当前位置
    mapStore.setCenter(row.currentPosition);
    // 设置缩放级别（可选）
    mapStore.setZoom(16);
    // 关闭抽屉
    emit('update:visible', false);
  } else {
    console.warn(`车辆 ${row.license} 的位置信息不完整，无法定位。`);
  }
};
</script>

<style scoped>
/* 可以根据需要添加一些样式 */
.el-form {
  padding: 0 20px;
}
.el-table {
  padding: 0 20px;
}
</style>