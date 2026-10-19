<template>
  <el-dialog v-model="visible" :title="dialogTitle" width="760px" :close-on-click-modal="false">
    <el-tabs v-model="activeTab">
      <!-- 表单模式 -->
      <el-tab-pane label="表单配置" name="form">
        <el-form label-width="120px" class="cfg-form">
          <el-divider content-position="left">基础</el-divider>
          <el-form-item :label="schemeMode ? '方案名' : '实验名称'">
            <el-input v-model="form.name" :placeholder="schemeMode ? '如：基准方案-20车' : '留空自动命名'" />
          </el-form-item>
          <el-form-item v-if="schemeMode" label="描述">
            <el-input v-model="form.description" type="textarea" :rows="2" placeholder="可选，方案用途说明" />
          </el-form-item>
          <el-form-item label="随机种子">
            <el-input-number v-model="form.seed" :min="0" :step="1" style="width: 220px" />
            <span class="tip">相同种子 + 相同参数 ⇒ 结果可复现</span>
          </el-form-item>

          <el-divider content-position="left">仿真</el-divider>
          <el-form-item label="虚拟时长(h)">
            <el-input-number v-model="form.durationHours" :min="0.5" :step="1" style="width: 220px" />
            <span class="tip">到达该虚拟时长实验自动完成</span>
          </el-form-item>
          <el-form-item label="时间加速比">
            <el-input-number v-model="form.speed" :min="1" :max="500" style="width: 220px" />
            <span class="tip">虚拟时间流逝速率（启动后仍可在实验页调节）</span>
          </el-form-item>

          <el-divider content-position="left">运力与负载</el-divider>
          <el-form-item label="车辆数量">
            <el-input-number v-model="form.vehicleCount" :min="0" style="width: 220px" />
            <span class="tip">0 = 使用沙箱车队模板全量</span>
          </el-form-item>
          <el-form-item label="日均订单量">
            <el-input-number v-model="form.dailyVolume" :min="0" style="width: 220px" />
            <span class="tip">0 = 沿用沙箱世界规则基准</span>
          </el-form-item>

          <el-divider content-position="left">算法超参（模拟退火）</el-divider>
          <el-form-item label="调度模式">
            <el-radio-group v-model="form.algoMode">
              <el-radio-button value="hybrid">hybrid(SA)</el-radio-button>
              <el-radio-button value="nf">nf(贪心)</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="初始温度">
            <el-input-number v-model="form.saInitialTemp" :min="1" style="width: 160px" />
          </el-form-item>
          <el-form-item label="冷却率">
            <el-input-number v-model="form.saCoolingRate" :min="0.5" :max="0.999" :step="0.01" style="width: 160px" />
          </el-form-item>
          <el-form-item label="终止温度">
            <el-input-number v-model="form.saMinTemp" :min="0.001" :step="0.01" style="width: 160px" />
          </el-form-item>
          <el-form-item label="每温迭代数">
            <el-input-number v-model="form.saItersPerTemp" :min="1" style="width: 160px" />
          </el-form-item>
          <el-form-item label="候选车辆K">
            <el-input-number v-model="form.candidateK" :min="1" style="width: 160px" />
          </el-form-item>

          <el-divider content-position="left">多目标优化（主要目标法 + 帕累托考量）</el-divider>
          <el-form-item label="优化模式">
            <el-radio-group v-model="form.objectiveMode">
              <el-radio-button value="primary">主要目标法</el-radio-button>
              <el-radio-button value="weighted">旧标量化(兼容)</el-radio-button>
            </el-radio-group>
            <span class="tip">{{ form.objectiveMode === 'primary' ? '主目标唯一硬性优化，其余指标作软性限制' : '沿用综合成本 F 单目标口径' }}</span>
          </el-form-item>
          <template v-if="form.objectiveMode === 'primary'">
            <el-form-item label="主要优化目标">
              <el-radio-group v-model="form.primaryObjective">
                <el-radio-button value="C2">C2 单位成本↓</el-radio-button>
                <el-radio-button value="U2">U2 装载率↑</el-radio-button>
                <el-radio-button value="U4">U4 空驶率↓</el-radio-button>
                <el-radio-button value="T2">T2 响应P95↓</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="劣化容差">
              <el-input-number v-model="form.objectiveTolerance" :min="0" :max="1" :step="0.05" style="width: 140px" />
              <span class="tip">非主目标相对当前值的劣化容差（自适应 ε 预算）</span>
            </el-form-item>
            <el-form-item label="绝对预算">
              <el-collapse class="budget-collapse">
                <el-collapse-item title="软约束绝对预算（缺省=自适应；0=默认；负数=关闭）" name="budgets">
                  <div class="budget-grid">
                    <div class="budget-item">
                      <span>U2 装载率下限</span>
                      <el-input-number v-model="form.budgetU2Min" :min="-1" :max="1" :step="0.05" :controls="false" />
                    </div>
                    <div class="budget-item">
                      <span>U4 空驶率上限</span>
                      <el-input-number v-model="form.budgetU4Max" :min="-1" :max="1" :step="0.05" :controls="false" />
                    </div>
                    <div class="budget-item">
                      <span>T2 响应P95上限(秒)</span>
                      <el-input-number v-model="form.budgetT2Max" :min="-1" :step="60" :controls="false" />
                    </div>
                    <div class="budget-item">
                      <span>C2 单位成本上限</span>
                      <el-input-number v-model="form.budgetC2Max" :min="-1" :step="0.1" :controls="false" />
                    </div>
                    <div class="budget-item">
                      <span>F1 基尼上限</span>
                      <el-input-number v-model="form.budgetF1Max" :min="-1" :max="1" :step="0.05" :controls="false" />
                    </div>
                    <div class="budget-item">
                      <span>G2 碳排上限</span>
                      <el-input-number v-model="form.budgetG2Max" :min="-1" :step="0.01" :controls="false" />
                    </div>
                  </div>
                </el-collapse-item>
              </el-collapse>
            </el-form-item>
          </template>

          <el-divider content-position="left">随机事件覆盖</el-divider>
          <el-form-item label="覆盖世界规则">
            <el-switch v-model="form.overrideEvents" />
            <span class="tip">默认沿用沙箱世界规则的概率模型</span>
          </el-form-item>
          <template v-if="form.overrideEvents">
            <el-form-item label="拒单概率">
              <el-input-number v-model="form.refuseProb" :min="0" :max="1" :step="0.01" style="width: 160px" />
            </el-form-item>
            <el-form-item label="天气概率/tick">
              <el-input-number v-model="form.weatherProb" :min="0" :step="0.0001" style="width: 160px" />
            </el-form-item>
            <el-form-item label="车祸概率/tick">
              <el-input-number v-model="form.accidentProb" :min="0" :step="0.0001" style="width: 160px" />
            </el-form-item>
            <el-form-item label="连续驾驶阈值(h)">
              <el-input-number v-model="form.restAfterHours" :min="0.5" :step="0.5" style="width: 160px" />
            </el-form-item>
          </template>
        </el-form>
      </el-tab-pane>

      <!-- TOML 高级模式 -->
      <el-tab-pane label="TOML 高级编辑" name="toml">
        <el-input v-model="tomlText" type="textarea" :rows="22" spellcheck="false" class="toml-editor" />
        <div class="tip" style="margin-top:6px">直接编辑实验参数 TOML 全文；切回"表单配置"会按表单内容重新生成</div>
      </el-tab-pane>
    </el-tabs>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button @click="applyToml" v-if="activeTab === 'toml'">应用到表单</el-button>
      <el-button type="primary" :loading="saving" @click="handleSubmit(false)">{{ schemeMode ? '创建方案' : '仅保存' }}</el-button>
      <el-button v-if="!schemeMode" type="success" :loading="saving" @click="handleSubmit(true)">
        {{ startLabel }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'

const props = defineProps({
  startLabel: { type: String, default: '保存并生成数据' },
  // 方案模式：新建实验方案（多一个描述字段，仅保存，不提供"生成数据并开始"入口）
  schemeMode: { type: Boolean, default: false },
})
const emit = defineEmits(['submit'])

const visible = ref(false)
const saving = ref(false)
const activeTab = ref('form')
const isEdit = ref(false)
const editId = ref(0)
const dialogTitle = computed(() => {
  if (props.schemeMode) return '新建实验方案'
  return isEdit.value ? '编辑实验参数' : '新建实验'
})

const defaultForm = () => ({
  name: '',
  description: '',
  seed: 12345,
  durationHours: 24,
  speed: 1,
  vehicleCount: 0,
  dailyVolume: 0,
  algoMode: 'hybrid',
  saInitialTemp: 1000,
  saCoolingRate: 0.95,
  saMinTemp: 0.1,
  saItersPerTemp: 50,
  candidateK: 5,
  objectiveMode: 'primary',
  primaryObjective: 'C2',
  objectiveTolerance: 0.15,
  budgetU2Min: 0,
  budgetU4Max: 0,
  budgetT2Max: 0,
  budgetC2Max: 0,
  budgetF1Max: 0.4,
  budgetG2Max: 0,
  overrideEvents: false,
  refuseProb: 0.06,
  weatherProb: 0.0005,
  accidentProb: 0.0002,
  restAfterHours: 4,
})
const form = ref(defaultForm())
const tomlText = ref('')

// 软约束绝对预算表单值 → TOML budgets 内联表（仅写非默认项：0=缺省自适应不落盘，
// 负数=显式关闭；F1 缺省 0.4 属默认值也不落盘，由后端默认参数兜底）
const budgetEntries = () => {
  const f = form.value
  const entries = []
  if (f.budgetU2Min !== 0) entries.push(`U2_min = ${f.budgetU2Min}`)
  if (f.budgetU4Max !== 0) entries.push(`U4_max = ${f.budgetU4Max}`)
  if (f.budgetT2Max !== 0) entries.push(`T2_p95_max = ${f.budgetT2Max}`)
  if (f.budgetC2Max !== 0) entries.push(`C2_max = ${f.budgetC2Max}`)
  if (f.budgetF1Max !== 0.4) entries.push(`F1_gini_max = ${f.budgetF1Max}`)
  if (f.budgetG2Max !== 0) entries.push(`G2_max = ${f.budgetG2Max}`)
  return entries
}

// 表单 → TOML（提交时以 TOML 为准；表单是 TOML 的可视化投影）
const buildToml = () => {
  const f = form.value
  let objectivesSection = `mode = "${f.objectiveMode}"`
  if (f.objectiveMode === 'primary') {
    objectivesSection += `
primary = "${f.primaryObjective}"
tolerance = ${f.objectiveTolerance}`
    const entries = budgetEntries()
    if (entries.length) objectivesSection += `\nbudgets = { ${entries.join(', ')} }`
  }
  let t = `seed = ${f.seed}

[simulation]
duration_hours = ${f.durationHours}
speed = ${f.speed}

[fleet]
vehicle_count = ${f.vehicleCount}

[orders]
daily_volume = ${f.dailyVolume}

[algorithm]
mode = "${f.algoMode}"
sa_initial_temp = ${f.saInitialTemp}
sa_cooling_rate = ${f.saCoolingRate}
sa_min_temp = ${f.saMinTemp}
sa_iters_per_temp = ${f.saItersPerTemp}
candidate_k = ${f.candidateK}

[objectives]
${objectivesSection}
`
  if (f.overrideEvents) {
    t += `
[random_events]
driver_refuse_prob = ${f.refuseProb}
weather_prob_per_tick = ${f.weatherProb}
accident_prob_per_tick = ${f.accidentProb}
rest_after_hours = ${f.restAfterHours}
rest_minutes = 20
refuse_max_per_shipment = 2
weather_delay_seconds = [1800, 7200]
maintenance_seconds = [3600, 21600]
level_factor = [1.4, 1.2, 1.0, 0.85, 0.7]
`
  }
  return t
}

// 简易 TOML → 表单（仅覆盖表单已有字段，解析失败不阻断）
const applyTomlToForm = (t) => {
  const f = form.value
  const num = (re) => { const m = t.match(re); return m ? Number(m[1]) : undefined }
  const numIn = (src, re) => { const m = src.match(re); return m ? Number(m[1]) : undefined }
  const str = (re) => { const m = t.match(re); return m ? m[1] : undefined }
  const v1 = num(/^seed\s*=\s*(\d+)/m); if (v1 !== undefined) f.seed = v1
  const v2 = num(/duration_hours\s*=\s*([\d.]+)/); if (v2 !== undefined) f.durationHours = v2
  const v3 = num(/speed\s*=\s*([\d.]+)/); if (v3 !== undefined) f.speed = v3
  const v4 = num(/vehicle_count\s*=\s*(\d+)/); if (v4 !== undefined) f.vehicleCount = v4
  const v5 = num(/daily_volume\s*=\s*(\d+)/); if (v5 !== undefined) f.dailyVolume = v5
  const v6 = str(/mode\s*=\s*"(\w+)"/); if (v6 && (v6 === 'hybrid' || v6 === 'nf')) f.algoMode = v6
  const v7 = num(/sa_initial_temp\s*=\s*([\d.]+)/); if (v7 !== undefined) f.saInitialTemp = v7
  const v8 = num(/sa_cooling_rate\s*=\s*([\d.]+)/); if (v8 !== undefined) f.saCoolingRate = v8
  const v9 = num(/sa_min_temp\s*=\s*([\d.]+)/); if (v9 !== undefined) f.saMinTemp = v9
  const v10 = num(/sa_iters_per_temp\s*=\s*(\d+)/); if (v10 !== undefined) f.saItersPerTemp = v10
  const v11 = num(/candidate_k\s*=\s*(\d+)/); if (v11 !== undefined) f.candidateK = v11
  // 多目标（主要目标法 + 帕累托考量）：在 [objectives] 节内解析
  const objPart = (t.split(/\[objectives\]/)[1] || '').split(/\n\s*\[/)[0]
  const om = objPart.match(/mode\s*=\s*"(primary|weighted)"/)
  if (om) f.objectiveMode = om[1]
  const pKey = objPart.match(/primary\s*=\s*"(C2|U2|U4|T2)"/); if (pKey) f.primaryObjective = pKey[1]
  const tol = numIn(objPart, /tolerance\s*=\s*(-?[\d.]+)/); if (tol !== undefined) f.objectiveTolerance = tol
  const bU2 = numIn(objPart, /U2_min\s*=\s*(-?[\d.]+)/); if (bU2 !== undefined) f.budgetU2Min = bU2
  const bU4 = numIn(objPart, /U4_max\s*=\s*(-?[\d.]+)/); if (bU4 !== undefined) f.budgetU4Max = bU4
  const bT2 = numIn(objPart, /T2_p95_max\s*=\s*(-?[\d.]+)/); if (bT2 !== undefined) f.budgetT2Max = bT2
  const bC2 = numIn(objPart, /C2_max\s*=\s*(-?[\d.]+)/); if (bC2 !== undefined) f.budgetC2Max = bC2
  const bF1 = numIn(objPart, /F1_gini_max\s*=\s*(-?[\d.]+)/); if (bF1 !== undefined) f.budgetF1Max = bF1
  const bG2 = numIn(objPart, /G2_max\s*=\s*(-?[\d.]+)/); if (bG2 !== undefined) f.budgetG2Max = bG2
  if (/\[random_events\]/.test(t)) {
    f.overrideEvents = true
    const rp = num(/driver_refuse_prob\s*=\s*([\d.]+)/); if (rp !== undefined) f.refuseProb = rp
    const wp = num(/weather_prob_per_tick\s*=\s*([\d.]+)/); if (wp !== undefined) f.weatherProb = wp
    const ap = num(/accident_prob_per_tick\s*=\s*([\d.]+)/); if (ap !== undefined) f.accidentProb = ap
    const rh = num(/rest_after_hours\s*=\s*([\d.]+)/); if (rh !== undefined) f.restAfterHours = rh
  } else {
    f.overrideEvents = false
  }
}

// 切到 TOML tab 时同步表单内容
watch(activeTab, (tab) => {
  if (tab === 'toml') tomlText.value = buildToml()
})

const applyToml = () => {
  applyTomlToForm(tomlText.value)
  activeTab.value = 'form'
  ElMessage.success('已应用到表单')
}

// 打开面板：mode='create' 新实验；mode='edit' 编辑已有实验参数；initialToml 预填（种子导入/再次运行）
const open = ({ mode = 'create', initialToml = '', name = '', id = 0, description = '' } = {}) => {
  isEdit.value = mode === 'edit'
  editId.value = id
  form.value = defaultForm()
  if (initialToml) applyTomlToForm(initialToml)
  if (name) form.value.name = name
  if (description) form.value.description = description
  tomlText.value = initialToml || buildToml()
  activeTab.value = 'form'
  visible.value = true
}

const handleSubmit = (andGenerate) => {
  const t = activeTab.value === 'toml' ? tomlText.value : buildToml()
  emit('submit', {
    id: editId.value,
    isEdit: isEdit.value,
    name: form.value.name,
    description: form.value.description,
    seed: form.value.seed,
    paramsToml: t,
    andGenerate,
    close: () => { visible.value = false },
    setSaving: (v) => { saving.value = v },
  })
}

defineExpose({ open })
</script>

<style scoped>
.cfg-form { max-height: 52vh; overflow-y: auto; padding-right: 12px; }
.tip { color: #909399; font-size: 12px; margin-left: 10px; }
.toml-editor :deep(textarea) { font-family: Consolas, monospace; font-size: 13px; }
.budget-collapse { width: 100%; border-top: none; }
.budget-collapse :deep(.el-collapse-item__header) { font-size: 12px; color: #909399; height: 34px; line-height: 34px; }
.budget-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px 16px; padding: 6px 2px; }
.budget-item { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: #606266; }
.budget-item :deep(.el-input-number) { width: 100%; }
</style>
