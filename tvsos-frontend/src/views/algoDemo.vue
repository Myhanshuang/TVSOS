<template>
  <div id="fouBorder" class="algo-demo-container">
    <div class="header">
      <h2>模拟退火调度算法核心优势与流程演示</h2>
      <p>对比系统在无调度与启用 SA 模拟退火+多维 Cost 调度下的性能进化。</p>
    </div>

    <!-- 图表对比区 -->
    <div class="metrics-comparison">
      <div 
        v-for="(item, index) in metricsList" 
        :key="index" 
        class="small-chart-box" 
        :ref="el => { if (el) chartRefs[index] = el }"
      ></div>
    </div>

    <!-- 算法动画演示区 -->
    <div class="animation-stage">
      <div class="stage-header">
        <h3>退火算法推演：寻找全局最优解</h3>
        <el-button type="primary" @click="startAnimation" :disabled="isAnimating">
          开始推演 (退火降温)
        </el-button>
      </div>
      <div class="stage-content" ref="lineChartRef"></div>
      <div class="iteration-info" v-if="isAnimating || iteration > 0">
        <p>当前迭代次数: <strong>{{ iteration }}</strong></p>
        <p>当前系统温度: <strong>{{ temperature.toFixed(2) }}</strong></p>
        <p>全局最少 Total Cost: <strong>{{ bestCost.toFixed(2) }}</strong></p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import * as echarts from 'echarts'

const chartRefs = ref([])
const lineChartRef = ref(null)
let barCharts = []
let lineChart = null
let timer = null // 定时器句柄
let resizeObserver = null // ResizeObserver句柄

const metricsList = ref([
  { title: '平均空载率 (%)', label: '平均空载率', before: 45.24, after: 21.45 },
  { title: '货物平均等待 (min)', label: '货物平均等待', before: 120.35, after: 45.12 },
  { title: '最长车辆闲置 (min)', label: '最长车辆闲置', before: 310.75, after: 110.28 },
  { title: '运输规划绕路比 (%)', label: '运输规划绕路比', before: 35.82, after: 5.18 },
  { title: '组合算法综合代价', label: '综合代价 (Cost)', before: 15124.63, after: 4425.81 }
])

// SA动画初始配置
const initSACost = 4725.81 // (4425.81 + 300)

// 状态
const isAnimating = ref(false)
const iteration = ref(0)
const temperature = ref(1000.0)
const bestCost = ref(initSACost)

onMounted(() => {
  nextTick(() => {
    // 延迟初始化，确保在路由切换导致的外层容器动画结束后再执行画布渲染
    setTimeout(() => {
      initBarCharts()
      if (lineChartRef.value) {
        initLineChart()
      }
      
      // 使用 ResizeObserver 监听容器真实尺寸变化，自动触发重绘，彻底解决白屏问题
      resizeObserver = new ResizeObserver(() => {
        barCharts.forEach(c => c?.resize())
        lineChart?.resize()
      })
      
      chartRefs.value.forEach(el => {
        if (el) resizeObserver.observe(el)
      })
      if (lineChartRef.value) {
        resizeObserver.observe(lineChartRef.value)
      }
    }, 200)
  })
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
  window.removeEventListener('resize', handleResize)
  barCharts.forEach(c => c?.dispose())
  lineChart?.dispose()
})

// 处理窗口大小变化，自适应重绘图表
const handleResize = () => {
  barCharts.forEach(c => c?.resize())
  lineChart?.resize()
}

// 初始化多个对比子图
const initBarCharts = () => {
  barCharts = chartRefs.value.map((el, idx) => {
    if (!el) return null;
    const chart = echarts.init(el)
    const item = metricsList.value[idx]
    const option = {
      title: { 
        text: item.title, 
        left: 'center', 
        textStyle: { fontSize: 14 } 
      },
      tooltip: { trigger: 'item' },
      grid: { left: '3%', right: '3%', bottom: '10%', containLabel: true },
      xAxis: { type: 'category', data: ['无干预/随机分派', 'SA 模拟退火调度'] },
      yAxis: { type: 'value' },
      series: [
        {
          name: item.label,
          type: 'bar',
          barWidth: '40%',
          label: { show: true, position: 'top' },
          data: [
            { value: item.before, itemStyle: { color: '#ee6666' } },
            { value: item.after, itemStyle: { color: '#5470c6' } }
          ]
        }
      ]
    }
    chart.setOption(option)
    return chart
  })
}

// 初始化退火算法收敛曲线折线图
const initLineChart = () => {
  lineChart = echarts.init(lineChartRef.value)
  const option = {
    title: { text: 'Total Cost 收敛曲线 (Metropolis准则接受度)', left: 'center' },
    tooltip: { trigger: 'axis' },
    legend: { data: ['当前状态代价', '历史最优代价'], top: '10%' },
    xAxis: { type: 'category', data: ['0'], name: '迭代(Iter)' },
    yAxis: { type: 'value', name: '系统代价 (Cost)', scale: true },
    series: [
      {
        name: '当前状态代价',
        data: [initSACost],
        type: 'line',
        smooth: true,
        itemStyle: { color: '#91cc75' },
        lineStyle: { width: 3 },
        areaStyle: { opacity: 0.3 }
      },
      {
        name: '历史最优代价',
        data: [initSACost],
        type: 'line',
        smooth: true,
        itemStyle: { color: '#ee6666' },
        lineStyle: { width: 3, type: 'dashed' }
      }
    ]
  }
  lineChart.setOption(option)
}

// 模拟退火算法推演动画，展示 Cost 指标跳变和快速收敛的过程
const startAnimation = () => {
  isAnimating.value = true
  iteration.value = 0
  temperature.value = 1000.0
  bestCost.value = initSACost

  const xAxisData = ['0']
  const seriesData = [initSACost]
  const bestSeriesData = [initSACost]

  let currentCost = initSACost

  if (timer) clearInterval(timer)
  timer = setInterval(() => {
    iteration.value += 1
    temperature.value *= 0.95 // 模拟冷却率

    // 随机产生一个邻居解的Cost
    const fluctuate = (Math.random() - 0.4) * (temperature.value * 2) 
    let nextCost = currentCost + fluctuate

    // 退火核心：差或好都按概率接受，高温时允许接受差解跳出局部最优
    if (nextCost < currentCost || Math.exp((currentCost - nextCost) / temperature.value) > Math.random()) {
      currentCost = nextCost
      if (currentCost < bestCost.value) {
        bestCost.value = currentCost
      }
    }

    xAxisData.push(iteration.value.toString())
    seriesData.push(currentCost.toFixed(0))
    bestSeriesData.push(bestCost.value.toFixed(0))

    if (lineChart && !lineChart.isDisposed()) {
      lineChart.setOption({
        xAxis: { data: xAxisData },
        series: [{ data: seriesData }, { data: bestSeriesData }]
      })
    }

    if (temperature.value <= 1.0 || iteration.value > 150) {
      clearInterval(timer)
      timer = null
      isAnimating.value = false
    }
  }, 50) // 每 50ms 一帧动画
}
</script>

<style scoped>
.algo-demo-container {
  padding: 20px;
  background-color: transparent;
  height: 100%;
  overflow-y: auto;
}
.header {
  position: relative;
  margin-bottom: 20px;
  text-align: center;
}
.header h2 {
  color: #333;
}
.metrics-comparison {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 20px;
  margin-bottom: 40px;
  padding: 0 5%;
}
.small-chart-box {
  width: calc(30% - 20px);
  min-width: 280px;
  height: 280px;
  background-color: #fff;
  padding: 15px;
  border-radius: 8px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.05);
}
.animation-stage {
  background-color: #fff;
  width: 90%;
  margin: 0 auto;
  padding: 20px;
  border-radius: 8px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.05);
}
.stage-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.stage-content {
  height: 400px;
}
.iteration-info {
  display: flex;
  justify-content: space-around;
  margin-top: 15px;
  background: #f0f2f5;
  padding: 15px;
  border-radius: 6px;
  color: #333;
}
</style>