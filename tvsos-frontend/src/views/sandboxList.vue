<template>
  <div class="sandbox-page">
    <div class="page-head">
      <div>
        <h2>沙箱管理</h2>
        <p class="page-sub">每个沙箱是一套独立的"世界设定"（基础设施 + 世界规则），沙箱内可创建多组实验方案与批次</p>
      </div>
      <el-button type="primary" :icon="Plus" @click="createVisible = true">新建沙箱</el-button>
    </div>

    <el-row :gutter="20" v-loading="loading">
      <el-col :span="8" v-for="sb in sandboxes" :key="sb.id">
        <el-card class="sb-card" shadow="hover" @click="openSandbox(sb)">
          <div class="sb-head">
            <span class="sb-name">{{ sb.name }}</span>
            <el-tag size="small" :type="sb.archived ? 'info' : 'success'">{{ sb.archived ? '已归档' : '活跃' }}</el-tag>
          </div>
          <p class="sb-desc">{{ sb.description || '（无描述）' }}</p>
          <div class="sb-meta">
            <span>算法版本 {{ sb.algoVersion || '-' }}</span>
            <span>创建于 {{ formatTime(sb.createdAt) }}</span>
          </div>
          <div class="sb-actions" @click.stop>
            <el-button size="small" type="primary" plain @click="openSandbox(sb)">进入沙箱</el-button>
            <el-button size="small" @click="goCompare(sb)">实验对比</el-button>
            <el-button size="small" type="danger" plain :disabled="sb.archived" @click="handleArchive(sb)">归档</el-button>
          </div>
        </el-card>
      </el-col>
      <el-col :span="24" v-if="!loading && sandboxes.length === 0">
        <el-empty description="还没有沙箱，点击右上角新建" />
      </el-col>
    </el-row>

    <!-- 新建沙箱对话框 -->
    <el-dialog v-model="createVisible" title="新建沙箱" width="640px">
      <el-form label-width="90px">
        <el-form-item label="名称" required>
          <el-input v-model="createForm.name" placeholder="如：成都城区配送沙箱" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="createForm.description" type="textarea" :rows="2" placeholder="沙箱用途说明" />
        </el-form-item>
        <el-form-item label="世界规则">
          <el-input v-model="createForm.worldRulesToml" type="textarea" :rows="12"
            placeholder="留空使用默认世界规则（地理/交通/时间/经济/订单/随机事件）" />
          <div class="form-tip">TOML 格式；留空自动生成默认值，创建后可再编辑</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { listSandboxes, createSandbox, archiveSandbox } from '@/api/sandbox'

const router = useRouter()
const sandboxes = ref([])
const loading = ref(false)
const createVisible = ref(false)
const creating = ref(false)
const createForm = ref({ name: '', description: '', worldRulesToml: '' })

const load = async () => {
  loading.value = true
  try {
    const res = await listSandboxes()
    if (res.data.code === 1) sandboxes.value = res.data.data || []
  } finally {
    loading.value = false
  }
}

const handleCreate = async () => {
  if (!createForm.value.name.trim()) {
    ElMessage.warning('请填写沙箱名称')
    return
  }
  creating.value = true
  try {
    const res = await createSandbox(createForm.value)
    if (res.data.code === 1) {
      ElMessage.success('沙箱已创建')
      createVisible.value = false
      createForm.value = { name: '', description: '', worldRulesToml: '' }
      load()
    }
  } finally {
    creating.value = false
  }
}

const handleArchive = async (sb) => {
  try {
    await ElMessageBox.confirm(`确定归档沙箱「${sb.name}」吗？归档后不再显示在默认列表。`, '归档沙箱', { type: 'warning' })
  } catch { return }
  const res = await archiveSandbox(sb.id)
  if (res.data.code === 1) {
    ElMessage.success('已归档')
    load()
  }
}

const openSandbox = (sb) => router.push(`/admin/sandbox/${sb.id}`)
const goCompare = (sb) => router.push(`/admin/sandbox/${sb.id}/compare`)
const formatTime = (t) => (t || '').replace('T', ' ').slice(0, 19)

onMounted(load)
</script>

<style scoped>
.sandbox-page { padding: 84px 32px 32px; min-height: 100vh; background: #f5f7fa; box-sizing: border-box; }
.page-head { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 20px; }
.page-sub { color: #909399; font-size: 13px; margin-top: 4px; }
.sb-card { cursor: pointer; margin-bottom: 20px; }
.sb-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.sb-name { font-size: 17px; font-weight: 600; }
.sb-desc { color: #606266; font-size: 13px; min-height: 36px; }
.sb-meta { display: flex; justify-content: space-between; color: #909399; font-size: 12px; margin: 8px 0 12px; }
.sb-actions { display: flex; gap: 8px; }
.form-tip { color: #909399; font-size: 12px; margin-top: 4px; }
</style>
