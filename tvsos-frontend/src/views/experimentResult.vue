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

      <!-- 终态指标全景：按指标体系维度分组可视化 + 构成图 + 事件时间线 + 原始字段折叠 -->
      <el-card style="margin-top:16px">
        <template #header>
          <div class="pan-head">
            <span>终态指标全量（summary）</span>
            <span class="pan-sub">按 T/U/C/A/M/R/G 维度分组 · 率值带比例条 · 底部可展开原始字段</span>
          </div>
        </template>

        <el-row :gutter="16">
          <!-- 左：分组指标卡 -->
          <el-col :span="16">
            <el-row :gutter="12">
              <el-col :span="12" v-for="g in groups" :key="g.title">
                <div class="m-group">
                  <div class="m-group-title">{{ g.title }}</div>
                  <div class="m-grid">
                    <div v-for="it in g.items" :key="it.label" class="m-item" :title="it.note || it.label">
                      <div class="m-label">{{ it.label }}</div>
                      <div class="m-value">
                        <template v-if="it.values">
                          {{ f(it.values[0]) }} / {{ f(it.values[1]) }} / {{ f(it.values[2]) }}<i class="m-unit">秒</i>
                        </template>
                        <template v-else>
                          {{ f(it.value) }}<i v-if="it.unit" class="m-unit">{{ it.unit }}</i>
                        </template>
                      </div>
                      <el-progress v-if="ratioOf(it) !== null" :percentage="ratioOf(it)" :stroke-width="5"
                        :show-text="false" :color="barColor(it)" />
                    </div>
                  </div>
                </div>
              </el-col>
            </el-row>
          </el-col>

          <!-- 右：构成图 + 响应分位条 -->
          <el-col :span="8">
            <div class="side-block">
              <div class="side-title">车队构成（辆）</div>
              <div ref="fleetRef" class="side-chart"></div>
            </div>
            <div class="side-block">
              <div class="side-title">货类需求（吨）</div>
              <div ref="cargoRef" class="side-chart"></div>
            </div>
            <div class="side-block">
              <div class="side-title">匹配响应分位（秒）</div>
              <div class="pct-bars">
                <div v-for="p in pctRows" :key="p.label" class="pct-row">
                  <span class="pct-label">{{ p.label }}</span>
                  <div class="pct-track"><div class="pct-fill" :style="{ width: p.width, background: p.color }"></div></div>
                  <span class="pct-val">{{ p.value }}</span>
                </div>
              </div>
            </div>
          </el-col>
        </el-row>

        <!-- 随机事件时间线 -->
        <div class="evt-block" v-if="events.length">
          <div class="evt-head">
            <span class="side-title">随机事件时间线（最近 {{ events.length }} 条）</span>
            <span class="evt-chips">
              <el-tag size="small" type="danger" effect="plain">事故 {{ f(summary.accident_count, 0) }}</el-tag>
              <el-tag size="small" type="warning" effect="plain">拒单 {{ f(summary.refuse_count, 0) }}</el-tag>
              <el-tag size="small" type="info" effect="plain">维修中 {{ f(summary.maintenance_vehicles, 0) }} 辆</el-tag>
              <el-tag size="small" type="success" effect="plain">休息 {{ f(summary.resting_drivers, 0) }} 人</el-tag>
            </span>
          </div>
          <el-timeline class="evt-timeline">
            <el-timeline-item v-for="ev in events" :key="ev.id" :timestamp="ev.event_time"
              :type="evTagType(ev.event_type)" :hollow="ev.status !== 'resolved'" placement="top">
              <span class="ev-license">{{ ev.license || ('车辆-' + ev.vehicle_id) }}</span>
              <span class="ev-desc">{{ ev.description }}</span>
              <span class="ev-dur" v-if="ev.duration_seconds > 0">持续 {{ Math.round(ev.duration_seconds / 60) }} min</span>
            </el-timeline-item>
          </el-timeline>
        </div>

        <el-collapse style="margin-top:8px">
          <el-collapse-item title="原始 summary 字段（源数据）" name="raw">
            <el-descriptions :column="3" border size="small">
              <el-descriptions-item v-for="(v, k) in summaryFlat" :key="k" :label="k">{{ v }}</el-descriptions-item>
            </el-descriptions>
          </el-collapse-item>
        </el-collapse>
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
const summary = ref({})
const summaryFlat = ref({})
const snapshots = ref([])
const convergence = ref([])
const trendRef = ref(null)
const convRef = ref(null)
const fleetRef = ref(null)
const cargoRef = ref(null)
let trendChart = null
let convChart = null
let fleetChart = null
let cargoChart = null

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

// ---- summary 全景可视化 ----
const f = (v, d = 2) => (typeof v === 'number' && Number.isFinite(v) ? Number(v.toFixed(d)) : (v ?? '—'))

// 比例条：0~1 的率值换算为百分比（越界截断），非数值返回 null 不渲染
const ratioOf = (it) => {
  const v = it.value
  if (typeof v !== 'number' || !Number.isFinite(v)) return null
  return Math.max(0, Math.min(100, v * 100))
}
// good=down 表示该指标越低越好（红色比例条），gini 超 0.4 软约束阈值转警示色
const barColor = (it) => {
  if (it.good === 'down') return '#f56c6c'
  if (it.warn !== undefined && it.value > it.warn) return '#e6a23c'
  return '#409eff'
}

const groups = computed(() => {
  const s = summary.value || {}
  const ratio = (good) => ({ ratio: true, good })
  return [
    {
      title: '吞吐与响应（T）',
      items: [
        { label: '总吞吐量 T0', value: s.total_throughput, unit: '吨', note: '全程累计完成运输吨数' },
        { label: '匹配吞吐量 T1', value: s.match_throughput, unit: '单/小时', note: '观测窗口匹配决策 / 虚拟时长' },
        { label: '响应 P50/P95/P99', values: [s.match_response_p50, s.match_response_p95, s.match_response_p99], note: '运单创建→分配车辆的分位响应(秒)' },
        { label: '匹配决策数', value: s.match_total_count, unit: '次' },
        { label: '运输效率', value: s.transport_efficiency, unit: '', ...ratio('up'), note: '按运能口径的综合利用率' },
        { label: '总运能', value: s.total_capacity_tm, unit: '吨·km', note: '累计吨公里' },
      ],
    },
    {
      title: '资源利用（U）',
      items: [
        { label: '装载率均值 U2', value: s.loading_rate_avg, ...ratio('up'), note: '实际载重/最大载重，越高越好' },
        { label: '装载率标准差', value: s.loading_rate_stddev, ...ratio('up'), note: '车队装载均衡度(越低越均衡)' },
        { label: '空驶率 U4', value: s.empty_rate, ...ratio('down'), note: '空驶里程/总里程，越低越好' },
        { label: '在线率 U3', value: s.online_rate, ...ratio('up'), note: '(运行+空闲+待发车)/总车辆' },
        { label: '基尼系数 F1', value: s.gini_coefficient, ratio: true, warn: 0.4, note: '车辆间负载均衡，软约束阈值0.4' },
        { label: '空满比 / 等运比', value: s.empty_full_ratio, unit: '', note: '空驶/重载、等待/运输高阶比例' },
      ],
    },
    {
      title: '成本效益（C）',
      items: [
        { label: '总成本 F', value: s.cost_value, unit: '¥', note: 'A–E 加权和，调度唯一优化目标' },
        { label: '单位成本 C2', value: s.unit_cost, unit: '¥/吨·km', note: '总成本/总吨公里，主目标↓' },
        { label: '基线改善 C1', value: s.baseline_improvement, unit: '%', note: 'SA相对贪心基线的成本改善' },
        { label: '待处理损耗', value: s.total_loss_th, unit: '吨·h', note: '货物积压时间加权损耗' },
      ],
    },
    {
      title: '算法效率（A）',
      items: [
        { label: '求解批次数', value: s.solve_batches, unit: '批' },
        { label: '单批均耗时', value: s.solve_time_avg_ms, unit: 'ms' },
        { label: '单批峰值耗时', value: s.solve_time_max_ms, unit: 'ms' },
        { label: '优化 gap A3', value: s.optimization_gap, unit: '%', note: 'SA解相对精确最优的差距' },
      ],
    },
    {
      title: '匹配质量（M）',
      items: [
        { label: '精确率 M1', value: s.match_precision, ...ratio('up'), note: '选中车=条件成本最优车' },
        { label: '召回率 M2', value: s.match_recall, ...ratio('up'), note: '未派单计为未命中' },
        { label: 'F1 M3', value: s.match_f1, ...ratio('up') },
        { label: 'MRR M4', value: s.match_mrr, note: '候选排名平均倒数' },
        { label: '已评估运单', value: s.match_evaluated, unit: '单' },
      ],
    },
    {
      title: '可靠性（R）',
      items: [
        { label: 'MTBF R1', value: s.mtbf_hours, unit: 'h', note: '平均无故障时间' },
        { label: 'MTTR R2', value: s.mttr_hours, unit: 'h', note: '平均修复时长' },
        { label: '服务可用性 R3', value: s.service_availability, ...ratio('up'), note: '1−维修停机/(车队×时长)' },
        { label: '事故数 R4', value: s.accident_count, unit: '次' },
      ],
    },
    {
      title: '绿色低碳（G）',
      items: [
        { label: '总碳排放 G1', value: s.carbon_total, unit: 'kg' },
        { label: '吨公里碳排 G2', value: s.carbon_per_ton_km, unit: 'kg/t·km', note: '软约束上限口径' },
      ],
    },
    {
      title: '事件与积压',
      items: [
        { label: '拒单次数 F2', value: s.refuse_count, unit: '次' },
        { label: '维修中车辆', value: s.maintenance_vehicles, unit: '辆' },
        { label: '休息中司机', value: s.resting_drivers, unit: '人' },
        { label: '待处理量', value: s.cargo_pending, unit: '吨' },
        { label: '总等待时间', value: s.total_waiting_time, unit: 'h' },
        { label: '总空驶里程', value: s.total_empty_mileage, unit: 'km' },
        { label: '装卸总耗时', value: s.work_time?.total_loading_time, unit: 'min' },
      ],
    },
  ]
})

// 右栏：响应分位条
const pctRows = computed(() => {
  const s = summary.value
  const raw = [s.match_response_p50, s.match_response_p95, s.match_response_p99]
  const max = Math.max(1, ...raw.filter((v) => typeof v === 'number' && Number.isFinite(v)))
  const colors = ['#409eff', '#e6a23c', '#f56c6c']
  return ['P50', 'P95', 'P99'].map((label, i) => ({
    label,
    value: f(raw[i], 1) + ' 秒',
    width: Math.max(2, ((raw[i] || 0) / max) * 100) + '%',
    color: colors[i],
  }))
})

const events = computed(() => (Array.isArray(summary.value.recent_events) ? summary.value.recent_events : []))
const evTagType = (t) => ({ accident: 'danger', weather: 'primary', refuse: 'warning', rest: 'success' }[t] || 'info')

const statusText = (s) => ({
  configured: '已配置', generating: '生成中', ready: '可启动',
  running: '运行中', finished: '已完成', failed: '失败', aborted: '已中止',
}[s] || s)
const formatTime = (t) => (t || '').replace('T', ' ').slice(0, 19)

const goReplay = () => router.push(`/admin/sandbox/${experiment.value.sandboxId}`)

// 类型/货类中文名映射（与实时看板口径一致）
const VEHICLE_TYPE_NAMES = { 1: '普通厢式货车', 2: '冷藏车', 3: '平板车', 4: '危化品罐车', 5: '高栏车', 6: '微型面包车' }
const CARGO_NAMES = { 1: '普通包裹', 2: '生鲜食品', 3: '大型机械', 4: '建材石料', 5: '危险化学品', 6: '农副产品', 7: '医药及疫苗', 8: '家用电器', 9: '汽车配件', 10: '生猪活禽' }
const typeName = (n) => { const m = String(n || '').match(/(\d+)$/); return (m && VEHICLE_TYPE_NAMES[+m[1]]) || n }
const cargoName = (n) => { const m = String(n || '').match(/(\d+)$/); return (m && CARGO_NAMES[+m[1]]) || n }

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
  const s = summary.value
  if (Array.isArray(s.vehicle_types) && s.vehicle_types.length && fleetRef.value) {
    fleetChart = echarts.init(fleetRef.value)
    fleetChart.setOption({
      tooltip: {},
      grid: { left: 6, right: 14, top: 22, bottom: 4, containLabel: true },
      xAxis: { type: 'category', data: s.vehicle_types.map((x) => typeName(x.name)), axisLabel: { fontSize: 10, interval: 0, rotate: 28 } },
      yAxis: { type: 'value', minInterval: 1, axisLabel: { fontSize: 10 }, splitLine: { lineStyle: { opacity: 0.2 } } },
      series: [{ type: 'bar', barWidth: 14, data: s.vehicle_types.map((x) => x.value), itemStyle: { color: '#409eff', borderRadius: [3, 3, 0, 0] } }],
    })
  }
  if (Array.isArray(s.cargo_demand) && s.cargo_demand.length && cargoRef.value) {
    cargoChart = echarts.init(cargoRef.value)
    cargoChart.setOption({
      tooltip: {},
      grid: { left: 4, right: 34, top: 6, bottom: 6, containLabel: true },
      xAxis: { type: 'value', axisLabel: { fontSize: 10 }, splitLine: { lineStyle: { opacity: 0.2 } } },
      yAxis: { type: 'category', data: s.cargo_demand.map((x) => cargoName(x.name)).reverse(), axisLabel: { fontSize: 10 } },
      series: [{ type: 'bar', barWidth: 10, data: [...s.cargo_demand.map((x) => x.value)].reverse(), itemStyle: { color: '#5470c6', borderRadius: [0, 3, 3, 0] } }],
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
      try { summary.value = JSON.parse(r.summaryJson || '{}') } catch { summary.value = {} }
      try { convergence.value = JSON.parse(r.convergence || '[]') } catch { convergence.value = [] }
      // 原始字段视图：嵌套对象拍平一层（JSON 串）
      const flat = {}
      for (const [k, v] of Object.entries(summary.value)) {
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
onUnmounted(() => {
  trendChart?.dispose(); convChart?.dispose(); fleetChart?.dispose(); cargoChart?.dispose()
})
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

.pan-head { display: flex; justify-content: space-between; align-items: baseline; }
.pan-sub { color: #909399; font-size: 12px; }

/* 分组指标卡 */
.m-group { background: #f8f9fb; border: 1px solid #ebeef5; border-radius: 8px; padding: 10px 12px; margin-bottom: 12px; }
.m-group-title { font-size: 13px; font-weight: 700; color: #2c3e50; margin-bottom: 8px; }
.m-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px 14px; }
.m-item { min-width: 0; }
.m-label { font-size: 11px; color: #909399; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.m-value { font-size: 16px; font-weight: 700; color: #303133; margin: 2px 0 4px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.m-unit { font-style: normal; font-size: 11px; font-weight: 400; color: #909399; margin-left: 4px; }

/* 右栏构成图与分位条 */
.side-block { margin-bottom: 6px; }
.side-title { font-size: 13px; font-weight: 700; color: #2c3e50; margin-bottom: 4px; }
.side-chart { height: 150px; width: 100%; }
.pct-bars { display: flex; flex-direction: column; gap: 8px; padding: 6px 2px 2px; }
.pct-row { display: flex; align-items: center; gap: 8px; }
.pct-label { font-size: 11px; color: #909399; width: 24px; flex-shrink: 0; }
.pct-track { flex: 1; height: 8px; background: #f0f2f5; border-radius: 4px; overflow: hidden; }
.pct-fill { height: 100%; border-radius: 4px; }
.pct-val { font-size: 11px; color: #606266; width: 52px; text-align: right; flex-shrink: 0; }

/* 事件时间线 */
.evt-block { margin-top: 12px; border-top: 1px solid #ebeef5; padding-top: 12px; }
.evt-head { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 8px; margin-bottom: 8px; }
.evt-chips { display: flex; gap: 8px; }
.evt-timeline { padding-left: 4px; }
.ev-license { font-weight: 600; color: #409eff; margin-right: 10px; }
.ev-desc { color: #303133; margin-right: 10px; }
.ev-dur { color: #909399; font-size: 12px; }
</style>
