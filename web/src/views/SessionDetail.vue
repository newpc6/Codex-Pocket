<template>
  <div class="session-detail-page" :class="{ 'is-mobile': isMobile }">
    <div v-if="summary" class="session-hero">
      <div class="hero-top">
        <button type="button" class="back-chip" @click="$router.push('/')">
          <el-icon><ArrowLeft /></el-icon>
          <span>返回会话</span>
        </button>

        <div class="hero-actions">
          <el-button :icon="Refresh" :loading="app.loading" @click="refreshPage()" circle size="small" />
          <el-dropdown trigger="click" @command="onAction">
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
                <el-dropdown-item command="rename">重命名</el-dropdown-item>
                <el-dropdown-item v-if="summary.agentId === 'codex'" command="fork">分支会话</el-dropdown-item>
                <el-dropdown-item v-if="summary.agentId === 'codex'" command="compact">压缩上下文</el-dropdown-item>
                <el-dropdown-item v-if="summary.agentId === 'codex'" command="rollback">回滚最近一轮</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>

      <div class="hero-main">
        <div class="hero-title-group">
          <div class="hero-name-row">
            <h1 class="hero-name">{{ displayName(summary) }}</h1>
            <el-tag :type="statusTagType(summary.status, summary.ended)" size="small" effect="light">
              {{ statusLabel(summary.status, summary.ended, summary.activeFlags?.length > 0) }}
            </el-tag>
            <div v-if="summary.lastTurnStatus === 'inProgress'" class="live-indicator">
              <span class="live-dot"></span>
              <span>执行中</span>
            </div>
          </div>

          <div class="hero-meta-row">
            <div class="hero-tags">
              <span class="hero-pill" :class="{ 'is-active': summary.loaded }">{{ summary.loaded ? '已接管' : '未接管' }}</span>
              <span v-if="summary.branch" class="hero-pill">{{ summary.branch }}</span>
              <span class="hero-pill">{{ lifecycleLabel(summary.lifecycleStage) }}</span>
            </div>
            <span class="hero-cwd">{{ summary.cwd }}</span>
          </div>

          <p v-if="summary.preview" class="hero-preview">
            {{ truncateText(summary.preview, 72) }}
          </p>
        </div>

        <div class="hero-status-card">
          <div class="hero-status-copy">
            <div class="hero-status-label">当前状态</div>
            <div class="hero-status-value">
              {{ summary.ended ? '会话已结束' : summary.loaded ? 'CodexFlow 正在托管' : '会话未接管' }}
            </div>
            <div class="hero-status-desc">
              {{ statusDescription(summary) }}
            </div>
          </div>

          <div class="hero-primary-actions">
            <el-button
              v-if="!summary.loaded && !summary.ended"
              type="primary"
              size="small"
              :loading="resuming"
              @click="handleResume"
            >
              接管会话
            </el-button>
            <el-button
              v-else-if="summary.ended"
              type="primary"
              size="small"
              :loading="resuming"
              @click="handleResume"
            >
              重新接管
            </el-button>
            <el-button
              v-else
              size="small"
              :loading="detaching"
              @click="handleDetach"
            >
              取消接管
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <div class="content-area">
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

      <div class="chat-shell">
        <div class="chat-toolbar">
          <div class="toolbar-left">
            <el-tag size="small" type="info" round>{{ detail?.totalTurns ?? orderedTurns.length }} 轮对话</el-tag>
            <span v-if="!followLiveOutput && latestTurn" class="follow-tip">已停留在历史位置</span>
          </div>
          <div class="toolbar-right">
            <el-button v-if="!followLiveOutput && latestTurn" size="small" text @click="jumpToLatest">回到最新</el-button>
          </div>
        </div>

        <div class="chat-area" ref="chatAreaRef" @scroll="onChatScroll">
          <div v-if="detail?.hasMoreHistory" class="history-load-row">
            <el-button text size="small" :loading="loadingHistory" @click="loadOlderTurns">加载更早对话</el-button>
          </div>

          <div v-if="detail && detail.turns.length === 0" class="empty-hint">
            {{ summary?.ended ? '会话已结束，没有更多对话。' : '还没有对话，在下方发送指令开始。' }}
          </div>

          <template v-if="orderedTurns.length > 0">
            <section v-for="turn in orderedTurns" :key="turn.id" class="turn-stream">
              <div class="turn-anchor">
                <span class="turn-title">Turn #{{ turnNumber(turn.id) }}</span>
                <span class="turn-meta">{{ turn.status === 'inProgress' ? '执行中' : formatTimestamp(turn.startedAt) }}</span>
              </div>

              <div
                v-for="(item, idx) in turn.items"
                :key="item.id || `${turn.id}-${idx}`"
                class="message-row"
                :class="messageSide(item.type)"
              >
                <div class="message-bubble" :class="bubbleClass(item.type)">
                  <div v-if="!isStructuredToolItem(item)" class="message-topline">
                    <span class="message-label">{{ itemLabel(item.type) }}</span>
                    <span v-if="item.status" class="message-status">{{ item.status }}</span>
                  </div>

                  <div
                    v-if="item.title && item.type !== 'userMessage' && item.type !== 'agentMessage' && !isStructuredToolItem(item)"
                    class="message-title"
                  >
                    {{ item.title }}
                  </div>

                  <template v-if="isStructuredToolItem(item)">
                    <div class="tool-card">
                      <div class="tool-summary">
                        <div class="tool-main">
                          <div class="tool-name">工具</div>
                          <div class="tool-headline">
                            <span class="tool-type">{{ toolDisplayName(item) }}</span>
                            <span
                              v-if="toolCommandTag(item)"
                              class="tool-command-tag"
                              :title="toolCommandTag(item)"
                            >
                              {{ toolCommandTag(item) }}
                            </span>
                          </div>
                        </div>
                      </div>

                      <details v-if="hasStructuredToolDetails(item)" class="tool-details">
                        <summary>查看原始内容</summary>
                        <div v-if="item.body" class="message-body is-code">
                          <pre>{{ item.body }}</pre>
                        </div>
                        <div v-if="item.auxiliary" class="message-aux tool-output">
                          <div class="tool-output-title">输出</div>
                          <pre>{{ item.auxiliary }}</pre>
                        </div>
                      </details>
                    </div>
                  </template>

                  <template v-else>
                    <div v-if="item.body" class="message-body" :class="{ 'is-code': isCodeType(item.type) }">
                      <pre v-if="isCodeType(item.type)">{{ item.body }}</pre>
                      <div v-else class="markdown-body">
                        <VueMarkdown :source="renderMarkdown(item.body)" :options="markdownOptions" />
                        <span v-if="isStreamingItem(turn, item)" class="typing-cursor">|</span>
                      </div>
                    </div>

                    <details v-if="item.auxiliary" class="message-aux">
                      <summary>详细输出</summary>
                      <pre>{{ item.auxiliary }}</pre>
                    </details>
                  </template>
                </div>
              </div>

              <div v-if="turn.diff" class="message-row side-left">
                <div class="message-bubble bubble-tool">
                  <div class="message-topline">
                    <span class="message-label">文件变更</span>
                  </div>
                  <pre class="diff-block">{{ turn.diff }}</pre>
                </div>
              </div>

              <div v-if="turn.error" class="message-row side-left">
                <div class="message-bubble bubble-error">
                  <div class="message-topline">
                    <span class="message-label">错误</span>
                  </div>
                  <div class="message-body">{{ turn.error }}</div>
                </div>
              </div>
            </section>
          </template>

          <div v-else-if="!app.loading && !detail" class="empty-hint">
            <el-icon class="is-loading" :size="20"><Loading /></el-icon>
            <span>正在加载…</span>
          </div>
        </div>
        <transition name="new-message-pill">
          <button
            v-if="showNewMessageHint"
            type="button"
            class="new-message-pill"
            @click="jumpToLatest"
          >
            有新消息，回到最新
          </button>
        </transition>
      </div>
    </div>

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
import { useRoute, useRouter } from 'vue-router'
import { useAppStore, type ApprovalRequest, type SessionSummary, type Turn, type TurnItem } from '../stores/app'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, Refresh, More, ArrowRight, Connection, SwitchButton, Loading } from '@element-plus/icons-vue'
import VueMarkdown from 'vue-markdown-render'
import {
  formatTimestamp, statusTagType, statusLabel, lifecycleLabel,
  lifecycleTagType, truncateText, sessionDisplayName,
} from '../utils/helpers'

const route = useRoute()
const router = useRouter()
const app = useAppStore()
const sessionId = route.params.id as string
const promptText = ref('')
const submitting = ref(false)
const resuming = ref(false)
const detaching = ref(false)
const chatAreaRef = ref<HTMLElement | null>(null)
const followLiveOutput = ref(true)
const loadingHistory = ref(false)
const pendingNewMessages = ref(0)

const markdownOptions = {
  html: false,
  breaks: true,
  linkify: true,
  typographer: true,
}

const localAssetBase = '/api/v1/assets/local-image'

const isMobile = ref(window.innerWidth <= 768)
function onResize() { isMobile.value = window.innerWidth <= 768 }
window.addEventListener('resize', onResize)

const detail = computed(() => app.sessionDetails[sessionId])
const summary = computed<SessionSummary | undefined>(() => {
  if (detail.value) return detail.value.summary
  return app.dashboard.sessions.find((s) => s.id === sessionId)
})

const sessionApprovals = computed(() => app.filteredApprovals.filter((a) => a.threadId === sessionId))
const orderedTurns = computed(() => detail.value?.turns || [])
const latestTurn = computed(() => orderedTurns.value[orderedTurns.value.length - 1])
const showNewMessageHint = computed(() => pendingNewMessages.value > 0 && !followLiveOutput.value)
const runningTurn = computed(() => {
  for (let i = orderedTurns.value.length - 1; i >= 0; i -= 1) {
    const turn = orderedTurns.value[i]
    if (turn.status === 'inProgress') return turn
  }
  return undefined
})
const isStreamingReply = computed(() => {
  const turn = runningTurn.value
  if (!turn) return false
  return turn.items?.some((item: TurnItem) => item.type === 'agentMessage' && item.body)
})

function displayName(s: SessionSummary) { return sessionDisplayName(s) }

function statusDescription(s: SessionSummary) {
  if (s.ended) return '当前会话已经结束，但历史内容仍然保留，可以重新接管继续工作。'
  if (s.loaded) return '当前会话由 CodexFlow 持续同步和控制，你可以在这里继续发送指令或中断执行。'
  return '当前会话还没有由 CodexFlow 托管，接管后可以继续执行并实时查看消息。'
}

function itemLabel(type: string): string {
  switch (type) {
    case 'userMessage': return '用户'
    case 'agentMessage': return 'Codex'
    case 'commandExecution': return '命令执行'
    case 'fileChange': return '文件变更'
    case 'reasoning': return '思考'
    case 'plan': return '计划'
    case 'mcpToolCall': return 'MCP 工具'
    case 'dynamicToolCall': return '工具'
    case 'collabAgentToolCall': return '协作'
    default: return type
  }
}

function messageSide(type: string) {
  return type === 'userMessage' ? 'side-right' : 'side-left'
}

function bubbleClass(type: string) {
  switch (type) {
    case 'userMessage': return 'bubble-user'
    case 'agentMessage': return 'bubble-agent'
    case 'commandExecution':
    case 'dynamicToolCall':
    case 'mcpToolCall':
    case 'collabAgentToolCall':
    case 'fileChange':
      return 'bubble-tool'
    case 'reasoning':
    case 'plan':
      return 'bubble-meta'
    default:
      return 'bubble-other'
  }
}

function isCodeType(type: string): boolean {
  return ['commandExecution', 'fileChange', 'mcpToolCall', 'dynamicToolCall'].includes(type)
}

function isStructuredToolItem(item: TurnItem): boolean {
  return item.type === 'commandExecution' || item.type === 'dynamicToolCall'
}

function toolDisplayName(item: TurnItem): string {
  if (item.type === 'commandExecution') return (item.title || 'shell_command').trim() || 'shell_command'
  const raw = item.title || item.type
  return raw.trim() || item.type
}

function toolCommandTag(item: TurnItem): string {
  const metadataCommand = (item.metadata?.command || '').trim()
  if (metadataCommand) return metadataCommand
  if (item.type === 'commandExecution') return (item.body || '').trim()
  if (item.type === 'dynamicToolCall' && toolDisplayName(item) === 'shell_command') {
    return extractCommandFromToolBody(item.body)
  }
  return ''
}

function extractCommandFromToolBody(body: string): string {
  const raw = (body || '').trim()
  if (!raw) return ''
  try {
    const decoded = JSON.parse(raw)
    return typeof decoded?.command === 'string' ? decoded.command.trim() : ''
  } catch {
    return ''
  }
}

function hasStructuredToolDetails(item: TurnItem): boolean {
  return Boolean((item.body && item.body.trim()) || (item.auxiliary && item.auxiliary.trim()))
}

function renderMarkdown(source: string): string {
  return normalizeAttachedImageSyntax(rewriteMarkdownImagePaths(source || ''))
}

function rewriteMarkdownImagePaths(source: string): string {
  const token = localStorage.getItem('cf_token') || ''
  return source.replace(/!\[([^\]]*)\]\(([^)]+)\)/g, (_full, alt: string, rawPath: string) => {
    const normalizedPath = normalizeImagePath(rawPath)
    if (!normalizedPath) return `![${alt}](${rawPath})`
    return `![${alt}](${buildLocalImageUrl(normalizedPath, token)})`
  })
}

function normalizeAttachedImageSyntax(source: string): string {
  const token = localStorage.getItem('cf_token') || ''
  return source.replace(/\[Attached image:\s*([^\]]+?)\]/g, (_full, rawPath: string) => {
    const normalizedPath = normalizeImagePath(rawPath)
    if (!normalizedPath) return _full
    return `\n\n![Attached image](${buildLocalImageUrl(normalizedPath, token)})\n\n`
  })
}

function normalizeImagePath(rawPath: string): string {
  const trimmed = rawPath.trim().replace(/^<|>$/g, '').replace(/^['"]|['"]$/g, '')
  if (!trimmed) return ''
  if (/^(https?:)?\/\//i.test(trimmed)) return trimmed
  if (/^(data:image\/)/i.test(trimmed)) return trimmed
  if (/^[A-Za-z]:[\\/]/.test(trimmed) || trimmed.startsWith('/')) return trimmed
  return ''
}

function buildLocalImageUrl(path: string, token: string): string {
  if (/^(https?:)?\/\//i.test(path) || /^(data:image\/)/i.test(path)) {
    return path
  }
  const params = new URLSearchParams({ path })
  if (token) params.set('token', token)
  return `${localAssetBase}?${params.toString()}`
}

function isStreamingItem(turn: Turn, item: TurnItem): boolean {
  return turn.status === 'inProgress' && item.type === 'agentMessage'
}

function turnNumber(id: string) {
  const idx = orderedTurns.value.findIndex((turn) => turn.id === id)
  return idx >= 0 ? idx + 1 : '?'
}

function scrollChatToBottom(force = false) {
  nextTick(() => {
    const el = chatAreaRef.value
    if (!el) return
    if (!force && !followLiveOutput.value) return
    el.scrollTop = el.scrollHeight
  })
}

function jumpToLatest() {
  followLiveOutput.value = true
  pendingNewMessages.value = 0
  scrollChatToBottom(true)
}

async function onChatScroll() {
  const el = chatAreaRef.value
  if (!el) return
  const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 80
  followLiveOutput.value = nearBottom
  if (nearBottom) {
    pendingNewMessages.value = 0
  }
  if (el.scrollTop < 40 && detail.value?.hasMoreHistory && !loadingHistory.value) {
    await loadOlderTurns()
  }
}

watch(orderedTurns, (next, prev) => {
  const prevLast = prev?.[prev.length - 1]
  const nextLast = next?.[next.length - 1]
  const latestChanged = !prevLast || !nextLast || prevLast.id !== nextLast.id || JSON.stringify(prevLast.items) !== JSON.stringify(nextLast.items)
  if (latestChanged && followLiveOutput.value) {
    scrollChatToBottom(true)
    pendingNewMessages.value = 0
  } else if (latestChanged) {
    pendingNewMessages.value += 1
  }
}, { deep: true })

async function refreshPage() {
  await app.refreshDashboard()
  await app.loadSession(sessionId)
}

async function refreshSessionWhenVisible() {
  if (document.visibilityState !== 'visible') return
  await app.loadSession(sessionId)
}

async function loadOlderTurns() {
  if (!detail.value?.hasMoreHistory || loadingHistory.value) return
  loadingHistory.value = true
  const el = chatAreaRef.value
  const beforeHeight = el?.scrollHeight || 0
  const beforeTop = el?.scrollTop || 0
  try {
    const nextOffset = Math.max((detail.value.offset || 0) - (detail.value.limit || 8), 0)
    await app.loadSession(sessionId, {
      offset: nextOffset,
      limit: detail.value.limit || 8,
      appendHistory: true,
    })
    await nextTick()
    if (el) {
      const delta = el.scrollHeight - beforeHeight
      el.scrollTop = beforeTop + delta
    }
  } finally {
    loadingHistory.value = false
  }
}

function onAction(cmd: string) {
  if (cmd === 'resume') handleResume()
  else if (cmd === 'detach') handleDetach()
  else if (cmd === 'end') handleEnd()
  else if (cmd === 'rename') handleRename()
  else if (cmd === 'fork') handleFork()
  else if (cmd === 'compact') handleCompact()
  else if (cmd === 'rollback') handleRollback()
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
    followLiveOutput.value = true
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

async function handleRename() {
  const currentName = summary.value ? displayName(summary.value) : ''
  try {
    const { value } = await ElMessageBox.prompt('给这个会话起一个更容易识别的名字', '重命名会话', {
      confirmButtonText: '保存',
      cancelButtonText: '取消',
      inputValue: currentName,
      inputPattern: /\S+/,
      inputErrorMessage: '名称不能为空',
    })
    const name = String(value || '').trim()
    if (!name) return
    await app.renameSession(sessionId, name)
    ElMessage.success('会话已重命名')
  } catch { /* cancelled */ }
}

async function handleFork() {
  try {
    await ElMessageBox.confirm('会基于当前历史创建一个新的 Codex 会话分支。', '分支会话', {
      confirmButtonText: '创建分支',
      cancelButtonText: '取消',
      type: 'info',
    })
    const forked = await app.forkSession(sessionId)
    ElMessage.success('分支会话已创建')
    router.push(`/session/${forked.id}`)
  } catch { /* cancelled */ }
}

async function handleCompact() {
  try {
    await ElMessageBox.confirm('Codex 会开始压缩当前会话上下文，过程会作为新的消息流显示。', '压缩上下文', {
      confirmButtonText: '开始压缩',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await app.compactSession(sessionId)
    ElMessage.success('已开始压缩上下文')
  } catch { /* cancelled */ }
}

async function handleRollback() {
  try {
    await ElMessageBox.confirm('会从 Codex 上下文中移除最近 1 轮，并写入回滚记录。这个操作无法在 CodexFlow 内撤销。', '回滚最近一轮', {
      confirmButtonText: '回滚',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await app.rollbackSession(sessionId, 1)
    ElMessage.success('已回滚最近一轮')
    await nextTick()
    scrollChatToBottom(true)
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
  document.addEventListener('visibilitychange', refreshSessionWhenVisible)
  window.addEventListener('focus', refreshSessionWhenVisible)
  scrollChatToBottom(true)
})

watch(summary, (next) => {
  if (!next?.loaded || !isMobile.value) return
  nextTick(() => {
    const input = document.querySelector('.session-detail-page .input-area')
    input?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  })
})

onUnmounted(() => {
  app.unregisterActiveSession(sessionId)
  document.removeEventListener('visibilitychange', refreshSessionWhenVisible)
  window.removeEventListener('focus', refreshSessionWhenVisible)
  window.removeEventListener('resize', onResize)
})
</script>

<style scoped>
.session-detail-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  margin: 0;
  overflow: hidden;
  min-height: 0;
}

.input-area {
  flex-shrink: 0;
}

.session-hero {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 14px;
  margin: 0 18px 8px;
  border: 1px solid var(--cf-border);
  border-radius: 14px;
  background:
    linear-gradient(140deg, rgba(51, 136, 255, 0.1) 0%, rgba(51, 136, 255, 0.02) 46%, rgba(255, 255, 255, 0.96) 100%),
    #fff;
  box-shadow: var(--cf-shadow-sm);
}

.hero-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.back-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 0;
  border-radius: 999px;
  padding: 6px 10px;
  background: rgba(255, 255, 255, 0.88);
  color: var(--cf-text-secondary);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  box-shadow: inset 0 0 0 1px rgba(205, 223, 255, 0.8);
}

.back-chip:hover {
  color: var(--cf-primary-dark);
  box-shadow: inset 0 0 0 1px rgba(121, 168, 255, 0.95);
}

.hero-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.hero-main {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 360px;
  gap: 12px;
  align-items: start;
}

.hero-title-group {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.hero-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.hero-name {
  font-size: 20px;
  line-height: 1.1;
  font-weight: 700;
  color: var(--cf-text-heavy);
}

.hero-meta-row {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 8px;
  flex-wrap: wrap;
}

.hero-cwd {
  font-size: 11px;
  color: var(--cf-text-secondary);
  font-family: monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
  order: 2;
}

.hero-tags {
  display: flex;
  align-items: center;
  gap: 5px;
  flex-wrap: wrap;
  order: 1;
}

.hero-pill {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  background: rgba(51, 136, 255, 0.08);
  color: var(--cf-primary-dark);
  font-size: 11px;
  font-weight: 600;
}

.hero-pill.is-active {
  background: rgba(19, 168, 107, 0.12);
  color: var(--cf-success);
}

.hero-preview {
  margin: 0;
  font-size: 11px;
  line-height: 1.45;
  color: var(--cf-text-secondary);
  max-width: 780px;
}

.hero-status-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 10px 12px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.85);
  border: 1px solid rgba(216, 230, 251, 0.95);
  box-shadow: 0 6px 14px rgba(15, 46, 106, 0.05);
}

.hero-status-copy {
  min-width: 0;
  flex: 1;
}

.hero-status-label {
  font-size: 11px;
  color: var(--cf-text-lighter);
  font-weight: 600;
}

.hero-status-value {
  font-size: 13px;
  line-height: 1.2;
  font-weight: 700;
  color: var(--cf-text-heavy);
  margin-top: 1px;
}

.hero-status-desc {
  font-size: 11px;
  line-height: 1.35;
  color: var(--cf-text-secondary);
}

.hero-primary-actions {
  display: flex;
  justify-content: flex-end;
  flex-shrink: 0;
}

.hero-primary-actions :deep(.el-button) {
  min-width: 116px;
  min-height: 30px;
  border-radius: 9px;
}

.hero-actions :deep(.el-button) {
  border-radius: 10px;
}

.live-indicator,
.streaming-hint {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--cf-warning);
}

.live-indicator {
  padding: 2px 7px;
  border-radius: 999px;
  background: rgba(245, 158, 11, 0.1);
  border: 1px solid rgba(245, 158, 11, 0.3);
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

.session-meta {
  padding: 8px 16px;
  background: var(--cf-card);
  border-bottom: 1px solid var(--cf-border-light);
  cursor: pointer;
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
  max-width: 420px;
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

.resume-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  background: #fffbeb;
  border-bottom: 1px solid #fde68a;
  font-size: 13px;
  color: #92400e;
}

.content-area {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 0 18px 0;
  background: linear-gradient(180deg, #eef5fd 0%, #e7f0fb 100%);
}

.approval-section {
  flex-shrink: 0;
  padding: 0 0 10px;
}

.approval-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: var(--cf-card);
  border-radius: 10px;
  border-left: 3px solid var(--cf-warning);
  margin-bottom: 6px;
  gap: 8px;
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

.chat-shell {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: rgba(248, 251, 255, 0.92);
  border: 1px solid #dce8f8;
  border-radius: 20px 20px 0 0;
  overflow: hidden;
  position: relative;
}

.chat-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px 10px;
  border-bottom: 1px solid rgba(220, 230, 246, 0.9);
  background: rgba(255, 255, 255, 0.85);
}

.toolbar-left,
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.follow-tip {
  font-size: 12px;
  color: var(--cf-text-secondary);
}

.chat-area {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 14px 18px 18px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.history-load-row,
.empty-hint {
  display: flex;
  justify-content: center;
}

.history-load-row {
  min-height: 24px;
}

.empty-hint {
  align-items: center;
  gap: 8px;
  padding: 40px 0;
  color: var(--cf-text-secondary);
  font-size: 14px;
}

.turn-stream {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.turn-anchor {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 12px;
  color: var(--cf-text-lighter);
}

.turn-title {
  font-weight: 600;
  color: var(--cf-text-secondary);
}

.message-row {
  display: flex;
  width: 100%;
}

.message-row.side-left {
  justify-content: flex-start;
}

.message-row.side-right {
  justify-content: flex-end;
}

.message-bubble {
  width: min(100%, 860px);
  border-radius: 18px;
  padding: 12px 14px;
  box-shadow: 0 10px 24px rgba(15, 46, 106, 0.04);
  border: 1px solid transparent;
}

.bubble-user {
  max-width: min(78%, 760px);
  background: #2f6fec;
  color: #fff;
  border-color: #2f6fec;
}

.bubble-user :deep(*) {
  color: #fff;
}

.bubble-agent {
  background: #ffffff;
  border-color: #d8e6fb;
}

.bubble-tool {
  background: #f8fbff;
  border-color: #d9e6f7;
}

.bubble-meta {
  background: #f7fafc;
  border-color: #e5ebf5;
}

.bubble-other {
  background: #ffffff;
  border-color: #e5e7eb;
}

.bubble-error {
  background: #fff5f5;
  border-color: #fecaca;
}

.message-topline {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 6px;
}

.message-label {
  font-size: 12px;
  font-weight: 700;
}

.message-status {
  font-size: 11px;
  opacity: 0.7;
}

.message-title {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 6px;
}

.message-body {
  font-size: 14px;
  line-height: 1.65;
  color: var(--cf-text-secondary);
}

.bubble-user .message-body {
  color: #fff;
}

.message-body.is-code pre,
.diff-block,
.message-aux pre {
  margin: 0;
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
  font-size: 12px;
  line-height: 1.55;
  white-space: pre-wrap;
}

.message-body.is-code pre,
.diff-block {
  padding: 10px 12px;
  border-radius: 10px;
  background: #0f172a;
  color: #e2e8f0;
}

.message-aux {
  margin-top: 10px;
}

.message-aux summary {
  cursor: pointer;
  font-size: 12px;
  color: var(--cf-text-secondary);
  margin-bottom: 8px;
}

.message-aux pre {
  max-height: 220px;
  overflow: auto;
  padding: 10px 12px;
  border-radius: 10px;
  background: rgba(15, 23, 42, 0.05);
  color: var(--cf-text-secondary);
}

.tool-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.tool-summary {
  display: flex;
  align-items: flex-start;
  justify-content: flex-start;
  gap: 8px;
}

.tool-main {
  min-width: 0;
  flex: 1;
}

.tool-name {
  font-size: 13px;
  font-weight: 700;
  color: var(--cf-text-heavy);
}

.tool-headline {
  margin-top: 2px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  max-width: 100%;
  font-size: 12px;
  line-height: 1.5;
  color: var(--cf-text-secondary);
}

.tool-type {
  flex-shrink: 0;
}

.tool-command-tag {
  display: inline-block;
  min-width: 0;
  max-width: min(100%, 560px);
  padding: 1px 8px;
  border-radius: 999px;
  background: rgba(51, 136, 255, 0.08);
  border: 1px solid rgba(151, 194, 255, 0.9);
  color: var(--cf-primary-dark);
  font-size: 11px;
  line-height: 1.6;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: middle;
}

.tool-details {
  border-top: 1px solid rgba(216, 230, 251, 0.9);
  padding-top: 10px;
}

.tool-details summary {
  cursor: pointer;
  font-size: 12px;
  color: var(--cf-text-secondary);
  font-weight: 600;
}

.tool-output {
  margin-top: 10px;
}

.tool-output-title {
  margin-bottom: 8px;
  font-size: 12px;
  font-weight: 700;
  color: var(--cf-text-secondary);
}

.markdown-body :deep(*) {
  word-break: break-word;
}

.markdown-body :deep(p),
.markdown-body :deep(ul),
.markdown-body :deep(ol),
.markdown-body :deep(blockquote),
.markdown-body :deep(pre),
.markdown-body :deep(table) {
  margin: 0 0 10px;
}

.markdown-body :deep(p:last-child),
.markdown-body :deep(ul:last-child),
.markdown-body :deep(ol:last-child),
.markdown-body :deep(blockquote:last-child),
.markdown-body :deep(pre:last-child),
.markdown-body :deep(table:last-child) {
  margin-bottom: 0;
}

.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: 20px;
}

.markdown-body :deep(code) {
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
  font-size: 12px;
  padding: 1px 4px;
  border-radius: 4px;
  background: rgba(15, 23, 42, 0.08);
}

.markdown-body :deep(pre) {
  overflow: auto;
  padding: 10px 12px;
  border-radius: 8px;
  background: rgba(15, 23, 42, 0.06);
}

.markdown-body :deep(img) {
  display: block;
  max-width: min(100%, 520px);
  width: auto;
  height: auto;
  margin: 12px 0 6px;
  border-radius: 14px;
  border: 1px solid rgba(216, 230, 251, 0.95);
  box-shadow: 0 10px 24px rgba(15, 46, 106, 0.08);
  background: #fff;
}

.typing-cursor {
  display: inline;
  color: var(--cf-primary);
  animation: blink-cursor 0.8s step-end infinite;
}

.new-message-pill {
  position: absolute;
  right: 18px;
  bottom: 18px;
  z-index: 4;
  border: 0;
  border-radius: 999px;
  padding: 10px 14px;
  background: #2f6fec;
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  box-shadow: 0 10px 24px rgba(47, 111, 236, 0.28);
  cursor: pointer;
}

.new-message-pill-enter-active,
.new-message-pill-leave-active {
  transition: all 0.2s ease;
}

.new-message-pill-enter-from,
.new-message-pill-leave-to {
  opacity: 0;
  transform: translateY(8px);
}

@keyframes blink-cursor {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

.input-area {
  padding: 10px 16px;
  background: var(--cf-card);
  border-top: 1px solid var(--cf-border-light);
  box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.04);
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

.session-detail-page.is-mobile {
  height: auto;
  min-height: 100%;
  overflow: visible;
}

.session-detail-page.is-mobile .session-hero {
  margin: 0 10px 12px;
  padding: 12px;
  border-radius: 16px;
}

.session-detail-page.is-mobile .hero-top,
.session-detail-page.is-mobile .hero-main {
  display: flex;
  flex-direction: column;
}

.session-detail-page.is-mobile .hero-status-card {
  flex-direction: column;
  align-items: stretch;
}

.session-detail-page.is-mobile .hero-primary-actions {
  justify-content: flex-start;
}

.session-detail-page.is-mobile .hero-name {
  font-size: 20px;
}

.session-detail-page.is-mobile .hero-preview {
  max-width: 100%;
}

.session-detail-page.is-mobile .hero-status-card {
  width: 100%;
}

.session-detail-page.is-mobile .content-area {
  overflow: visible;
  min-height: auto;
  padding: 0 0 0;
  background: transparent;
}

.session-detail-page.is-mobile .chat-shell {
  border-radius: 14px;
}

.session-detail-page.is-mobile .chat-area {
  padding: 10px 12px 14px;
}

.session-detail-page.is-mobile .new-message-pill {
  right: 12px;
  bottom: 12px;
}

.session-detail-page.is-mobile .message-bubble,
.session-detail-page.is-mobile .bubble-user {
  max-width: 100%;
  width: 100%;
}

.session-detail-page.is-mobile .input-area {
  position: sticky;
  bottom: 0;
  z-index: 5;
  padding: 8px 10px;
  box-shadow: 0 -6px 18px rgba(15, 46, 106, 0.08);
}

.session-detail-page.is-mobile .input-row :deep(.el-textarea__inner) {
  font-size: 16px;
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
