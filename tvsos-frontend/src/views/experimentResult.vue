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
      <div class="head-actions">
        <el-dropdown @command="handleExportReport">
          <el-button type="primary">
            导出报告<el-icon class="el-icon--right"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="html">HTML 可读报告（可打印 PDF）</el-dropdown-item>
              <el-dropdown-item command="md">Markdown 报告</el-dropdown-item>
              <el-dropdown-item command="json" divided>JSON 原始全量数据</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-button type="primary" plain @click="goReplay" v-if="experiment">以此参数再次运行</el-button>
      </div>
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
import { ArrowLeft, ArrowDown } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
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

// ---- 结果格式化导出：HTML 可读报告（可打印PDF）/ Markdown / JSON 原始全量 ----
const esc = (s) => String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;')
const mdCell = (s) => String(s ?? '').replace(/\|/g, '\\|').replace(/\r?\n/g, ' ')
const safeName = (s) => String(s || '').replace(/[\\/:*?"<>|]/g, '_')

const downloadText = (filename, content, mime) => {
  const blob = new Blob([content], { type: mime + ';charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

const groupItemText = (it) => {
  if (it.values) return f(it.values[0]) + ' / ' + f(it.values[1]) + ' / ' + f(it.values[2]) + ' 秒'
  const t = f(it.value)
  return it.unit ? t + ' ' + it.unit : String(t)
}

// 收敛序列等值压缩：只在 bestCost 变化时记一行（首末必含），等值段可由相邻点推断
const compressConvergence = (arr) => {
  if (!Array.isArray(arr) || !arr.length) return null
  const points = []
  let prev = null
  arr.forEach((c, i) => {
    const v = Number(c.bestCost)
    if (prev === null || v !== prev) { points.push({ idx: i + 1, temp: c.temp, bestCost: v }); prev = v }
  })
  const lastEntry = arr[arr.length - 1]
  if (points[points.length - 1].idx !== arr.length) points.push({ idx: arr.length, temp: lastEntry.temp, bestCost: Number(lastEntry.bestCost) })
  const costs = arr.map((c) => Number(c.bestCost))
  return { total: arr.length, points, first: costs[0], min: Math.min(...costs), last: costs[costs.length - 1] }
}

const buildHtmlReport = () => {
  const e = experiment.value || {}
  const s = summary.value || {}
  const conv = compressConvergence(convergence.value)
  const out = []
  out.push('<h1>TVSOS 沙箱实验报告</h1>')
  out.push('<div class="meta">实验 <b>#' + esc(e.id) + ' ' + esc(e.name) + '</b> · 种子 ' + esc(e.seed) + ' · 状态 ' + esc(statusText(e.status)) + ' · 创建于 ' + esc(formatTime(e.createdAt)) + ' · 沙箱 #' + esc(e.sandboxId) + '<br>虚拟终点 ' + esc(s.virtual_time || '-') + '（进度 ' + Math.round((s.progress || 0) * 100) + '%） · 导出于 ' + new Date().toLocaleString('zh-CN') + '</div>')

  out.push('<h2>一、主目标指标</h2>')
  out.push('<table><tr><th>指标</th><th>值</th></tr>')
  for (const m of primaryCards.value) out.push('<tr><td>' + esc(m.label) + '</td><td class="num">' + esc(m.value) + '</td></tr>')
  out.push('</table>')

  out.push('<h2>二、终态指标全景（按维度分组）</h2>')
  for (const g of groups.value) {
    out.push('<h3>' + esc(g.title) + '</h3>')
    out.push('<table><tr><th>指标</th><th>值</th><th>口径说明</th></tr>')
    for (const it of g.items) {
      out.push('<tr><td>' + esc(it.label) + '</td><td class="num">' + esc(groupItemText(it)) + '</td><td>' + esc(it.note || '') + '</td></tr>')
    }
    out.push('</table>')
  }

  out.push('<h2>三、随机事件流（最近 ' + events.value.length + ' 条）</h2>')
  if (events.value.length) {
    out.push('<table><tr><th>虚拟时刻</th><th>类型</th><th>车牌</th><th>描述</th><th>持续</th></tr>')
    for (const ev of events.value) {
      const dur = ev.duration_seconds > 0 ? Math.round(ev.duration_seconds / 60) + ' min' : '—'
      out.push('<tr><td>' + esc(ev.event_time) + '</td><td><span class="tag ' + esc(ev.event_type) + '">' + esc(ev.event_type) + '</span></td><td>' + esc(ev.license || '车辆-' + ev.vehicle_id) + '</td><td>' + esc(ev.description) + '</td><td class="num">' + dur + '</td></tr>')
    }
    out.push('</table>')
  } else {
    out.push('<p>本实验无随机事件记录。</p>')
  }

  out.push('<h2>四、SA 收敛曲线（等值压缩）</h2>')
  if (conv) {
    out.push('<div class="meta">共 ' + conv.total + ' 条记录 · 初始 ' + conv.first + ' · 最优 ' + f(conv.min, 2) + ' · 终值 ' + conv.last + ' · bestCost 变化点 ' + conv.points.length + ' 个（等值段已压缩，完整逐条序列见 JSON 导出）</div>')
    out.push('<table><tr><th>序</th><th>温度</th><th>bestCost</th></tr>')
    for (const p of conv.points) out.push('<tr><td class="num">' + p.idx + '</td><td class="num">' + f(p.temp, 2) + '</td><td class="num">' + f(p.bestCost, 2) + '</td></tr>')
    out.push('</table>')
  } else {
    out.push('<p>无收敛记录。</p>')
  }

  out.push('<h2>五、指标时间序列</h2>')
  if (snapshots.value.length) {
    const keys = [...new Set(snapshots.value.flatMap((o) => Object.keys(o)))].filter((k) => k !== 'id' && k !== 'experimentId')
    out.push('<table><tr>' + keys.map((k) => '<th>' + esc(k) + '</th>').join('') + '</tr>')
    for (const snap of snapshots.value) out.push('<tr>' + keys.map((k) => '<td class="num">' + esc(snap[k] ?? '') + '</td>').join('') + '</tr>')
    out.push('</table>')
  } else {
    out.push('<p>本实验无快照序列数据。</p>')
  }

  out.push('<h2>六、参数快照（TOML）</h2>')
  out.push('<pre>' + esc(e.paramsToml || '') + '</pre>')
  out.push('<footer>TVSOS 数据实验沙箱 · 本报告由实验终态数据在前端生成 · 浏览器打印（Ctrl+P）可另存为 PDF</footer>')

  return '<!DOCTYPE html><html lang="zh"><head><meta charset="utf-8"><title>TVSOS 实验报告 #' + esc(e.id) + ' ' + esc(e.name) + '</title><style>'
    + 'body{font-family:"Segoe UI","Microsoft YaHei",sans-serif;max-width:960px;margin:0 auto;padding:40px 32px;color:#303133;line-height:1.6}'
    + 'h1{font-size:24px;border-bottom:3px solid #409eff;padding-bottom:8px}'
    + '.meta{color:#909399;font-size:13px;margin:8px 0 24px}'
    + 'h2{font-size:18px;margin-top:28px;border-left:4px solid #409eff;padding-left:10px}'
    + 'h3{font-size:15px;margin:16px 0 8px;color:#2c3e50}'
    + 'table{border-collapse:collapse;width:100%;margin:8px 0 16px;font-size:13px}'
    + 'th,td{border:1px solid #dcdfe6;padding:6px 10px;text-align:left;word-break:break-all}'
    + 'th{background:#f5f7fa}tr:nth-child(even) td{background:#fafafa}'
    + 'td.num{text-align:right;font-variant-numeric:tabular-nums;white-space:nowrap}'
    + '.tag{display:inline-block;padding:1px 8px;border-radius:10px;font-size:12px}'
    + '.tag.accident{background:#fef0f0;color:#f56c6c}.tag.weather{background:#ecf5ff;color:#409eff}'
    + '.tag.refuse{background:#fdf6ec;color:#e6a23c}.tag.rest{background:#f0f9eb;color:#67c23a}.tag{background:#f4f4f5;color:#909399}'
    + 'pre{background:#fafafa;border:1px solid #ebeef5;border-radius:6px;padding:12px;font-size:12px;overflow-x:auto}'
    + 'footer{margin-top:32px;color:#c0c4cc;font-size:12px;text-align:center}'
    + '@media print{body{padding:0}}'
    + '</style></head><body>' + out.join('\n') + '</body></html>'
}

const buildMarkdownReport = () => {
  const e = experiment.value || {}
  const s = summary.value || {}
  const conv = compressConvergence(convergence.value)
  const out = []
  out.push('# TVSOS 沙箱实验报告')
  out.push('')
  out.push('> 实验 #' + mdCell(e.id) + ' ' + mdCell(e.name) + ' · 种子 ' + mdCell(e.seed) + ' · 状态 ' + mdCell(statusText(e.status)) + ' · 创建于 ' + mdCell(formatTime(e.createdAt)) + ' · 沙箱 #' + mdCell(e.sandboxId))
  out.push('> 虚拟终点 ' + mdCell(s.virtual_time || '-') + '（进度 ' + Math.round((s.progress || 0) * 100) + '%） · 导出于 ' + new Date().toLocaleString('zh-CN'))
  out.push('')
  out.push('## 一、主目标指标')
  out.push('')
  out.push('| 指标 | 值 |')
  out.push('| --- | ---: |')
  for (const m of primaryCards.value) out.push('| ' + mdCell(m.label) + ' | ' + mdCell(m.value) + ' |')
  out.push('')
  out.push('## 二、终态指标全景（按维度分组）')
  for (const g of groups.value) {
    out.push('')
    out.push('### ' + mdCell(g.title))
    out.push('')
    out.push('| 指标 | 值 | 口径说明 |')
    out.push('| --- | ---: | --- |')
    for (const it of g.items) out.push('| ' + mdCell(it.label) + ' | ' + mdCell(groupItemText(it)) + ' | ' + mdCell(it.note || '') + ' |')
  }
  out.push('')
  out.push('## 三、随机事件流（最近 ' + events.value.length + ' 条）')
  out.push('')
  if (events.value.length) {
    out.push('| 虚拟时刻 | 类型 | 车牌 | 描述 | 持续 |')
    out.push('| --- | --- | --- | --- | ---: |')
    for (const ev of events.value) {
      const dur = ev.duration_seconds > 0 ? Math.round(ev.duration_seconds / 60) + ' min' : '—'
      out.push('| ' + mdCell(ev.event_time) + ' | ' + mdCell(ev.event_type) + ' | ' + mdCell(ev.license || '车辆-' + ev.vehicle_id) + ' | ' + mdCell(ev.description) + ' | ' + dur + ' |')
    }
  } else {
    out.push('本实验无随机事件记录。')
  }
  out.push('')
  out.push('## 四、SA 收敛曲线（等值压缩）')
  out.push('')
  if (conv) {
    out.push('共 ' + conv.total + ' 条记录 · 初始 ' + conv.first + ' · 最优 ' + f(conv.min, 2) + ' · 终值 ' + conv.last + ' · bestCost 变化点 ' + conv.points.length + ' 个（等值段已压缩，完整序列见 JSON 导出）')
    out.push('')
    out.push('| 序 | 温度 | bestCost |')
    out.push('| ---: | ---: | ---: |')
    for (const p of conv.points) out.push('| ' + p.idx + ' | ' + f(p.temp, 2) + ' | ' + f(p.bestCost, 2) + ' |')
  } else {
    out.push('无收敛记录。')
  }
  out.push('')
  out.push('## 五、指标时间序列')
  out.push('')
  if (snapshots.value.length) {
    const keys = [...new Set(snapshots.value.flatMap((o) => Object.keys(o)))].filter((k) => k !== 'id' && k !== 'experimentId')
    out.push('| ' + keys.join(' | ') + ' |')
    out.push('| ' + keys.map(() => '---:').join(' | ') + ' |')
    for (const snap of snapshots.value) out.push('| ' + keys.map((k) => mdCell(snap[k] ?? '')).join(' | ') + ' |')
  } else {
    out.push('本实验无快照序列数据。')
  }
  out.push('')
  out.push('## 六、参数快照（TOML）')
  out.push('')
  out.push('```toml')
  out.push(e.paramsToml || '')
  out.push('```')
  out.push('')
  out.push('---')
  out.push('*TVSOS 数据实验沙箱 · 本报告由实验终态数据在前端生成*')
  return out.join('\n')
}

const handleExportReport = (fmt) => {
  if (!experiment.value) return
  const base = '实验报告-' + experiment.value.id + '-' + safeName(experiment.value.name)
  if (fmt === 'html') {
    downloadText(base + '.html', buildHtmlReport(), 'text/html')
    ElMessage.success('HTML 报告已导出（浏览器打开，Ctrl+P 可另存 PDF）')
  } else if (fmt === 'md') {
    downloadText(base + '.md', buildMarkdownReport(), 'text/markdown')
    ElMessage.success('Markdown 报告已导出')
  } else {
    const payload = {
      generator: 'TVSOS 沙箱实验报告导出',
      exportedAt: new Date().toISOString(),
      experiment: experiment.value,
      primaryMetrics: primary.value,
      summary: summary.value,
      convergence: convergence.value,
      snapshots: snapshots.value,
    }
    downloadText(base + '.json', JSON.stringify(payload, null, 2), 'application/json')
    ElMessage.success('JSON 原始全量数据已导出')
  }
}

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
.head-actions { display: flex; gap: 8px; align-items: flex-start; }
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
