<template>
  <div class="dashboard-page">
    <div class="page-title">
      <div class="page-title-heading">会话管理</div>
      <div class="page-title-extra">
        <el-dropdown @command="onAgentSwitch">
          <el-button>
            <el-icon><Connection /></el-icon>
            {{ currentAgentName }} <el-icon><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item v-for="a in agents" :key="a.id" :command="a.id" :disabled="!a.available">
                <el-icon v-if="a.id === app.selectedAgentId"><Check /></el-icon>
                {{ a.name }}
                <span v-if="!a.available" style="color: #999; margin-left: 4px">(不可用)</span>
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-button type="primary" :icon="Plus" @click="showNewSession = true">新建会话</el-button>
      </div>
    </div>

    <div class="stat-grid">
      <div class="stat-card tone-blue" :class="{ 'is-selected': filterByLifecycle === '' }" @click="setLifecycleFilter('')">
        <div class="stat-head">
          <span>总会话</span>
          <el-button type="primary" link size="small">查看</el-button>
        </div>
        <div class="stat-value">{{ stats.totalSessions }}</div>
        <div class="stat-desc">所有已发现的会话</div>
      </div>
      <div class="stat-card tone-green" :class="{ 'is-selected': filterByLifecycle === 'managed' }" @click="setLifecycleFilter('managed')">
        <div class="stat-head">
          <span>已接管</span>
          <el-button type="primary" link size="small">筛选</el-button>
        </div>
        <div class="stat-value">{{ stats.loadedSessions }}</div>
        <div class="stat-desc">正在由 CodexPocket 托管</div>
      </div>
      <div class="stat-card tone-cyan" :class="{ 'is-selected': filterByLifecycle === 'active' }" @click="setLifecycleFilter('active')">
        <div class="stat-head">
          <span>运行中</span>
          <el-button type="primary" link size="small">筛选</el-button>
        </div>
        <div class="stat-value">{{ stats.activeSessions }}</div>
        <div class="stat-desc">当前正在执行任务</div>
      </div>
      <div class="stat-card tone-orange" @click="$router.push('/approvals')">
        <div class="stat-head">
          <span>待审批</span>
          <el-button type="primary" link size="small">进入</el-button>
        </div>
        <div class="stat-value">{{ stats.pendingApprovals }}</div>
        <div class="stat-desc">等待审批处理的请求</div>
      </div>
    </div>

    <el-alert v-if="app.error" :title="app.error" type="error" show-icon :closable="false" style="margin-bottom: 16px" />
    <el-alert v-if="app.filteredApprovals.length > 0" type="warning" :closable="false" style="margin-bottom: 16px">
      <template #title>当前有 {{ app.filteredApprovals.length }} 个审批等待处理。
        <el-button type="primary" link @click="$router.push('/approvals')">前往审批</el-button>
      </template>
    </el-alert>

    <el-card shadow="never" style="border-radius: var(--cf-radius); margin-bottom: 18px;">
      <div class="filter-bar">
        <el-input v-model="searchQuery" placeholder="搜索会话名称、路径、分支..." prefix-icon="Search" clearable
          class="search-box" />
        <el-select v-model="filterByLifecycle" placeholder="生命周期" clearable style="width: 140px">
          <el-option label="运行中" value="active" />
          <el-option label="已接管" value="managed" />
          <el-option label="已结束" value="ended" />
          <el-option label="可接管" value="runtime_available" />
          <el-option label="已发现" value="discovered" />
          <el-option label="历史" value="history_only" />
        </el-select>
        <el-select v-model="sortBy" placeholder="排序" style="width: 140px">
          <el-option label="最近更新" value="updatedAt" />
          <el-option label="名称" value="name" />
          <el-option label="状态" value="status" />
        </el-select>
        <div class="filter-bar-right">
          <div v-if="app.sseConnected" class="sse-badge">
            <span class="sse-dot"></span>
            实时
          </div>
          <el-button :icon="Refresh" :loading="app.loading" @click="app.refreshDashboard()" circle />
        </div>
      </div>
    </el-card>

    <div class="session-results">
      <div v-if="folderGroups.length > 0" class="folder-session-list">
        <section v-for="group in folderGroups" :key="group.key" class="folder-group">
          <div class="folder-heading">
            <div class="folder-title-block">
              <div class="folder-title-row">
                <el-icon><FolderOpened /></el-icon>
                <span class="folder-title">{{ group.name }}</span>
                <el-tag size="small" type="info" round>{{ group.sessions.length }}</el-tag>
              </div>
              <div class="folder-path">{{ group.path }}</div>
            </div>
            <div class="folder-heading-actions">
              <div class="folder-stats">
                <span v-if="group.activeCount > 0" class="folder-stat is-active">{{ group.activeCount }} 运行中</span>
                <span v-if="group.managedCount > 0" class="folder-stat">{{ group.managedCount }} 已接管</span>
              </div>
              <el-button size="small" text :icon="Plus" @click.stop="openNewSessionForFolder(group.path)">
                新建对话
              </el-button>
            </div>
          </div>

          <div class="folder-session-items">
            <button
              v-for="session in group.sessions"
              :key="session.id"
              type="button"
              class="session-row"
              :class="{ 'is-running': isSessionActive(session), 'is-managed': session.loaded }"
              @click="goDetail(session.id)"
            >
              <span class="session-state-dot"></span>
              <span class="session-row-main">
                <span class="session-row-title">{{ displayName(session) }}</span>
                <span v-if="session.preview" class="session-row-preview">{{ truncateText(session.preview, 92) }}</span>
              </span>
              <span class="session-row-tags">
                <el-tag v-if="session.branch" size="small" effect="plain">{{ session.branch }}</el-tag>
                <el-tag size="small" :type="lifecycleTagType(session.lifecycleStage)" effect="light">
                  {{ lifecycleLabel(session.lifecycleStage) }}
                </el-tag>
                <el-tag :type="statusTagType(session.status, session.ended)" size="small" effect="light">
                  {{ statusLabel(session.status, session.ended, session.activeFlags?.length > 0) }}
                </el-tag>
              </span>
              <span class="session-row-time">{{ formatTimestamp(session.updatedAt) }}</span>
            </button>
          </div>
        </section>
      </div>

      <div v-else-if="!app.loading" class="session-empty-panel">
        <el-empty description="没有匹配的会话" />
      </div>
    </div>

    <el-dialog v-model="showNewSession" title="新建会话" width="520px" class="new-session-dialog" :close-on-click-modal="false">
      <el-form :model="newForm" label-width="80px">
        <el-form-item label="工作目录">
          <div class="cwd-field">
            <el-input v-model="newForm.cwd" placeholder="D:\project\myapp" />
            <el-button :icon="FolderOpened" @click="showDirectoryPicker = true">选择目录</el-button>
          </div>
        </el-form-item>
        <el-form-item label="提示词">
          <el-input v-model="newForm.prompt" type="textarea" :rows="3" placeholder="输入你的指令..." />
        </el-form-item>
        <el-form-item label="Agent">
          <el-select v-model="newForm.agentId">
            <el-option v-for="a in agents" :key="a.id" :label="a.name" :value="a.id" :disabled="!a.available" />
          </el-select>
        </el-form-item>
        <el-form-item label="模式">
          <el-segmented v-model="newForm.preset" :options="presetSegmentOptions" @change="applyPreset" />
        </el-form-item>
        <el-collapse class="advanced-session-options">
          <el-collapse-item name="advanced" title="高级选项">
            <div class="advanced-grid">
              <label>
                <span>模型</span>
                <el-select v-model="newForm.model" placeholder="默认模型">
                  <el-option v-for="m in sessionOptions.models" :key="m.id || 'default'" :label="m.name" :value="m.id" />
                </el-select>
              </label>
              <label>
                <span>推理强度</span>
                <el-select v-model="newForm.reasoningEffort" placeholder="默认">
                  <el-option v-for="r in sessionOptions.reasoningEfforts" :key="r.id || 'default'" :label="r.name" :value="r.id" />
                </el-select>
              </label>
              <label>
                <span>协作模式</span>
                <el-select v-model="newForm.collaborationMode" placeholder="默认协作">
                  <el-option v-for="m in sessionOptions.collaborationModes" :key="m.id" :label="m.name" :value="m.id" />
                </el-select>
              </label>
            </div>
          </el-collapse-item>
        </el-collapse>
      </el-form>
      <template #footer>
        <el-button @click="showNewSession = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>

    <DirectoryPickerDialog
      v-model="showDirectoryPicker"
      :initial-path="newForm.cwd"
      @select="onDirectorySelected"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useAppStore, type SessionSummary } from '../stores/app'
import { Refresh, Plus, FolderOpened } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import {
  formatTimestamp, statusTagType, statusLabel, lifecycleLabel,
  lifecycleTagType, truncateText, sessionDisplayName,
} from '../utils/helpers'
import DirectoryPickerDialog from '../components/DirectoryPickerDialog.vue'

const router = useRouter()
const app = useAppStore()
const showNewSession = ref(false)
const showDirectoryPicker = ref(false)
const creating = ref(false)
const newForm = reactive({
  cwd: '',
  prompt: '',
  agentId: 'codex',
  preset: 'balanced',
  model: '',
  reasoningEffort: 'medium',
  collaborationMode: 'default',
})
const searchQuery = ref('')
const filterByLifecycle = ref('')
const sortBy = ref('updatedAt')

const stats = computed(() => app.dashboard.stats)
const agents = computed(() => app.dashboard.agents || [])
const sessionOptions = computed(() => app.dashboard.options)
const presetSegmentOptions = computed(() => sessionOptions.value.presets.map((preset) => ({
  label: preset.name,
  value: preset.id,
})))
const currentAgentName = computed(() => {
  const a = agents.value.find((a) => a.id === app.selectedAgentId)
  return a?.name || 'Codex'
})

const filteredAndSearchedSessions = computed(() => {
  let sessions = app.filteredSessions
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase()
    sessions = sessions.filter((s) =>
      sessionDisplayName(s).toLowerCase().includes(q) ||
      s.cwd?.toLowerCase().includes(q) ||
      s.branch?.toLowerCase().includes(q) ||
      s.preview?.toLowerCase().includes(q)
    )
  }
  if (filterByLifecycle.value) {
    if (filterByLifecycle.value === 'active') {
      sessions = sessions.filter(isSessionActive)
    } else {
      sessions = sessions.filter((s) => s.lifecycleStage === filterByLifecycle.value)
    }
  }
  if (sortBy.value === 'name') {
    sessions = [...sessions].sort((a, b) => sessionDisplayName(a).localeCompare(sessionDisplayName(b)))
  } else if (sortBy.value === 'status') {
    sessions = [...sessions].sort((a, b) => a.status.localeCompare(b.status))
  }
  return sessions
})

const folderGroups = computed(() => {
  const map = new Map<string, {
    key: string
    path: string
    name: string
    sessions: SessionSummary[]
    activeCount: number
    managedCount: number
    latestUpdatedAt: number
  }>()

  for (const session of filteredAndSearchedSessions.value) {
    const key = normalizeFolderKey(session.cwd)
    const existing = map.get(key) || {
      key,
      path: session.cwd || '未知目录',
      name: folderName(session.cwd),
      sessions: [],
      activeCount: 0,
      managedCount: 0,
      latestUpdatedAt: 0,
    }
    existing.sessions.push(session)
    if (isSessionActive(session)) existing.activeCount += 1
    if (session.loaded) existing.managedCount += 1
    existing.latestUpdatedAt = Math.max(existing.latestUpdatedAt, session.updatedAt || 0)
    map.set(key, existing)
  }

  return [...map.values()]
    .map((group) => ({
      ...group,
      sessions: [...group.sessions].sort((a, b) => (b.updatedAt || 0) - (a.updatedAt || 0)),
    }))
    .sort((a, b) => (b.activeCount - a.activeCount) || (b.latestUpdatedAt - a.latestUpdatedAt))
})

function displayName(session: SessionSummary) {
  return sessionDisplayName(session)
}

function isSessionActive(session: SessionSummary) {
  if (session.ended) return false
  return session.status === 'active'
    || session.status === 'inProgress'
    || session.lastTurnStatus === 'inProgress'
    || (session.activeFlags?.length || 0) > 0
}

function normalizeFolderKey(cwd: string) {
  const trimmed = (cwd || '').trim()
  if (!trimmed) return 'unknown'
  return trimmed.replace(/[\\/]+$/, '').toLowerCase()
}

function folderName(cwd: string) {
  const trimmed = (cwd || '').trim().replace(/[\\/]+$/, '')
  if (!trimmed) return '未知目录'
  const parts = trimmed.split(/[\\/]/).filter(Boolean)
  return parts[parts.length - 1] || trimmed
}

function onAgentSwitch(id: string) {
  app.selectedAgentId = id
}

function goDetail(id: string) {
  router.push(`/session/${id}`)
}

function setLifecycleFilter(value: string) {
  if (!value) {
    filterByLifecycle.value = ''
    return
  }
  filterByLifecycle.value = filterByLifecycle.value === value ? '' : value
}

function openNewSessionForFolder(cwd: string) {
  newForm.cwd = cwd || ''
  newForm.prompt = ''
  newForm.agentId = app.selectedAgentId || 'codex'
  applyPreset(newForm.preset)
  showNewSession.value = true
}

function onDirectorySelected(path: string) {
  newForm.cwd = path
}

async function handleCreate() {
  if (!newForm.cwd.trim() || !newForm.prompt.trim()) {
    ElMessage.warning('请填写工作目录和提示词')
    return
  }
  creating.value = true
  try {
    const s = await app.startSession(newForm.cwd, newForm.prompt, newForm.agentId, {
      model: newForm.model,
      reasoningEffort: newForm.reasoningEffort,
      collaborationMode: newForm.collaborationMode,
    })
    showNewSession.value = false
    newForm.cwd = ''
    newForm.prompt = ''
    ElMessage.success('会话已创建')
    router.push(`/session/${s.id}`)
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '创建失败')
  } finally {
    creating.value = false
  }
}

function applyPreset(value: string | number) {
  const preset = sessionOptions.value.presets.find((item) => item.id === String(value))
  if (!preset) return
  newForm.model = preset.model || ''
  newForm.reasoningEffort = preset.reasoningEffort || ''
  newForm.collaborationMode = preset.collaborationMode || 'default'
}

onMounted(() => {
  app.refreshDashboard()
  app.loadOptions().catch(() => {})
})

onUnmounted(() => {
})
</script>

<style scoped>
.dashboard-page {
  max-width: 1200px;
  margin: 0 auto;
}

.stat-card {
  border: 1px solid transparent;
}

.stat-card.is-selected {
  border-color: rgba(51, 136, 255, 0.45);
  box-shadow: 0 12px 26px rgba(51, 136, 255, 0.1);
}

.filter-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.cwd-field {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  width: 100%;
}

.new-session-dialog :deep(.el-segmented) {
  width: 100%;
}

.new-session-dialog :deep(.el-segmented__item) {
  min-width: 0;
  flex: 1;
}

.advanced-session-options {
  margin-left: 80px;
  border-top: 0;
  border-bottom: 0;
}

.advanced-session-options :deep(.el-collapse-item__header) {
  height: 34px;
  color: var(--cf-text-secondary);
  font-size: 13px;
}

.advanced-session-options :deep(.el-collapse-item__wrap) {
  border-bottom: 0;
}

.advanced-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 10px;
}

.advanced-grid label {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  align-items: center;
  gap: 8px;
  color: var(--cf-text-secondary);
  font-size: 13px;
}

.filter-bar-right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 8px;
}

.folder-session-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.session-results {
  min-height: 360px;
}

.session-empty-panel {
  display: grid;
  place-items: center;
  min-height: 360px;
  border: 1px solid var(--cf-border);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.68);
  box-shadow: var(--cf-shadow-sm);
}

.folder-group {
  border: 1px solid var(--cf-border);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.88);
  box-shadow: var(--cf-shadow-sm);
  overflow: hidden;
}

.folder-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 54px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--cf-border-light);
  background: linear-gradient(180deg, #fbfdff 0%, #f4f8fd 100%);
}

.folder-title-block {
  min-width: 0;
}

.folder-title-row {
  display: flex;
  align-items: center;
  gap: 7px;
  color: var(--cf-text-heavy);
  font-weight: 700;
}

.folder-title-row .el-icon {
  color: var(--cf-text-secondary);
}

.folder-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.folder-path {
  margin-top: 3px;
  color: var(--cf-text-secondary);
  font-family: monospace;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.folder-stats {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  color: var(--cf-text-secondary);
  font-size: 12px;
}

.folder-stat {
  padding: 2px 8px;
  border-radius: 999px;
  background: #eef4fb;
}

.folder-stat.is-active {
  color: #b45309;
  background: #fff3dc;
}

.folder-heading-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.folder-heading-actions :deep(.el-button) {
  padding: 4px 8px;
  border-radius: 8px;
  color: var(--cf-primary-dark);
  font-weight: 600;
}

.folder-session-items {
  padding: 6px;
}

.session-row {
  display: grid;
  grid-template-columns: 8px minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 46px;
  padding: 7px 10px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: pointer;
  transition: background 0.18s ease;
}

.session-row:hover {
  background: #eef6ff;
}

.session-row.is-running {
  background: #fff8eb;
}

.session-row.is-running:hover {
  background: #fff1d7;
}

.session-state-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #c8d3e1;
}

.session-row.is-managed .session-state-dot {
  background: var(--cf-success);
}

.session-row.is-running .session-state-dot {
  background: var(--cf-warning);
  box-shadow: 0 0 0 4px rgba(245, 158, 11, 0.14);
}

.session-row-main {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: 2px;
}

.session-row-title {
  color: var(--cf-text-heavy);
  font-weight: 650;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-row-preview {
  color: var(--cf-text-secondary);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-row-tags {
  display: flex;
  align-items: center;
  gap: 5px;
  flex-shrink: 0;
}

.session-row-time {
  color: var(--cf-text-secondary);
  font-size: 12px;
  white-space: nowrap;
}

.sse-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 600;
  color: var(--cf-success);
  padding: 3px 10px;
  border-radius: 10px;
  background: rgba(19, 168, 107, 0.08);
  border: 1px solid rgba(19, 168, 107, 0.2);
}

.sse-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--cf-success);
  animation: sse-pulse 2s ease-in-out infinite;
}

@keyframes sse-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

@media (max-width: 768px) {
  .filter-bar {
    gap: 8px;
  }

  .filter-bar .search-box {
    width: 100%;
    order: -1;
  }

  .filter-bar .el-select {
    width: calc(50% - 4px) !important;
  }

  .filter-bar-right {
    margin-left: 0;
    width: 100%;
    justify-content: flex-end;
  }

  .cwd-field {
    grid-template-columns: 1fr;
  }

  .advanced-session-options {
    margin-left: 0;
  }

  .advanced-grid label {
    grid-template-columns: 1fr;
    align-items: stretch;
  }

  .folder-heading {
    align-items: flex-start;
    flex-direction: column;
  }

  .folder-stats {
    flex-wrap: wrap;
  }

  .folder-heading-actions {
    width: 100%;
    justify-content: space-between;
  }

  .session-row {
    grid-template-columns: 8px minmax(0, 1fr);
    gap: 8px;
  }

  .session-row-tags,
  .session-row-time {
    grid-column: 2;
  }

  .session-row-tags {
    flex-wrap: wrap;
  }
}
</style>
