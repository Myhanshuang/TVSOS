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

          <el-divider content-position="left">求解算法（可更换插件）</el-divider>
          <el-form-item label="求解算法">
            <el-select v-model="form.solver" style="width: 280px">
              <el-option v-for="s in solverOptions" :key="s.value" :value="s.value" :label="s.label" />
            </el-select>
            <span class="tip">{{ solverTip }}</span>
          </el-form-item>

          <!-- SA 专有超参 -->
          <template v-if="form.solver === 'sa'">
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
          </template>

          <!-- 种群算法通用超参 -->
          <template v-if="isPopulationSolver">
            <el-form-item label="种群规模">
              <el-input-number v-model="form.population" :min="0" :step="5" style="width: 160px" />
              <span class="tip">0 = 自动（nsga3/moead 取权重向量数，其余 40）</span>
            </el-form-item>
            <el-form-item label="进化代数">
              <el-input-number v-model="form.generations" :min="1" :step="10" style="width: 160px" />
            </el-form-item>
            <el-form-item label="交叉概率">
              <el-input-number v-model="form.crossoverProb" :min="0" :max="1" :step="0.05" style="width: 160px" />
            </el-form-item>
            <el-form-item label="变异概率">
              <el-input-number v-model="form.mutationProb" :min="0" :max="1" :step="0.05" style="width: 160px" />
              <span class="tip">按每个体期望变异基因数折算为逐基因概率</span>
            </el-form-item>
          </template>

          <!-- NSGA-III / MOEA/D 专有 -->
          <template v-if="form.solver === 'nsga3' || form.solver === 'moead'">
            <el-form-item label="参考点划分数">
              <el-input-number v-model="form.refDivisions" :min="2" :step="1" style="width: 160px" />
              <span class="tip">Das-Dennis H（4 目标下 H=4 → 35 个参考点/权重）</span>
            </el-form-item>
          </template>
          <template v-if="form.solver === 'moead'">
            <el-form-item label="邻域大小">
              <el-input-number v-model="form.neighborhood" :min="2" :step="1" style="width: 160px" />
              <span class="tip">按权重欧氏距离取近邻子问题数</span>
            </el-form-item>
            <el-form-item label="聚合函数">
              <el-select v-model="form.decomposition" style="width: 180px">
                <el-option value="tchebycheff" label="tchebycheff 切比雪夫" />
                <el-option value="weighted_sum" label="weighted_sum 加权和" />
                <el-option value="pbi" label="pbi 边界惩罚" />
              </el-select>
            </el-form-item>
          </template>

          <!-- SPEA2 / MOPSO 专有 -->
          <template v-if="form.solver === 'spea2' || form.solver === 'mopso'">
            <el-form-item label="档案容量">
              <el-input-number v-model="form.archiveSize" :min="5" :step="5" style="width: 160px" />
              <span class="tip">外部非支配档案规模上限</span>
            </el-form-item>
          </template>
          <template v-if="form.solver === 'mopso'">
            <el-form-item label="惯性权重w">
              <el-input-number v-model="form.inertiaWeight" :min="0.1" :max="0.9" :step="0.05" style="width: 160px" />
            </el-form-item>
            <el-form-item label="认知系数c1">
              <el-input-number v-model="form.cognitiveCoef" :min="0.5" :max="3" :step="0.1" style="width: 160px" />
            </el-form-item>
            <el-form-item label="社会系数c2">
              <el-input-number v-model="form.socialCoef" :min="0.5" :max="3" :step="0.1" style="width: 160px" />
            </el-form-item>
          </template>
          <el-form-item v-if="form.solver === 'sa' || form.solver === 'nf'" label="候选车辆K">
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
  solver: 'sa',
  saInitialTemp: 1000,
  saCoolingRate: 0.95,
  saMinTemp: 0.1,
  saItersPerTemp: 50,
  candidateK: 5,
  population: 40,
  generations: 60,
  crossoverProb: 0.9,
  mutationProb: 0.25,
  refDivisions: 4,
  neighborhood: 10,
  decomposition: 'tchebycheff',
  archiveSize: 30,
  inertiaWeight: 0.4,
  cognitiveCoef: 1.5,
  socialCoef: 1.5,
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

// 求解器插件选项（与后端 solverRegistry 同源）
const solverOptions = [
  { value: 'sa', label: 'SA 模拟退火（ε-约束+帕累托）' },
  { value: 'nsga2', label: 'NSGA-II（非支配排序+拥挤度）' },
  { value: 'nsga3', label: 'NSGA-III（参考点小生境）' },
  { value: 'moead', label: 'MOEA/D（分解+邻域进化）' },
  { value: 'spea2', label: 'SPEA2（强度适应度档案）' },
  { value: 'mopso', label: 'MOPSO（多目标粒子群）' },
  { value: 'nf', label: 'NF 贪心首可行（确定性基线）' },
]
const isPopulationSolver = computed(() =>
  ['nsga2', 'nsga3', 'moead', 'spea2', 'mopso'].includes(form.value.solver))
const solverTip = computed(() => {
  const tips = {
    sa: '温度调度的单点搜索，接受准则叠加帕累托支配',
    nsga2: '经典多目标遗传算法，种群并行搜索 ε-可行前沿',
    nsga3: '参考点多样性保持，适合 4 目标及以上',
    moead: '按权重向量分解为标量子问题，邻域协同进化',
    spea2: '档案型进化算法，密度截断保持前沿多样性',
    mopso: '粒子群离散编码，外档案 leader 引导',
    nf: '无迭代基线，C1 改善度的参照口径',
  }
  return tips[form.value.solver] || ''
})

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

// [algorithm] 段：solver + 按算法写超参（SA 专有 + 种群通用 + 算法专有）
const algorithmSection = () => {
  const f = form.value
  let s = `solver = "${f.solver}"`
  if (f.solver === 'sa') {
    s += `\nsa_initial_temp = ${f.saInitialTemp}
sa_cooling_rate = ${f.saCoolingRate}
sa_min_temp = ${f.saMinTemp}
sa_iters_per_temp = ${f.saItersPerTemp}
candidate_k = ${f.candidateK}`
  }
  if (isPopulationSolver.value) {
    s += `\npopulation = ${f.population}
generations = ${f.generations}
crossover_prob = ${f.crossoverProb}
mutation_prob = ${f.mutationProb}`
  }
  if (f.solver === 'nsga3' || f.solver === 'moead') {
    s += `\nref_divisions = ${f.refDivisions}`
  }
  if (f.solver === 'moead') {
    s += `\nneighborhood = ${f.neighborhood}
decomposition = "${f.decomposition}"`
  }
  if (f.solver === 'spea2' || f.solver === 'mopso') {
    s += `\narchive_size = ${f.archiveSize}`
  }
  if (f.solver === 'mopso') {
    s += `\ninertia_weight = ${f.inertiaWeight}
cognitive_coef = ${f.cognitiveCoef}
social_coef = ${f.socialCoef}`
  }
  if (f.solver === 'nf') {
    s += `\ncandidate_k = ${f.candidateK}`
  }
  return s
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
${algorithmSection()}

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
  // 求解算法：在 [algorithm] 节内解析 solver；旧版 mode=nf 兼容映射
  const algoPart = (t.split(/\[algorithm\]/)[1] || '').split(/\n\s*\[/)[0]
  const sv = algoPart.match(/solver\s*=\s*"(sa|nsga2|nsga3|moead|spea2|mopso|nf)"/)
  if (sv) f.solver = sv[1]
  else if (/mode\s*=\s*"nf"/.test(algoPart)) f.solver = 'nf'
  const v7 = num(/sa_initial_temp\s*=\s*([\d.]+)/); if (v7 !== undefined) f.saInitialTemp = v7
  const v8 = num(/sa_cooling_rate\s*=\s*([\d.]+)/); if (v8 !== undefined) f.saCoolingRate = v8
  const v9 = num(/sa_min_temp\s*=\s*([\d.]+)/); if (v9 !== undefined) f.saMinTemp = v9
  const v10 = num(/sa_iters_per_temp\s*=\s*(\d+)/); if (v10 !== undefined) f.saItersPerTemp = v10
  const v11 = num(/candidate_k\s*=\s*(\d+)/); if (v11 !== undefined) f.candidateK = v11
  const pop = numIn(algoPart, /population\s*=\s*(\d+)/); if (pop !== undefined) f.population = pop
  const gen = numIn(algoPart, /generations\s*=\s*(\d+)/); if (gen !== undefined) f.generations = gen
  const cx = numIn(algoPart, /crossover_prob\s*=\s*([\d.]+)/); if (cx !== undefined) f.crossoverProb = cx
  const mut = numIn(algoPart, /mutation_prob\s*=\s*([\d.]+)/); if (mut !== undefined) f.mutationProb = mut
  const ref = numIn(algoPart, /ref_divisions\s*=\s*(\d+)/); if (ref !== undefined) f.refDivisions = ref
  const nb = numIn(algoPart, /neighborhood\s*=\s*(\d+)/); if (nb !== undefined) f.neighborhood = nb
  const dc = algoPart.match(/decomposition\s*=\s*"(tchebycheff|weighted_sum|pbi)"/); if (dc) f.decomposition = dc[1]
  const arc = numIn(algoPart, /archive_size\s*=\s*(\d+)/); if (arc !== undefined) f.archiveSize = arc
  const iw = numIn(algoPart, /inertia_weight\s*=\s*([\d.]+)/); if (iw !== undefined) f.inertiaWeight = iw
  const c1 = numIn(algoPart, /cognitive_coef\s*=\s*([\d.]+)/); if (c1 !== undefined) f.cognitiveCoef = c1
  const c2 = numIn(algoPart, /social_coef\s*=\s*([\d.]+)/); if (c2 !== undefined) f.socialCoef = c2
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
