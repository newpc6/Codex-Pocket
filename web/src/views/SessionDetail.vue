<template>
  <div class="session-detail-page">
    <div class="page-title">
      <div class="page-title-heading">
        <el-button :icon="ArrowLeft" @click="$router.push('/')" text style="margin-right: 8px" />
        {{ summary ? displayName(summary) : '会话详情' }}
      </div>
      <div class="page-title-extra">
        <el-tag v-if="summary" :type="statusTagType(summary.status, summary.ended)" size="small">
          {{ statusLabel(summary.status, summary.ended, summary.activeFlags?.length > 0) }}
        </el-tag>
        <div v-if="summary && summary.lastTurnStatus === 'inProgress'" class="live-indicator">
          <span class="live-dot"></span>
          <span>实时执行中</span>
        </div>
        <el-button :icon="Refresh" :loading="app.loading" @click="refreshPage()" circle />
      </div>
    </div>

    <div v-if="summary" class="card">
      <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px">
        <div>
          <div style="font-size: 16px; font-weight: 700; color: var(--cf-text-heavy)">{{ displayName(summary) }}</div>
          <div style="font-size: 12px; color: var(--cf-text-secondary); font-family: monospace; margin-top: 4px">
            {{ summary.cwd }}
          </div>
        </div>
      </div>
      <div style="display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 12px">
        <el-tag size="small" :type="summary.loaded ? 'success' : ''">
          {{ summary.loaded ? '已接管' : '未接管' }}
        </el-tag>
        <el-tag size="small" type="info">{{ summary.source }}</el-tag>
        <el-tag v-if="summary.branch" size="small">{{ summary.branch }}</el-tag>
        <el-tag size="small">{{ summary.modelProvider }}</el-tag>
        <el-tag size="small" :type="lifecycleTagType(summary.lifecycleStage)">
          {{ lifecycleLabel(summary.lifecycleStage) }}
        </el-tag>
      </div>
      <div v-if="summary.preview" style="font-size: 13px; color: var(--cf-text-secondary); margin-bottom: 12px">
        {{ truncateText(summary.preview, 200) }}
      </div>
      <div style="font-size: 11px; color: var(--cf-text-secondary)">
        更新 {{ formatTimestamp(summary.updatedAt) }}
      </div>
    </div>

    <div v-if="summary && !summary.loaded && !summary.ended" class="card" style="text-align: center">
      <p style="margin-bottom: 12px; color: var(--cf-text-secondary)">这个会话尚未接管，接管后可以继续执行。</p>
      <el-button type="primary" :loading="resuming" @click="handleResume">接管会话</el-button>
    </div>

    <div v-if="summary && summary.ended" class="card" style="text-align: center">
      <p style="margin-bottom: 12px; color: var(--cf-text-secondary)">这个会话已结束，可以重新接管继续。</p>
      <el-button type="primary" :loading="resuming" @click="handleResume">重新接管</el-button>
    </div>

    <div v-if="summary && summary.loaded && !summary.ended" class="card">
      <div style="font-size: 15px; font-weight: 700; color: var(--cf-text-heavy); margin-bottom: 12px">发送指令</div>
      <el-input v-model="promptText" type="textarea" :rows="3" placeholder="输入指令..." :disabled="submitting" />
      <div style="display: flex; gap: 8px; margin-top: 12px">
        <el-button type="primary" :loading="submitting" @click="handleSubmit"
          :disabled="!promptText.trim()">
          {{ summary.lastTurnStatus === 'inProgress' ? 'Steer' : '发送' }}
        </el-button>
        <el-button v-if="summary.lastTurnStatus === 'inProgress'" type="warning" @click="handleInterrupt">
          中断
        </el-button>
        <el-button type="danger" @click="handleEnd">结束会话</el-button>
      </div>
    </div>

    <div v-if="sessionApprovals.length > 0" style="margin-top: 16px">
      <div style="font-size: 16px; font-weight: 700; color: var(--cf-text-heavy); margin-bottom: 12px">待审批</div>
      <div v-for="approval in sessionApprovals" :key="approval.id" class="approval-card">
        <div style="display: flex; justify-content: space-between; align-items: flex-start">
          <div>
            <div style="font-size: 14px; font-weight: 600">{{ approval.kind }}</div>
            <div style="font-size: 13px; color: var(--cf-text-secondary); margin-top: 4px">
              {{ approval.reason || approval.summary }}
            </div>
          </div>
          <div style="display: flex; gap: 8px">
            <el-button size="small" type="success" @click="handleApproval(approval, true)">批准</el-button>
            <el-button size="small" type="danger" @click="handleApproval(approval, false)">拒绝</el-button>
          </div>
        </div>
      </div>
    </div>

    <div v-if="detail" style="margin-top: 16px">
      <div v-if="detail.turns.length === 0" class="card">
        <p style="color: var(--cf-text-secondary)">
          {{ summary?.ended ? '这个会话已经结束，没有更多 turn。' : '这个会话还没有 turn，可以直接发送指令开始。' }}
        </p>
      </div>

      <template v-if="orderedTurns.length > 0">
        <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px">
          <div style="font-size: 16px; font-weight: 700; color: var(--cf-text-heavy)">
            对话记录
            <el-tag size="small" type="info" round style="margin-left: 8px">{{ orderedTurns.length }}</el-tag>
          </div>
          <div style="display: flex; gap: 8px">
            <el-button size="small" @click="expandAll">全部展开</el-button>
            <el-button size="small" @click="collapseAll">全部折叠</el-button>
          </div>
        </div>
        <TurnCard v-for="(turn, i) in orderedTurns" :key="turn.id" :turn="turn" :index="i" :ref="(el: any) => setTurnRef(turn.id, el)" />
      </template>
    </div>

    <div v-else-if="!app.loading" class="card" style="text-align: center">
      <el-icon class="is-loading" :size="24"><Loading /></el-icon>
      <span style="margin-left: 8px; color: var(--cf-text-secondary)">正在加载会话详情…</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useAppStore, type ApprovalRequest, type SessionSummary } from '../stores/app'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, Refresh } from '@element-plus/icons-vue'
import {
  formatTimestamp, statusTagType, statusLabel, lifecycleLabel,
  lifecycleTagType, truncateText, sessionDisplayName,
} from '../utils/helpers'
import TurnCard from '../components/TurnCard.vue'

const route = useRoute()
const app = useAppStore()
const sessionId = route.params.id as string
const promptText = ref('')
const submitting = ref(false)
const resuming = ref(false)

const detail = computed(() => app.sessionDetails[sessionId])
const summary = computed<SessionSummary | undefined>(() => {
  if (detail.value) return detail.value.summary
  return app.dashboard.sessions.find((s) => s.id === sessionId)
})

const sessionApprovals = computed(() => app.filteredApprovals.filter((a) => a.threadId === sessionId))

const orderedTurns = computed(() => {
  if (!detail.value) return []
  return [...detail.value.turns].reverse()
})

const turnRefs = ref<Record<string, any>>({})

function setTurnRef(id: string, el: any) {
  if (el) turnRefs.value[id] = el
}

function expandAll() {
  Object.values(turnRefs.value).forEach((comp: any) => {
    if (comp?.expanded !== undefined) comp.expanded = true
  })
}

function collapseAll() {
  Object.values(turnRefs.value).forEach((comp: any) => {
    if (comp?.expanded !== undefined) comp.expanded = false
  })
}

function displayName(s: SessionSummary) { return sessionDisplayName(s) }

async function refreshPage() {
  await app.refreshDashboard()
  await app.loadSession(sessionId)
}

async function handleResume() {
  resuming.value = true
  try {
    await app.resumeSession(sessionId)
    ElMessage.success('会话已接管')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '接管失败')
  } finally {
    resuming.value = false
  }
}

async function handleSubmit() {
  if (!promptText.value.trim()) return
  submitting.value = true
  try {
    const s = summary.value
    if (s?.lastTurnStatus === 'inProgress' && s.lastTurnId) {
      await app.steerTurn(sessionId, s.lastTurnId, promptText.value)
    } else {
      await app.startTurn(sessionId, promptText.value)
    }
    promptText.value = ''
    ElMessage.success('指令已发送')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '发送失败')
  } finally {
    submitting.value = false
  }
}

async function handleInterrupt() {
  const s = summary.value
  if (!s?.lastTurnId) return
  try {
    await app.interruptTurn(sessionId, s.lastTurnId)
    ElMessage.success('已中断')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '中断失败')
  }
}

async function handleEnd() {
  try {
    await ElMessageBox.confirm('确定要结束这个会话吗？', '确认')
    await app.endSession(sessionId)
    ElMessage.success('会话已结束')
  } catch { /* cancelled */ }
}

async function handleApproval(approval: ApprovalRequest, accept: boolean) {
  try {
    let result: Record<string, any>
    if (approval.kind === 'command' || approval.kind === 'fileChange') {
      result = { decision: accept ? 'accept' : 'deny' }
    } else if (approval.kind === 'permissions') {
      result = accept
        ? { permissions: approval.params?.permissions || {}, scope: 'session' }
        : { permissions: null, scope: null }
    } else if (approval.kind === 'userInput') {
      const { value } = await ElMessageBox.prompt('请输入回复', '用户输入', {
        confirmButtonText: '提交',
        cancelButtonText: '取消',
      })
      const questionId = approval.params?.questions?.[0]?.id || 'reply'
      result = { answers: { [questionId]: { answers: [value] } } }
    } else {
      result = { decision: accept ? 'accept' : 'deny' }
    }
    await app.resolveApproval(approval.id, result)
    ElMessage.success(accept ? '已批准' : '已拒绝')
  } catch { /* cancelled */ }
}

onMounted(async () => {
  await refreshPage()
  // Register this session as active so SSE events trigger targeted refresh
  app.registerActiveSession(sessionId)
})

onUnmounted(() => {
  app.unregisterActiveSession(sessionId)
})
</script>

<style scoped>
.session-detail-page {
  max-width: 1200px;
  margin: 0 auto;
}

.live-indicator {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--cf-warning);
  padding: 4px 12px;
  border-radius: 12px;
  background: rgba(245, 158, 11, 0.1);
  border: 1px solid rgba(245, 158, 11, 0.3);
}

.live-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--cf-warning);
  animation: live-pulse 1.5s ease-in-out infinite;
}

@keyframes live-pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.4; transform: scale(0.8); }
}
</style>
