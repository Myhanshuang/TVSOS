<script setup>
/**
 * 地图数据覆盖层（凹字形布局）
 * 六个面板环绕地图四周，覆盖在地图图层之上（不改变地图窗口实际大小）：
 *   左上=运行健康图；左下=运力效率趋势图；右上=最值Top3；右中=车辆状态饼图；
 *   右下=调配效率雷达图；正下=最终优化指标值 + 随机事件流。
 * 显隐由 dataPanel Store（导航“显示数据”按钮）控制，与车辆/POI信息面板互斥：
 * 信息面板打开时本层渐出，关闭后自动渐入回落。
 */
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import * as echarts from 'echarts'
import { storeToRefs } from 'pinia'
import { useImformStore, useDataPanelStore, useUserStore } from '@/stores'
import { getRealtimeDashboard } from '@/api/report'
import { getVehiclesData } from '@/api/vehicle'

const imform = useImformStore()
const dataPanel = useDataPanelStore()
const userStore = useUserStore()
const { imformIf } = storeToRefs(imform)
const { showDataPanels } = storeToRefs(dataPanel)

/** 面板最终可见性：总开关打开 且 车辆/POI 信息面板未占用 */
const panelsShown = computed(() => showDataPanels.value && !imformIf.value)

const summary = ref({})
const trendLabels = ref([])
const trends = ref({})
const wsConnected = ref(false)

const API_CONFIG = { refreshRate: 15000, reconnectDelay: 3000 }
let statsSocket = null
let reconnectTimer = null
let refreshTimer = null

const fmtNum = (v) => new Intl.NumberFormat().format(Math.round(Number(v) || 0))
const fmt1 = (v) => Number(v || 0).toFixed(1)
const fmt2 = (v) => Number(v || 0).toFixed(2)

const healthHead = computed(() => {
  const s = summary.value || {}
  return `吞吐 ${fmtNum(s.total_throughput)}t · 积压 ${fmtNum(s.cargo_pending)}t · P95 ${fmt1(s.match_response_p95)}s`
})

// ---- 图表实例 ----
const healthRef = ref(null)
const effRef = ref(null)
const pieRef = ref(null)
const radarRef = ref(null)
const charts = { health: null, eff: null, pie: null, radar: null }

// ---- 事件标记：{ 类型: [事件虚拟时刻HH:MM:SS, ...] }，渲染时对齐到趋势标签 ----
const eventMarks = ref({ accident: [], refuse: [], weather: [] })
const seenEventIds = new Set()

const vehicleStatusMap = computed(() => {
  const map = {}
  for (const it of (summary.value.vehicle_status || [])) map[it.name] = Number(it.value) || 0
  return map
})

const VEHICLE_STATUS_TEXT = { 1: '行驶中', 2: '空闲', 3: '待发车', 4: '维修中' }

const trackEvents = () => {
  const events = Array.isArray(summary.value.recent_events) ? summary.value.recent_events : []
  for (const ev of events) {
    if (!ev || seenEventIds.has(ev.id)) continue
    seenEventIds.add(ev.id)
    const bucket = eventMarks.value[ev.event_type]
    if (!bucket) continue
    const t = String(ev.event_time || '').slice(11) // 对齐后端趋势标签 HH:MM:SS
    if (t) bucket.push(t)
  }
  for (const k of Object.keys(eventMarks.value)) {
    if (eventMarks.value[k].length > 120) eventMarks.value[k] = eventMarks.value[k].slice(-120)
  }
}

const marksToData = (bucket) => {
  const out = []
  for (const t of eventMarks.value[bucket] || []) {
    const idx = trendLabels.value.indexOf(t)
    if (idx >= 0) out.push([idx, 0])
  }
  return out
}

const initCharts = () => {
  if (!charts.health && healthRef.value) charts.health = echarts.init(healthRef.value)
  if (!charts.eff && effRef.value) charts.eff = echarts.init(effRef.value)
  if (!charts.pie && pieRef.value) charts.pie = echarts.init(pieRef.value)
  if (!charts.radar && radarRef.value) charts.radar = echarts.init(radarRef.value)
}

const renderHealth = () => {
  if (!charts.health) return
  const t = trends.value || {}
  charts.health.setOption({
    animation: false,
    grid: { left: 46, right: 40, top: 26, bottom: 22 },
    tooltip: { trigger: 'axis' },
    legend: { show: false },
    xAxis: { type: 'category', boundaryGap: false, data: trendLabels.value, axisLabel: { fontSize: 10, color: '#909399' } },
    yAxis: [
      { type: 'value', name: '吨', nameTextStyle: { fontSize: 10 }, axisLabel: { fontSize: 10 }, splitLine: { lineStyle: { opacity: 0.25 } } },
      { type: 'value', name: '秒', nameTextStyle: { fontSize: 10 }, axisLabel: { fontSize: 10 }, splitLine: { show: false } }
    ],
    series: [
      { name: '累计吞吐', type: 'line', smooth: true, symbol: 'none', data: t.throughput || [], lineStyle: { width: 2 }, areaStyle: { opacity: 0.15 }, itemStyle: { color: '#2f54eb' } },
      { name: '待处理量', type: 'line', smooth: true, symbol: 'none', data: t.pending || [], lineStyle: { width: 1.5 }, areaStyle: { opacity: 0.25 }, itemStyle: { color: '#faad14' } },
      { name: 'P95响应', type: 'line', smooth: true, symbol: 'none', yAxisIndex: 1, data: t.p95 || [], lineStyle: { width: 1.5, type: 'dashed' }, itemStyle: { color: '#f5222d' } },
      { name: '事故', type: 'scatter', symbol: 'triangle', symbolSize: 9, itemStyle: { color: '#f5222d' }, data: marksToData('accident'), tooltip: { formatter: () => '事故' } },
      { name: '拒单', type: 'scatter', symbol: 'circle', symbolSize: 7, itemStyle: { color: '#faad14' }, data: marksToData('refuse'), tooltip: { formatter: () => '拒单' } },
      { name: '天气', type: 'scatter', symbol: 'diamond', symbolSize: 8, itemStyle: { color: '#2f54eb' }, data: marksToData('weather'), tooltip: { formatter: () => '天气' } }
    ]
  }, { notMerge: true })
}

const renderEff = () => {
  if (!charts.eff) return
  const t = trends.value || {}
  const sm = vehicleStatusMap.value
  charts.eff.setOption({
    animation: false,
    legend: { top: 0, textStyle: { fontSize: 10 }, itemWidth: 12, itemHeight: 8 },
    tooltip: { trigger: 'axis' },
    grid: [
      { left: 40, right: 14, top: 24, height: '48%' },
      { left: 40, right: 14, top: '70%', bottom: 20 }
    ],
    xAxis: [
      { type: 'category', gridIndex: 0, boundaryGap: false, data: trendLabels.value, axisLabel: { fontSize: 10, color: '#909399' } },
      { type: 'category', gridIndex: 1, data: ['车辆状态'], axisLabel: { fontSize: 10 }, axisTick: { show: false } }
    ],
    yAxis: [
      { type: 'value', gridIndex: 0, max: 100, name: '%', nameTextStyle: { fontSize: 10 }, axisLabel: { fontSize: 10 }, splitLine: { lineStyle: { opacity: 0.25 } } },
      { type: 'value', gridIndex: 1, name: '辆', nameTextStyle: { fontSize: 10 }, axisLabel: { fontSize: 10 }, splitLine: { show: false }, minInterval: 1 }
    ],
    series: [
      { name: '装载率', type: 'line', smooth: true, symbol: 'none', xAxisIndex: 0, yAxisIndex: 0, data: t.loading_rate || [], lineStyle: { width: 1.8 }, itemStyle: { color: '#1890ff' } },
      { name: '空驶率', type: 'line', smooth: true, symbol: 'none', xAxisIndex: 0, yAxisIndex: 0, data: t.empty_rate || [], lineStyle: { width: 1.8 }, itemStyle: { color: '#faad14' } },
      { name: '在线率', type: 'line', smooth: true, symbol: 'none', xAxisIndex: 0, yAxisIndex: 0, data: t.online_rate || [], lineStyle: { width: 1 }, areaStyle: { opacity: 0.15 }, itemStyle: { color: '#95de64' } },
      { name: '行驶中', type: 'bar', stack: 'vs', xAxisIndex: 1, yAxisIndex: 1, barWidth: 26, data: [sm['行驶中'] || 0], itemStyle: { color: '#1890ff' } },
      { name: '空闲中', type: 'bar', stack: 'vs', xAxisIndex: 1, yAxisIndex: 1, barWidth: 26, data: [sm['空闲中'] || 0], itemStyle: { color: '#95de64' } },
      { name: '待发车', type: 'bar', stack: 'vs', xAxisIndex: 1, yAxisIndex: 1, barWidth: 26, data: [sm['待发车'] || 0], itemStyle: { color: '#13c2c2' } },
      { name: '维修中', type: 'bar', stack: 'vs', xAxisIndex: 1, yAxisIndex: 1, barWidth: 26, data: [sm['维修中'] || 0], itemStyle: { color: '#f5222d' } }
    ]
  }, { notMerge: true })
}

const renderPie = () => {
  if (!charts.pie) return
  charts.pie.setOption({
    animation: false,
    tooltip: { trigger: 'item' },
    legend: { bottom: 0, textStyle: { fontSize: 10 }, itemWidth: 12, itemHeight: 8 },
    series: [{
      type: 'pie',
      radius: ['38%', '62%'],
      center: ['50%', '44%'],
      label: { fontSize: 10, formatter: '{b}\n{c}辆' },
      data: [
        { name: '行驶中', value: vehicleStatusMap.value['行驶中'] || 0, itemStyle: { color: '#1890ff' } },
        { name: '空闲中', value: vehicleStatusMap.value['空闲中'] || 0, itemStyle: { color: '#95de64' } },
        { name: '待发车', value: vehicleStatusMap.value['待发车'] || 0, itemStyle: { color: '#13c2c2' } },
        { name: '维修中', value: vehicleStatusMap.value['维修中'] || 0, itemStyle: { color: '#f5222d' } }
      ]
    }]
  }, { notMerge: true })
}

const renderRadar = () => {
  if (!charts.radar) return
  const r = summary.value.radar || {}
  charts.radar.setOption({
    animation: false,
    tooltip: {},
    radar: {
      indicator: [
        { name: '里程利用率', max: 100 },
        { name: '装载率', max: 100 },
        { name: '派单率', max: 100 },
        { name: '货物流动率', max: 100 },
        { name: '运力利用率', max: 100 }
      ],
      radius: '60%',
      center: ['50%', '54%'],
      axisName: { fontSize: 10, color: '#606266' },
      splitNumber: 4
    },
    series: [{
      type: 'radar',
      symbolSize: 3,
      data: [{
        name: '调配效率',
        value: [
          Number(r.mileage_utilization || 0),
          Number(r.loading_rate || 0),
          Number(r.dispatch_rate || 0),
          Number(r.cargo_flow_rate || 0),
          Number(r.capacity_utilization || 0)
        ],
        lineStyle: { color: '#2f54eb', width: 2 },
        itemStyle: { color: '#2f54eb' },
        areaStyle: { opacity: 0.18 }
      }]
    }]
  }, { notMerge: true })
}

const renderCharts = () => {
  initCharts()
  renderHealth()
  renderEff()
  renderPie()
  renderRadar()
}

// ---- 快照应用（WS 推送 / 轮询兜底共用入口） ----
const applySnapshot = (data) => {
  if (!data) return
  summary.value = data.summary || {}
  trendLabels.value = Array.isArray(data.trend_labels) ? data.trend_labels : []
  trends.value = data.detail_trends || {}
  trackEvents()
  nextTick(renderCharts)
}

const getWsBase = () => {
  const envBase = import.meta.env.VITE_WS_BASE
  if (envBase) return envBase.replace(/\/$/, '')
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws`
}

// WS 握手无法自定义头，token 经 query 传递（后端 AuthRequired 兼容 ?token=）
const withToken = (url) => {
  const sep = url.includes('?') ? '&' : '?'
  return `${url}${sep}token=${userStore.token}`
}

const connectStatsSocket = () => {
  if (statsSocket) { statsSocket.close(); statsSocket = null }
  statsSocket = new WebSocket(withToken(`${getWsBase()}/stats`))
  statsSocket.onopen = () => { wsConnected.value = true }
  statsSocket.onmessage = (evt) => {
    let msg
    try { msg = JSON.parse(evt.data) } catch { return }
    if (msg?.event === 'snapshot' && msg?.payload) applySnapshot(msg.payload)
  }
  statsSocket.onclose = () => {
    wsConnected.value = false
    statsSocket = null
    if (reconnectTimer) clearTimeout(reconnectTimer)
    reconnectTimer = setTimeout(connectStatsSocket, API_CONFIG.reconnectDelay)
  }
  statsSocket.onerror = () => { statsSocket?.close() }
}

const fetchData = async () => {
  try {
    const res = await getRealtimeDashboard()
    applySnapshot(res?.data?.data)
  } catch (e) { /* 忽略瞬时错误 */ }
}

// 面板重新拉入后需要重算图表尺寸（覆盖层位移期间尺寸不变，但保险 resize 一次）
watch(panelsShown, (shown) => {
  if (shown) nextTick(() => { renderCharts(); Object.values(charts).forEach((c) => c?.resize()) })
})

const handleResize = () => { Object.values(charts).forEach((c) => c?.resize()) }

onMounted(() => {
  fetchData()
  connectStatsSocket()
  refreshTimer = setInterval(() => { if (!wsConnected.value) fetchData() }, API_CONFIG.refreshRate)
  window.addEventListener('resize', handleResize)
  nextTick(initCharts)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  if (refreshTimer) clearInterval(refreshTimer)
  if (reconnectTimer) clearTimeout(reconnectTimer)
  if (statsSocket) statsSocket.close()
  Object.keys(charts).forEach((k) => { charts[k]?.dispose(); charts[k] = null })
})

// ---- 最值统计：Top3 点击展开完整车辆列表 ----
const topDialogVisible = ref(false)
const topDialogTitle = ref('')
const topDialogRows = ref([])
const topDialogLoading = ref(false)

const openTopDialog = async (metric) => {
  topDialogTitle.value = metric === 'load' ? '车辆载货重量榜（全部车辆）' : '车辆载货时长榜（全部车辆）'
  topDialogVisible.value = true
  topDialogLoading.value = true
  try {
    const res = await getVehiclesData()
    const rows = (res?.data?.data || []).map((v) => ({
      license: v.license || ('车辆-' + v.id),
      loadTons: Number(((v.size || 0) / 1000).toFixed(2)),
      loadedHours: Number((v.loadedDuration || 0).toFixed(2)),
      statusText: VEHICLE_STATUS_TEXT[v.status] || '未知'
    }))
    rows.sort((a, b) => (metric === 'load' ? b.loadTons - a.loadTons : b.loadedHours - a.loadedHours))
    topDialogRows.value = rows
  } finally {
    topDialogLoading.value = false
  }
}
</script>

<template>
  <div class="data-overlay" :class="{ shown: panelsShown }">
    <!-- 左侧整板（全高，向右过渡滑入）：运行健康 + 运力效率趋势 -->
    <section class="block b-left" style="--d: 0ms">
      <div class="sec sec-health">
        <div class="sec-head">
          <span class="sec-title">运行健康</span>
          <span class="sec-kpi">{{ healthHead }}</span>
        </div>
        <div ref="healthRef" class="sec-chart"></div>
      </div>
      <div class="h-divider"></div>
      <div class="sec sec-eff">
        <div class="sec-head">
          <span class="sec-title">运力效率趋势</span>
          <span class="sec-sub">U2 装载率 · U4 空驶率 · U3 在线率</span>
        </div>
        <div ref="effRef" class="sec-chart"></div>
      </div>
    </section>

    <!-- 右侧整板（全高，向左过渡滑入）：最值统计 + 车辆状态饼 + 调配效率雷达 -->
    <section class="block b-right" style="--d: 140ms">
      <div class="sec sec-top">
        <div class="sec-head">
          <span class="sec-title">最值统计</span>
          <span class="sec-sub">点击查看完整列表</span>
        </div>
        <div class="top-grid">
          <div class="top-col" @click="openTopDialog('load')" title="点击查看完整车辆列表">
            <div class="top-col-title">Top 载货最重</div>
            <ul class="top-list">
              <li v-for="(v, i) in (summary.top_load_vehicles || [])" :key="'l' + i">
                <i class="rank" :class="'r' + (i + 1)">{{ i + 1 }}</i>
                <span class="plate">{{ v.license }}</span>
                <b>{{ fmt2(v.value) }} t</b>
              </li>
              <li v-if="!(summary.top_load_vehicles || []).length" class="empty">暂无数据</li>
            </ul>
          </div>
          <div class="top-col" @click="openTopDialog('time')" title="点击查看完整车辆列表">
            <div class="top-col-title">Top 载货时间最长</div>
            <ul class="top-list">
              <li v-for="(v, i) in (summary.top_time_vehicles || [])" :key="'t' + i">
                <i class="rank" :class="'r' + (i + 1)">{{ i + 1 }}</i>
                <span class="plate">{{ v.license }}</span>
                <b>{{ fmt2(v.value) }} h</b>
              </li>
              <li v-if="!(summary.top_time_vehicles || []).length" class="empty">暂无数据</li>
            </ul>
          </div>
        </div>
      </div>
      <div class="h-divider"></div>
      <div class="sec sec-pie">
        <div class="sec-head">
          <span class="sec-title">车辆状态分布</span>
        </div>
        <div ref="pieRef" class="sec-chart"></div>
      </div>
      <div class="h-divider"></div>
      <div class="sec sec-radar">
        <div class="sec-head">
          <span class="sec-title">调配效率雷达</span>
        </div>
        <div ref="radarRef" class="sec-chart"></div>
      </div>
    </section>

    <!-- 底部整板（夹于左右整板之间，向上过渡滑入）：最终优化指标值 + 随机事件流 -->
    <section class="block b-bottom" style="--d: 70ms">
      <div class="opt-half">
        <div class="sec-head">
          <span class="sec-title">最终优化指标值</span>
          <span class="sec-sub">优化目标：综合成本 F</span>
        </div>
        <div class="opt-grid">
          <div class="opt-item">
            <span class="opt-label">F 综合成本</span>
            <b class="opt-value cost">¥{{ fmtNum(summary.cost_value) }}</b>
          </div>
          <div class="opt-item">
            <span class="opt-label">C2 单位成本</span>
            <b class="opt-value">{{ fmt2(summary.unit_cost) }}</b>
          </div>
          <div class="opt-item">
            <span class="opt-label">C1 基线改善</span>
            <b class="opt-value">{{ fmt1(summary.baseline_improvement) }}%</b>
          </div>
          <div class="opt-item">
            <span class="opt-label">A3 优化 gap</span>
            <b class="opt-value">{{ fmt2(summary.optimization_gap) }}%</b>
          </div>
        </div>
      </div>
      <div class="v-divider"></div>
      <div class="event-half">
        <div class="sec-head">
          <span class="sec-title">随机事件流</span>
          <span class="sec-sub">事故 {{ fmtNum(summary.accident_count) }} · 拒单 {{ fmtNum(summary.refuse_count) }}</span>
        </div>
        <div class="event-list">
          <div v-if="!(summary.recent_events || []).length" class="event-empty">暂无随机事件记录</div>
          <div v-for="ev in (summary.recent_events || [])" :key="ev.id" class="event-item">
            <span class="ev-badge" :class="ev.event_type">{{ ev.event_type }}</span>
            <span class="ev-desc">{{ ev.description }}</span>
            <span class="ev-meta">{{ ev.license || ('车辆-' + ev.vehicle_id) }} · {{ ev.event_time }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- 完整车辆榜弹窗 -->
    <el-dialog v-model="topDialogVisible" :title="topDialogTitle" width="640px" append-to-body>
      <el-table :data="topDialogRows" v-loading="topDialogLoading" size="small" max-height="420">
        <el-table-column type="index" label="#" width="50" />
        <el-table-column prop="license" label="车牌" min-width="110" />
        <el-table-column prop="loadTons" label="当前载货 (t)" sortable min-width="110" />
        <el-table-column prop="loadedHours" label="载货时间 (h)" sortable min-width="110" />
        <el-table-column prop="statusText" label="状态" min-width="90" />
      </el-table>
    </el-dialog>
  </div>
</template>

<style scoped>
/* 覆盖层：绝对铺满地图容器，自身不拦截事件，仅各整板可交互 */
/* --side-w：左右整板统一宽度（等宽），底板 left/right 取同值以保证零空隙贴合 */
.data-overlay {
  --side-w: min(26vw, 400px);
  position: absolute;
  inset: 0;
  z-index: 600;
  pointer-events: none;
  overflow: hidden;
}

/* 三大整板：左/右全高贴边，底板夹其间——完全打开后板块之间零空隙（边缘完全贴合） */
.block {
  position: absolute;
  pointer-events: auto;
  background: rgba(255, 255, 255, 0.94);
  backdrop-filter: blur(10px);
  box-shadow: 0 0 24px rgba(0, 0, 0, 0.16);
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition: transform 0.45s cubic-bezier(0.22, 0.61, 0.36, 1), opacity 0.4s ease;
  transition-delay: 0ms;
}

.b-left {
  left: 0;
  top: 0;
  bottom: 0;
  width: var(--side-w);
  padding: 12px 14px 14px;
}

.b-right {
  right: 0;
  top: 0;
  bottom: 0;
  width: var(--side-w);
  padding: 12px 14px 14px;
}

.b-bottom {
  left: var(--side-w);
  right: var(--side-w);
  bottom: 0;
  height: 23%;
  min-height: 150px;
  flex-direction: row;
  padding: 10px 16px 12px;
}

/* 入场：左板向右滑入、底板向上滑入、右板向左滑入（错峰渐进）；退场无延迟统一渐出 */
.data-overlay.shown .block { transition-delay: var(--d, 0ms); }
.data-overlay:not(.shown) .block { opacity: 0; pointer-events: none; transition-delay: 0ms; }
.data-overlay:not(.shown) .b-left { transform: translateX(-100%); }
.data-overlay:not(.shown) .b-bottom { transform: translateY(100%); }
.data-overlay:not(.shown) .b-right { transform: translateX(100%); }

/* 板内分区 */
.sec { flex: 1; min-height: 0; display: flex; flex-direction: column; }
.sec-health { flex: 1.15; }
.sec-eff { flex: 1; }
.sec-top { flex: 0 0 auto; }
.sec-pie { flex: 1.2; }
.sec-radar { flex: 1; }

.h-divider { height: 1px; background: #ebeef5; margin: 8px 0; flex-shrink: 0; }
.v-divider { width: 1px; background: #ebeef5; margin: 0 14px; flex-shrink: 0; }

.sec-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 8px;
  margin-bottom: 4px;
  flex-shrink: 0;
}
.sec-title { font-size: 13px; font-weight: 700; color: #2c3e50; white-space: nowrap; }
.sec-kpi { font-size: 11px; color: #909399; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.sec-sub { font-size: 11px; color: #909399; white-space: nowrap; }
.sec-chart { flex: 1; min-height: 0; width: 100%; }

/* 最值统计 */
.top-grid { display: flex; gap: 12px; }
.top-col { flex: 1; min-width: 0; cursor: pointer; }
.top-col:hover .top-col-title { color: #1890ff; }
.top-col-title { font-size: 12px; font-weight: 600; color: #606266; margin-bottom: 4px; }
.top-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 6px; }
.top-list li { display: flex; align-items: center; gap: 6px; font-size: 12px; color: #303133; }
.top-list .rank {
  width: 16px; height: 16px; border-radius: 4px; font-style: normal; font-size: 11px;
  display: inline-flex; align-items: center; justify-content: center;
  background: #e4e7ed; color: #606266; flex-shrink: 0;
}
.top-list .rank.r1 { background: #f5222d; color: #fff; }
.top-list .rank.r2 { background: #fa8c16; color: #fff; }
.top-list .rank.r3 { background: #faad14; color: #fff; }
.top-list .plate { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.top-list b { font-size: 12px; }
.top-list .empty { color: #909399; font-size: 12px; }

/* 最终优化指标值 */
.opt-half { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.opt-grid { flex: 1; display: grid; grid-template-columns: 1fr 1fr; gap: 8px; align-content: center; }
.opt-item {
  background: #f5f7fa; border-radius: 8px; padding: 6px 10px;
  display: flex; flex-direction: column; gap: 2px;
}
.opt-label { font-size: 11px; color: #909399; }
.opt-value { font-size: 16px; color: #2c3e50; }
.opt-value.cost { color: #e67e22; }

/* 随机事件流 */
.event-half { flex: 1.2; min-width: 0; display: flex; flex-direction: column; }
.event-list { flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 5px; min-height: 0; }
.event-empty { font-size: 12px; color: #909399; padding: 8px 0; }
.event-item {
  display: flex; align-items: center; gap: 8px; font-size: 12px;
  background: #f5f7fa; border-radius: 6px; padding: 4px 8px;
}
.ev-badge {
  font-size: 10px; padding: 1px 8px; border-radius: 8px;
  background: #8c8c8c; color: #fff; text-transform: uppercase; flex-shrink: 0;
}
.ev-badge.refuse { background: #faad14; }
.ev-badge.rest { background: #13c2c2; }
.ev-badge.weather { background: #2f54eb; }
.ev-badge.accident { background: #f5222d; }
.ev-desc { flex: 1; color: #303133; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ev-meta { color: #909399; font-size: 11px; flex-shrink: 0; }
</style>
