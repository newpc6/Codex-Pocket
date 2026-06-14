<template>
  <div class="session-detail-page" :class="{ 'is-mobile': isMobile }">
    <!-- 顶部信息栏（紧凑） -->
    <div class="session-header-bar">
      <div class="header-left">
        <el-button :icon="ArrowLeft" @click="$router.push('/')" text size="small" />
        <span class="session-name">{{ summary ? displayName(summary) : '会话详情' }}</span>
        <el-tag v-if="summary" :type="statusTagType(summary.status, summary.ended)" size="small">
          {{ statusLabel(summary.status, summary.ended, summary.activeFlags?.length > 0) }}
        </el-tag>
        <div v-if="summary && summary.lastTurnStatus === 'inProgress'" class="live-indicator">
          <span class="live-dot"></span>
          <span>执行中</span>
        </div>
      </div>
      <div class="header-right">
        <el-button :icon="Refresh" :loading="app.loading" @click="refreshPage()" circle size="small" />
        <el-dropdown v-if="summary" trigger="click" @command="onAction">
          <el-button size="small"><el-icon><More /></el-icon></el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item v-if="!summary.loaded && !summary.ended" command="resume">
                <el-icon><Connection /></el-icon> 接管会话
              </el-dropdown-item>
              <el-dropdown-item v-if="summary.ended" command="resume">
                <el-icon><Connection /></el-icon> 重新接管
              </el-dropdown-item>
              <el-dropdown-item v-if="summary.loaded && !summary.ended" command="detach">
                <el-icon><SwitchButton /></el-icon> 取消接管
              </el-dropdown-item>
              <el-dropdown-item v-if="summary.loaded && !summary.ended" command="end">
                <el-icon><SwitchButton /></el-icon> 结束会话
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>

    <!-- 会话元信息（可折叠） -->
    <div v-if="summary" class="session-meta" :class="{ 'is-collapsed': metaCollapsed }" @click="metaCollapsed = !metaCollapsed">
      <div class="meta-row">
        <span class="meta-cwd">{{ summary.cwd }}</span>
        <div class="meta-tags">
          <el-tag size="small" :type="summary.loaded ? 'success' : ''">{{ summary.loaded ? '已接管' : '未接管' }}</el-tag>
          <el-tag v-if="summary.branch" size="small">{{ summary.branch }}</el-tag>
          <el-tag size="small" :type="lifecycleTagType(summary.lifecycleStage)">{{ lifecycleLabel(summary.lifecycleStage) }}</el-tag>
        </div>
        <el-icon class="meta-arrow" :class="{ 'is-up': !metaCollapsed }"><ArrowRight /></el-icon>
      </div>
      <div v-if="!metaCollapsed && summary.preview" class="meta-preview">{{ truncateText(summary.preview, 200) }}</div>
    </div>

    <!-- 未接管提示 -->
    <div v-if="summary && !summary.loaded && !summary.ended" class="resume-banner">
      <span>会话未接管，接管后可继续执行</span>
      <el-button type="primary" size="small" :loading="resuming" @click="handleResume">接管</el-button>
    </div>
    <div v-if="summary && summary.loaded && !summary.ended" class="resume-banner">
      <span>当前会话正在由 CodexFlow 托管。取消接管后，会话会回到已发现状态。</span>
      <el-button size="small" :loading="detaching" @click="handleDetach">取消接管</el-button>
    </div>
    <div v-if="summary && summary.ended" class="resume-banner">
      <span>会话已结束，可重新接管继续</span>
      <el-button type="primary" size="small" :loading="resuming" @click="handleResume">重新接管</el-button>
    </div>

    <div class="content-area">
      <!-- 审批区域 -->
      <div v-if="sessionApprovals.length > 0" class="approval-section">
        <div v-for="approval in sessionApprovals" :key="approval.id" class="approval-card">
          <div class="approval-info">
            <div class="approval-kind">{{ approval.kind }}</div>
            <div class="approval-reason">{{ approval.reason || approval.summary }}</div>
          </div>
          <div class="approval-actions">
            <el-button
              v-for="choice in approvalChoices(approval)"
              :key="choice.value"
              size="small"
              :type="choice.type"
              @click="handleApprovalChoice(approval, choice.value)"
            >
              {{ choice.label }}
            </el-button>
          </div>
        </div>
      </div>

      <!-- 对话记录（中间滚动区域） -->
      <div class="chat-area" ref="chatAreaRef" @scroll="onChatScroll">
        <div v-if="detail && detail.turns.length === 0" class="empty-hint">
          {{ summary?.ended ? '会话已结束，没有更多对话。' : '还没有对话，在下方发送指令开始。' }}
        </div>

        <template v-if="orderedTurns.length > 0">
          <div class="chat-toolbar">
            <el-tag size="small" type="info" round>{{ orderedTurns.length }} 轮对话</el-tag>
            <div style="display: flex; gap: 4px">
              <el-button size="small" text @click="expandAll">展开</el-button>
              <el-button size="small" text @click="collapseAll">折叠</el-button>
            </div>
          </div>
          <TurnCard v-for="(turn, i) in orderedTurns" :key="turn.id" :turn="turn" :index="i" :ref="(el: any) => setTurnRef(turn.id, el)" />
        </template>

        <div v-else-if="!app.loading && !detail" class="empty-hint">
          <el-icon class="is-loading" :size="20"><Loading /></el-icon>
          <span>正在加载…</span>
        </div>
      </div>
    </div>

    <!-- 底部输入区域（固定） -->
    <div v-if="summary && summary.loaded && !summary.ended" class="input-area">
      <div v-if="isStreamingReply" class="streaming-hint">
        <span class="live-dot"></span>
        Codex 正在回复
      </div>
      <div class="input-row">
        <el-input
          v-model="promptText"
          type="textarea"
          :autosize="{ minRows: 1, maxRows: 4 }"
          placeholder="输入指令…"
          :disabled="submitting"
          @keydown.enter.exact.prevent="handleSubmit"
        />
        <el-button type="primary" :loading="submitting" @click="handleSubmit"
          :disabled="!promptText.trim()" class="send-btn">
          {{ summary.lastTurnStatus === 'inProgress' ? 'Steer' : '发送' }}
        </el-button>
        <el-button v-if="summary.lastTurnStatus === 'inProgress'" type="warning" size="small" @click="handleInterrupt">
          中断
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useAppStore, type ApprovalRequest, type SessionSummary } from '../stores/app'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, Refresh, More, ArrowRight, Connection, SwitchButton } from '@element-plus/icons-vue'
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
const detaching = ref(false)
const metaCollapsed = ref(true)
const chatAreaRef = ref<HTMLElement | null>(null)
const followLiveOutput = ref(true)

const isMobile = ref(window.innerWidth <= 768)
function onResize() { isMobile.value = window.innerWidth <= 768 }
window.addEventListener('resize', onResize)

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
const runningTurn = computed(() => orderedTurns.value.find((turn) => turn.status === 'inProgress'))
const isStreamingReply = computed(() => {
  const turn = runningTurn.value
  if (!turn) return false
  return turn.items?.some((item) => item.type === 'agentMessage' && item.body)
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

function scrollChatToBottom() {
  nextTick(() => {
    if (chatAreaRef.value) {
      chatAreaRef.value.scrollTop = chatAreaRef.value.scrollHeight
    }
  })
}

function onChatScroll() {
  const el = chatAreaRef.value
  if (!el) return
  followLiveOutput.value = el.scrollHeight - el.scrollTop - el.clientHeight < 80
}

// Follow live Codex output while the user stays near the latest message.
watch(orderedTurns, () => {
  if (followLiveOutput.value || runningTurn.value) scrollChatToBottom()
}, { deep: true })

async function refreshPage() {
  await app.refreshDashboard()
  await app.loadSession(sessionId)
}

function onAction(cmd: string) {
  if (cmd === 'resume') handleResume()
  else if (cmd === 'detach') handleDetach()
  else if (cmd === 'end') handleEnd()
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

async function handleDetach() {
  detaching.value = true
  try {
    await ElMessageBox.confirm('确定要取消接管这个会话吗？', '确认')
    await app.detachSession(sessionId)
    ElMessage.success('已取消接管')
  } catch { /* cancelled */ }
  finally {
    detaching.value = false
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

function approvalChoices(approval: ApprovalRequest) {
  if (approval.kind === 'userInput') {
    return [{ value: 'answer', label: '回复', type: 'primary' }]
  }
  const choices = approval.choices?.length ? approval.choices : ['accept', 'decline']
  return choices.map((choice) => ({
    value: choice,
    label: choiceLabel(choice),
    type: choiceType(choice),
  }))
}

function choiceLabel(choice: string) {
  switch (choice) {
    case 'accept': return '批准本次'
    case 'acceptForSession': return '本会话批准'
    case 'decline': return '拒绝'
    case 'deny': return '拒绝'
    case 'cancel': return '取消'
    case 'session': return '允许本会话'
    case 'turn': return '允许本轮'
    case 'answer': return '回复'
    default: return choice
  }
}

function choiceType(choice: string) {
  switch (choice) {
    case 'accept':
    case 'acceptForSession':
    case 'session':
    case 'turn':
      return 'success'
    case 'decline':
    case 'deny':
    case 'cancel':
      return 'danger'
    default:
      return 'primary'
  }
}

async function handleApprovalChoice(approval: ApprovalRequest, decision: string) {
  try {
    let result: Record<string, any>
    if (approval.kind === 'command' || approval.kind === 'fileChange' || approval.kind === 'generic') {
      result = { decision }
    } else if (approval.kind === 'permissions') {
      result = decision === 'session' || decision === 'turn'
        ? { permissions: approval.params?.permissions || {}, scope: decision }
        : { permissions: null, scope: null }
    } else if (approval.kind === 'userInput') {
      const { value } = await ElMessageBox.prompt('请输入回复', '用户输入', {
        confirmButtonText: '提交',
        cancelButtonText: '取消',
      })
      const questionId = approval.params?.questions?.[0]?.id || 'reply'
      result = { answers: { [questionId]: { answers: [value] } } }
    } else {
      result = { decision }
    }
    await app.resolveApproval(approval.id, result)
    ElMessage.success('审批已提交')
  } catch { /* cancelled */ }
}

onMounted(async () => {
  await refreshPage()
  app.registerActiveSession(sessionId)
  scrollChatToBottom()
})

onUnmounted(() => {
  app.unregisterActiveSession(sessionId)
  window.removeEventListener('resize', onResize)
})
</script>

<style scoped>
.session-detail-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  max-width: 1200px;
  margin: 0 auto;
  overflow: hidden;
  min-height: 0;
}

/* ---- 顶部信息栏 ---- */
.session-header-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  background: var(--cf-card);
  border-bottom: 1px solid var(--cf-border-light);
  flex-shrink: 0;
  gap: 8px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 1;
}

.session-name {
  font-size: 15px;
  font-weight: 700;
  color: var(--cf-text-heavy);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.live-indicator {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  font-weight: 600;
  color: var(--cf-warning);
  padding: 2px 8px;
  border-radius: 10px;
  background: rgba(245, 158, 11, 0.1);
  border: 1px solid rgba(245, 158, 11, 0.3);
  flex-shrink: 0;
}

.live-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--cf-warning);
  animation: live-pulse 1.5s ease-in-out infinite;
}

@keyframes live-pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.4; transform: scale(0.8); }
}

/* ---- 元信息 ---- */
.session-meta {
  padding: 8px 16px;
  background: var(--cf-card);
  border-bottom: 1px solid var(--cf-border-light);
  cursor: pointer;
  flex-shrink: 0;
  transition: background 0.15s ease;
}

.session-meta:hover {
  background: #f8fafd;
}

.meta-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.meta-cwd {
  font-size: 12px;
  color: var(--cf-text-secondary);
  font-family: monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 400px;
}

.meta-tags {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.meta-arrow {
  margin-left: auto;
  transition: transform 0.2s ease;
  color: var(--cf-text-lighter);
  font-size: 12px;
}

.meta-arrow.is-up {
  transform: rotate(90deg);
}

.meta-preview {
  font-size: 12px;
  color: var(--cf-text-secondary);
  margin-top: 6px;
  line-height: 1.5;
}

.session-meta.is-collapsed .meta-preview {
  display: none;
}

/* ---- 接管提示 ---- */
.resume-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  background: #fffbeb;
  border-bottom: 1px solid #fde68a;
  font-size: 13px;
  color: #92400e;
  flex-shrink: 0;
}

/* ---- 审批 ---- */
.approval-section {
  flex-shrink: 0;
  padding: 8px 16px;
  background: #fffbeb;
  border-bottom: 1px solid #fde68a;
}

.content-area {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.approval-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: var(--cf-card);
  border-radius: var(--cf-radius-sm);
  border-left: 3px solid var(--cf-warning);
  margin-bottom: 6px;
  gap: 8px;
}

.approval-card:last-child {
  margin-bottom: 0;
}

.approval-info {
  min-width: 0;
  flex: 1;
}

.approval-kind {
  font-size: 13px;
  font-weight: 600;
}

.approval-reason {
  font-size: 12px;
  color: var(--cf-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.approval-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

/* ---- 聊天区域 ---- */
.chat-area {
  flex: 1;
  overflow-y: auto;
  padding: 12px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 0;
}

.chat-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
  flex-shrink: 0;
}

.empty-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 40px 0;
  color: var(--cf-text-secondary);
  font-size: 14px;
}

/* ---- 底部输入 ---- */
.input-area {
  flex-shrink: 0;
  padding: 10px 16px;
  background: var(--cf-card);
  border-top: 1px solid var(--cf-border-light);
  box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.04);
}

.streaming-hint {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
  color: var(--cf-warning);
  font-size: 12px;
  font-weight: 600;
}

.input-row {
  display: flex;
  align-items: flex-end;
  gap: 8px;
}

.input-row :deep(.el-textarea__inner) {
  border-radius: 12px;
  padding: 8px 12px;
  font-size: 14px;
  resize: none;
}

.send-btn {
  border-radius: 12px;
  height: 36px;
  flex-shrink: 0;
}

/* ---- 手机适配 ---- */
.session-detail-page.is-mobile .session-header-bar {
  padding: 8px 12px;
}

.session-detail-page.is-mobile .session-name {
  font-size: 14px;
  max-width: 160px;
}

.session-detail-page.is-mobile .meta-cwd {
  max-width: 200px;
  font-size: 11px;
}

.session-detail-page.is-mobile .chat-area {
  padding: 8px 10px;
}

.session-detail-page.is-mobile .input-area {
  padding: 8px 10px;
}

.session-detail-page.is-mobile .input-row :deep(.el-textarea__inner) {
  font-size: 16px; /* prevent iOS zoom */
}

.session-detail-page.is-mobile .send-btn {
  height: 34px;
}

.session-detail-page.is-mobile .approval-card {
  flex-direction: column;
  align-items: flex-start;
}

.session-detail-page.is-mobile .approval-actions {
  margin-top: 6px;
  width: 100%;
  justify-content: flex-end;
}
</style>
