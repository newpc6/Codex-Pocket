<template>
  <div class="page-container">
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px">
      <div style="display: flex; align-items: center; gap: 12px">
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
      </div>
      <el-button :icon="Refresh" :loading="app.loading" @click="app.refreshDashboard()">刷新</el-button>
    </div>

    <div class="stat-grid">
      <div class="stat-card">
        <div class="stat-value" style="color: var(--cf-primary)">{{ stats.totalSessions }}</div>
        <div class="stat-label">总会话</div>
      </div>
      <div class="stat-card">
        <div class="stat-value" style="color: var(--cf-success)">{{ stats.loadedSessions }}</div>
        <div class="stat-label">已加载</div>
      </div>
      <div class="stat-card">
        <div class="stat-value" style="color: var(--cf-info)">{{ stats.activeSessions }}</div>
        <div class="stat-label">运行中</div>
      </div>
      <div class="stat-card">
        <div class="stat-value" style="color: var(--cf-warning)">{{ stats.pendingApprovals }}</div>
        <div class="stat-label">待审批</div>
      </div>
    </div>

    <el-alert v-if="app.error" :title="app.error" type="error" show-icon :closable="false" style="margin-bottom: 16px" />

    <el-alert v-if="app.filteredApprovals.length > 0" type="warning" :closable="false" style="margin-bottom: 16px">
      <template #title>当前有 {{ app.filteredApprovals.length }} 个审批等待处理。</template>
    </el-alert>

    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px">
      <div style="font-size: 16px; font-weight: 600">
        列表 <span style="font-size: 12px; color: var(--cf-text-secondary)">{{ app.filteredSessions.length }}</span>
      </div>
      <el-button type="primary" size="small" :icon="Plus" @click="showNewSession = true">新建</el-button>
    </div>

    <template v-for="(group, key) in visibleGroups" :key="key">
      <div style="margin-top: 16px; margin-bottom: 8px">
        <span style="font-size: 14px; font-weight: 600">{{ groupTitles[key as keyof typeof groupTitles] }}</span>
        <span style="font-size: 12px; color: var(--cf-text-secondary); margin-left: 8px">{{ group.length }}</span>
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

    <el-empty v-if="app.filteredSessions.length === 0 && !app.loading" description="暂时没有会话" />

    <el-dialog v-model="showNewSession" title="新建会话" width="480px">
      <el-form :model="newForm" label-width="80px">
        <el-form-item label="工作目录">
          <el-input v-model="newForm.cwd" placeholder="D:\project\myapp" />
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
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useAppStore, type SessionSummary } from '../stores/app'
import { Refresh, Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import {
  formatTimestamp, statusTagType, statusLabel, lifecycleLabel,
  lifecycleTagType, truncateText, sessionDisplayName,
} from '../utils/helpers'

const router = useRouter()
const app = useAppStore()
const showNewSession = ref(false)
const creating = ref(false)
const newForm = reactive({ cwd: '', prompt: '', agentId: 'codex' })

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

const visibleGroups = computed(() => {
  const groups = app.sessionGroups
  const result: Record<string, SessionSummary[]> = {}
  for (const [key, sessions] of Object.entries(groups)) {
    if (sessions.length > 0) result[key] = sessions
  }
  return result
})

function displayName(session: SessionSummary) {
  return sessionDisplayName(session)
}

function onAgentSwitch(id: string) {
  app.selectedAgentId = id
}

function goDetail(id: string) {
  router.push(`/session/${id}`)
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
</script>
