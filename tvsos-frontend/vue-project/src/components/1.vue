<script setup>
import { onMounted, onBeforeUnmount, ref, nextTick } from 'vue'
import {
  getPoiTybe,
  getVehicleCategory,
  getVehicleSum,
  getPoiSum,
  getDriverSum,
  getCargoSizeSum
} from '@/api/report'
import * as echarts from 'echarts'

// 统计数据（四张卡片）
const vehicleSum = ref(0)
const PoiSum = ref(0)
const driverSum = ref(0)
const cargoSizeSum = ref(0)

const barRef = ref(null)
const pieRef = ref(null)

let barChart = null
let pieChart = null
let ro = null
let timer = null

const PoiTypeData = ref([]) // 后端返回 [xAxisArray, yArray]
const VehicleCategoryData = ref([]) // 后端返回 [{name, count}, ...]

function normalizePieData(raw) {
  if (!Array.isArray(raw)) return []
  return raw
    .map((it) => ({
      name: String(it?.name ?? '未知'),
      value: Number(it?.count ?? 0)
    }))
    .filter((x) => x.name && Number.isFinite(x.value))
}

// ✅ 同时拉取：4个统计 + 2个图表数据（并发）
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

  // 图表数据
  PoiTypeData.value = poiRes?.data?.data ?? []
  VehicleCategoryData.value = vehRes?.data?.data ?? []

  // 卡片数据（假设 data.data 是数字）
  vehicleSum.value = Number(vehicleSumRes?.data?.data ?? 0)
  PoiSum.value = Number(poiSumRes?.data?.data ?? 0)
  driverSum.value = Number(driverSumRes?.data?.data ?? 0)
  cargoSizeSum.value = Number(cargoSizeSumRes?.data?.data ?? 0)
}

function buildBarOption() {
  const xData = Array.isArray(PoiTypeData.value?.[0]) ? PoiTypeData.value[0] : []
  const yData = Array.isArray(PoiTypeData.value?.[1]) ? PoiTypeData.value[1] : []

  return {
    title: { text: 'POI 各类型数量柱状图', left: 'center', top: 10 },
    tooltip: { trigger: 'axis' },
    grid: { left: 40, right: 20, top: 60, bottom: 40, containLabel: true },
    xAxis: { type: 'category', data: xData, axisLabel: { interval: 0 } },
    yAxis: { type: 'value', minInterval: 1 },
    series: [{ type: 'bar', data: yData }]
  }
}

function buildPieOption() {
  let pieData = normalizePieData(VehicleCategoryData.value)
  const sum = pieData.reduce((s, x) => s + (x.value || 0), 0)
  if (!pieData.length || sum === 0) pieData = [{ name: '暂无数据', value: 1 }]

  return {
    title: { text: '车辆类型饼状图', left: 'center', top: 10 },
    tooltip: { trigger: 'item' },
    legend: { orient: 'vertical', right: 10, top: 'middle' }, // 右侧图示
    series: [
      {
        name: '车辆类型',
        type: 'pie',
        radius: ['40%', '70%'],
        center: ['40%', '55%'], // 给右侧 legend 留空间
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

function initCharts() {
  if (barChart || pieChart) return

  barChart = echarts.init(barRef.value, null, { renderer: 'svg' })
  pieChart = echarts.init(pieRef.value, null, { renderer: 'svg' })

  ro = new ResizeObserver(() => {
    barChart?.resize()
    pieChart?.resize()
  })
  ro.observe(barRef.value)
  ro.observe(pieRef.value)
}

function updateCharts() {
  if (!barChart || !pieChart) return
  if (barRef.value?.clientWidth === 0 || barRef.value?.clientHeight === 0) return
  if (pieRef.value?.clientWidth === 0 || pieRef.value?.clientHeight === 0) return

  barChart.setOption(buildBarOption(), true)
  pieChart.setOption(buildPieOption(), true)

  barChart.resize()
  pieChart.resize()
}

async function pollOnce() {
  try {
    await fetchAllOnce()   // ✅ 同时更新卡片+图
    updateCharts()
  } catch (e) {
    console.error('轮询请求失败：', e)
  }
}

onMounted(async () => {
  await nextTick()
  initCharts()

  await pollOnce()
  timer = window.setInterval(pollOnce, 5000)
})

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
      <div class="hangCards">
        <div class="card">
          <div class="cardTitle">车辆总数</div>
          <div class="cardValue">{{ vehicleSum }}</div>
        </div>

        <div class="card">
          <div class="cardTitle">POI 总数</div>
          <div class="cardValue">{{ PoiSum }}</div>
        </div>

        <div class="card">
          <div class="cardTitle">司机总数</div>
          <div class="cardValue">{{ driverSum }}</div>
        </div>

        <div class="card">
          <div class="cardTitle">货物总量</div>
          <div class="cardValue">{{ cargoSizeSum }} <span class="unit">kg</span></div>
        </div>
      </div>

      <div class="hangCharts">
        <div class="chart">
          <div ref="barRef" class="chartDom"></div>
        </div>

        <div class="chart">
          <div ref="pieRef" class="chartDom"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
#secBorder {
  margin: 0px;
  padding: 0px;
  width: 99vw;
  height: 88vh;
  overflow: hidden;
}
.Border {
  margin: 5px 10%;
  width: 80%;
  height: calc(100% - 10px);
}
.hangCards {
  display: flex;
  height: 25%;
  width: 100%;
}
.card {
  margin: 2px 6px;
  width: 100%;
  height: calc(100% - 4px);
  background-color: white;
  border-radius: 6px;
  box-shadow: 2px 2px 2px #e1e1e1, -2px -2px 2px #e1e1e1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 12px 14px;
}
.cardTitle {
  font-size: 14px;
  color: #666;
  margin-bottom: 10px;
}
.cardValue {
  font-size: 28px;
  font-weight: 700;
  color: #2c3e50;
  line-height: 1;
}
.unit {
  font-size: 14px;
  font-weight: 500;
  color: #888;
  margin-left: 6px;
}

.hangCharts {
  display: flex;
  height: 75%;
  width: 100%;
}
.chart {
  width: 50%;
  height: calc(100% - 12px);
  margin: 8px 6px 4px 6px;
  background-color: white;
  border-radius: 6px;
  box-shadow: 2px 2px 2px #e1e1e1, -2px -2px 2px #e1e1e1;
  display: flex;
}
.chartDom {
  width: 100%;
  height: 100%;
}
</style>
