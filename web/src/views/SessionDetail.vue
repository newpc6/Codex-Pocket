<template>
  <div class="page-container">
    <div style="margin-bottom: 16px">
      <el-button :icon="ArrowLeft" @click="$router.push('/')">返回</el-button>
    </div>

    <div v-if="summary" class="card">
      <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px">
        <div>
          <div style="font-size: 16px; font-weight: 600">{{ displayName(summary) }}</div>
          <div style="font-size: 12px; color: var(--cf-text-secondary); font-family: monospace; margin-top: 4px">
            {{ summary.cwd }}
          </div>
        </div>
        <el-tag :type="statusTagType(summary.status, summary.ended)" size="small">
          {{ statusLabel(summary.status, summary.ended, summary.activeFlags?.length > 0) }}
        </el-tag>
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
      <div style="font-size: 14px; font-weight: 600; margin-bottom: 12px">发送指令</div>
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
      <div style="font-size: 16px; font-weight: 600; margin-bottom: 12px">待审批</div>
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

      <template v-if="activeTurn">
        <div style="font-size: 16px; font-weight: 600; margin-bottom: 12px">当前运行中</div>
        <TurnCard :turn="activeTurn" />
      </template>

      <template v-if="recentTurns.length > 0">
        <div style="font-size: 16px; font-weight: 600; margin-bottom: 12px; margin-top: 16px">最近的 turn</div>
        <TurnCard v-for="turn in recentTurns" :key="turn.id" :turn="turn" />
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
let pollTimer: ReturnType<typeof setInterval> | null = null

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

const activeTurn = computed(() => orderedTurns.value.find((t) => t.status === 'inProgress'))
const recentTurns = computed(() => orderedTurns.value.filter((t) => t.status !== 'inProgress'))

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
  pollTimer = setInterval(async () => {
    const s = summary.value
    if (s?.ended) return
    if (s?.lastTurnStatus === 'inProgress' || sessionApprovals.value.length > 0) {
      await refreshPage()
    }
  }, 3000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>
