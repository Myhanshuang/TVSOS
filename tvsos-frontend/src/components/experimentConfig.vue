<template>
  <el-dialog v-model="visible" :title="isEdit ? '编辑实验参数' : '新建实验'" width="760px" :close-on-click-modal="false">
    <el-tabs v-model="activeTab">
      <!-- 表单模式 -->
      <el-tab-pane label="表单配置" name="form">
        <el-form label-width="120px" class="cfg-form">
          <el-divider content-position="left">基础</el-divider>
          <el-form-item label="实验名称">
            <el-input v-model="form.name" placeholder="留空自动命名" />
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

          <el-divider content-position="left">优化目标权重（全局优化指标）</el-divider>
          <el-form-item label="C2 单位成本↓">
            <el-input-number v-model="form.wC2" :min="0" :step="0.1" style="width: 140px" />
          </el-form-item>
          <el-form-item label="U2 装载率↑">
            <el-input-number v-model="form.wU2" :min="0" :step="0.1" style="width: 140px" />
          </el-form-item>
          <el-form-item label="U4 空驶率↓">
            <el-input-number v-model="form.wU4" :min="0" :step="0.1" style="width: 140px" />
          </el-form-item>
          <el-form-item label="T2 响应P95↓">
            <el-input-number v-model="form.wT2" :min="0" :step="0.1" style="width: 140px" />
          </el-form-item>

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
      <el-button type="primary" :loading="saving" @click="handleSubmit(false)">仅保存</el-button>
      <el-button type="success" :loading="saving" @click="handleSubmit(true)">
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
})
const emit = defineEmits(['submit'])

const visible = ref(false)
const saving = ref(false)
const activeTab = ref('form')
const isEdit = ref(false)
const editId = ref(0)

const defaultForm = () => ({
  name: '',
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
  wC2: 1.0, wU2: 0.5, wU4: 0.3, wT2: 0.3,
  overrideEvents: false,
  refuseProb: 0.06,
  weatherProb: 0.0005,
  accidentProb: 0.0002,
  restAfterHours: 4,
})
const form = ref(defaultForm())
const tomlText = ref('')

// 表单 → TOML（提交时以 TOML 为准；表单是 TOML 的可视化投影）
const buildToml = () => {
  const f = form.value
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
weights = { C2 = ${f.wC2}, U2 = ${f.wU2}, U4 = ${f.wU4}, T2 = ${f.wT2} }
soft_constraints = { F1_gini_max = 0.4 }
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
  const str = (re) => { const m = t.match(re); return m ? m[1] : undefined }
  const v1 = num(/^seed\s*=\s*(\d+)/m); if (v1 !== undefined) f.seed = v1
  const v2 = num(/duration_hours\s*=\s*([\d.]+)/); if (v2 !== undefined) f.durationHours = v2
  const v3 = num(/speed\s*=\s*([\d.]+)/); if (v3 !== undefined) f.speed = v3
  const v4 = num(/vehicle_count\s*=\s*(\d+)/); if (v4 !== undefined) f.vehicleCount = v4
  const v5 = num(/daily_volume\s*=\s*(\d+)/); if (v5 !== undefined) f.dailyVolume = v5
  const v6 = str(/mode\s*=\s*"(\w+)"/); if (v6) f.algoMode = v6
  const v7 = num(/sa_initial_temp\s*=\s*([\d.]+)/); if (v7 !== undefined) f.saInitialTemp = v7
  const v8 = num(/sa_cooling_rate\s*=\s*([\d.]+)/); if (v8 !== undefined) f.saCoolingRate = v8
  const v9 = num(/sa_min_temp\s*=\s*([\d.]+)/); if (v9 !== undefined) f.saMinTemp = v9
  const v10 = num(/sa_iters_per_temp\s*=\s*(\d+)/); if (v10 !== undefined) f.saItersPerTemp = v10
  const v11 = num(/candidate_k\s*=\s*(\d+)/); if (v11 !== undefined) f.candidateK = v11
  const wC2 = num(/C2\s*=\s*([\d.]+)/); if (wC2 !== undefined) f.wC2 = wC2
  const wU2 = num(/U2\s*=\s*([\d.]+)/); if (wU2 !== undefined) f.wU2 = wU2
  const wU4 = num(/U4\s*=\s*([\d.]+)/); if (wU4 !== undefined) f.wU4 = wU4
  const wT2 = num(/T2\s*=\s*([\d.]+)/); if (wT2 !== undefined) f.wT2 = wT2
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
const open = ({ mode = 'create', initialToml = '', name = '', id = 0 } = {}) => {
  isEdit.value = mode === 'edit'
  editId.value = id
  form.value = defaultForm()
  if (initialToml) applyTomlToForm(initialToml)
  if (name) form.value.name = name
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
</style>
