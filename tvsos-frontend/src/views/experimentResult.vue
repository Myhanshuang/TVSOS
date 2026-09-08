<template>
  <div class="result-page" v-loading="loading">
    <div class="page-head">
      <div>
        <el-button link :icon="ArrowLeft" @click="router.back()">返回</el-button>
        <h2>实验结果：{{ experiment?.name || '#' + expId }}</h2>
        <p class="page-sub" v-if="experiment">
          种子 {{ experiment.seed }} · {{ statusText(experiment.status) }} ·
          创建于 {{ formatTime(experiment.createdAt) }} · 沙箱 #{{ experiment.sandboxId }}
        </p>
      </div>
      <el-button type="primary" plain @click="goReplay" v-if="experiment">以此参数再次运行</el-button>
    </div>

    <template v-if="primary">
      <!-- 主目标卡片（对齐结果数据层） -->
      <el-row :gutter="16" class="metric-cards">
        <el-col :span="4" v-for="m in primaryCards" :key="m.label">
          <el-card shadow="hover" class="metric-card">
            <div class="metric-value">{{ m.value }}</div>
            <div class="metric-label">{{ m.label }}</div>
          </el-card>
        </el-col>
      </el-row>

      <el-row :gutter="16">
        <el-col :span="12">
          <el-card>
            <template #header>指标时间序列（虚拟时钟）</template>
            <div ref="trendRef" class="chart"></div>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card>
            <template #header>SA 收敛曲线（算法性能）</template>
            <div ref="convRef" class="chart"></div>
          </el-card>
        </el-col>
      </el-row>

      <el-card style="margin-top:16px">
        <template #header>终态指标全量（summary）</template>
        <el-descriptions :column="3" border size="small">
          <el-descriptions-item v-for="(v, k) in summaryFlat" :key="k" :label="k">{{ v }}</el-descriptions-item>
        </el-descriptions>
      </el-card>

      <el-card style="margin-top:16px">
        <template #header>参数快照（TOML）</template>
        <pre class="toml-view">{{ experiment.paramsToml }}</pre>
      </el-card>
    </template>

    <el-empty v-else-if="!loading" description="该实验尚无终态结果（可能未完成或已中止）" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft } from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import { getExperimentResult } from '@/api/sandbox'

const route = useRoute()
const router = useRouter()
const expId = Number(route.params.id)

const loading = ref(false)
const experiment = ref(null)
const primary = ref(null)
const summaryFlat = ref({})
const snapshots = ref([])
const convergence = ref([])
const trendRef = ref(null)
const convRef = ref(null)
let trendChart = null
let convChart = null

const primaryCards = computed(() => {
  if (!primary.value) return []
  const p = primary.value
  const fmt = (v, d = 3) => (typeof v === 'number' ? Number(v.toFixed(d)) : v)
  return [
    { label: 'C2 单位成本', value: fmt(p.C2_unit_cost) },
    { label: 'U2 装载率', value: fmt(p.U2_loading_rate) },
    { label: 'U4 空驶率', value: fmt(p.U4_empty_rate) },
    { label: 'T2 响应P95(s)', value: fmt(p.T2_match_response_p95) },
    { label: 'F1 基尼系数', value: fmt(p.F1_gini) },
    { label: 'G2 吨公里碳排', value: fmt(p.G2_carbon_per_ton_km) },
    { label: '总成本', value: fmt(p.total_cost, 2) },
    { label: '总吞吐(吨)', value: fmt(p.total_throughput, 2) },
    { label: '车祸次数', value: p.accident_count },
    { label: '拒单次数', value: p.refuse_count },
    { label: '虚拟tick数', value: p.virtual_ticks },
    { label: 'SA均耗时(ms)', value: fmt(p.solve_time_avg_ms, 2) },
  ]
})

const statusText = (s) => ({
  configured: '已配置', generating: '生成中', ready: '可启动',
  running: '运行中', finished: '已完成', failed: '失败', aborted: '已中止',
}[s] || s)
const formatTime = (t) => (t || '').replace('T', ' ').slice(0, 19)

const goReplay = () => router.push(`/admin/sandbox/${experiment.value.sandboxId}`)

const renderCharts = () => {
  if (snapshots.value.length > 0 && trendRef.value) {
    trendChart = echarts.init(trendRef.value)
    const times = snapshots.value.map(s => (s.snapshotTime || '').slice(11, 19))
    trendChart.setOption({
      tooltip: { trigger: 'axis' },
      legend: {},
      grid: { left: 50, right: 20, top: 40, bottom: 30 },
      xAxis: { type: 'category', data: times },
      yAxis: [{ type: 'value' }],
      series: [
        { name: '总成本', type: 'line', data: snapshots.value.map(s => s.costValue), smooth: true },
        { name: '总吞吐', type: 'line', data: snapshots.value.map(s => s.totalThroughput), smooth: true },
        { name: '空驶率', type: 'line', data: snapshots.value.map(s => s.emptyRate), smooth: true },
      ],
    })
  }
  if (convergence.value.length > 0 && convRef.value) {
    convChart = echarts.init(convRef.value)
    convChart.setOption({
      tooltip: { trigger: 'axis' },
      grid: { left: 60, right: 20, top: 30, bottom: 30 },
      xAxis: { type: 'category', name: '批次/温度序', data: convergence.value.map((_, i) => i + 1) },
      yAxis: { type: 'log', name: 'bestCost' },
      series: [{ name: 'SA最优cost', type: 'line', data: convergence.value.map(c => c.bestCost), smooth: true }],
    })
  }
}

const load = async () => {
  loading.value = true
  try {
    const res = await getExperimentResult(expId)
    if (res.data.code === 1) {
      experiment.value = res.data.data.experiment
      snapshots.value = res.data.data.snapshots || []
      const r = res.data.data.result
      try { primary.value = JSON.parse(r.primaryMetrics || '{}') } catch { primary.value = {} }
      try { summaryFlat.value = JSON.parse(r.summaryJson || '{}') } catch { summaryFlat.value = {} }
      try { convergence.value = JSON.parse(r.convergence || '[]') } catch { convergence.value = [] }
      // summary 里嵌套对象拍平一层
      const flat = {}
      for (const [k, v] of Object.entries(summaryFlat.value)) {
        flat[k] = typeof v === 'object' && v !== null ? JSON.stringify(v) : v
      }
      summaryFlat.value = flat
      setTimeout(renderCharts, 0)
    }
  } finally {
    loading.value = false
  }
}

onMounted(load)
onUnmounted(() => { trendChart?.dispose(); convChart?.dispose() })
</script>

<style scoped>
.result-page { padding: 84px 32px 32px; min-height: 100vh; background: #f5f7fa; box-sizing: border-box; }
.page-head { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 16px; }
.page-head h2 { margin: 4px 0; }
.page-sub { color: #909399; font-size: 13px; }
.metric-cards { margin-bottom: 16px; }
.metric-card { text-align: center; margin-bottom: 12px; }
.metric-value { font-size: 22px; font-weight: 700; color: #409eff; }
.metric-label { color: #909399; font-size: 12px; margin-top: 4px; }
.chart { height: 300px; }
.toml-view { background: #fafafa; padding: 12px; border-radius: 4px; font-size: 12px; max-height: 300px; overflow: auto; }
</style>
