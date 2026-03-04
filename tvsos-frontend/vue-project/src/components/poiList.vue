<!-- POI 列表抽屉组件：提供 POI 的筛选、列表展示及地图联动功能 -->
<template>
  <!-- Element Plus 抽屉组件：从右侧滑出，占比 50% -->
  <el-drawer title="poi列表" direction="rtl" size="50%" :model-value="visible"
    @update:model-value="$emit('update:visible', $event)">

    <!-- 筛选表单区域 -->
    <el-form inline>
      <!-- 状态筛选下拉框 -->
      <el-form-item label="状态">
        <el-select placeholder="请选择" style="width: 100px" v-model="selectedStatus">
          <el-option label="关闭" value="0"></el-option>
          <el-option label="正常" value="1"></el-option>
        </el-select>
      </el-form-item>
      <!-- 类型筛选下拉框 -->
      <el-form-item label="类型">
        <el-select style="width: 100px" placeholder="请选择" v-model="selectedType">
          <el-option label="加油站" value="加油站"></el-option>
          <el-option label="加气站" value="加气站"></el-option>
          <el-option label="其它能源站" value="其它能源站"></el-option>
          <el-option label="工厂" value="工厂"></el-option>
          <el-option label="公司企业" value="公司企业"></el-option>
          <el-option label="购物中心" value="购物中心"></el-option>
          <el-option label="家具建材市场" value="家具建材市场"></el-option>
        </el-select>
      </el-form-item>
      <!-- 操作按钮 -->
      <el-form-item>
        <el-button type="primary" @click="handleSearch">搜索</el-button>
        <el-button @click="handleReset">重置</el-button>
      </el-form-item>
    </el-form>

    <!-- POI 数据表格 -->
    <el-table :data="poiList" @row-click="handleRowClick">
      <!-- 名称列：点击可跳转 -->
      <el-table-column label="名字" prop="name">
        <template #default="{ row }">
          <el-link type="primary" :underline="false">{{ row.name }}</el-link>
        </template>
      </el-table-column>

      <!-- 类型列：根据 ID 映射中文标签 -->
      <el-table-column label="类型" prop="tybe">
        <template #default="{ row }">
          <el-tag>{{ typeMap[row.tybe] || '未知类型' }}</el-tag>
        </template>
      </el-table-column>

      <!-- 状态列：动态颜色标签 -->
      <el-table-column label="状态" prop="status">
        <template #default="{ row }">
          <el-tag :type="row.status === 0 ? 'success' : 'danger'">
            {{ statusMap[row.status] || '未知' }}
          </el-tag>
        </template>
      </el-table-column>
    </el-table>

    <!-- 分页器：前端模拟分页控制 -->
    <el-pagination v-model:current-page="params.pagenum" v-model:page-size="params.pagesize" :page-sizes="[2, 3, 5, 10]"
      :background="true" layout="jumper,total, sizes,prev, pager, next" :total="total" pager-count="4"
      @size-change="onSizeChange" @current-change="onCurrentChange" style="margin-top: 20px; justify-content: end;" />

  </el-drawer>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useMapStore, useImformStore, usePoiBoxStore } from '@/stores'
import request from '@/utils/request.js'

/** 实例化相关 Store */
const poiBox = usePoiBoxStore()
const imform = useImformStore()

/** 定义 Props：控制显示隐藏 */
defineProps({
  visible: {
    type: Boolean,
    default: false
  }
})

/** 定义自定义事件：同步显隐状态 */
const emit = defineEmits(['update:visible'])

const mapStore = useMapStore()

/** 
 * 处理行点击事件：地图位置联动 
 * 1. 移动地图中心 2. 设置缩放 3. 触发点位闪烁 4. 打开右侧详情面板 5. 关闭当前列表抽屉
 */
const handleRowClick = (row) => {
  mapStore.setCenter([row.lon, row.lat])
  mapStore.setZoom(15)
  mapStore.setBlinkingPoi(row)
  poiBox.recentPoiChange(row)
  imform.imformShow("poi", null)
  emit('update:visible', false)
}

/** 响应式筛选条件 */
const selectedStatus = ref('')
const selectedType = ref('')

/** 表单显示数据及分页参数 */
const poiList = ref([])
const params = ref({
  pagenum: 1,
  pagesize: 10
})
const total = ref(0) // 总数，用于计算分页

/** POI 类型码映射表 */
const typeMap = {
  1: '加油站',
  2: '加气站',
  3: '其它能源站',
  4: '工厂',
  5: '公司企业',
  6: '购物中心',
  7: '家具建材市场'
}

/** POI 状态码映射表 */
const statusMap = {
  0: '关闭',
  1: '正常',
}

/** 组件挂载后初始化加载数据 */
onMounted(() => {
  loadData()
})

/** 分页：每页条数改变回调 */
const onSizeChange = (newSize) => {
  params.value.pagesize = newSize
  params.value.pagenum = 1
  loadData()
}

/** 重置筛选条件及分页状态 */
const handleReset = () => {
  selectedStatus.value = ''
  selectedType.value = ''

  params.value.pagenum = 1
  params.value.pagesize = 10

  loadData()
  console.log('重置筛选条件')
}

/** 分页：页码改变回调 */
const onCurrentChange = (newPage) => {
  params.value.pagenum = newPage
  loadData()
}

/** 执行搜索逻辑 */
const handleSearch = () => {
  params.value.pagenum = 1 // 搜索需切回第一页
  loadData()
}

/**
 * 加载数据主方法
 * 包含：中文类型转 ID、后端全量请求、前端逻辑切片分页
 */
const loadData = async () => {
  // 1. 类型预处理：根据选中的中文找 ID
  let typeId = null
  if (selectedType.value) {
    typeId = Object.keys(typeMap).find(key => typeMap[key] === selectedType.value)
  }

  try {
    // 2. 发起 Axios GET 请求
    const res = await request.get('/poi', {
      params: {
        status: selectedStatus.value === '' ? null : selectedStatus.value,
        tybe: typeId // 注：后端字段名为 tybe
      }
    })

    // 3. 处理响应数据
    if (res.data.code === 1 || res.data.code === 0 || res.data.code === 200) {

      const allData = res.data.data || []

      // --- 模拟前端分页逻辑实现 ---

      // A. 更新总记录数 (Element Pagination 依赖此值计算页数)
      total.value = allData.length

      // B. 计算截取碎片的起始与结束索引
      const startIndex = (params.value.pagenum - 1) * params.value.pagesize
      const endIndex = startIndex + params.value.pagesize

      // C. 使用 slice 截取当前页需要显示的子集
      poiList.value = allData.slice(startIndex, endIndex)

    } else {
      console.error('获取POI列表失败:', res.data.message || res.data.msg)
    }
  } catch (error) {
    console.error('请求异常:', error)
  }
}
</script>