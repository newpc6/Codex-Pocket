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

export interface SessionDetail {
  summary: SessionSummary
  turns: Turn[]
  totalTurns: number
  offset: number
  limit: number
  hasMoreHistory: boolean
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

export const useAppStore = defineStore('app', () => {
  const dashboard = ref<DashboardData>({
    agent: { connected: false, startedAt: '', listenAddr: '', codexBinaryPath: '' },
    agents: [],
    defaultAgent: 'codex',
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

  // Track which sessions are currently being viewed (for targeted refresh)
  const activeSessionIds = ref<Set<string>>(new Set())
  const sessionRefreshTimers = new Map<string, ReturnType<typeof setTimeout>>()
  const activeSessionPollers = new Map<string, ReturnType<typeof setInterval>>()
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

  async function refreshDashboard() {
    loading.value = true
    error.value = ''
    try {
      const res = await api.get<DashboardData>('/dashboard')
      dashboard.value = res.data
      syncSelectedAgent(res.data)
    } catch (e: any) {
      error.value = e.response?.data?.error || e.message
    } finally {
      loading.value = false
    }
  }

  function syncSelectedAgent(data: DashboardData) {
    const available = (data.agents || []).filter((a) => a.available).map((a) => a.id)
    if (!available.includes(selectedAgentId.value)) {
      const def = data.defaultAgent?.toLowerCase()
      selectedAgentId.value = available.includes(def) ? def : (available.includes('codex') ? 'codex' : available[0] || 'codex')
    }
  }

  async function loadSession(id: string, options?: { offset?: number; limit?: number; appendHistory?: boolean }) {
    try {
      const params: Record<string, number> = {}
      if (typeof options?.offset === 'number') params.offset = options.offset
      if (typeof options?.limit === 'number') params.limit = options.limit
      const res = await api.get<SessionDetail>(`/sessions/${id}`, { params })
      if (options?.appendHistory && sessionDetails.value[id]) {
        const existing = sessionDetails.value[id]
        const mergedTurns = [...res.data.turns, ...existing.turns]
        sessionDetails.value[id] = {
          ...res.data,
          turns: mergedTurns,
        }
        return
      }
      if (!options?.appendHistory && sessionDetails.value[id]) {
        const existing = sessionDetails.value[id]
        const keepCount = res.data.offset - existing.offset
        if (keepCount > 0 && existing.turns.length >= keepCount) {
          sessionDetails.value[id] = {
            ...res.data,
            turns: [...existing.turns.slice(0, keepCount), ...res.data.turns],
            offset: existing.offset,
            hasMoreHistory: existing.offset > 0,
          }
          return
        }
      }
      sessionDetails.value[id] = res.data
    } catch (e: any) {
      error.value = e.response?.data?.error || e.message
    }
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

  async function startTurn(sessionId: string, prompt: string, imageUploadIds: string[] = []) {
    const inputs: Array<Record<string, string>> = []
    if (prompt.trim()) inputs.push({ type: 'text', text: prompt.trim() })
    for (const uid of imageUploadIds) inputs.push({ type: 'image', uploadId: uid })
    await api.post(`/sessions/${sessionId}/turns/start`, { prompt, inputs })
    await refreshDashboard()
    await loadSession(sessionId)
  }

  async function steerTurn(sessionId: string, turnId: string, prompt: string, imageUploadIds: string[] = []) {
    const inputs: Array<Record<string, string>> = []
    if (prompt.trim()) inputs.push({ type: 'text', text: prompt.trim() })
    for (const uid of imageUploadIds) inputs.push({ type: 'image', uploadId: uid })
    await api.post(`/sessions/${sessionId}/turns/steer`, { turnId, prompt, inputs })
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

  async function startSession(cwd: string, prompt: string, agentId: string) {
    const res = await api.post('/sessions', { action: 'start', cwd, prompt, agent: agentId })
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
        if (method === 'agentMessage/delta') {
          applyAgentMessageDelta(threadId, params)
        } else {
          scheduleSessionLoad(threadId, 80)
        }
      }

      // Always refresh dashboard on significant events
      if ([
        'turn/started', 'turn/completed',
        'thread/started', 'thread/status/changed', 'thread/closed',
        'item/started', 'item/completed',
      ].includes(method)) {
        await refreshDashboard()
      }
    })

    // Approval events
    sseService.on('approval.created', async () => {
      await refreshDashboard()
    })

    sseService.on('approval.resolved', async () => {
      await refreshDashboard()
    })

    // Session lifecycle events
    sseService.on('session.created', async () => {
      await refreshDashboard()
    })

    sseService.on('session.resumed', async () => {
      await refreshDashboard()
    })

    sseService.on('session.ended', async () => {
      await refreshDashboard()
    })

    sseService.on('session.archived', async () => {
      await refreshDashboard()
    })

    // Turn events
    sseService.on('turn.started', async (event: SSEEvent) => {
      const threadId = event.payload?.threadId as string
      if (threadId && (activeSessionIds.value.has(threadId) || !!sessionDetails.value[threadId])) {
        await loadSession(threadId)
      }
      await refreshDashboard()
    })

    sseService.on('turn.steered', async (event: SSEEvent) => {
      const threadId = event.payload?.threadId as string
      if (threadId && (activeSessionIds.value.has(threadId) || !!sessionDetails.value[threadId])) {
        await loadSession(threadId)
      }
    })

    sseService.on('turn.interrupted', async (event: SSEEvent) => {
      const threadId = event.payload?.threadId as string
      if (threadId && (activeSessionIds.value.has(threadId) || !!sessionDetails.value[threadId])) {
        await loadSession(threadId)
      }
      await refreshDashboard()
    })

    // Sessions refreshed
    sseService.on('sessions.refreshed', async () => {
      await refreshDashboard()
    })
  }

  function disconnectSSE() {
    sseService.disconnect()
    for (const timer of sessionRefreshTimers.values()) clearTimeout(timer)
    sessionRefreshTimers.clear()
    for (const poller of activeSessionPollers.values()) clearInterval(poller)
    activeSessionPollers.clear()
  }

  function registerActiveSession(id: string) {
    activeSessionIds.value.add(id)
    if (activeSessionPollers.has(id)) return
    const poller = setInterval(async () => {
      if (!activeSessionIds.value.has(id)) return
      const summary = dashboard.value.sessions.find((s) => s.id === id)
      const knownDetail = sessionDetails.value[id]
      const aggressive = !!(summary?.loaded
        || summary?.lastTurnStatus === 'inProgress'
        || summary?.status === 'active'
        || summary?.lifecycleStage === 'history_only'
        || knownDetail?.summary?.lastTurnStatus === 'inProgress')
      if (!aggressive && !knownDetail) return
      await loadSession(id)
      if (aggressive) {
        await refreshDashboard()
      }
    }, 600)
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
    dashboard, sessionDetails, selectedAgentId, loading, error,
    sseConnected, sseStatus, lastEvent, activeSessionIds,
    filteredSessions, filteredApprovals, sessionGroups, isAgentOnline,
    refreshDashboard, loadSession, resumeSession, detachSession, endSession, archiveSession,
    renameSession, forkSession, compactSession, rollbackSession,
    startTurn, steerTurn, interruptTurn, resolveApproval, startSession,
    connectSSE, disconnectSSE, registerActiveSession, unregisterActiveSession,
  }
})
