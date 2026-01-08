<script setup>
import { onMounted, ref, watch } from 'vue'
import { getShipmentList } from '@/api/shipment'
import dayjs from 'dayjs'

const numTask = ref()
const taskData = ref([])
const status = ref()

// taskData 的 status 字段映射
const statusMap = {
  1: { text: '待拆分', type: 'info' },
  2: { text: '待调度', type: 'warning' },
  3: { text: '运输中', type: '' }, 
  4: { text: '完成', type: 'success' },
  5: { text: '取消', type: 'danger' },
  6: { text: '异常', type: 'danger' }
}

// 定义格式化函数
const formatTime = (timeStr) => {
  if (!timeStr) return '--'
  return dayjs(timeStr).format('YYYY-MM-DD HH:mm:ss')
}

function reloadShipmentList(){
  getShipmentList({ num: status.value, status: status.value }).then((res) => {
    if (res.data?.code === 1 && res.data.data?.length) {
      taskData.value = res.data?.data
      console.log("获取任务:", taskData.value.length);
    } else {
      console.log("获取任务失败");
    }
    });
}

onMounted(() => {
  reloadShipmentList()
})


</script>

<template>
  <div id="thiBorder">
    <div class="box">
      <div class="selectBox">
        <span style="margin-left: 15px; font-size: 14px;">订单号</span>
        <el-input
          v-model="numTask"
          placeholder="请输入订单号"
          class="numInput"
        />
        <span style="margin-left: 15px; font-size: 14px;">状态筛选</span>
        <el-select
          placeholder="请选择"
          class="statusSelect"
          v-model="status"
        >
          <el-option label="全部" value=""></el-option>
          <el-option label="待拆分" :value="1"></el-option>
          <el-option label="待调度" :value="2"></el-option>
          <el-option label="运输中" :value="3"></el-option>
          <el-option label="完成" :value="4"></el-option>
          <el-option label="取消" :value="5"></el-option>
          <el-option label="异常" :value="6"></el-option>
        </el-select>
        <el-button type="primary" @click="reloadShipmentList" class="reloadBox">查询</el-button>
      </div>
      <el-table :data="taskData" class="elBox" stripe="true" width="100%">
        <el-table-column prop="num" label="订单编号" width="90">
        </el-table-column>
        <el-table-column label="货源地" width="180">
          <el-table-column prop="beginLon" label="经度" width="120">
          </el-table-column>
          <el-table-column prop="beginLat" label="纬度" width="120">
          </el-table-column>
        </el-table-column>
        <el-table-column label="目的地">
          <el-table-column prop="endLon" label="经度" width="120">
          </el-table-column>
          <el-table-column prop="endLat" label="纬度" width="120">
          </el-table-column>
        </el-table-column>
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
#thiBorder {
  margin: 0px;
  padding: 0px;
  width: 99vw;
  height: 88vh;
  /* border: 1px solid black; */
  font-family: 'Segoe UI', 'Arial', 'PingFang SC', 'Hiragino Sans GB', '微软雅黑', sans-serif;
  color: #2c3e50;
}

.box {
  display: inline-block;
  width: calc(100% - 400px);
  height: calc(100% - 60px);
  margin: 0px 180px;
  padding: 30px 20px ;
  font-family: inherit;
  /* border: black 1px solid; */
  border-radius: 12px;
  background: #ffffff;
box-shadow:  7px 7px 10px #bebebe,
             -7px -7px 10px #ffffff;
}

.selectBox {
  display: inline-block;
  width: 100%;
  height: 50px;
  margin-bottom: 10px;
}

.elBox {
  display: inline-block;
  width: 100%;
  height: calc(100% - 60px);
}

.numInput {
  display: inline-block;
  width: 200px;
  height: 100%;
  margin-left: 20px;
  padding: 5px;
  font-size: 14px;
}

.statusSelect {
  display: inline-block;
  width: 100px;
  height: 100%;
  margin-left: 20px;
  margin-top: 10px;
  font-size: 14px;
}

.reloadBox {
  display: inline-block;
  margin-left: 20px;
  margin-bottom: 5px;
  font-size: 14px;
}
</style>