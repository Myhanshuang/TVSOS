<!-- 订单管理组件：用于查询、筛选及展示物流订单的详细信息 -->
<script setup>
import { onMounted, ref } from 'vue'
import { getShipmentList } from '@/api/shipment'
import dayjs from 'dayjs'

/** 搜索及数据响应式变量 */
const status = ref()        // 筛选状态：1-6 的整数值
const numTask = ref()       // 搜索订单号
const taskData = ref([])    // 表格展示的订单列表数据
const tableKey = ref(0)     // 表格更新标识符，用于强制重绘表格

/** 
 * 订单状态配置映射表
 * 将后端返回的数字状态转换为对应的中文文本及 Element Plus 标签类型 
 */
const statusMap = {
  1: { text: '待调度', type: 'info' },
  2: { text: '待发车', type: 'warning' },
  3: { text: '等待中', type: 'warning' },
  4: { text: '运输中', type: '' },
  5: { text: '已完成', type: 'success' },
  6: { text: '异常', type: 'danger' }
}

/**
 * 时间格式化工具函数
 * @param {string} timeStr - 原始时间字符串
 * @returns {string} 格式化后的时间 YYYY-MM-DD HH:mm:ss
 */
const formatTime = (timeStr) => {
  if (!timeStr) return '--'
  return dayjs(timeStr).format('YYYY-MM-DD HH:mm:ss')
}

/**
 * 核心方法：重载或获取订单列表
 * 根据当前选择的订单号和状态发起 API 请求
 */
async function reloadShipmentList() {
  const res = await getShipmentList({ num: numTask.value, status: status.value })
  // 确保列表为数组格式
  const list = Array.isArray(res?.data?.data) ? res.data.data : []
  taskData.value = [...list]
  // 变更 key 会触发 el-table 的完整渲染生命周期
  tableKey.value++
}

/** 生命周期：组件首次挂载后加载初始数据 */
onMounted(() => {
  reloadShipmentList()
})

</script>

<template>
  <div id="thiBorder">
    <!-- 主体卡片容器 -->
    <div class="box">
      <!-- 搜索筛选栏 -->
      <div class="selectBox">
        <span style="margin-left: 15px; font-size: 14px;">订单号</span>
        <el-input v-model="numTask" placeholder="请输入订单号" class="numInput" />

        <span style="margin-left: 15px; font-size: 14px;">状态筛选</span>
        <el-select placeholder="请选择" class="statusSelect" v-model="status">
          <el-option label="全部" value=""></el-option>
          <el-option label="待调度" :value="1"></el-option>
          <el-option label="待发车" :value="2"></el-option>
          <el-option label="等待中" :value="3"></el-option>
          <el-option label="运输中" :value="4"></el-option>
          <el-option label="已完成" :value="5"></el-option>
          <el-option label="异常" :value="6"></el-option>
        </el-select>

        <el-button type="primary" @click="reloadShipmentList" class="reloadBox">查询</el-button>
      </div>

      <!-- 订单数据展示表格 -->
      <el-table :data="taskData" class="elBox" stripe width="100%" :key="tableKey" row-key="num">
        <!-- 订单编号列 -->
        <el-table-column prop="num" label="订单编号" width="90">
        </el-table-column>

        <!-- 货源地坐标组（多级表头） -->
        <el-table-column label="货源地" width="180">
          <el-table-column prop="beginLon" label="经度" width="120">
          </el-table-column>
          <el-table-column prop="beginLat" label="纬度" width="120">
          </el-table-column>
        </el-table-column>

        <!-- 目的地坐标组（多级表头） -->
        <el-table-column label="目的地">
          <el-table-column prop="endLon" label="经度" width="120">
          </el-table-column>
          <el-table-column prop="endLat" label="纬度" width="120">
          </el-table-column>
        </el-table-column>

        <!-- 时间类列：均通过 formatTime 进行格式化 -->
        <el-table-column prop="estBeginTime" label="预计出发时间按">
          <template #default="scope">
            {{ formatTime(scope.row.estBeginTime) }}
          </template>
        </el-table-column>

        <el-table-column prop="estEndTime" label="预计结束时间按">
          <template #default="scope">
            {{ formatTime(scope.row.estEndTime) }}
          </template>
        </el-table-column>

        <el-table-column prop="createTime" label="订单创建时间">
          <template #default="scope">
            {{ formatTime(scope.row.createTime) }}
          </template>
        </el-table-column>

        <!-- 订单状态列：固定在右侧，显示彩色标签 -->
        <el-table-column prop="status" label="订单状态" width="120" fixed="right">
          <template #default="scope">
            <el-tag :type="statusMap[scope.row.status]?.type || 'info'">
              {{ statusMap[scope.row.status]?.text || '未知状态' }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<style scoped>
/* 组件外层包围盒 */
#thiBorder {
  margin: 0px;
  padding: 0px;
  width: 99vw;
  height: 88vh;
  font-family: 'Segoe UI', 'Arial', 'PingFang SC', 'Hiragino Sans GB', '微软雅黑', sans-serif;
  color: #2c3e50;
}

/* 主展示区域：白色卡片样式 */
.box {
  display: inline-block;
  width: calc(100% - 400px);
  height: calc(100% - 60px);
  margin: 0px 180px;
  padding: 30px 20px;
  font-family: inherit;
  border-radius: 12px;
  background: #ffffff;
  box-shadow: 7px 7px 10px #bebebe,
    -7px -7px 10px #ffffff;
}

/* 筛选表单行布局 */
.selectBox {
  display: inline-block;
  width: 100%;
  height: 50px;
  margin-bottom: 10px;
}

/* 表格主体高度控制 */
.elBox {
  display: inline-block;
  width: 100%;
  height: calc(100% - 60px);
}

/* 订单号输入框尺寸 */
.numInput {
  display: inline-block;
  width: 200px;
  height: 100%;
  margin-left: 20px;
  padding: 5px;
  font-size: 14px;
}

/* 状态选择框尺寸 */
.statusSelect {
  display: inline-block;
  width: 100px;
  height: 100%;
  margin-left: 20px;
  margin-top: 10px;
  font-size: 14px;
}

/* 查询按钮间距 */
.reloadBox {
  display: inline-block;
  margin-left: 20px;
  margin-bottom: 5px;
  font-size: 14px;
}

/* 预留图表样式 */
#barChart {
  width: 500px;
  height: 300px;
}
</style>