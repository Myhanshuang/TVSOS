<script setup>
import { onMounted, ref } from 'vue'

const taskData = ref()


</script>

<template>
  <div id="thiBorder">
    <div class="box">
      <el-table :data="taskData" class="elBox" stripe="true">
        <el-table-column prop="date" label="订单编号" width="150">
        </el-table-column>
        <el-table-column prop="name" label="货物类型" width="120">
        </el-table-column>
        <el-table-column prop="name" label="货源地" width="180">
        </el-table-column>
        <el-table-column prop="name" label="目的地" width="180">
        </el-table-column>
        <el-table-column prop="name" label="订单状态" width="120">
        </el-table-column>
        <el-table-column prop="name" label="创建时间" width="180">
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

.elBox {
  display: inline-block;
  width: 100%;
  height: 100%;
}
</style>