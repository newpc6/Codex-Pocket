import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../utils/api'
import { sseService, type SSEEvent } from '../utils/sse'

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
  const lastEvent = ref<SSEEvent | null>(null)

  // Track which sessions are currently being viewed (for targeted refresh)
  const activeSessionIds = ref<Set<string>>(new Set())

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

  async function loadSession(id: string) {
    try {
      const res = await api.get<SessionDetail>(`/sessions/${id}`)
      sessionDetails.value[id] = res.data
    } catch (e: any) {
      error.value = e.response?.data?.error || e.message
    }
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

  async function archiveSession(id: string) {
    await api.post(`/sessions/${id}/archive`)
    delete sessionDetails.value[id]
    await refreshDashboard()
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
    sseService.connect()
    sseConnected.value = true

    // Wildcard handler: update lastEvent for any event
    sseService.on('*', (event: SSEEvent) => {
      lastEvent.value = event
    })

    // Codex notifications: turn progress, plan updates, diff updates, etc.
    sseService.on('codex.notification', async (event: SSEEvent) => {
      const method = event.payload?.method as string
      if (!method) return

      // Refresh session detail if it's a turn-related notification for an active session
      const threadId = event.payload?.params?.threadId as string
      if (threadId && activeSessionIds.value.has(threadId)) {
        await loadSession(threadId)
      }

      // Always refresh dashboard on significant events
      if ([
        'turn/started', 'turn/completed', 'turn/diff/updated', 'turn/plan/updated',
        'thread/started', 'thread/status/changed', 'thread/closed',
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
      if (threadId && activeSessionIds.value.has(threadId)) {
        await loadSession(threadId)
      }
      await refreshDashboard()
    })

    sseService.on('turn.steered', async (event: SSEEvent) => {
      const threadId = event.payload?.threadId as string
      if (threadId && activeSessionIds.value.has(threadId)) {
        await loadSession(threadId)
      }
    })

    sseService.on('turn.interrupted', async (event: SSEEvent) => {
      const threadId = event.payload?.threadId as string
      if (threadId && activeSessionIds.value.has(threadId)) {
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
    sseConnected.value = false
  }

  function registerActiveSession(id: string) {
    activeSessionIds.value.add(id)
  }

  function unregisterActiveSession(id: string) {
    activeSessionIds.value.delete(id)
  }

  return {
    dashboard, sessionDetails, selectedAgentId, loading, error,
    sseConnected, lastEvent, activeSessionIds,
    filteredSessions, filteredApprovals, sessionGroups, isAgentOnline,
    refreshDashboard, loadSession, resumeSession, endSession, archiveSession,
    startTurn, steerTurn, interruptTurn, resolveApproval, startSession,
    connectSSE, disconnectSSE, registerActiveSession, unregisterActiveSession,
  }
})
