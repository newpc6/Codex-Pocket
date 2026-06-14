export interface SSEEvent {
  type: string
  timestamp: string
  payload: any
}

type EventHandler = (event: SSEEvent) => void

class SSEService {
  private source: EventSource | null = null
  private handlers: Map<string, Set<EventHandler>> = new Map()
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private reconnectAttempts = 0
  private maxReconnectDelay = 30000
  private disposed = false

  connect() {
    if (this.source) return
    this.disposed = false

    const token = localStorage.getItem('cf_token')
    // EventSource doesn't support custom headers, so we pass token as query param
    const url = token
      ? `/api/v1/events?token=${encodeURIComponent(token)}`
      : '/api/v1/events'

    this.source = new EventSource(url)

    this.source.onopen = () => {
      this.reconnectAttempts = 0
    }

    this.source.onerror = () => {
      this.source?.close()
      this.source = null
      this.scheduleReconnect()
    }

    // Listen for named events from the server
    this.source.addEventListener('codex.notification', (e: MessageEvent) => {
      this.emit('codex.notification', this.parseEvent(e))
    })

    this.source.addEventListener('approval.created', (e: MessageEvent) => {
      this.emit('approval.created', this.parseEvent(e))
    })

    this.source.addEventListener('approval.resolved', (e: MessageEvent) => {
      this.emit('approval.resolved', this.parseEvent(e))
    })

    this.source.addEventListener('session.created', (e: MessageEvent) => {
      this.emit('session.created', this.parseEvent(e))
    })

    this.source.addEventListener('session.resumed', (e: MessageEvent) => {
      this.emit('session.resumed', this.parseEvent(e))
    })

    this.source.addEventListener('session.ended', (e: MessageEvent) => {
      this.emit('session.ended', this.parseEvent(e))
    })

    this.source.addEventListener('session.archived', (e: MessageEvent) => {
      this.emit('session.archived', this.parseEvent(e))
    })

    this.source.addEventListener('turn.started', (e: MessageEvent) => {
      this.emit('turn.started', this.parseEvent(e))
    })

    this.source.addEventListener('turn.steered', (e: MessageEvent) => {
      this.emit('turn.steered', this.parseEvent(e))
    })

    this.source.addEventListener('turn.interrupted', (e: MessageEvent) => {
      this.emit('turn.interrupted', this.parseEvent(e))
    })

    this.source.addEventListener('sessions.refreshed', (e: MessageEvent) => {
      this.emit('sessions.refreshed', this.parseEvent(e))
    })
  }

  disconnect() {
    this.disposed = true
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    this.source?.close()
    this.source = null
  }

  on(eventType: string, handler: EventHandler) {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, new Set())
    }
    this.handlers.get(eventType)!.add(handler)
  }

  off(eventType: string, handler: EventHandler) {
    this.handlers.get(eventType)?.delete(handler)
  }

  private emit(type: string, event: SSEEvent) {
    // Emit to specific handlers
    this.handlers.get(type)?.forEach((h) => h(event))
    // Emit to wildcard handlers
    this.handlers.get('*')?.forEach((h) => h(event))
  }

  private parseEvent(e: MessageEvent): SSEEvent {
    try {
      return JSON.parse(e.data)
    } catch {
      return { type: 'unknown', timestamp: new Date().toISOString(), payload: e.data }
    }
  }

  private scheduleReconnect() {
    if (this.disposed) return
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), this.maxReconnectDelay)
    this.reconnectAttempts++
    this.reconnectTimer = setTimeout(() => {
      this.connect()
    }, delay)
  }
}

export const sseService = new SSEService()
