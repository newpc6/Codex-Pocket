import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../utils/api'
import { sseService, type SSEEvent, type SSEStatus } from '../utils/sse'

export interface AgentInfo {
  id: string
  name: string
  available: boolean
  default: boolean
}

export interface SessionOptionItem {
  id: string
  name: string
  description?: string
  default?: boolean
}

export interface SessionPreset extends SessionOptionItem {
  model: string
  reasoningEffort: string
  collaborationMode: string
}

export interface SessionOptions {
  models: SessionOptionItem[]
  reasoningEfforts: SessionOptionItem[]
  collaborationModes: SessionOptionItem[]
  presets: SessionPreset[]
}

export interface SessionSummary {
  id: string
  agentId: string
  name: string
  agentNickname: string
  cwd: string
  preview: string
  source: string
  branch: string
  modelProvider: string
  loaded: boolean
  ended: boolean
  status: string
  lifecycleStage: string
  updatedAt: number
  activeFlags: string[]
  lastTurnStatus: string
  lastTurnId: string
}

export interface ApprovalRequest {
  id: string
  threadId: string
  kind: string
  reason: string
  summary: string
  choices: string[]
  createdAt: string
  params: Record<string, any>
}

export interface TurnItem {
  id: string
  type: string
  title: string
  body: string
  status: string
  auxiliary: string
  metadata?: Record<string, string>
}

export interface Turn {
  id: string
  status: string
  startedAt: number
  durationMs: number
  planExplanation: string
  plan: Array<{ step: string; status: string }>
  diff: string
  error: string
  items: TurnItem[]
}

export interface SessionGoal {
  objective: string
  status: string
  tokenBudget: number
  tokensUsed: number
  timeUsedSeconds: number
}

export interface SessionDetail {
  summary: SessionSummary
  goal?: SessionGoal
  turns: Turn[]
  totalTurns: number
  offset: number
  limit: number
  hasMoreHistory: boolean
}

export interface ChangedFile {
  path: string
  oldPath?: string
  status: string
  additions: number
  deletions: number
  binary: boolean
  untracked: boolean
}

export interface ChangedFileDetail extends ChangedFile {
  diff: string
  content: string
  truncated: boolean
  readable: boolean
  error?: string
}

export interface SessionChanges {
  scope: string
  ref: string
  base: string
  turnId?: string
  cwd: string
  summary: {
    files: number
    additions: number
    deletions: number
    untracked: number
  }
  files: ChangedFile[]
  diff: string
  file?: ChangedFileDetail
  generated: number
}

export interface DashboardData {
  agent: {
    connected: boolean
    startedAt: string
    listenAddr: string
    codexBinaryPath: string
  }
  agents: AgentInfo[]
  defaultAgent: string
  options: SessionOptions
  stats: {
    totalSessions: number
    loadedSessions: number
    activeSessions: number
    pendingApprovals: number
  }
  sessions: SessionSummary[]
  approvals: ApprovalRequest[]
}

function parseNotificationParams(event: SSEEvent): Record<string, any> {
  const payload = event.payload || {}
  const params = payload.params

  if (typeof params === 'string') {
    try {
      const decoded = JSON.parse(params)
      return decoded && typeof decoded === 'object' ? decoded : {}
    } catch {
      return {}
    }
  }

  if (params && typeof params === 'object') {
    return params as Record<string, any>
  }

  return {}
}

function parseNotificationThreadId(event: SSEEvent): string {
  const payload = event.payload || {}
  const directThreadId = typeof payload.threadId === 'string' ? payload.threadId : ''
  if (directThreadId) return directThreadId

  const params = parseNotificationParams(event)
  return typeof params.threadId === 'string' ? params.threadId : ''
}

function defaultSessionOptions(): SessionOptions {
  return {
    models: [
      { id: '', name: 'Codex 默认', description: '沿用当前 Codex 配置', default: true },
      { id: 'gpt-5-codex', name: 'GPT-5 Codex', description: '适合认真改代码和审查' },
      { id: 'gpt-5-mini', name: 'GPT-5 Mini', description: '适合快速任务' },
    ],
    reasoningEfforts: [
      { id: '', name: '默认', default: true },
      { id: 'minimal', name: '快一点' },
      { id: 'medium', name: '认真改' },
      { id: 'high', name: '深度审查' },
    ],
    collaborationModes: [
      { id: 'default', name: '默认协作', description: '可以分析、修改并验证', default: true },
      { id: 'plan', name: '只分析不改', description: '先给计划和建议' },
      { id: 'review', name: '代码审查', description: '优先找风险和测试缺口' },
    ],
    presets: [
      { id: 'balanced', name: '认真改', description: '适合日常开发', model: '', reasoningEffort: 'medium', collaborationMode: 'default' },
      { id: 'fast', name: '快一点', description: '轻量修改和问答', model: '', reasoningEffort: 'minimal', collaborationMode: 'default' },
      { id: 'review', name: '代码审查', description: '只看风险和回归', model: '', reasoningEffort: 'high', collaborationMode: 'review' },
      { id: 'analysis', name: '只分析不改', description: '先讨论方案', model: '', reasoningEffort: 'medium', collaborationMode: 'plan' },
    ],
  }
}

function normalizeSessionOptions(input?: SessionOptions): SessionOptions {
  const fallback = defaultSessionOptions()
  return {
    models: input?.models?.length ? input.models : fallback.models,
    reasoningEfforts: input?.reasoningEfforts?.length ? input.reasoningEfforts : fallback.reasoningEfforts,
    collaborationModes: input?.collaborationModes?.length ? input.collaborationModes : fallback.collaborationModes,
    presets: input?.presets?.length ? input.presets : fallback.presets,
  }
}

export const useAppStore = defineStore('app', () => {
  const dashboard = ref<DashboardData>({
    agent: { connected: false, startedAt: '', listenAddr: '', codexBinaryPath: '' },
    agents: [],
    defaultAgent: 'codex',
    options: defaultSessionOptions(),
    stats: { totalSessions: 0, loadedSessions: 0, activeSessions: 0, pendingApprovals: 0 },
    sessions: [],
    approvals: [],
  })
  const sessionDetails = ref<Record<string, SessionDetail>>({})
  const selectedAgentId = ref('codex')
  const loading = ref(false)
  const error = ref('')
  const sseConnected = ref(false)
  const sseStatus = ref<SSEStatus>('disconnected')
  const lastEvent = ref<SSEEvent | null>(null)
  const compactingSessionIds = ref<Set<string>>(new Set())

  // Track which sessions are currently being viewed (for targeted refresh)
  const activeSessionIds = ref<Set<string>>(new Set())
  const sessionRefreshTimers = new Map<string, ReturnType<typeof setTimeout>>()
  const activeSessionPollers = new Map<string, ReturnType<typeof setInterval>>()
  const sessionLoadInFlight = new Map<string, Promise<void>>()
  const localPromptItemsBySession = new Map<string, TurnItem[]>()
  let dashboardRefreshInFlight: Promise<void> | null = null
  let dashboardRefreshTimer: ReturnType<typeof setTimeout> | null = null
  let lastDashboardRefreshAt = 0
  let sseHandlersBound = false

  const filteredSessions = computed(() => {
    return dashboard.value.sessions.filter((s) => s.agentId === selectedAgentId.value)
  })

  const filteredApprovals = computed(() => {
    const ids = new Set(filteredSessions.value.map((s) => s.id))
    return dashboard.value.approvals.filter((a) => ids.has(a.threadId))
  })

  const sessionGroups = computed(() => {
    const sessions = filteredSessions.value
    return {
      managed: sessions.filter((s) => s.lifecycleStage === 'managed'),
      ended: sessions.filter((s) => s.lifecycleStage === 'ended'),
      runtimeAvailable: sessions.filter((s) => s.lifecycleStage === 'runtime_available'),
      discovered: sessions.filter((s) => s.lifecycleStage === 'discovered'),
      historyOnly: sessions.filter((s) => s.lifecycleStage === 'history_only'),
    }
  })

  const isAgentOnline = computed(() => dashboard.value.agent.connected)

  async function refreshDashboard(options?: { force?: boolean }) {
    const now = Date.now()
    if (!options?.force && dashboardRefreshInFlight) {
      return dashboardRefreshInFlight
    }
    if (!options?.force && now - lastDashboardRefreshAt < 500) {
      return dashboardRefreshInFlight || Promise.resolve()
    }

    loading.value = true
    error.value = ''
    dashboardRefreshInFlight = (async () => {
      const res = await api.get<DashboardData>('/dashboard')
      dashboard.value = { ...res.data, options: normalizeSessionOptions(res.data.options) }
      syncSelectedAgent(res.data)
      lastDashboardRefreshAt = Date.now()
    })()

    try {
      await dashboardRefreshInFlight
    } catch (e: any) {
      error.value = e.response?.data?.error || e.message
    } finally {
      loading.value = false
      dashboardRefreshInFlight = null
    }
  }

  function syncSelectedAgent(data: DashboardData) {
    const available = (data.agents || []).filter((a) => a.available).map((a) => a.id)
    if (!available.includes(selectedAgentId.value)) {
      const def = data.defaultAgent?.toLowerCase()
      selectedAgentId.value = available.includes(def) ? def : (available.includes('codex') ? 'codex' : available[0] || 'codex')
    }
  }

  async function loadSession(id: string, options?: { offset?: number; limit?: number; appendHistory?: boolean; fast?: boolean }) {
    const canReuse = !options?.appendHistory && typeof options?.offset !== 'number' && typeof options?.limit !== 'number'
    const inFlightKey = `${id}:${options?.fast ? 'fast' : 'full'}`
    if (canReuse) {
      const pending = sessionLoadInFlight.get(inFlightKey)
      if (pending) return pending
    }

    const run = async () => {
      try {
        const params: Record<string, number> = {}
        if (typeof options?.offset === 'number') params.offset = options.offset
        if (typeof options?.limit === 'number') params.limit = options.limit
        if (options?.fast) params.fast = 1
        const res = await api.get<SessionDetail>(`/sessions/${id}`, { params })
        const nextDetail = withLocalPromptItems(id, res.data)
        if (options?.appendHistory && sessionDetails.value[id]) {
          const existing = sessionDetails.value[id]
          const mergedTurns = [...nextDetail.turns, ...existing.turns]
          sessionDetails.value[id] = {
            ...nextDetail,
            turns: mergedTurns,
          }
          return
        }
        if (!options?.appendHistory && sessionDetails.value[id]) {
          const existing = sessionDetails.value[id]
          const keepCount = nextDetail.offset - existing.offset
          if (keepCount > 0 && existing.turns.length >= keepCount) {
            sessionDetails.value[id] = {
              ...nextDetail,
              turns: [...existing.turns.slice(0, keepCount), ...nextDetail.turns],
              offset: existing.offset,
              hasMoreHistory: existing.offset > 0,
            }
            return
          }
        }
        sessionDetails.value[id] = nextDetail
      } catch (e: any) {
        error.value = e.response?.data?.error || e.message
      }
    }

    const promise = run()
    if (canReuse) {
      sessionLoadInFlight.set(inFlightKey, promise)
      promise.finally(() => {
        if (sessionLoadInFlight.get(inFlightKey) === promise) {
          sessionLoadInFlight.delete(inFlightKey)
        }
      })
    }
    return promise
  }

  function scheduleSessionLoad(id: string, delay = 120) {
    if (!id) return
    const existing = sessionRefreshTimers.get(id)
    if (existing) clearTimeout(existing)
    sessionRefreshTimers.set(id, setTimeout(async () => {
      sessionRefreshTimers.delete(id)
      await loadSession(id)
    }, delay))
  }

  function replaceSessionDetail(id: string, detail: SessionDetail) {
    sessionDetails.value[id] = detail
  }

  function markSessionCompacting(id: string, compacting: boolean) {
    if (!id) return
    const next = new Set(compactingSessionIds.value)
    if (compacting) next.add(id)
    else next.delete(id)
    compactingSessionIds.value = next
  }

  function scheduleDashboardRefresh(delay = 500) {
    if (dashboardRefreshTimer) clearTimeout(dashboardRefreshTimer)
    dashboardRefreshTimer = setTimeout(() => {
      dashboardRefreshTimer = null
      void refreshDashboard()
    }, delay)
  }

  function ensureSessionTurn(detail: SessionDetail, turnId: string): Turn {
    let turn = detail.turns.find((entry) => entry.id === turnId)
    if (turn) return turn

    turn = {
      id: turnId,
      status: 'inProgress',
      startedAt: Date.now(),
      durationMs: 0,
      planExplanation: '',
      plan: [],
      diff: '',
      error: '',
      items: [],
    }
    detail.turns.push(turn)
    detail.totalTurns = Math.max(detail.totalTurns || 0, detail.turns.length)
    return turn
  }

  function inputItemsToTurnItems(input: Array<Record<string, string>>): TurnItem[] {
    const textParts = input
      .filter((entry) => entry.type === 'text' && entry.text?.trim())
      .map((entry) => entry.text.trim())
    const imageParts = input
      .filter((entry) => entry.type === 'image' && entry.uploadId?.trim())
      .map((entry) => `[Attached image: upload:${entry.uploadId.trim()}]`)
    const body = [...textParts, ...imageParts].join('\n\n')
    if (!body) return []
    return [{
      id: `local-user-${Date.now()}`,
      type: 'userMessage',
      title: 'User Prompt',
      body,
      status: '',
      auxiliary: '',
      metadata: { localInput: 'true' },
    }]
  }

  function rememberLocalPromptItems(sessionId: string, input: Array<Record<string, string>>) {
    const items = inputItemsToTurnItems(input)
    if (items.length === 0) return
    localPromptItemsBySession.set(sessionId, items)
    try {
      sessionStorage.setItem(`cf_local_prompt:${sessionId}`, JSON.stringify(items))
    } catch {
      // Best-effort cache so the just-created prompt survives a route refresh.
    }
  }

  function localPromptItems(sessionId: string): TurnItem[] {
    const cached = localPromptItemsBySession.get(sessionId)
    if (cached?.length) return cached
    try {
      const raw = sessionStorage.getItem(`cf_local_prompt:${sessionId}`)
      if (!raw) return []
      const parsed = JSON.parse(raw)
      if (!Array.isArray(parsed)) return []
      const items = parsed.filter((item) => item?.type === 'userMessage' && item?.body)
      if (items.length) localPromptItemsBySession.set(sessionId, items)
      return items
    } catch {
      return []
    }
  }

  function withLocalPromptItems(sessionId: string, detail: SessionDetail): SessionDetail {
    const items = localPromptItems(sessionId)
    if (items.length === 0 || detail.turns.length === 0) return detail

    const lastTurnIndex = detail.summary.lastTurnId
      ? detail.turns.findIndex((turn) => turn.id === detail.summary.lastTurnId)
      : -1
    const turnIndex = lastTurnIndex >= 0 ? lastTurnIndex : detail.turns.length - 1
    const turns = detail.turns.map((turn, index) => {
      if (index !== turnIndex) return turn
      const existingBodies = new Set(turn.items.filter((item) => item.type === 'userMessage').map((item) => item.body))
      const missing = items.filter((item) => !existingBodies.has(item.body))
      if (missing.length === 0) return turn
      return {
        ...turn,
        items: [...missing, ...turn.items],
      }
    })
    return { ...detail, turns }
  }

  function appendLocalUserInput(threadId: string, turnId: string, input: Array<Record<string, string>>) {
    const detail = sessionDetails.value[threadId]
    if (!detail) return
    const turn = ensureSessionTurn(detail, turnId)
    turn.status = 'inProgress'
    detail.summary.lastTurnId = turnId
    detail.summary.lastTurnStatus = 'inProgress'
    detail.summary.loaded = true
    const newItems = inputItemsToTurnItems(input)
    if (newItems.length === 0) return
    const hasUserMessage = turn.items.some((item) => item.type === 'userMessage' && item.body === newItems[0].body)
    if (!hasUserMessage) {
      turn.items.push(...newItems)
    }
  }

  function applyAgentMessageDelta(threadId: string, params: Record<string, any>) {
    const detail = sessionDetails.value[threadId]
    if (!detail) return

    const turnId = typeof params.turnId === 'string' ? params.turnId : ''
    const itemId = typeof params.itemId === 'string' ? params.itemId : ''
    const delta = typeof params.delta === 'string' ? params.delta : ''
    if (!turnId || !itemId || !delta) return

    const turn = ensureSessionTurn(detail, turnId)
    turn.status = 'inProgress'
    detail.summary.lastTurnId = turnId
    detail.summary.lastTurnStatus = 'inProgress'

    let item = turn.items.find((entry) => entry.id === itemId)
    if (!item) {
      item = {
        id: itemId,
        type: 'agentMessage',
        title: 'Agent',
        body: '',
        status: '',
        auxiliary: '',
      }
      turn.items.push(item)
    }

    if (item.type !== 'agentMessage') {
      item.type = 'agentMessage'
    }
    item.body = `${item.body || ''}${delta}`
  }

  async function resumeSession(id: string) {
    const res = await api.post(`/sessions/${id}/resume`)
    await refreshDashboard()
    await loadSession(id)
    return res.data
  }

  async function endSession(id: string) {
    await api.post(`/sessions/${id}/end`)
    await refreshDashboard()
    await loadSession(id)
  }

  async function detachSession(id: string) {
    await api.post(`/sessions/${id}/detach`)
    await refreshDashboard()
    await loadSession(id)
  }

  async function archiveSession(id: string) {
    await api.post(`/sessions/${id}/archive`)
    delete sessionDetails.value[id]
    await refreshDashboard()
  }

  async function renameSession(id: string, name: string) {
    const res = await api.post<SessionSummary>(`/sessions/${id}/rename`, { name })
    await refreshDashboard()
    await loadSession(id)
    return res.data
  }

  async function forkSession(id: string) {
    const res = await api.post<SessionSummary>(`/sessions/${id}/fork`)
    await refreshDashboard()
    await loadSession(res.data.id)
    return res.data
  }

  async function compactSession(id: string) {
    markSessionCompacting(id, true)
    await api.post(`/sessions/${id}/compact`)
    await refreshDashboard()
    await loadSession(id)
  }

  async function rollbackSession(id: string, numTurns: number) {
    const res = await api.post<SessionDetail>(`/sessions/${id}/rollback`, { numTurns })
    sessionDetails.value[id] = res.data
    await refreshDashboard()
    return res.data
  }

  async function setSessionGoal(id: string, objective: string, tokenBudget = 0) {
    const res = await api.post<SessionGoal>(`/sessions/${id}/goal`, {
      objective,
      status: 'active',
      tokenBudget,
    })
    if (sessionDetails.value[id]) {
      sessionDetails.value[id].goal = res.data
    }
    return res.data
  }

  async function clearSessionGoal(id: string) {
    await api.post(`/sessions/${id}/goal/clear`)
    if (sessionDetails.value[id]) {
      delete sessionDetails.value[id].goal
    }
  }

  async function startTurn(sessionId: string, prompt: string, imageUploadIds: string[] = []) {
    const inputs: Array<Record<string, string>> = []
    if (prompt.trim()) inputs.push({ type: 'text', text: prompt.trim() })
    for (const uid of imageUploadIds) inputs.push({ type: 'image', uploadId: uid })
    const res = await api.post<Turn>(`/sessions/${sessionId}/turns/start`, { prompt, inputs })
    if (res.data?.id) {
      appendLocalUserInput(sessionId, res.data.id, inputs)
    }
    await refreshDashboard()
    await loadSession(sessionId)
  }

  async function steerTurn(sessionId: string, turnId: string, prompt: string, imageUploadIds: string[] = []) {
    const inputs: Array<Record<string, string>> = []
    if (prompt.trim()) inputs.push({ type: 'text', text: prompt.trim() })
    for (const uid of imageUploadIds) inputs.push({ type: 'image', uploadId: uid })
    await api.post(`/sessions/${sessionId}/turns/steer`, { turnId, prompt, inputs })
    appendLocalUserInput(sessionId, turnId, inputs)
    await refreshDashboard()
    await loadSession(sessionId)
  }

  async function interruptTurn(sessionId: string, turnId: string) {
    await api.post(`/sessions/${sessionId}/turns/interrupt`, { turnId })
    await refreshDashboard()
  }

  async function resolveApproval(id: string, result: Record<string, any>) {
    await api.post(`/approvals/${id}/resolve`, { result })
    await refreshDashboard()
  }

  async function loadOptions() {
    const res = await api.get<SessionOptions>('/options')
    dashboard.value.options = normalizeSessionOptions(res.data)
    return dashboard.value.options
  }

  async function loadSessionChanges(sessionId: string, params?: { scope?: string; ref?: string; base?: string; turnId?: string; file?: string }) {
    const res = await api.get<SessionChanges>(`/sessions/${sessionId}/changes`, { params })
    return res.data
  }

  async function revertSessionChanges(sessionId: string, files: string[]) {
    const res = await api.post<SessionChanges>(`/sessions/${sessionId}/changes/revert`, { files })
    await loadSession(sessionId)
    await refreshDashboard()
    return res.data
  }

  async function startReview(sessionId: string, params?: { scope?: string; ref?: string; base?: string; turnId?: string }) {
    const res = await api.post<Turn>(`/sessions/${sessionId}/review`, params || {})
    if (res.data?.id) {
      appendLocalUserInput(sessionId, res.data.id, [{ type: 'text', text: '审查改动' }])
    }
    await refreshDashboard()
    await loadSession(sessionId)
    return res.data
  }

  async function startSession(cwd: string, prompt: string, agentId: string, options?: {
    model?: string
    reasoningEffort?: string
    collaborationMode?: string
  }) {
    const res = await api.post('/sessions', {
      action: 'start',
      cwd,
      prompt,
      agent: agentId,
      model: options?.model || '',
      reasoningEffort: options?.reasoningEffort || '',
      collaborationMode: options?.collaborationMode || '',
    })
    const inputs: Array<Record<string, string>> = []
    if (prompt.trim()) inputs.push({ type: 'text', text: prompt.trim() })
    if (res.data?.id) rememberLocalPromptItems(res.data.id, inputs)
    await refreshDashboard()
    await loadSession(res.data.id)
    return res.data
  }

  // ---- SSE Integration ----

  function connectSSE() {
    bindSSEHandlers()
    sseService.connect()
  }

  function bindSSEHandlers() {
    if (sseHandlersBound) return
    sseHandlersBound = true

    sseService.onStatus((status) => {
      sseStatus.value = status
      sseConnected.value = status === 'connected'
    })

    // Wildcard handler: update lastEvent for any event
    sseService.on('*', (event: SSEEvent) => {
      lastEvent.value = event
    })

    // Codex notifications: turn progress, plan updates, diff updates, etc.
    sseService.on('codex.notification', async (event: SSEEvent) => {
      const method = event.payload?.method as string
      if (!method) return

      const threadId = parseNotificationThreadId(event)
      if (threadId && (activeSessionIds.value.has(threadId) || !!sessionDetails.value[threadId])) {
        const params = parseNotificationParams(event)
        if (method === 'agentMessage/delta' || method === 'item/agentMessage/delta') {
          applyAgentMessageDelta(threadId, params)
        } else {
          scheduleSessionLoad(threadId, 250)
        }
      }

      // Always refresh dashboard on significant events
      if ([
        'turn/started', 'turn/completed',
        'thread/started', 'thread/status/changed', 'thread/closed',
        'thread/goal/updated', 'thread/goal/cleared',
        'item/started', 'item/completed',
      ].includes(method)) {
        scheduleDashboardRefresh(350)
      }
    })

    // Approval events
    sseService.on('approval.created', async () => {
      scheduleDashboardRefresh(300)
    })

    sseService.on('approval.resolved', async () => {
      scheduleDashboardRefresh(300)
    })

    // Session lifecycle events
    sseService.on('session.created', async () => {
      scheduleDashboardRefresh(300)
    })

    sseService.on('session.resumed', async () => {
      scheduleDashboardRefresh(300)
    })

    sseService.on('session.detached', async (event: SSEEvent) => {
      const threadId = event.payload?.threadId as string
      if (threadId) markSessionCompacting(threadId, false)
      if (threadId && (activeSessionIds.value.has(threadId) || !!sessionDetails.value[threadId])) {
        await loadSession(threadId)
      }
      scheduleDashboardRefresh(300)
    })

    sseService.on('session.ended', async () => {
      scheduleDashboardRefresh(300)
    })

    sseService.on('session.archived', async () => {
      scheduleDashboardRefresh(300)
    })

    // Turn events
    sseService.on('turn.started', async (event: SSEEvent) => {
      const threadId = event.payload?.threadId as string
      if (threadId) markSessionCompacting(threadId, false)
      if (threadId && (activeSessionIds.value.has(threadId) || !!sessionDetails.value[threadId])) {
        await loadSession(threadId)
      }
      scheduleDashboardRefresh(300)
    })

    sseService.on('turn.steered', async (event: SSEEvent) => {
      const threadId = event.payload?.threadId as string
      if (threadId && (activeSessionIds.value.has(threadId) || !!sessionDetails.value[threadId])) {
        await loadSession(threadId)
      }
    })

    sseService.on('turn.interrupted', async (event: SSEEvent) => {
      const threadId = event.payload?.threadId as string
      if (threadId) markSessionCompacting(threadId, false)
      if (threadId && (activeSessionIds.value.has(threadId) || !!sessionDetails.value[threadId])) {
        await loadSession(threadId)
      }
      scheduleDashboardRefresh(300)
    })

    sseService.on('turn.completed', async (event: SSEEvent) => {
      const threadId = event.payload?.threadId as string
      if (threadId) markSessionCompacting(threadId, false)
      if (threadId && (activeSessionIds.value.has(threadId) || !!sessionDetails.value[threadId])) {
        await loadSession(threadId)
      }
      scheduleDashboardRefresh(300)
    })

    // Sessions refreshed
    sseService.on('sessions.refreshed', async () => {
      scheduleDashboardRefresh(500)
    })

    sseService.on('session.compacting', async (event: SSEEvent) => {
      const threadId = event.payload?.threadId as string
      if (!threadId) return
      markSessionCompacting(threadId, true)
      if (activeSessionIds.value.has(threadId) || !!sessionDetails.value[threadId]) {
        scheduleSessionLoad(threadId, 350)
      }
    })
  }

  function disconnectSSE() {
    sseService.disconnect()
    for (const timer of sessionRefreshTimers.values()) clearTimeout(timer)
    sessionRefreshTimers.clear()
    if (dashboardRefreshTimer) {
      clearTimeout(dashboardRefreshTimer)
      dashboardRefreshTimer = null
    }
    for (const poller of activeSessionPollers.values()) clearInterval(poller)
    activeSessionPollers.clear()
  }

  function registerActiveSession(id: string) {
    activeSessionIds.value.add(id)
    if (activeSessionPollers.has(id)) return
    const poller = setInterval(async () => {
      if (!activeSessionIds.value.has(id)) return
      if (document.visibilityState !== 'visible') return
      const summary = dashboard.value.sessions.find((s) => s.id === id)
      const knownDetail = sessionDetails.value[id]
      const aggressive = !!(summary?.loaded
        || summary?.lastTurnStatus === 'inProgress'
        || summary?.status === 'active'
        || summary?.lifecycleStage === 'history_only'
        || knownDetail?.summary?.lastTurnStatus === 'inProgress')
      if (!aggressive && !knownDetail) return
      if (sseConnected.value) return
      await loadSession(id)
      if (aggressive && !sseConnected.value) {
        await refreshDashboard()
      }
    }, 2500)
    activeSessionPollers.set(id, poller)
  }

  function unregisterActiveSession(id: string) {
    activeSessionIds.value.delete(id)
    const poller = activeSessionPollers.get(id)
    if (poller) {
      clearInterval(poller)
      activeSessionPollers.delete(id)
    }
  }

  return {
    dashboard, sessionDetails, selectedAgentId, loading, error, compactingSessionIds,
    sseConnected, sseStatus, lastEvent, activeSessionIds,
    filteredSessions, filteredApprovals, sessionGroups, isAgentOnline,
    refreshDashboard, loadSession, resumeSession, detachSession, endSession, archiveSession,
    renameSession, forkSession, compactSession, rollbackSession,
    setSessionGoal, clearSessionGoal,
    startTurn, steerTurn, interruptTurn, resolveApproval, startSession,
    loadOptions, loadSessionChanges, revertSessionChanges, startReview,
    replaceSessionDetail,
    connectSSE, disconnectSSE, registerActiveSession, unregisterActiveSession,
  }
})
