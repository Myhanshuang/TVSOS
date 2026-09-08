<template>
  <div class="compare-page">
    <div class="page-head">
      <div>
        <el-button link :icon="ArrowLeft" @click="router.push(`/admin/sandbox/${sandboxId}`)">返回沙箱</el-button>
        <h2>实验对比</h2>
        <p class="page-sub">选择本沙箱已完成实验进行横向对比；也可输入其他沙箱的实验 ID 跨沙箱对比</p>
      </div>
    </div>

    <el-card>
      <template #header>选择实验</template>
      <el-select v-model="selectedIds" multiple placeholder="选择已完成实验" style="width: 60%">
        <el-option v-for="e in finishedExperiments" :key="e.id" :value="e.id"
          :label="`#${e.id} ${e.name} (seed=${e.seed})`" />
      </el-select>
      <el-input v-model="extraIds" placeholder="追加实验ID，逗号分隔（跨沙箱）" style="width: 25%; margin-left: 8px" />
      <el-button type="primary" style="margin-left: 8px" @click="load" :disabled="effectiveIds.length < 1">对比</el-button>
    </el-card>

    <template v-if="loaded">
      <!-- 主目标雷达图（归一化） -->
      <el-row :gutter="16" style="margin-top:16px">
        <el-col :span="12">
          <el-card>
            <template #header>主目标雷达（归一化，越大越优口径已折算）</template>
            <div ref="radarRef" class="chart"></div>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card>
            <template #header>关键指标柱状对比</template>
            <div ref="barRef" class="chart"></div>
          </el-card>
        </el-col>
      </el-row>

      <!-- 结果明细表 -->
      <el-card style="margin-top:16px">
        <template #header>结果明细</template>
        <el-table :data="rows" border size="small">
          <el-table-column label="指标" prop="metric" width="200" fixed />
          <el-table-column v-for="c in comparedExps" :key="c.id" :label="`#${c.id} ${c.name}`" min-width="140">
            <template #default="{ row }">{{ row['exp' + c.id] }}</template>
          </el-table-column>
        </el-table>
      </el-card>

      <!-- 参数差异 -->
      <el-card style="margin-top:16px">
        <template #header>参数 TOML 差异（相对第一个实验）</template>
        <div v-for="d in paramDiffs" :key="d.id" class="diff-block">
          <h4>#{{ d.id }} {{ d.name }}</h4>
          <div v-if="d.diffs.length === 0" class="same-tip">参数完全一致</div>
          <div v-for="(line, i) in d.diffs" :key="i" class="diff-line">{{ line }}</div>
        </div>
      </el-card>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft } from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import { listExperiments, compareExperiments } from '@/api/sandbox'

const route = useRoute()
const router = useRouter()
const sandboxId = Number(route.params.id)

const experiments = ref([])
const selectedIds = ref([])
const extraIds = ref('')
const loaded = ref(false)
const comparedExps = ref([])
const comparedResults = ref({}) // expId -> primaryMetrics object
const rows = ref([])
const paramDiffs = ref([])
const radarRef = ref(null)
const barRef = ref(null)
let radarChart = null
let barChart = null

const finishedExperiments = computed(() => experiments.value.filter(e => e.status === 'finished'))

const effectiveIds = computed(() => {
  const ids = [...selectedIds.value]
  for (const s of extraIds.value.split(',')) {
    const v = Number(s.trim())
    if (v > 0 && !ids.includes(v)) ids.push(v)
  }
  return ids
})

const METRICS = [
  { key: 'C2_unit_cost', label: 'C2 单位成本', lower: true },
  { key: 'U2_loading_rate', label: 'U2 装载率', lower: false },
  { key: 'U4_empty_rate', label: 'U4 空驶率', lower: true },
  { key: 'T2_match_response_p95', label: 'T2 响应P95', lower: true },
  { key: 'F1_gini', label: 'F1 基尼', lower: true },
  { key: 'G2_carbon_per_ton_km', label: 'G2 碳排', lower: true },
  { key: 'total_cost', label: '总成本', lower: true },
  { key: 'total_throughput', label: '总吞吐', lower: false },
]

const load = async () => {
  const res = await compareExperiments(effectiveIds.value)
  if (res.data.code !== 1) return
  comparedExps.value = res.data.data.experiments || []
  const resultMap = {}
  for (const r of res.data.data.results || []) {
    try { resultMap[r.experimentId] = JSON.parse(r.primaryMetrics || '{}') } catch { resultMap[r.experimentId] = {} }
  }
  comparedResults.value = resultMap

  // 明细表
  rows.value = METRICS.map(m => {
    const row = { metric: m.label }
    for (const e of comparedExps.value) {
      const v = resultMap[e.id]?.[m.key]
      row['exp' + e.id] = typeof v === 'number' ? Number(v.toFixed(4)) : '-'
    }
    return row
  })

  // 参数 diff（相对第一个实验，逐行对比）
  const baseLines = (comparedExps.value[0]?.paramsToml || '').split('\n')
  paramDiffs.value = comparedExps.value.slice(1).map(e => {
    const lines = (e.paramsToml || '').split('\n')
    const diffs = []
    const allKeys = new Set([...baseLines, ...lines])
    // 简单行集合差异
    const baseSet = new Map()
    for (const l of baseLines) { const k = l.split('=')[0]?.trim(); if (k && !k.startsWith('#') && l.includes('=')) baseSet.set(k, l.trim()) }
    const thisMap = new Map()
    for (const l of lines) { const k = l.split('=')[0]?.trim(); if (k && !k.startsWith('#') && l.includes('=')) thisMap.set(k, l.trim()) }
    for (const [k, v] of thisMap) {
      if (baseSet.get(k) !== v) diffs.push(`本实验: ${v}   |   基准: ${baseSet.get(k) || '（无）'}`)
    }
    for (const [k, v] of baseSet) {
      if (!thisMap.has(k)) diffs.push(`本实验缺失: ${v}`)
    }
    return { id: e.id, name: e.name, diffs }
  })

  setTimeout(renderCharts, 0)
  loaded.value = true
}

const renderCharts = () => {
  const exps = comparedExps.value
  const pm = comparedResults.value
  // 雷达：每维归一化到 0~1，"越低越好"取倒数方向折算为"越大越优"
  const radarMetrics = METRICS.slice(0, 6)
  const ranges = radarMetrics.map(m => {
    const vals = exps.map(e => pm[e.id]?.[m.key]).filter(v => typeof v === 'number')
    return { min: Math.min(...vals, 0), max: Math.max(...vals, 1e-9) }
  })
  if (radarRef.value) {
    radarChart = echarts.init(radarRef.value)
    radarChart.setOption({
      tooltip: {},
      legend: { bottom: 0 },
      radar: {
        indicator: radarMetrics.map(m => ({ name: m.label, max: 1 })),
      },
      series: [{
        type: 'radar',
        data: exps.map(e => ({
          name: `#${e.id} ${e.name}`,
          value: radarMetrics.map((m, i) => {
            const v = pm[e.id]?.[m.key]
            if (typeof v !== 'number') return 0
            const { min, max } = ranges[i]
            const norm = max === min ? 1 : (v - min) / (max - min)
            return m.lower ? 1 - norm : norm
          }),
        })),
      }],
    })
  }
  if (barRef.value) {
    barChart = echarts.init(barRef.value)
    const barMetrics = [METRICS[6], METRICS[7]]
    barChart.setOption({
      tooltip: {},
      legend: { bottom: 0 },
      grid: { left: 60, right: 20, top: 30, bottom: 50 },
      xAxis: { type: 'category', data: barMetrics.map(m => m.label) },
      yAxis: { type: 'value' },
      series: exps.map(e => ({
        name: `#${e.id} ${e.name}`,
        type: 'bar',
        data: barMetrics.map(m => Number((pm[e.id]?.[m.key] || 0).toFixed(3))),
      })),
    })
  }
}

onMounted(async () => {
  const res = await listExperiments(sandboxId)
  if (res.data.code === 1) experiments.value = res.data.data || []
})
onUnmounted(() => { radarChart?.dispose(); barChart?.dispose() })
</script>

<style scoped>
.compare-page { padding: 84px 32px 32px; min-height: 100vh; background: #f5f7fa; box-sizing: border-box; }
.page-head { margin-bottom: 16px; }
.page-head h2 { margin: 4px 0; }
.page-sub { color: #909399; font-size: 13px; }
.chart { height: 320px; }
.diff-block { margin-bottom: 12px; }
.diff-line { font-family: Consolas, monospace; font-size: 12px; color: #e6a23c; }
.same-tip { color: #67c23a; font-size: 13px; }
</style>
