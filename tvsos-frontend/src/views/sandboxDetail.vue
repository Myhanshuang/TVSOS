<template>
  <div class="detail-page" v-loading="loading">
    <div class="page-head">
      <div>
        <el-button link :icon="ArrowLeft" @click="router.push('/admin/sandboxes')">返回沙箱列表</el-button>
        <h2>{{ sandbox?.name || '沙箱详情' }}</h2>
        <p class="page-sub">{{ sandbox?.description }}</p>
      </div>
      <div class="head-actions">
        <el-button @click="goCompare">实验对比</el-button>
        <el-button :icon="Download" @click="importVisible = true">导入</el-button>
        <el-button type="primary" :icon="Plus" @click="openCreateExperiment">新建实验</el-button>
      </div>
    </div>

    <el-row :gutter="20">
      <!-- 左：实验方案 -->
      <el-col :span="8">
        <el-card>
          <template #header>
            <div class="card-head">
              <span>实验方案（参数模板）</span>
              <el-button size="small" :icon="Plus" @click="openCreateScheme">新建方案</el-button>
            </div>
          </template>
          <div v-if="schemes.length === 0" class="empty-tip">暂无方案，可直接新建实验</div>
          <div v-for="s in schemes" :key="s.id" class="scheme-item">
            <div class="scheme-info">
              <div class="scheme-name">{{ s.name }}</div>
              <div class="scheme-desc">{{ s.description || '（无描述）' }}</div>
            </div>
            <div class="scheme-ops">
              <el-button size="small" type="primary" plain @click="runScheme(s)">再次运行</el-button>
              <el-button size="small" type="danger" plain @click="handleDeleteScheme(s)">删除</el-button>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 右：实验记录 -->
      <el-col :span="16">
        <el-card>
          <template #header><span>实验记录（批次）</span></template>
          <el-table :data="experiments" style="width: 100%" v-if="experiments.length > 0">
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="name" label="实验名" min-width="130" show-overflow-tooltip />
            <el-table-column prop="seed" label="种子" width="90" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="statusType(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="进度" width="90">
              <template #default="{ row }">
                <span v-if="row.status === 'running'">{{ Math.round((row.progress || 0) * 100) }}%</span>
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column label="创建时间" width="150">
              <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="320" fixed="right">
              <template #default="{ row }">
                <el-button v-if="row.status === 'configured'" size="small" @click="editExperiment(row)">改参数</el-button>
                <el-button v-if="['configured', 'ready'].includes(row.status)" size="small" type="warning" plain
                  @click="handleGenerate(row)">生成数据</el-button>
                <el-button v-if="row.status === 'ready'" size="small" type="success" plain
                  @click="handleStart(row)">开始实验</el-button>
                <el-button v-if="row.status === 'running'" size="small" type="primary"
                  @click="goExperiment(row)">进入实验</el-button>
                <el-button v-if="row.status === 'finished'" size="small" type="primary" plain
                  @click="goResult(row)">查看结果</el-button>
                <el-button size="small" :icon="Download" circle title="导出参数TOML"
                  @click="handleExport(row)" />
                <el-button size="small" :icon="RefreshRight" circle title="以此参数再次运行"
                  @click="rerunExperiment(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div v-else class="empty-tip">暂无实验记录，点击右上角"新建实验"开始</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 实验参数配置面板 -->
    <experiment-config ref="configPanel" @submit="handleConfigSubmit" start-label="生成数据并开始实验" />

    <!-- 新建方案：图形化表单与 TOML 共存配置（复用实验配置面板的方案模式） -->
    <experiment-config ref="schemePanel" scheme-mode @submit="handleSchemeSubmit" />

    <!-- 种子导入对话框 -->
    <el-dialog v-model="importVisible" title="导入参数 / 种子包" width="560px">
      <p class="tip">粘贴"导出参数TOML"下载的文件内容（按参数创建新实验），或完整 JSON 种子包（同参数同种子复现）；生成数据后即可运行</p>
      <el-input v-model="importText" type="textarea" :rows="10" spellcheck="false"
        placeholder='{"seed":12345,"dataVersion":1,"paramsToml":"..."}' />
      <template #footer>
        <el-button @click="importVisible = false">取消</el-button>
        <el-button type="primary" @click="handleImport">导入</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, ArrowLeft, Download, RefreshRight } from '@element-plus/icons-vue'
import experimentConfig from '@/components/experimentConfig.vue'
import {
  getSandbox, listSchemes, createScheme, deleteScheme,
  listExperiments, createExperiment, updateExperimentParams,
  generateExperimentData, startExperiment, importExperiment,
} from '@/api/sandbox'

const route = useRoute()
const router = useRouter()
const sandboxId = Number(route.params.id)

const sandbox = ref(null)
const schemes = ref([])
const experiments = ref([])
const loading = ref(false)
const configPanel = ref(null)

const schemePanel = ref(null)
const importVisible = ref(false)
const importText = ref('')

const load = async () => {
  loading.value = true
  try {
    // 三个请求各自兜错：任一瞬时网络失败只影响对应区块并自动重试一次，
    // 不再让整个 Promise.all 一起失败把页面打成空白（曾表现为退出实验后“暂无实验”）
    const fetchAll = async () => {
      const [sbRes, scRes, exRes] = await Promise.all([
        getSandbox(sandboxId).catch(() => null),
        listSchemes(sandboxId).catch(() => null),
        listExperiments(sandboxId).catch(() => null),
      ])
      if (sbRes?.data?.code === 1) sandbox.value = sbRes.data.data
      if (scRes?.data?.code === 1) schemes.value = scRes.data.data || []
      if (exRes?.data?.code === 1) experiments.value = exRes.data.data || []
      return !(sbRes?.data?.code === 1 && scRes?.data?.code === 1 && exRes?.data?.code === 1)
    }
    const failed = await fetchAll()
    if (failed) {
      // 路由切换高峰期的瞬时失败：延迟后自动重试一次
      await new Promise((resolve) => setTimeout(resolve, 600))
      const stillFailed = await fetchAll()
      if (stillFailed) ElMessage.error('沙箱数据加载失败，请刷新页面重试')
    }
  } finally {
    loading.value = false
  }
}

const statusType = (s) => ({
  configured: 'info', generating: 'warning', ready: 'warning',
  running: 'success', finished: 'primary', failed: 'danger', aborted: 'info',
}[s] || 'info')
const statusText = (s) => ({
  configured: '已配置', generating: '生成中', ready: '可启动',
  running: '运行中', finished: '已完成', failed: '失败', aborted: '已中止',
}[s] || s)
const formatTime = (t) => (t || '').replace('T', ' ').slice(0, 19)

// ---- 实验创建/编辑/运行 ----
const openCreateExperiment = () => configPanel.value.open({ mode: 'create' })
const openCreateScheme = () => schemePanel.value.open({ mode: 'create' })
const editExperiment = (row) => configPanel.value.open({ mode: 'edit', id: row.id, name: row.name, initialToml: row.paramsToml })
const rerunExperiment = (row) => configPanel.value.open({ mode: 'create', name: row.name + '-复跑', initialToml: row.paramsToml })
const runScheme = (s) => configPanel.value.open({ mode: 'create', name: s.name + '-' + new Date().toTimeString().slice(0, 5), initialToml: s.paramsToml })

const handleConfigSubmit = async ({ id, isEdit, name, seed, paramsToml, andGenerate, close, setSaving }) => {
  setSaving(true)
  try {
    let expId = id
    if (isEdit) {
      const res = await updateExperimentParams(id, { name, paramsToml, seed })
      if (res.data.code !== 1) return
    } else {
      const res = await createExperiment(sandboxId, { name, paramsToml, seed })
      if (res.data.code !== 1) return
      expId = res.data.data.id
    }
    if (andGenerate) {
      const genRes = await generateExperimentData(expId)
      if (genRes.data.code !== 1) { load(); return }
      ElMessage.success(`数据生成完成：${genRes.data.data.orderCount} 订单 / ${genRes.data.data.eventCount} 事件，正在启动实验`)
      const startRes = await startExperiment(expId)
      if (startRes.data.code === 1) {
        close()
        router.push(`/admin/experiment/${expId}`)
        return
      }
      load()
      return
    }
    ElMessage.success('已保存')
    close()
    load()
  } finally {
    setSaving(false)
  }
}

const handleGenerate = async (row) => {
  const res = await generateExperimentData(row.id)
  if (res.data.code === 1) {
    ElMessage.success(`生成完成：${res.data.data.orderCount} 订单 / ${res.data.data.eventCount} 事件`)
    load()
  }
}

const handleStart = async (row) => {
  const res = await startExperiment(row.id)
  if (res.data.code === 1) router.push(`/admin/experiment/${row.id}`)
}

// ---- 参数导出/导入 ----
// 导出参数 TOML：作为独立 .toml 文件在浏览器中下载（替代原"复制 JSON 种子包到剪贴板"）
const handleExport = (row) => {
  const toml = row.paramsToml || ''
  const safeName = (row.name || 'experiment-' + row.id).replace(/[\\/:*?"<>|]/g, '_')
  const blob = new Blob([toml], { type: 'application/toml;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = safeName + '.toml'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
  ElMessage.success('已导出参数文件：' + safeName + '.toml')
}

// 导入支持两种格式：JSON 种子包（原有）/ 参数 TOML（新导出格式，按参数直接创建实验）
const handleImport = async () => {
  const text = importText.value.trim()
  if (!text) {
    ElMessage.warning('请粘贴参数 TOML 或 JSON 种子包')
    return
  }
  const done = () => {
    importVisible.value = false
    importText.value = ''
    load()
  }
  if (text.startsWith('{')) {
    let payload
    try {
      payload = JSON.parse(text)
    } catch {
      ElMessage.error('JSON 解析失败，请检查种子包格式')
      return
    }
    const res = await importExperiment(sandboxId, payload)
    if (res.data.code === 1) {
      ElMessage.success(`已导入为实验 #${res.data.data.id}，请生成数据后启动`)
      done()
    }
    return
  }
  // 参数 TOML：种子取 TOML 首行 seed 命名，后端解析校验后创建实验
  const seedMatch = text.match(/^seed\s*=\s*(\d+)/m)
  const res = await createExperiment(sandboxId, {
    name: seedMatch ? `导入参数-${seedMatch[1]}` : '导入参数',
    paramsToml: text,
  })
  if (res.data.code === 1) {
    ElMessage.success(`已按 TOML 参数创建实验 #${res.data.data.id}，生成数据后即可运行`)
    done()
  }
}

// ---- 方案 ----
// 新建方案：图形化表单与 TOML 双向共存（experimentConfig 方案模式），提交创建参数模板
const handleSchemeSubmit = async ({ name, description, paramsToml, close, setSaving }) => {
  if (!name || !name.trim()) {
    ElMessage.warning('请填写方案名')
    return
  }
  setSaving(true)
  try {
    const res = await createScheme(sandboxId, { name, description, paramsToml })
    if (res.data.code === 1) {
      ElMessage.success('方案已创建')
      close()
      load()
    }
  } finally {
    setSaving(false)
  }
}

const handleDeleteScheme = async (s) => {
  try {
    await ElMessageBox.confirm(`确定删除方案「${s.name}」吗？已产生的实验批次不受影响。`, '删除方案', { type: 'warning' })
  } catch { return }
  const res = await deleteScheme(s.id)
  if (res.data.code === 1) {
    ElMessage.success('已删除')
    load()
  }
}

// ---- 跳转 ----
const goExperiment = (row) => router.push(`/admin/experiment/${row.id}`)
const goResult = (row) => router.push(`/admin/experiment/${row.id}/result`)
const goCompare = () => router.push(`/admin/sandbox/${sandboxId}/compare`)

onMounted(load)
</script>

<style scoped>
.detail-page { padding: 84px 32px 32px; min-height: 100vh; background: #f5f7fa; box-sizing: border-box; }
.page-head { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 20px; }
.page-head h2 { margin: 4px 0; }
.page-sub { color: #909399; font-size: 13px; }
.head-actions { display: flex; gap: 8px; }
.card-head { display: flex; justify-content: space-between; align-items: center; }
.scheme-item { display: flex; justify-content: space-between; align-items: center; padding: 10px 0; border-bottom: 1px solid #f0f0f0; }
.scheme-name { font-weight: 600; }
.scheme-desc { color: #909399; font-size: 12px; }
.scheme-ops { display: flex; gap: 6px; }
.empty-tip { color: #909399; text-align: center; padding: 24px 0; }
.tip { color: #909399; font-size: 12px; margin-bottom: 8px; }
</style>
