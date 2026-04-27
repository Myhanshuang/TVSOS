<!-- 统计分析组件：展示车辆、POI、司机等汇总数据，并提供柱状图和饼图的可视化分析 -->
<script setup>
import { onMounted, onBeforeUnmount, ref, nextTick } from 'vue'
import { getPoiTybe, getVehicleCategory, getVehicleSum, getPoiSum, getDriverSum, getCargoSizeSum } from '@/api/report'
import * as echarts from 'echarts'

/** 顶部卡片汇总数据变量 */
const vehicleSum = ref(0)       // 车辆总数
const poiSum = ref(0)           // POI 点位总数
const driverSum = ref(0)        // 司机总数
const cargoSizeSum = ref(0)     // 运输中货物总重量

/** ECharts 容器 DOM 引用 */
const barRef = ref(null)        // 柱状图容器
const pieRef = ref(null)        // 饼图容器

/** 图表实例及辅助变量 */
let barChart = null             // 柱状图 ECharts 实例
let pieChart = null             // 饼图 ECharts 实例
let ro = null                   // 响应式尺寸监听器 (ResizeObserver)
let timer = null                // 自动刷新定时器

/** 图表原始数据存储 */
const PoiTypeData = ref([])     // POI 各类型统计数据
const VehicleCategoryData = ref([]) // 车辆分类统计数据 

/**
 * 格式化饼图所需的数据结构
 * @param {Array} raw 原始 API 返回数据
 * @returns {Array} 符合 ECharts pie series.data 格式的数据
 */
function normalizePieData(raw) {
  if (!Array.isArray(raw)) return []
  return raw
    .map((it) => ({
      name: String(it?.name ?? '未知'),
      value: Number(it?.count ?? 0)
    }))
    .filter((x) => x.name && Number.isFinite(x.value))
}

/**
 * 并发获取所有统计数据。
 * 使用 Promise.all 同时发起 6 个 API 请求以提高效率
 */
async function fetchAllOnce() {
  const [
    poiRes,
    vehRes,
    vehicleSumRes,
    poiSumRes,
    driverSumRes,
    cargoSizeSumRes
  ] = await Promise.all([
    getPoiTybe(),
    getVehicleCategory(),
    getVehicleSum(),
    getPoiSum(),
    getDriverSum(),
    getCargoSizeSum()
  ])

  // 更新响应式数据，若接口失败则回退到默认值
  PoiTypeData.value = poiRes?.data?.data ?? []
  VehicleCategoryData.value = vehRes?.data?.data ?? []
  vehicleSum.value = Number(vehicleSumRes?.data?.data ?? 0)
  poiSum.value = Number(poiSumRes?.data?.data ?? 0)
  driverSum.value = Number(driverSumRes?.data?.data ?? 0)
  cargoSizeSum.value = Number(cargoSizeSumRes?.data?.data ?? 0)
}

/**
 * 构建 POI 类型柱状图的 ECharts 配置项
 */
function buildBarOption() {
  // 后端返回格式通常为二维数组：[0]是名称数组，[1]是数值数组
  const xData = Array.isArray(PoiTypeData.value?.[0]) ? PoiTypeData.value[0] : []
  const yData = Array.isArray(PoiTypeData.value?.[1]) ? PoiTypeData.value[1] : []

  return {
    title: {
      text: 'POI 各类型数量柱状图',
      left: 'center',
      top: 10
    },
    tooltip: { trigger: 'axis' },
    grid: { left: 40, right: 20, top: 60, bottom: 40, containLabel: true },
    xAxis: { type: 'category', data: xData, axisLabel: { interval: 0 } },
    yAxis: { type: 'value', minInterval: 1 },
    series: [{ type: 'bar', data: yData }]
  }
}

/**
 * 构建车辆类型饼状图的 ECharts 配置项
 */
function buildPieOption() {
  let pieData = normalizePieData(VehicleCategoryData.value)
  const sum = pieData.reduce((s, x) => s + (x.value || 0), 0)

  // 若无数据，展示“暂无数据”占位圆环
  if (!pieData.length || sum === 0) pieData = [{ name: '暂无数据', value: 1 }]

  return {
    title: {
      text: '车辆类型饼状图',
      left: 'center',
      top: 10
    },
    tooltip: { trigger: 'item' },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'middle'
    },
    series: [
      {
        name: '车辆类型',
        type: 'pie',
        radius: ['40%', '70%'], // 环形图效果
        center: ['40%', '55%'],
        avoidLabelOverlap: false,
        itemStyle: { borderRadius: 10, borderColor: '#fff', borderWidth: 2 },
        label: { show: false, position: 'center' },
        emphasis: { label: { show: true, fontSize: 28, fontWeight: 'bold' } },
        labelLine: { show: false },
        data: pieData
      }
    ]
  }
}

/**
 * 初始化图表实例并绑定尺寸自适应监听器
 */
function initCharts() {
  if (barChart || pieChart) return

  // 使用 SVG 渲染器渲染图表
  barChart = echarts.init(barRef.value, null, { renderer: 'svg' })
  pieChart = echarts.init(pieRef.value, null, { renderer: 'svg' })

  // 监听容器 DOM 尺寸变化，自动调整图表大小
  ro = new ResizeObserver(() => {
    barChart?.resize()
    pieChart?.resize()
  })
  ro.observe(barRef.value)
  ro.observe(pieRef.value)
}

/**
 * 执行数据更新，并重绘图表
 */
function updateCharts() {
  // 防御性校验，确保 DOM 已经渲染且容器有宽高
  if (!barChart || !pieChart) return
  if (barRef.value?.clientWidth === 0 || barRef.value?.clientHeight === 0) return
  if (pieRef.value?.clientWidth === 0 || pieRef.value?.clientHeight === 0) return

  // 设置配置项（开启 notMerge 模式以清空之前状态）
  barChart.setOption(buildBarOption(), true)
  pieChart.setOption(buildPieOption(), true)

  barChart.resize()
  pieChart.resize()
}

/**
 * 单次轮询任务：拉取数据并更新 UI
 */
async function pollOnce() {
  try {
    await fetchAllOnce()
    updateCharts()
  } catch (e) {
    console.error('轮询请求失败：', e)
  }
}

/** 生命周期挂载：流程初始化 */
onMounted(async () => {
  // 等待 DOM 序列更新完成
  await nextTick()
  initCharts()
  await pollOnce()
  // 开启 5 秒一次的数据自动刷新
  timer = window.setInterval(pollOnce, 5000)
})

/** 生命周期卸载：清理资源，防止内存泄漏 */
onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
  ro?.disconnect()
  ro = null

  barChart?.dispose()
  pieChart?.dispose()
  barChart = null
  pieChart = null
})
</script>

<template>
  <div id="secBorder">
    <div class="Border">
      <!-- 顶部汇总指标卡片区域 -->
      <div class="hangCards">
        <div class="card" data-hover-text="查看详情">
          <div class="card-title">车辆数量</div>
          <div class="data">{{ vehicleSum }} 辆</div>
        </div>
        <div class="card" data-hover-text="点位分布">
          <div class="card-title">POI 点总量</div>
          <div class="data">{{ poiSum }} 个</div>
        </div>
        <div class="card"  data-hover-text="人员管理">
          <div class="card-title">司机数量</div>
          <div class="data">{{ driverSum }} 位</div>
        </div>
        <div class="card" data-hover-text="物流监控">
          <div class="card-title" >运输中的货物总量</div>
          <div class="data">{{ cargoSizeSum }} kg</div>
        </div>
      </div>

      <!-- 下部分可视化图表区域 -->
      <div class="hangCharts">
        <!-- 柱状图模块 -->
        <div class="chart">
          <div ref="barRef" class="chartDom"></div>
        </div>

        <!-- 饼图模块 -->
        <div class="chart">
          <div ref="pieRef" class="chartDom"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* 外部整体容器：占满屏幕主体区域且禁止外部溢出 */
#secBorder {
  margin: 0px;
  padding: 0px;
  width: 99vw;
  height: 88vh;
  overflow: hidden;
}

/* 内部限制宽度并居中 */
.Border {
  margin: 5px 10%;
  width: 80%;
  height: calc(100% - 10px);
}

/* 顶部指标行布局 */
.hangCards {
  display: flex;
  height: 25%;
  width: 100%;
}

/* 指标卡片基础样式及阴影 */
.card {
  margin: 2px 6px;
  width: 100%;
  height: calc(100% - 4px);
  background-color: white;
  border-radius: 6px;
  box-shadow: 2px 2px 2px #e1e1e1, -2px -2px 2px #e1e1e1;

  /* 为了让右下角的符号能够绝对定位 */
  position: relative; 
  overflow: hidden;
  transition: all 0.3s ease;
  cursor: pointer;
}

/* 设置文字样式 */
.card::before {
  content: attr(data-hover-text); 
  position: absolute;
  bottom: 15px;
  right: 15px; 
  font-size: 14px;
  font-weight: 500;
  color: #363636;
  
  opacity: 0;
  transform: translateX(60px);
  transition: all 0.3s ease;
  white-space: nowrap;
}

/* 悬停时，图标向左移，文字浮现 */
.card:hover::before {
  opacity: 1;
  transform: translateX(0);
}


/* 确保图标不会挡住文字 */
.card:hover::after {
  opacity: 1;
  /* 悬停时图标也向左移一点，给整体留出空间 */
  transform: translateX(-60px); 

}

/* 利用伪元素生成右下角的 > */
.card::after {
  content: "";
  position: absolute;
  bottom: 12px;
  right: 12px;
  width: 24px;
  height: 24px;
  /* 使用你项目中的图标路径 */
  background: url("../../public/images/arrow-right.svg") no-repeat center;
  background-size: contain;
  opacity: 0.5;
  transition: 0.3s;
}

/* 添加一个悬停变色效果 */
.card:hover {
  transform: translateY(-5px);
  box-shadow: 4px 4px 10px #d1d1d1;
}

/* 图表行布局 */
.hangCharts {
  display: flex;
  height: 75%;
  width: 100%;
}

/* 图表容器背景及布局控制 */
.chart {
  width: 50%;
  height: calc(100% - 12px);
  margin: 8px 6px 4px 6px;
  background-color: white;
  border-radius: 6px;
  box-shadow: 2px 2px 2px #e1e1e1, -2px -2px 2px #e1e1e1;
  display: flex;
}

/* ECharts 实际绘图 DOM 样式 */
.chartDom {
  width: 100%;
  height: 100%;
}

/* 指标卡片标题样式 */
.card-title {
  margin: 15px 0px 15px 20px;
  padding: 0px;
  font-size: 24px;
  font-weight: bold;
  color: #606060;
}

/* 指标数值文本样式 */
.data {
  margin: 0px;
  padding: 0px;
  text-align: center;
  font-size: 30px;
  font-weight: bold;
  color: #333333;
}
</style>