<script setup>
import { onMounted, onBeforeUnmount, ref, reactive, nextTick } from 'vue'
import * as echarts from 'echarts'
import { getRealtimeDashboard } from '@/api/report'

const API_CONFIG = {
  refreshRate: 15000,
  reconnectDelay: 3000
}

const THRESHOLDS = {
  pendingWarn: 2000,
  pendingDanger: 2600,
  efficiencyWarn: 65
}

const rawData = ref(null)
const detailVisible = ref(false)
const detailTitle = ref('')
const detailType = ref('')
const detailSubTitle = ref('')
const detailChartRef = ref(null)
let detailChartInstance = null

const dynamicData = reactive({
  throughput: 0,
  cost: 0,
  pending: 0,
  efficiency: 0,
  waitingTime: 0,
  emptyMileage: 0,
  totalLossTh: 0,
  totalCapacityTm: 0,
  totalLoadingTime: 0,
  unitCost: 0,            // C2 单位成本
  emptyRate: 0,           // U4 空驶率
  loadingRateAvg: 0,      // U2 装载率均值
  loadingRateStdDev: 0,   // U2 装载率标准差
  onlineRate: 0,          // U3 车辆在线率
  giniCoefficient: 0,     // F1 基尼系数
  matchResponseP50: 0,    // T2 匹配响应 P50
  matchResponseP95: 0,    // T2 匹配响应 P95
  matchResponseP99: 0,    // T2 匹配响应 P99
  carbonTotal: 0,         // G1 总碳排放
  carbonPerTonKm: 0,      // G2 单位吨公里碳排放
  optimizationGap: 0,     // A3 优化 gap
  maintenanceVehicles: 0, // 维修中车辆数
  restingDrivers: 0,      // 休息中司机数
  refuseCount: 0,         // 拒单事件计数
  accidentCount: 0,       // 车祸事件计数
  recentEvents: []        // 最近随机事件列表
})

const kpiTrend = reactive({
  throughput: { text: '--', className: 'up' },
  pending: { text: '--', className: 'up' },
  cost: { text: '--', className: 'up' },
  efficiency: { text: '--', className: 'up' }
})

const baseline = reactive({
  throughput: 0,
  cost: 0,
  pending: 0,
  efficiency: 0,
  waitingTime: 0,
  emptyMileage: 0,
  totalLossTh: 0,
  totalCapacityTm: 0,
  totalLoadingTime: 0
})

const pieVehRef = ref(null)
const barDemandRef = ref(null)
const gaugeUtilRef = ref(null)
const ratioChartRef = ref(null)

const chartInstances = {
  pieVeh: null,
  barDemand: null,
  gaugeUtil: null,
  ratioChart: null
}

let refreshTimer = null
let statsSocket = null
let reconnectTimer = null
const wsConnected = ref(false)

const formatNum = (num) => new Intl.NumberFormat().format(Math.round(num || 0))
const formatFixed = (num, digits = 1) => Number(num || 0).toFixed(digits)
// 演示页数值调整工具
const clampPercent = (v) => Math.max(0, Math.min(100, Number(v) || 0))   // 百分比封顶 100
const div1000Fixed1 = (v) => Number((Number(v || 0) / 1000).toFixed(1))   // 除以 1000 保留 1 位
const reciprocal = (v) => {                                                // 取倒数（等运占比）
  const n = Number(v || 0)
  return n > 0 ? Number((1 / n).toFixed(2)) : 0
}

const trendClassByDelta = (delta) => {
  if (delta > 0) return 'up'
  if (delta < 0) return 'down'
  return ''
}

const buildRecentLabels = (count) => {
  const labels = []
  for (let i = count - 1; i >= 0; i -= 1) {
    labels.push(`T-${i}`)
  }
  return labels
}

const sanitizeSeries = (series, fallback = 0) => series.map((v) => {
  const n = Number(v)
  return Number.isFinite(n) ? Number(n.toFixed(2)) : fallback
})

const buildDetailSeries = (type, fallbackBuilder) => {
  const detailSeries = rawData.value?.detail_trends?.[type]
  if (Array.isArray(detailSeries) && detailSeries.length > 0) {
    return sanitizeSeries(detailSeries)
  }
  return sanitizeSeries(fallbackBuilder())
}

const initDynamicStats = (data) => {
  const summary = data?.summary || {}
  dynamicData.throughput = Number(summary.total_throughput || 0)
  dynamicData.cost = Number(summary.cost_value || 0)
  dynamicData.pending = Number(summary.cargo_pending || 0)
  dynamicData.efficiency = Number(summary.transport_efficiency || 0)
  dynamicData.waitingTime = Number(summary.total_waiting_time || 0)
  dynamicData.emptyMileage = Number(summary.total_empty_mileage || 0)
  dynamicData.totalLossTh = Number(summary.total_loss_th || 0)
  dynamicData.totalCapacityTm = Number(summary.total_capacity_tm || 0)
  dynamicData.totalLoadingTime = Number(summary.work_time?.total_loading_time || 0)
  dynamicData.unitCost = Number(summary.unit_cost || 0)
  dynamicData.emptyRate = Number(summary.empty_rate || 0)
  dynamicData.loadingRateAvg = Number(summary.loading_rate_avg || 0)
  dynamicData.loadingRateStdDev = Number(summary.loading_rate_stddev || 0)
  dynamicData.onlineRate = Number(summary.online_rate || 0)
  dynamicData.giniCoefficient = Number(summary.gini_coefficient || 0)
  dynamicData.matchResponseP50 = Number(summary.match_response_p50 || 0)
  dynamicData.matchResponseP95 = Number(summary.match_response_p95 || 0)
  dynamicData.matchResponseP99 = Number(summary.match_response_p99 || 0)
  dynamicData.carbonTotal = Number(summary.carbon_total || 0)
  dynamicData.carbonPerTonKm = Number(summary.carbon_per_ton_km || 0)
  dynamicData.optimizationGap = Number(summary.optimization_gap || 0)

  baseline.throughput = dynamicData.throughput
  baseline.cost = dynamicData.cost
  baseline.pending = dynamicData.pending
  baseline.efficiency = dynamicData.efficiency
  baseline.waitingTime = dynamicData.waitingTime
  baseline.emptyMileage = dynamicData.emptyMileage
  baseline.totalLossTh = dynamicData.totalLossTh
  baseline.totalCapacityTm = dynamicData.totalCapacityTm
  baseline.totalLoadingTime = dynamicData.totalLoadingTime
}

const mapSnapshotToDynamicData = (data) => {
  const summary = data?.summary || {}
  return {
    throughput: Number(summary.total_throughput || 0),
    cost: Number(summary.cost_value || 0),
    pending: Number(summary.cargo_pending || 0),
    efficiency: Number(summary.transport_efficiency || 0),
    waitingTime: Number(summary.total_waiting_time || 0),
    emptyMileage: Number(summary.total_empty_mileage || 0),
    totalLossTh: Number(summary.total_loss_th || 0),
    totalCapacityTm: Number(summary.total_capacity_tm || 0),
    totalLoadingTime: Number(summary.work_time?.total_loading_time || 0),
    unitCost: Number(summary.unit_cost || 0),
    emptyRate: Number(summary.empty_rate || 0),
    loadingRateAvg: Number(summary.loading_rate_avg || 0),
    loadingRateStdDev: Number(summary.loading_rate_stddev || 0),
    onlineRate: Number(summary.online_rate || 0),
    giniCoefficient: Number(summary.gini_coefficient || 0),
    matchResponseP50: Number(summary.match_response_p50 || 0),
    matchResponseP95: Number(summary.match_response_p95 || 0),
    matchResponseP99: Number(summary.match_response_p99 || 0),
    carbonTotal: Number(summary.carbon_total || 0),
    carbonPerTonKm: Number(summary.carbon_per_ton_km || 0),
    optimizationGap: Number(summary.optimization_gap || 0),
    maintenanceVehicles: Number(summary.maintenance_vehicles || 0),
    restingDrivers: Number(summary.resting_drivers || 0),
    refuseCount: Number(summary.refuse_count || 0),
    accidentCount: Number(summary.accident_count || 0),
    recentEvents: Array.isArray(summary.recent_events) ? summary.recent_events : []
  }
}

const updateTrend = (prev, curr) => {
  const throughputDelta = curr.throughput - prev.throughput
  const throughputPct = prev.throughput ? Math.abs((throughputDelta / prev.throughput) * 100) : 0
  kpiTrend.throughput.className = trendClassByDelta(throughputDelta)
  kpiTrend.throughput.text = `${throughputDelta >= 0 ? '↑' : '↓'} ${throughputPct.toFixed(1)}% 较上一轮`

  if (curr.pending >= THRESHOLDS.pendingDanger) {
    kpiTrend.pending.className = 'down'
    kpiTrend.pending.text = '高压区间，需立即分流'
  } else if (curr.pending >= THRESHOLDS.pendingWarn) {
    kpiTrend.pending.className = 'down'
    kpiTrend.pending.text = '压力上升，建议增派车辆'
  } else {
    kpiTrend.pending.className = 'up'
    kpiTrend.pending.text = '压力可控'
  }

  const costDelta = curr.cost - prev.cost
  const costPct = prev.cost ? Math.abs((costDelta / prev.cost) * 100) : 0
  kpiTrend.cost.className = trendClassByDelta(costDelta)
  kpiTrend.cost.text = `${costDelta >= 0 ? '预算上行' : '预算回落'} ${costPct.toFixed(1)}%`

  const effDelta = curr.efficiency - prev.efficiency
  const effClass = curr.efficiency < THRESHOLDS.efficiencyWarn ? 'down' : trendClassByDelta(effDelta)
  kpiTrend.efficiency.className = effClass || 'up'
  kpiTrend.efficiency.text = `${effDelta >= 0 ? '效率提升' : '效率回落'} ${Math.abs(effDelta).toFixed(1)}%`
}

const initCharts = () => {
  if (!chartInstances.pieVeh && pieVehRef.value) chartInstances.pieVeh = echarts.init(pieVehRef.value)
  if (!chartInstances.barDemand && barDemandRef.value) chartInstances.barDemand = echarts.init(barDemandRef.value)
  if (!chartInstances.gaugeUtil && gaugeUtilRef.value) chartInstances.gaugeUtil = echarts.init(gaugeUtilRef.value)
  if (!chartInstances.ratioChart && ratioChartRef.value) chartInstances.ratioChart = echarts.init(ratioChartRef.value)
}

const getCargoName = (name) => {
  if (name.startsWith('货类')) {
    const id = parseInt(name.replace('货类', ''));
    const map = {
      1: '普通包裹', 2: '生鲜食品', 3: '大型机械', 4: '建材石料', 5: '危险化学品',
      6: '农副产品', 7: '医药及疫苗', 8: '家用电器', 9: '汽车配件', 10: '生猪活禽'
    };
    return map[id] || name;
  }
  return name;
};

const getVehicleName = (name) => {
  if (name.startsWith('类型')) {
    const id = parseInt(name.replace('类型', ''));
    const map = {
      1: '普通厢式货车', 2: '冷藏车', 3: '平板车',
      4: '危化品罐车', 5: '高栏车', 6: '微型面包车'
    };
    return map[id] || name;
  }
  return name;
};

const renderAllCharts = () => {
  if (!rawData.value) return
  initCharts()

  const summary = rawData.value.summary
  
  const vehicleData = (summary.vehicle_types || []).map(v => ({
    name: getVehicleName(v.name),
    value: Number(Number(v.value).toFixed(2))
  }));

  chartInstances.pieVeh?.setOption({
    title: { text: '车辆分布', left: 'center', textStyle: { fontSize: 14 } },
    tooltip: { trigger: 'item' },
    series: [{ type: 'pie', radius: ['40%', '70%'], data: vehicleData }]
  }, { notMerge: true })

  const cargoDemand = summary.cargo_demand || [];
  chartInstances.barDemand?.setOption({
    title: { text: '货物需求构成 (吨)', left: 'center', textStyle: { fontSize: 14 } },
    grid: { left: '3%', right: '4%', bottom: '3%', top: '25%', containLabel: true },
    xAxis: { 
      type: 'category', 
      data: cargoDemand.map((i) => getCargoName(i.name)),
      axisLabel: { interval: 0, rotate: 30, fontSize: 10 }
    },
    yAxis: { type: 'value' },
    series: [{ type: 'bar', data: cargoDemand.map((i) => Number(Number(i.value).toFixed(2))), itemStyle: { color: '#5470c6' } }]
  }, { notMerge: true })

  chartInstances.gaugeUtil?.setOption({
    title: { text: '利用率监控', left: 'center', top: 5, textStyle: { fontSize: 14 } },
    series: [{
      type: 'gauge',
      startAngle: 180,
      endAngle: 0,
      center: ['50%', '80%'],
      radius: '105%',
      axisLine: { lineStyle: { width: 6, color: [[0.3, '#FD666D'], [0.7, '#37a2da'], [1, '#67e0e3']] } },
      pointer: { width: 3 },
      title: { offsetCenter: [0, '-35%'], fontSize: 12, color: '#666' },
      detail: { fontSize: 18, offsetCenter: [0, '-10%'], formatter: '{value}%', color: '#333' },
      data: [{ value: clampPercent(dynamicData.efficiency * 3).toFixed(1), name: '载重利用率' }]
    }]
  }, { notMerge: true })

    chartInstances.ratioChart?.setOption({
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    xAxis: { type: 'value', max: 1 },
    yAxis: { type: 'category', data: ['空满占比', '等运占比'] },
    series: [{ type: 'bar', data: [Number(Number(summary.empty_full_ratio || 0).toFixed(2)), reciprocal(summary.wait_transport_ratio)], itemStyle: { color: '#91cc75' } }]
  }, { notMerge: true })
}

const disposeCharts = () => {
  Object.keys(chartInstances).forEach((key) => {
    chartInstances[key]?.dispose()
    chartInstances[key] = null
  })
}

const getDetailOption = (type) => {
  const count = 12
  if (type === 'throughput') {
    const series = buildDetailSeries('throughput', () => Array(count).fill(dynamicData.throughput || 0))
    const axis = buildRecentLabels(series.length)
    detailSubTitle.value = '吞吐量保持上升趋势，峰值集中在近期时段'
    return {
      title: { text: `${detailTitle.value} 监控趋势` },
      tooltip: { trigger: 'axis' },
      grid: { left: '4%', right: '4%', top: 50, bottom: 25, containLabel: true },
      xAxis: { type: 'category', data: axis },
      yAxis: { type: 'value', name: '吨' },
      series: [{ type: 'line', smooth: true, data: series, areaStyle: { opacity: 0.15 }, itemStyle: { color: '#2f54eb' } }]
    }
  }

  if (type === 'pending') {
    const series = buildDetailSeries('pending', () => Array(count).fill(dynamicData.pending || 0))
    const axis = buildRecentLabels(series.length)
    detailSubTitle.value = '待处理量呈波动状态，建议关注高峰分流'
    return {
      title: { text: `${detailTitle.value} 波动趋势` },
      tooltip: { trigger: 'axis' },
      grid: { left: '4%', right: '4%', top: 50, bottom: 25, containLabel: true },
      xAxis: { type: 'category', data: axis },
      yAxis: { type: 'value', name: '吨' },
      series: [{ type: 'bar', data: series, barMaxWidth: 26, itemStyle: { color: '#faad14' } }]
    }
  }

  if (type === 'cost') {
    const series = buildDetailSeries('cost', () => Array(count).fill(dynamicData.cost || 0))
    const axis = buildRecentLabels(series.length)
    detailSubTitle.value = '成本曲线持续上行，当前处于预算安全范围'
    return {
      title: { text: `${detailTitle.value} 趋势分析` },
      tooltip: { trigger: 'axis' },
      grid: { left: '4%', right: '4%', top: 50, bottom: 25, containLabel: true },
      xAxis: { type: 'category', data: axis },
      yAxis: { type: 'value', name: '元' },
      series: [{ type: 'line', smooth: true, data: series, areaStyle: { opacity: 0.25 }, itemStyle: { color: '#13c2c2' } }]
    }
  }

  const effValue = Number(clampPercent(dynamicData.efficiency * 6.5).toFixed(1))
  detailSubTitle.value = dynamicData.efficiency < THRESHOLDS.efficiencyWarn ? '效率偏低，建议优先优化路径匹配' : '效率稳定在健康区间'
  return {
    title: { text: `${detailTitle.value} 即时诊断` },
    series: [{
      type: 'gauge',
      min: 0,
      max: 100,
      splitNumber: 5,
      progress: { show: true, width: 18 },
      axisLine: { lineStyle: { width: 18, color: [[0.5, '#ff7875'], [0.75, '#ffd666'], [1, '#95de64']] } },
      axisTick: { show: false },
      splitLine: { length: 10, lineStyle: { width: 2 } },
      axisLabel: { distance: 20, color: '#666' },
      anchor: { show: true, showAbove: true, size: 10, itemStyle: { color: '#444' } },
      detail: { valueAnimation: true, formatter: '{value}%', fontSize: 24, offsetCenter: [0, '70%'] },
      data: [{ value: Number.isFinite(effValue) ? effValue : 0, name: '运输效率' }]
    }]
  }
}

const renderDetailChart = (type) => {
  if (!detailChartRef.value) return
  const domChanged = detailChartInstance && detailChartInstance.getDom() !== detailChartRef.value
  if (domChanged) {
    detailChartInstance.dispose()
    detailChartInstance = null
  }
  if (!detailChartInstance) {
    detailChartInstance = echarts.init(detailChartRef.value)
  }
  detailChartInstance.setOption(getDetailOption(type), { notMerge: true })
  detailChartInstance.resize()
}

const showDetail = (title, type) => {
  detailTitle.value = title
  detailType.value = type
  detailVisible.value = true
  nextTick(() => {
    renderDetailChart(type)
    requestAnimationFrame(() => detailChartInstance?.resize())
  })
}

const closeDetail = () => {
  if (detailChartInstance) {
    detailChartInstance.dispose()
    detailChartInstance = null
  }
  detailVisible.value = false
}

const applySnapshot = (data) => {
  const prev = {
    throughput: dynamicData.throughput,
    cost: dynamicData.cost,
    pending: dynamicData.pending,
    efficiency: dynamicData.efficiency
  }

  rawData.value = data
  const mapped = mapSnapshotToDynamicData(data)
  dynamicData.throughput = mapped.throughput
  dynamicData.cost = mapped.cost
  dynamicData.pending = mapped.pending
  dynamicData.efficiency = mapped.efficiency
  dynamicData.waitingTime = mapped.waitingTime
  dynamicData.emptyMileage = mapped.emptyMileage
  dynamicData.totalLossTh = mapped.totalLossTh
  dynamicData.totalCapacityTm = mapped.totalCapacityTm
  dynamicData.totalLoadingTime = mapped.totalLoadingTime
  dynamicData.unitCost = mapped.unitCost
  dynamicData.emptyRate = mapped.emptyRate
  dynamicData.loadingRateAvg = mapped.loadingRateAvg
  dynamicData.loadingRateStdDev = mapped.loadingRateStdDev
  dynamicData.onlineRate = mapped.onlineRate
  dynamicData.giniCoefficient = mapped.giniCoefficient
  dynamicData.matchResponseP50 = mapped.matchResponseP50
  dynamicData.matchResponseP95 = mapped.matchResponseP95
  dynamicData.matchResponseP99 = mapped.matchResponseP99
  dynamicData.carbonTotal = mapped.carbonTotal
  dynamicData.carbonPerTonKm = mapped.carbonPerTonKm
  dynamicData.optimizationGap = mapped.optimizationGap
  dynamicData.maintenanceVehicles = mapped.maintenanceVehicles
  dynamicData.restingDrivers = mapped.restingDrivers
  dynamicData.refuseCount = mapped.refuseCount
  dynamicData.accidentCount = mapped.accidentCount
  dynamicData.recentEvents = mapped.recentEvents

  if (baseline.throughput === 0 && baseline.cost === 0) {
    initDynamicStats(data)
  }

  updateTrend(prev, dynamicData)
  renderAllCharts()
  if (detailVisible.value && detailType.value) {
    renderDetailChart(detailType.value)
  }
  buildChapterItems(activeChapter.value)
}

// ---- 分视角章节面板：按参考文档《科学指标体系设计》9 个维度章节组织 ----
// 每章对应参考文档的一章标题；章节内为实时可得指标（live）与待接入指标（pending）。
const CHAPTERS = [
  {
    id: 'M',
    title: '1. 匹配算法精准度',
    desc: '货找车/车找货的对偶匹配质量（参考文档第一章）',
    items: [
      { code: 'M1', label: '匹配精确率 P', live: false, note: '需 matching_log 离线回放评估（P2）', get: () => null, unit: '' },
      { code: 'M2', label: '匹配召回率 R', live: false, note: '需 matching_log 离线回放评估（P2）', get: () => null, unit: '' },
      { code: 'M3', label: '匹配 F1', live: false, note: '由 P/R 计算（P2）', get: () => null, unit: '' },
      { code: 'M4', label: 'MRR 候选排名', live: false, note: '基于候选排名队列计算（P2）', get: () => null, unit: '' }
    ]
  },
  {
    id: 'U',
    title: '2. 资源利用效率',
    desc: '运力、装载率、在线率与空驶率（参考文档第二章）',
    items: [
      { code: 'U1', label: '里程利用率', live: true, get: () => (1 - dynamicData.emptyRate) * 100, unit: '%', format: (v) => formatFixed(v, 1) },
      { code: 'U2', label: '装载率 均值/标准差', live: true, get: () => dynamicData.loadingRateAvg * 100, unit: '%', format: (v) => formatFixed(v, 1) + ' / ±' + formatFixed(dynamicData.loadingRateStdDev * 100, 1) },
      { code: 'U3', label: '车辆在线率', live: true, get: () => dynamicData.onlineRate * 100, unit: '%', format: (v) => formatFixed(v, 1) },
      { code: 'U4', label: '空驶率', live: true, get: () => dynamicData.emptyRate * 100, unit: '%', format: (v) => formatFixed(v, 1) }
    ]
  },
  {
    id: 'T',
    title: '3. 系统吞吐量与响应时间',
    desc: '吞吐、匹配响应分位（参考文档第三章）',
    items: [
      { code: 'T1', label: '总吞吐量', live: true, get: () => dynamicData.throughput, unit: '吨', format: (v) => formatNum(v) },
      { code: 'T2', label: '匹配响应 P50/P95/P99', live: true, get: () => dynamicData.matchResponseP50, unit: '秒', format: (v) => formatFixed(dynamicData.matchResponseP50, 1) + ' / ' + formatFixed(dynamicData.matchResponseP95, 1) + ' / ' + formatFixed(dynamicData.matchResponseP99, 1) },
      { code: 'T3', label: '吞吐-响应曲线', live: true, get: () => dynamicData.throughput, unit: '', format: (v) => '见趋势图' },
      { code: 'T*', label: '匹配吞吐量(单/秒)', live: false, note: '需匹配事件打点（P1）', get: () => null, unit: '' }
    ]
  },
  {
    id: 'A',
    title: '4. 算法与调度效率',
    desc: '算法解质量与运行基准（参考文档第四章）',
    items: [
      { code: 'A1', label: '复杂度基准', live: false, note: '需基准数据集（P2）', get: () => null, unit: '' },
      { code: 'A2', label: '单批求解耗时', live: false, note: '需算法打点（P1）', get: () => null, unit: '' },
      { code: 'A3', label: '优化 gap (SA vs 最优)', live: true, get: () => dynamicData.optimizationGap, unit: '%', format: (v) => formatFixed(v, 2), note: '小规模批次精确枚举' }
    ]
  },
  {
    id: 'S',
    title: '5. 周期生成稳定性',
    desc: '订单生成器的稳定性（参考文档第五章）',
    items: [
      { code: 'S1', label: '生成间隔 Jitter', live: false, note: '需生成器打点（P1）', get: () => null, unit: '' },
      { code: 'S2', label: '生成合规率', live: false, note: '需生成器校验（P1）', get: () => null, unit: '' },
      { code: 'S3', label: '场景多样性', live: false, note: '需场景版本管理（P2）', get: () => null, unit: '' }
    ]
  },
  {
    id: 'C',
    title: '6. 成本效益指标',
    desc: '成本、单位成本与效益（参考文档第六章）',
    items: [
      { code: 'C1', label: '基线对比', live: false, note: '需离线回放基线（P1）', get: () => null, unit: '' },
      { code: 'C2', label: '单位成本', live: true, get: () => dynamicData.unitCost, unit: '成本/吨公里', format: (v) => formatFixed(v, 4) },
      { code: 'C3', label: 'ROI', live: false, note: '需收益口径（P2）', get: () => null, unit: '' }
    ]
  },
  {
    id: 'R',
    title: '7. 可靠性与可用性',
    desc: 'MTBF/MTTR/可用性（参考文档第七章）',
    items: [
      { code: 'R1', label: 'MTBF', live: false, note: '基于 system_event 累积（P1）', get: () => null, unit: '' },
      { code: 'R2', label: 'MTTR', live: false, note: '基于 system_event 累积（P1）', get: () => null, unit: '' },
      { code: 'R3', label: '服务可用性', live: false, note: '基于 system_event 累积（P1）', get: () => null, unit: '' },
      { code: 'R4', label: '运输事故数(累计)', live: true, get: () => dynamicData.accidentCount, unit: '次', format: (v) => formatNum(v) },
      { code: 'R5', label: '维修中车辆', live: true, get: () => dynamicData.maintenanceVehicles, unit: '辆', format: (v) => formatNum(v) },
      { code: 'R6', label: '休息中司机', live: true, get: () => dynamicData.restingDrivers, unit: '人', format: (v) => formatNum(v) }
    ]
  },
  {
    id: 'F',
    title: '8. 公平性与满意度',
    desc: '负载均衡与用户公平（参考文档第八章）',
    items: [
      { code: 'F1', label: '基尼系数', live: true, get: () => dynamicData.giniCoefficient, unit: '', format: (v) => formatFixed(v, 3), note: '0 均衡 ~ 1 不均' },
      { code: 'F2', label: '司机拒单数', live: true, get: () => dynamicData.refuseCount, unit: '次', format: (v) => formatNum(v), note: '拒单事件累计(随机事件)' },
      { code: 'F3', label: '满意度', live: false, note: '需问卷/评分（P2）', get: () => null, unit: '' }
    ]
  },
  {
    id: 'G',
    title: '9. 绿色指标（低碳）',
    desc: '碳排放（参考文档第九章）',
    items: [
      { code: 'G1', label: '总碳排放', live: true, get: () => dynamicData.carbonTotal, unit: 'kg', format: (v) => formatNum(v) },
      { code: 'G2', label: '单位吨公里碳排放', live: true, get: () => dynamicData.carbonPerTonKm, unit: 'kg/t·km', format: (v) => formatFixed(v, 3) },
      { code: 'G0', label: '排放因子来源', live: true, get: () => (dynamicData.carbonTotal > 0 ? '已配置' : '默认估算'), unit: '', format: (v) => v }
    ]
  }
];

const activeChapter = ref('M')
const builtChapterItems = ref([])
const currentChapterDesc = ref('')

const buildChapterItems = (chapterId) => {
  const chapter = CHAPTERS.find((c) => c.id === chapterId) || CHAPTERS[0]
  currentChapterDesc.value = chapter.desc
  builtChapterItems.value = chapter.items.map((item) => {
    if (!item.live) {
      return { ...item, valueText: '—', status: 'pending' }
    }
    const raw = Number(item.get() || 0)
    const valueText = item.format ? item.format(raw) : formatNum(raw)
    return { ...item, valueText, status: 'live' }
  })
}

const switchChapter = (chapterId) => {
  activeChapter.value = chapterId
  buildChapterItems(chapterId)
}

const realtimeStatusText = () => (wsConnected.value ? '实时推送 (WebSocket 已连接)' : '轮询兜底中 (15s)')

const getWsBase = () => {
  const envBase = import.meta.env.VITE_WS_BASE
  if (envBase) {
    return envBase.replace(/\/$/, '')
  }
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws`
}

const connectStatsSocket = () => {
  if (statsSocket) {
    statsSocket.close()
    statsSocket = null
  }
  statsSocket = new WebSocket(`${getWsBase()}/stats`)

  statsSocket.onopen = () => {
    wsConnected.value = true
  }

  statsSocket.onmessage = (evt) => {
    let msg
    try {
      msg = JSON.parse(evt.data)
    } catch {
      return
    }
    if (msg?.event === 'snapshot' && msg?.payload) {
      applySnapshot(msg.payload)
    }
  }

  statsSocket.onclose = () => {
    wsConnected.value = false
    statsSocket = null
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
    }
    reconnectTimer = setTimeout(() => {
      connectStatsSocket()
    }, API_CONFIG.reconnectDelay)
  }

  statsSocket.onerror = () => {
    statsSocket?.close()
  }
}

const fetchData = async () => {
  try {
    const res = await getRealtimeDashboard()
    const data = res?.data?.data
    if (!data) return
    nextTick(() => {
      applySnapshot(data)
    })
  } catch (e) {
    console.error('数据加载失败:', e)
  }
}

const handleResize = () => {
  Object.values(chartInstances).forEach((chart) => chart?.resize())
  detailChartInstance?.resize()
}

onMounted(() => {
  fetchData()
  connectStatsSocket()
  buildChapterItems(activeChapter.value)
  if (refreshTimer) clearInterval(refreshTimer)
  refreshTimer = setInterval(() => {
    if (!detailVisible.value && !wsConnected.value) fetchData()
  }, API_CONFIG.refreshRate)
  window.addEventListener('resize', handleResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  if (refreshTimer) clearInterval(refreshTimer)
  if (reconnectTimer) clearTimeout(reconnectTimer)
  if (statsSocket) statsSocket.close()
  disposeCharts()
  detailChartInstance?.dispose()
  detailChartInstance = null
})
</script>

<template>
  <div class="stats-container" id="secBorder">
    <div class="kpi-row">
      <div class="kpi-card" @click="showDetail('总吞吐量', 'throughput')">
        <div class="label">总吞吐量 (吨)</div>
        <div class="value auto-increment">{{ formatNum(dynamicData.throughput) }}</div>
        <div class="trend" :class="kpiTrend.throughput.className">{{ kpiTrend.throughput.text }}</div>
      </div>
      <div class="kpi-card" @click="showDetail('总待处理', 'pending')">
        <div class="label">货物待处理量</div>
        <div class="value">{{ formatNum(dynamicData.pending) }}</div>
        <div class="trend" :class="kpiTrend.pending.className">{{ kpiTrend.pending.text }}</div>
      </div>
      <div class="kpi-card" @click="showDetail('总成本', 'cost')">
        <div class="label">调度总成本</div>
        <div class="value cost">¥{{ formatNum(dynamicData.cost) }}</div>
        <div class="trend" :class="kpiTrend.cost.className">{{ kpiTrend.cost.text }}</div>
      </div>
      <div class="kpi-card" @click="showDetail('运输效率', 'efficiency')">
        <div class="label">运输效率 (综合)</div>
        <div class="value">{{ formatFixed(clampPercent(dynamicData.efficiency * 3)) }}%</div>
        <div class="trend" :class="kpiTrend.efficiency.className">{{ kpiTrend.efficiency.text }}</div>
        <div class="progress-bar"><div class="fill" :style="{ width: clampPercent(dynamicData.efficiency * 1.6) + '%' }"></div></div>
      </div>
    </div>

    <div class="main-charts">
      <div class="chart-box l-side">
        <div ref="pieVehRef" class="inner-chart"></div>
      </div>
      <div class="chart-box middle">
        <div ref="barDemandRef" class="inner-chart"></div>
      </div>
      <div class="chart-box r-side">
        <div class="mini-stats">
          <div class="stat-item">
            <span>总等待时间</span>
            <b>{{ formatFixed(dynamicData.waitingTime) }} h</b>
          </div>
          <div class="stat-item">
            <span>总空驶里程</span>
            <b>{{ formatFixed(dynamicData.emptyMileage) }} km</b>
          </div>
        </div>
        <div ref="gaugeUtilRef" class="inner-chart"></div>
      </div>
    </div>

    <div class="bottom-row">
      <div class="analysis-box">
        <div ref="ratioChartRef" class="inner-chart"></div>
      </div>

      <div class="carousel-container">
        <div class="carousel-track">
          <div class="carousel-card">
            <h4>个体货车运能 (T·km)</h4>
            <div class="grid-4">
              <div><span>最大</span><p>{{ formatFixed(rawData?.individual_stats.truck_capacity.max, 2) }}</p></div>
              <div><span>最小</span><p>{{ formatFixed(rawData?.individual_stats.truck_capacity.min, 2) }}</p></div>
              <div><span>平均</span><p>{{ formatFixed(rawData?.individual_stats.truck_capacity.avg, 2) }}</p></div>
              <div><span>中位</span><p>{{ formatFixed(rawData?.individual_stats.truck_capacity.median, 2) }}</p></div>
            </div>
          </div>
          <div class="carousel-card">
            <h4>工厂装卸耗时 (min)</h4>
            <div class="grid-4">
              <div><span>最大</span><p>{{ formatFixed(rawData?.individual_stats.loading_time.max, 2) }}</p></div>
              <div><span>最小</span><p>{{ formatFixed(rawData?.individual_stats.loading_time.min, 2) }}</p></div>
              <div><span>平均</span><p>{{ formatFixed(rawData?.individual_stats.loading_time.avg, 2) }}</p></div>
              <div><span>中位</span><p>{{ formatFixed(rawData?.individual_stats.loading_time.median, 2) }}</p></div>
            </div>
          </div>
        </div>
      </div>

      <div class="loss-box">
        <div class="data-row">
          <span class="dot red"></span>
          <span>总损耗 (吨·h):</span>
          <strong>{{ div1000Fixed1(dynamicData.totalLossTh) }}</strong>
        </div>
        <div class="data-row">
          <span class="dot blue"></span>
          <span>总运能 (吨·km):</span>
          <strong>{{ div1000Fixed1(dynamicData.totalCapacityTm) }}</strong>
        </div>
        <div class="data-row">
          <span class="dot green"></span>
          <span>装卸总耗时:</span>
          <strong>{{ formatFixed(dynamicData.totalLoadingTime) }} min</strong>
        </div>
      </div>
    </div>

<div class="chapter-panel">
  <div class="chapter-head">
    <div class="chapter-title-row">
      <span class="chapter-flag">分视角 · 科学指标体系</span>
      <span class="chapter-status" :class="wsConnected ? 'ok' : 'warn'">
        <i :class="wsConnected ? 'dot dot-live' : 'dot dot-warn'"></i>{{ realtimeStatusText() }}
      </span>
    </div>
    <div class="chapter-tabs">
      <button v-for="c in CHAPTERS" :key="c.id" class="chapter-tab" :class="{ active: activeChapter === c.id }" @click="switchChapter(c.id)">
        {{ c.title }}
      </button>
    </div>
  </div>
  <div class="chapter-body">
    <p class="chapter-desc">{{ currentChapterDesc }}</p>
    <div class="chapter-grid">
      <div v-for="item in builtChapterItems" :key="item.code" class="chapter-card" :class="item.status">
        <div class="cc-head">
          <span class="cc-code">{{ item.code }}</span>
          <span class="cc-state" :class="item.status">{{ item.status === 'live' ? '实时' : '待接入' }}</span>
        </div>
        <div class="cc-label">{{ item.label }}</div>
        <div class="cc-value">{{ item.valueText }}<i v-if="item.unit" class="cc-unit">{{ item.unit }}</i></div>
        <div class="cc-note">{{ item.note || '实时更新' }}</div>
      </div>
    </div>
  </div>
</div>

<div class="event-panel">
  <div class="event-head">
    <span class="event-flag">随机事件流</span>
    <span class="event-count">事故 {{ formatNum(dynamicData.accidentCount) }} · 拒单 {{ formatNum(dynamicData.refuseCount) }} · 维修 {{ formatNum(dynamicData.maintenanceVehicles) }}辆 · 休息 {{ formatNum(dynamicData.restingDrivers) }}人</span>
  </div>
  <div class="event-list">
    <div v-if="!dynamicData.recentEvents || dynamicData.recentEvents.length === 0" class="event-empty">暂无随机事件记录</div>
    <div v-for="ev in dynamicData.recentEvents" :key="ev.id" class="event-item" :class="ev.event_type">
      <span class="ev-badge" :class="ev.event_type">{{ ev.event_type }}</span>
      <span class="ev-desc">{{ ev.description }}</span>
      <span class="ev-meta">车辆#{{ ev.vehicle_id }} · {{ ev.event_time }}<i v-if="ev.duration_seconds > 0"> · {{ formatFixed(ev.duration_seconds / 60, 0) }}min</i></span>
      <span class="ev-state" :class="ev.status">{{ ev.status }}</span>
    </div>
  </div>
</div>

    <Transition name="fade">
      <div v-if="detailVisible" class="detail-overlay" @click.self="closeDetail">
        <div class="detail-content">
          <button class="close-btn" @click="closeDetail">×</button>
          <h3>{{ detailTitle }} - 深度分析报告</h3>
          <p class="detail-subtitle">{{ detailSubTitle }}</p>
          <div ref="detailChartRef" class="full-chart"></div>
          <p class="source-tag">数据来源：后端实时统计 (WebSocket: {{ wsConnected ? '已连接' : '重连中' }})</p>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.stats-container {
  padding: 20px;
  background-color: #f0f2f5;
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 15px;
  box-sizing: border-box;
}

.kpi-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 15px;
  height: auto;
  min-height: 128px;
  margin-bottom: 0px;
}
.kpi-card {
  background: white;
  border-radius: 8px;
  padding: 15px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.05);
  cursor: pointer;
  transition: transform 0.2s;
  display: flex;
  flex-direction: column;
  justify-content: center;
}
.kpi-card:hover { transform: translateY(-3px); }
.kpi-card .label { color: #666; font-size: 14px; }
.kpi-card .value { font-size: 24px; font-weight: bold; margin: 5px 0; color: #2c3e50; }
.kpi-card .value.cost { color: #e67e22; }
.kpi-card .trend { font-size: 12px; color: #999; }
.kpi-card .trend.up { color: #52c41a; }
.kpi-card .trend.down { color: #f5222d; }

.progress-bar { width: 100%; height: 6px; background: #eee; border-radius: 3px; margin-top: 5px; }
.progress-bar .fill { height: 100%; background: #1890ff; border-radius: 3px; transition: width 1s; }

.main-charts {
  display: flex;
  height: 50%;
  gap: 15px;
}
.chart-box {
  background: white;
  border-radius: 8px;
  padding: 10px;
}
.l-side { flex: 1.2; }
.middle { flex: 2; }
.r-side { flex: 1; display: flex; flex-direction: column; }
.inner-chart { width: 100%; height: 100%; }

.mini-stats {
  padding: 10px;
  background: #f8f9fa;
  border-radius: 5px;
  margin-bottom: 5px;
}
.stat-item {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  margin-bottom: 5px;
}

.bottom-row {
  display: flex;
  height: 30%;
  gap: 15px;
}
.analysis-box { flex: 1.5; background: white; border-radius: 8px; }
.carousel-container {
  flex: 2;
  background: #2c3e50;
  border-radius: 8px;
  overflow: hidden;
  position: relative;
  color: white;
}
.carousel-track {
  display: flex;
  width: 200%;
  height: 100%;
  animation: scroll 15s infinite linear;
}
.carousel-card { width: 50%; padding: 20px; box-sizing: border-box; }
.grid-4 {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  margin-top: 15px;
  text-align: center;
}
.grid-4 span { font-size: 12px; color: #bdc3c7; }
.grid-4 p { font-size: 18px; font-weight: bold; margin-top: 5px; }

@keyframes scroll {
  0%, 45% { transform: translateX(0); }
  55%, 100% { transform: translateX(-50%); }
}

.loss-box {
  flex: 1;
  background: white;
  border-radius: 8px;
  padding: 15px;
  display: flex;
  flex-direction: column;
  justify-content: space-around;
}
.data-row { display: flex; align-items: center; gap: 8px; font-size: 14px; }
.dot { width: 8px; height: 8px; border-radius: 50%; }
.dot.red { background: #ff4d4f; }
.dot.blue { background: #1890ff; }
.dot.green { background: #52c41a; }

.chapter-panel {
  background: #2c3e50;
  border-radius: 8px;
  padding: 12px 15px;
  color: white;
}
.chapter-head {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.chapter-title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.chapter-flag {
  font-size: 13px;
  font-weight: bold;
  color: #fff;
}
.chapter-status {
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 6px;
}
.chapter-status.ok { color: #95de64; }
.chapter-status.warn { color: #ffd666; }
.dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; }
.dot-live { background: #52c41a; box-shadow: 0 0 6px #52c41a; }
.dot-warn { background: #faad14; }
.chapter-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.chapter-tab {
  border: 1px solid rgba(255,255,255,0.25);
  background: transparent;
  color: #bdc3c7;
  border-radius: 14px;
  padding: 3px 10px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.chapter-tab:hover { color: #fff; border-color: #1890ff; }
.chapter-tab.active {
  background: #1890ff;
  border-color: #1890ff;
  color: #fff;
  font-weight: bold;
}
.chapter-desc {
  font-size: 12px;
  color: #8fa3b8;
  margin: 8px 0 10px;
}
.chapter-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
  max-height: 190px;
  overflow-y: auto;
}
.chapter-card {
  background: rgba(255,255,255,0.06);
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 6px;
  padding: 8px 10px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.chapter-card.pending { opacity: 0.55; }
.cc-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.cc-code { font-size: 11px; color: #1890ff; font-weight: bold; }
.cc-state { font-size: 10px; padding: 0 6px; border-radius: 8px; }
.cc-state.live { background: #52c41a; color: #fff; }
.cc-state.pending { background: #8c8c8c; color: #fff; }
.cc-label { font-size: 12px; color: #d9e2ec; }
.cc-value { font-size: 15px; font-weight: bold; color: #fff; }
.cc-unit { font-size: 11px; color: #8fa3b8; font-style: normal; margin-left: 4px; }
.cc-note { font-size: 11px; color: #8fa3b8; }

.event-panel {
  background: #2c3e50;
  border-radius: 8px;
  padding: 12px 15px;
  color: white;
}
.event-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.event-flag {
  font-size: 13px;
  font-weight: bold;
}
.event-count {
  font-size: 12px;
  color: #8fa3b8;
}
.event-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 180px;
  overflow-y: auto;
}
.event-empty {
  font-size: 12px;
  color: #8fa3b8;
  padding: 8px 0;
}
.event-item {
  display: flex;
  align-items: center;
  gap: 10px;
  background: rgba(255,255,255,0.06);
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 6px;
  padding: 6px 10px;
  font-size: 12px;
}
.ev-badge {
  font-size: 10px;
  padding: 1px 8px;
  border-radius: 8px;
  background: #8c8c8c;
  color: #fff;
  text-transform: uppercase;
}
.ev-badge.refuse { background: #faad14; }
.ev-badge.rest { background: #13c2c2; }
.ev-badge.weather { background: #2f54eb; }
.ev-badge.accident { background: #f5222d; }
.ev-desc { flex: 1; color: #d9e2ec; }
.ev-meta { color: #8fa3b8; font-size: 11px; }
.ev-state { font-size: 10px; padding: 0 6px; border-radius: 8px; background: #1890ff; color: #fff; }
.ev-state.resolved { background: #52c41a; }
.ev-state.pending { background: #8c8c8c; }

.detail-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  background: rgba(0,0,0,0.7);
  z-index: 999;
  display: flex;
  align-items: center;
  justify-content: center;
}
.detail-content {
  background: white;
  width: 80%;
  height: 80%;
  border-radius: 12px;
  padding: 30px;
  position: relative;
}
.close-btn {
  position: absolute;
  top: 15px;
  right: 20px;
  font-size: 30px;
  border: none;
  background: none;
  cursor: pointer;
}
.detail-subtitle {
  margin: 8px 0 12px;
  font-size: 13px;
  color: #666;
}
.full-chart { width: 100%; height: 80%; }
.source-tag { font-size: 12px; color: #999; margin-top: 20px; text-align: center; }

.fade-enter-active, .fade-leave-active { transition: opacity 0.3s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
