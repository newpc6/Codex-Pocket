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
      <div class="stat-card tone-blue" @click="filterByLifecycle = ''">
        <div class="stat-head">
          <span>总会话</span>
          <el-button type="primary" link size="small">查看</el-button>
        </div>
        <div class="stat-value">{{ stats.totalSessions }}</div>
        <div class="stat-desc">所有已发现的会话</div>
      </div>
      <div class="stat-card tone-green" @click="filterByLifecycle = 'managed'">
        <div class="stat-head">
          <span>已接管</span>
          <el-button type="primary" link size="small">筛选</el-button>
        </div>
        <div class="stat-value">{{ stats.loadedSessions }}</div>
        <div class="stat-desc">正在由 CodexFlow 托管</div>
      </div>
      <div class="stat-card tone-cyan" @click="filterByLifecycle = 'active'">
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

    <template v-for="(group, key) in visibleGroups" :key="key">
      <div style="margin-top: 16px; margin-bottom: 8px; display: flex; align-items: center; gap: 8px;">
        <span style="font-size: 16px; font-weight: 700; color: var(--cf-text-heavy)">{{ groupTitles[key as keyof typeof groupTitles] }}</span>
        <el-tag size="small" type="info">{{ group.length }}</el-tag>
      </div>
      <p style="font-size: 13px; color: var(--cf-text-secondary); margin-bottom: 10px">{{ groupHelpers[key as keyof typeof groupHelpers] }}</p>
      <div v-for="session in group" :key="session.id" class="session-item" @click="goDetail(session.id)">
        <div class="session-header">
          <div class="session-name">{{ displayName(session) }}</div>
          <el-tag :type="statusTagType(session.status, session.ended)" size="small">
            {{ statusLabel(session.status, session.ended, session.activeFlags?.length > 0) }}
          </el-tag>
        </div>
        <div class="session-cwd">{{ session.cwd }}</div>
        <div v-if="session.preview" class="session-preview">{{ truncateText(session.preview) }}</div>
        <div class="session-tags">
          <el-tag size="small" :type="session.loaded ? 'success' : ''">
            {{ session.loaded ? '已接管' : '未接管' }}
          </el-tag>
          <el-tag size="small" type="info">{{ session.source }}</el-tag>
          <el-tag v-if="session.branch" size="small">{{ session.branch }}</el-tag>
          <el-tag size="small" :type="lifecycleTagType(session.lifecycleStage)">
            {{ lifecycleLabel(session.lifecycleStage) }}
          </el-tag>
        </div>
        <div style="font-size: 11px; color: var(--cf-text-secondary); margin-top: 8px">
          更新 {{ formatTimestamp(session.updatedAt) }}
        </div>
      </div>
    </template>

    <el-empty v-if="filteredAndSearchedSessions.length === 0 && !app.loading" description="没有匹配的会话" />

    <el-dialog v-model="showNewSession" title="新建会话" width="480px" :close-on-click-modal="false">
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
const newForm = reactive({ cwd: '', prompt: '', agentId: 'codex' })
const searchQuery = ref('')
const filterByLifecycle = ref('')
const sortBy = ref('updatedAt')

const stats = computed(() => app.dashboard.stats)
const agents = computed(() => app.dashboard.agents || [])
const currentAgentName = computed(() => {
  const a = agents.value.find((a) => a.id === app.selectedAgentId)
  return a?.name || 'Codex'
})

const groupTitles: Record<string, string> = {
  managed: '已接管',
  ended: '已结束',
  runtimeAvailable: '待接管',
  discovered: '已发现',
  historyOnly: '历史会话',
}

const groupHelpers: Record<string, string> = {
  managed: '这些会话已经由 CodexFlow 后台托管，可以直接继续操作。',
  ended: '这些会话已经从 CodexFlow 托管态退出。需要继续时，再重新接管。',
  runtimeAvailable: '这些会话当前未接管，但运行时仍可继续接管。',
  discovered: '这些会话已被发现，但尚未接管。点击可查看详情，接管后即可继续执行。',
  historyOnly: '这些只是已发现的真实会话记录。先接管，才可以继续执行。',
}

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

const visibleGroups = computed(() => {
  const sessions = filteredAndSearchedSessions.value
  const groups: Record<string, SessionSummary[]> = {
    managed: [],
    ended: [],
    runtimeAvailable: [],
    discovered: [],
    historyOnly: [],
  }
  for (const s of sessions) {
    const key = s.lifecycleStage === 'runtime_available' ? 'runtimeAvailable'
      : s.lifecycleStage === 'history_only' ? 'historyOnly'
      : s.lifecycleStage
    if (groups[key]) groups[key].push(s)
    else groups.discovered.push(s)
  }
  const result: Record<string, SessionSummary[]> = {}
  for (const [key, val] of Object.entries(groups)) {
    if (val.length > 0) result[key] = val
  }
  return result
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

function onAgentSwitch(id: string) {
  app.selectedAgentId = id
}

function goDetail(id: string) {
  router.push(`/session/${id}`)
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
    const s = await app.startSession(newForm.cwd, newForm.prompt, newForm.agentId)
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

onMounted(() => {
  app.refreshDashboard()
})

onUnmounted(() => {
})
</script>

<style scoped>
.dashboard-page {
  max-width: 1200px;
  margin: 0 auto;
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

.filter-bar-right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 8px;
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
}
</style>
