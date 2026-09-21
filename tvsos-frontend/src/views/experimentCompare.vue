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

      <!-- 参数差异（git diff 视图） -->
      <el-card style="margin-top:16px">
        <template #header>参数 TOML 差异（git diff · 基准 = 第一个实验）</template>
        <div v-for="d in paramDiffs" :key="d.id" class="diff-block">
          <div class="diff-title">
            <h4>#{{ d.id }} {{ d.name }}</h4>
            <el-tag v-if="d.hunks.length === 0" size="small" type="success" effect="plain">参数完全一致</el-tag>
            <el-tag v-else size="small" type="warning" effect="plain">+{{ d.addCount }} / -{{ d.delCount }} 行变更</el-tag>
          </div>
          <div v-if="d.hunks.length" class="diff-view">
            <template v-for="(h, hi) in d.hunks" :key="hi">
              <div class="d-hunk">{{ h.header }}</div>
              <div v-for="(ln, li) in h.lines" :key="hi + '-' + li" class="d-line" :class="ln.type">
                <span class="d-no">{{ ln.aNo ?? '' }}</span>
                <span class="d-no">{{ ln.bNo ?? '' }}</span>
                <span class="d-mk">{{ ln.marker }}</span>
                <span class="d-txt">{{ ln.text || ' ' }}</span>
              </div>
            </template>
          </div>
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

// ---- git 风格行级 diff（LCS）----
// diffLines：输出 { type: 'equal'|'del'|'add', text, aNo, bNo } 序列（del 属基准侧，add 属对比侧）
const diffLines = (a, b) => {
  const n = a.length
  const m = b.length
  // LCS 长度矩阵（自底向上），TOML 行数小，O(n·m) 足够
  const dp = Array.from({ length: n + 1 }, () => new Array(m + 1).fill(0))
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      dp[i][j] = a[i] === b[j] ? dp[i + 1][j + 1] + 1 : Math.max(dp[i + 1][j], dp[i][j + 1])
    }
  }
  const out = []
  let i = 0
  let j = 0
  while (i < n && j < m) {
    if (a[i] === b[j]) { out.push({ type: 'equal', text: a[i], aNo: i + 1, bNo: j + 1 }); i++; j++ }
    else if (dp[i + 1][j] >= dp[i][j + 1]) { out.push({ type: 'del', text: a[i], aNo: i + 1 }); i++ }
    else { out.push({ type: 'add', text: b[j], bNo: j + 1 }); j++ }
  }
  while (i < n) { out.push({ type: 'del', text: a[i], aNo: i + 1 }); i++ }
  while (j < m) { out.push({ type: 'add', text: b[j], bNo: j + 1 }); j++ }
  return out
}

// buildHunks：把 diff 序列切成带 3 行上下文的 hunk，产出 git unified 头 @@ -a,n +b,n @@
const DIFF_CONTEXT = 3
const buildHunks = (diffs) => {
  const changed = []
  diffs.forEach((d, idx) => { if (d.type !== 'equal') changed.push(idx) })
  if (!changed.length) return []
  // 相邻变更间隔 <= 2*CONTEXT+1 归并进同一个 hunk
  const groups = []
  let g = [changed[0]]
  for (let k = 1; k < changed.length; k++) {
    if (changed[k] - changed[k - 1] <= DIFF_CONTEXT * 2 + 1) g.push(changed[k])
    else { groups.push(g); g = [changed[k]] }
  }
  groups.push(g)
  return groups.map((gr) => {
    const from = Math.max(0, gr[0] - DIFF_CONTEXT)
    const to = Math.min(diffs.length - 1, gr[gr.length - 1] + DIFF_CONTEXT)
    const slice = diffs.slice(from, to + 1)
    const aLines = slice.filter((d) => d.type !== 'add')
    const bLines = slice.filter((d) => d.type !== 'del')
    // 纯新增 hunk 无基准侧行号时，取前一行基准侧行号作锚点（git -n,0 口径）
    const aStart = aLines.length ? aLines[0].aNo : (from > 0 ? diffs[from - 1].aNo : 0)
    const bStart = bLines.length ? bLines[0].bNo : (from > 0 ? diffs[from - 1].bNo : 0)
    return {
      header: `@@ -${aStart},${aLines.length} +${bStart},${bLines.length} @@`,
      lines: slice.map((d) => ({ ...d, marker: d.type === 'add' ? '+' : d.type === 'del' ? '-' : ' ' })),
    }
  })
}

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

  // 参数 diff：git 风格统一 diff（基准 = 第一个实验）
  const baseToml = comparedExps.value[0]?.paramsToml || ''
  const splitLines = (t) => {
    const ls = (t || '').split('\n')
    while (ls.length && !ls[ls.length - 1].trim()) ls.pop()
    return ls
  }
  paramDiffs.value = comparedExps.value.slice(1).map(e => {
    const hunks = buildHunks(diffLines(splitLines(baseToml), splitLines(e.paramsToml)))
    let addCount = 0
    let delCount = 0
    for (const h of hunks) for (const ln of h.lines) {
      if (ln.type === 'add') addCount++
      else if (ln.type === 'del') delCount++
    }
    return { id: e.id, name: e.name, hunks, addCount, delCount }
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
.diff-block { margin-bottom: 14px; }
.diff-title { display: flex; align-items: center; gap: 10px; margin-bottom: 6px; }
.diff-title h4 { margin: 0; }

/* git diff 统一视图：双侧行号槽 + 变更标记 + 红删绿增 */
.diff-view {
  border: 1px solid #ebeef5; border-radius: 6px; overflow-x: auto;
  font-family: Consolas, 'Courier New', monospace; font-size: 12px; background: #fff;
}
.d-hunk {
  background: #f5f7fa; color: #909399; padding: 2px 10px;
  border-top: 1px solid #ebeef5; border-bottom: 1px solid #ebeef5;
}
.d-line { display: flex; white-space: pre; }
.d-line .d-no { width: 38px; flex-shrink: 0; text-align: right; color: #c0c4cc; padding-right: 8px; user-select: none; }
.d-line .d-mk { width: 16px; flex-shrink: 0; text-align: center; user-select: none; color: #909399; }
.d-line .d-txt { flex: 1; padding-right: 10px; color: #303133; }
.d-line.del { background: #fef0f0; }
.d-line.del .d-mk, .d-line.del .d-txt { color: #f56c6c; }
.d-line.add { background: #f0f9eb; }
.d-line.add .d-mk, .d-line.add .d-txt { color: #67c23a; }
</style>
