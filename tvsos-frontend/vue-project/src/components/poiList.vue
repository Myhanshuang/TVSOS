<template>
  <el-drawer title="poi列表" direction="rtl" size="50%" :model-value="visible"
    @update:model-value="$emit('update:visible', $event)">
     
      <el-form inline>
        <el-form-item label="城市">
          <el-select placeholder="请选择" style="width: 100px" v-model="selectedCity">
            <el-option label="成都" value="成都"></el-option>
            <el-option label="重庆" value="重庆"></el-option>
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
      <el-table :data="poiList"  @row-click="handleRowClick">
        <el-table-column label="名字" prop="name">
          <template #default="{ row }">
            <el-link type="primary" :underline="false">{{ row.name }}</el-link>
          </template>
        </el-table-column>
        <el-table-column label="类型" prop="type">
          <template #default="{ row }">
            {{ getTypeText(row.type) }}
          </template>
        </el-table-column>
        <el-table-column label="城市">
          成都
        </el-table-column>
        <el-table-column label="状态" prop="status">
          <template #default="{ row }">
            {{ getStatusText(row.status) }}
          </template>
        </el-table-column>
      </el-table>
      <!-- 分页区域 -->
      <el-pagination v-model:current-page="params.pagenum" v-model:page-size="params.pagesize"
        :page-sizes="[2, 3, 5, 10]" :background="true" layout="jumper,total, sizes,prev, pager, next" :total="total"
        @size-change="onSizeChange" @current-change="onCurrentChange" style="margin-top: 20px; justify-content: end;" />
   
  </el-drawer>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useMapStore } from '@/stores'

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
  // 关闭抽屉
  emit('update:visible', false)
}
// 选中的城市和类型
const selectedCity = ref('')
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
  0: '正常',
  1: '关闭',

}

// 获取类型文本
const getTypeText = (type) => {
  return typeMap[type] || '未知类型'
}

// 获取状态文本
const getStatusText = (status) => {
  return statusMap[status] || '未知状态'
}

// 模拟API数据
const mockPoiData = [
  {
    "id": 1,
    "name": "中国石油中和加油站",
    "lon": 104.091008,
    "lat": 30.561087,
    "type": 1,
    "status": 0
  },
  {
    "id": 2,
    "name": "延长壳牌盛锦三街加油站",
    "lon": 104.0354,
    "lat": 30.563117,
    "type": 1,
    "status": 0
  },
  {
    "id": 3,
    "name": "成都通能盛锦CNG加气站",
    "lon": 104.035513,
    "lat": 30.56348,
    "type": 1,
    "status": 0
  },
  {
    "id": 4,
    "name": "中国石油麻柳湾加油站",
    "lon": 104.087907,
    "lat": 30.596824,
    "type": 1,
    "status": 0
  },
  {
    "id": 5,
    "name": "中国石油元华加油站",
    "lon": 104.050117,
    "lat": 30.601513,
    "type": 1,
    "status": 0
  },
  {
    "id": 6,
    "name": "延长壳牌加油站(成都高新益新大道站)",
    "lon": 104.033537,
    "lat": 30.587778,
    "type": 1,
    "status": 0
  },
  {
    "id": 7,
    "name": "延长壳牌拓新西二街加油站",
    "lon": 104.053281,
    "lat": 30.542733,
    "type": 1,
    "status": 0
  },
  {
    "id": 8,
    "name": "中国石油油料中和加油站(红星路南延线)",
    "lon": 104.088515,
    "lat": 30.547206,
    "type": 1,
    "status": 0
  },
  {
    "id": 9,
    "name": "中国石化加油站",
    "lon": 104.026775,
    "lat": 30.567829,
    "type": 1,
    "status": 0
  },
  {
    "id": 10,
    "name": "长城加油站",
    "lon": 104.026451,
    "lat": 30.580316,
    "type": 2,
    "status": 0
  },
  {
    "id": 11,
    "name": "桂溪南站加油站",
    "lon": 104.060099,
    "lat": 30.608153,
    "type": 1,
    "status": 0
  },
  {
    "id": 12,
    "name": "成都高新区兴达加油站",
    "lon": 104.030415,
    "lat": 30.5908,
    "type": 1,
    "status": 0
  },
  {
    "id": 13,
    "name": "延长壳牌加油站(锦华路站)",
    "lon": 104.097729,
    "lat": 30.597947,
    "type": 1,
    "status": 0
  },
  {
    "id": 14,
    "name": "中国石化金晖加油站",
    "lon": 104.098723,
    "lat": 30.598139,
    "type": 1,
    "status": 0
  },
  {
    "id": 15,
    "name": "中国石化棬子树加油加气站",
    "lon": 104.102801,
    "lat": 30.594122,
    "type": 1,
    "status": 0
  },
  {
    "id": 16,
    "name": "延长壳牌大源组团二加油站",
    "lon": 104.034767,
    "lat": 30.544123,
    "type": 1,
    "status": 0
  },
  {
    "id": 17,
    "name": "中国石油棕树加油站(长寿路)",
    "lon": 104.076729,
    "lat": 30.611847,
    "type": 3,
    "status": 0
  },
  {
    "id": 18,
    "name": "延长壳牌加油站(成都市高新区成都高新中和站)",
    "lon": 104.097542,
    "lat": 30.542025,
    "type": 1,
    "status": 0
  },
  {
    "id": 19,
    "name": "中国石油成双加油站",
    "lon": 104.018722,
    "lat": 30.575009,
    "type": 1,
    "status": 0
  },
  {
    "id": 20,
    "name": "中国石油剑南加油站",
    "lon": 104.047121,
    "lat": 30.534593,
    "type": 1,
    "status": 0
  },
  {
    "id": 21,
    "name": "延长壳牌加油站(成新站)",
    "lon": 104.034885,
    "lat": 30.608623,
    "type": 1,
    "status": 0
  },
  {
    "id": 22,
    "name": "桂溪加油站",
    "lon": 104.077479,
    "lat": 30.618361,
    "type": 1,
    "status": 0
  },
  {
    "id": 23,
    "name": "道森能源长风加油站",
    "lon": 104.02022,
    "lat": 30.550826,
    "type": 1,
    "status": 0
  },
  {
    "id": 24,
    "name": "延长壳牌高新大道加油站",
    "lon": 104.041715,
    "lat": 30.617425,
    "type": 4,
    "status": 0
  },
  {
    "id": 25,
    "name": "中国石油高新天山加油站",
    "lon": 104.054007,
    "lat": 30.622034,
    "type": 1,
    "status": 0
  },
  {
    "id": 26,
    "name": "中国石化高新A加油站",
    "lon": 104.027133,
    "lat": 30.608959,
    "type": 1,
    "status": 0
  },
  {
    "id": 27,
    "name": "中国石化高新B加油站",
    "lon": 104.025772,
    "lat": 30.608241,
    "type": 7,
    "status": 0
  },
  {
    "id": 28,
    "name": "成通石化高新成通加油站(天府大道南段辅路)",
    "lon": 104.070777,
    "lat": 30.515983,
    "type": 6,
    "status": 0
  },
  {
    "id": 29,
    "name": "中国国际能源成都蓉电加油站",
    "lon": 104.098661,
    "lat": 30.622705,
    "type": 1,
    "status": 0
  },
  {
    "id": 30,
    "name": "中国石油成都交投娇子内侧加油站",
    "lon": 104.122029,
    "lat": 30.598541,
    "type": 5,
    "status": 0
  },
  {
    "id": 31,
    "name": "中国石油娇子三环(外侧)加油站",
    "lon": 104.123477,
    "lat": 30.597027,
    "type": 1,
    "status": 0
  }
]

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
  selectedCity.value = ''
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
const loadData = () => {
  console.log('搜索条件:', {
    city: selectedCity.value,
    type: selectedType.value,
    page: params.value.pagenum,
    pageSize: params.value.pagesize
  })

  // 这里模拟根据筛选条件过滤数据
  let filteredData = [...mockPoiData]

  // 实际使用时，这些过滤逻辑应该在API层面处理
  // 这里只是前端模拟
  if (selectedCity.value) {
    // 模拟城市过滤 - 实际应该调用API
    console.log('按城市过滤:', selectedCity.value)
  }

  if (selectedType.value) {
    // 找到对应的类型值
    const typeValue = Object.keys(typeMap).find(key => typeMap[key] === selectedType.value)
    if (typeValue) {
      filteredData = filteredData.filter(item => item.type === parseInt(typeValue))
    }
  }

  // 模拟分页
  const startIndex = (params.value.pagenum - 1) * params.value.pagesize
  const endIndex = startIndex + params.value.pagesize
  poiList.value = filteredData.slice(startIndex, endIndex)
  total.value = filteredData.length

  console.log('加载数据，参数:', params.value)
}
</script>