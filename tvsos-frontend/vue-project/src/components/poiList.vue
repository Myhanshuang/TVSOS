<template>
  <el-drawer title="poi列表" direction="rtl" size="50%" :model-value="visible"
    @update:model-value="$emit('update:visible', $event)">
     
      <el-form inline>
        <el-form-item label="状态">
          <el-select placeholder="请选择" style="width: 100px" v-model="selectedStatus">
            <el-option label="关闭" value="0"></el-option>
            <el-option label="正常" value="1"></el-option>
          </el-select>
        </el-form-item>
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
        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>
<el-table :data="poiList" @row-click="handleRowClick">
        <el-table-column label="名字" prop="name">
          <template #default="{ row }">
            <el-link type="primary" :underline="false">{{ row.name }}</el-link>
          </template>
        </el-table-column>

        <!-- 修改1：类型列 -->
        <!-- 后端返回字段是 tybe (注意拼写)，我们需要用 typeMap 把数字转成文字 -->
        <el-table-column label="类型" prop="tybe">
          <template #default="{ row }">
            <el-tag>{{ typeMap[row.tybe] || '未知类型' }}</el-tag>
          </template>
        </el-table-column>

     

        <!-- 修改2：状态列 -->
        <!-- 后端返回 status 是 0 或 1，需转成文字 -->
        <el-table-column label="状态" prop="status">
          <template #default="{ row }">
            <el-tag :type="row.status === 0 ? 'success' : 'danger'">
              {{ statusMap[row.status] || '未知' }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
      <!-- 分页区域 -->
      <el-pagination v-model:current-page="params.pagenum" v-model:page-size="params.pagesize"
        :page-sizes="[2, 3, 5, 10]" :background="true" layout="jumper,total, sizes,prev, pager, next" :total="total"  pager-count="4"
        @size-change="onSizeChange" @current-change="onCurrentChange" style="margin-top: 20px; justify-content: end;" />
   
  </el-drawer>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useMapStore, useImformStore, usePoiBoxStore } from '@/stores'
import request from '@/utils/request.js'
const poiBox = usePoiBoxStore()
const imform = useImformStore()

defineProps({
  visible: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:visible'])

const mapStore = useMapStore()

// 修改表格点击事件
const handleRowClick = (row) => {
  // 设置地图中心点到选中的POI
  mapStore.setCenter([row.lon, row.lat])
  mapStore.setZoom(15)
  // 设置闪烁的POI
  mapStore.setBlinkingPoi(row)
  // 设置列表点击点为最近点击的POI
  poiBox.recentPoiChange(row)
  // 设置信息框内容为POI，并显示
  imform.imformShow("poi", null)
  // 关闭抽屉
  emit('update:visible', false)
}
// 选中的城市和类型
const selectedStatus = ref('') // <--- 修改：selectedCity 改为 selectedStatus
const selectedType = ref('')

// POI数据
const poiList = ref([])
const params = ref({
  pagenum: 1,
  pagesize: 10
})
const total = ref(0)

// 类型映射
const typeMap = {
  1: '加油站',
  2: '加气站',
  3: '其它能源站',
  4: '工厂',
  5: '公司企业',
  6: '购物中心',
  7: '家具建材市场'
}

// 状态映射
const statusMap = {
  0: '关闭',
  1: '正常',

}



 

// 初始化数据
onMounted(() => {
  // 这里模拟从API获取数据
  loadData()
})

// 分页方法
const onSizeChange = (newSize) => {
  params.value.pagesize = newSize
  params.value.pagenum = 1
  // 这里可以调用API重新获取数据
  loadData()
}
// 重置处理 - 修复后的版本
const handleReset = () => {
  // 清空筛选条件
  selectedStatus.value = ''
  selectedType.value = ''

  // 重置分页参数
  params.value.pagenum = 1
  params.value.pagesize = 10

  // 重新加载数据（显示所有数据）
  loadData()

  console.log('重置筛选条件')
}

const onCurrentChange = (newPage) => {
  params.value.pagenum = newPage
  // 这里可以调用API重新获取数据
  loadData()
}
// 搜索处理
const handleSearch = () => {
  params.value.pagenum = 1 // 重置到第一页
  loadData()
}
// 加载数据方法（实际使用时替换为API调用）
// ... 其他 import 代码 ...

// 修改 loadData 方法
const loadData = async () => {
  // 1. 类型转换：将下拉框选中的中文类型（如"加油站"）转换为接口需要的数字 ID（tybe）
  // 假设 typeMap 结构为 { 1: '加油站', 2: '加气站' ... }
  let typeId = null
  if (selectedType.value) {
    // 查找 value 等于 selectedType.value 的 key
    typeId = Object.keys(typeMap).find(key => typeMap[key] === selectedType.value)
  }

  try {
    // 2. 发起请求
    // 注意：后端不支持分页，所以不要传 page/size，只传筛选条件
    const res = await request.get('/poi', { // 👈 请将 '/poi/list' 替换为你真实的 API 路径
      params: {
        // 接口文档参数：status (integer)
        status: selectedStatus.value === '' ? null : selectedStatus.value,
        
        // 接口文档参数：tybe (integer) -> 注意文档里写的是 tybe 不是 type
        tybe: typeId 
        
        // name 参数如果你有搜索框也可以加上，没有就不用传
        // name: searchName.value 
      }
    })

    // 3. 处理响应
    // 兼容判断：虽然 Swagger 示例写 code:0，但很多通用框架成功是 1，请根据实际情况调整
    if (res.data.code === 1 || res.data.code === 0 || res.data.code === 200) {
      
      // 获取所有符合条件的数据
      const allData = res.data.data || []
      
      // --- 前端分页逻辑开始 ---
      
      // A. 设置总条数
      total.value = allData.length

      // B. 计算当前页的数据范围
      // 例如：第1页，10条/页 -> startIndex=0, endIndex=10
      const startIndex = (params.value.pagenum - 1) * params.value.pagesize
      const endIndex = startIndex + params.value.pagesize

      // C. 截取当前页的数据赋值给表格
      poiList.value = allData.slice(startIndex, endIndex)
      
      // --- 前端分页逻辑结束 ---

    } else {
      console.error('获取POI列表失败:', res.data.message || res.data.msg)
    }
  } catch (error) {
    console.error('请求异常:', error)
  }
}
</script>