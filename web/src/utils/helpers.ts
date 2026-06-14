export function formatTimestamp(ts: number | undefined): string {
  if (!ts || ts <= 0) return '未知'
  const date = new Date(ts * 1000)
  const now = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')

  const timeStr = `${pad(date.getHours())}:${pad(date.getMinutes())}`
  const dateStr = `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`

  const isToday =
    date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth() &&
    date.getDate() === now.getDate()

  if (isToday) return `今天 ${timeStr}`

  const yesterday = new Date(now)
  yesterday.setDate(yesterday.getDate() - 1)
  const isYesterday =
    date.getFullYear() === yesterday.getFullYear() &&
    date.getMonth() === yesterday.getMonth() &&
    date.getDate() === yesterday.getDate()

  if (isYesterday) return `昨天 ${timeStr}`
  if (date.getFullYear() === now.getFullYear()) return `${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${timeStr}`
  return `${dateStr} ${timeStr}`
}

export function statusTagType(status: string, ended: boolean): string {
  if (ended) return 'info'
  switch (status) {
    case 'active': return 'success'
    case 'inProgress': return 'warning'
    case 'notLoaded': return ''
    case 'idle': return 'info'
    default: return 'info'
  }
}

export function statusLabel(status: string, ended: boolean, hasWaiting: boolean): string {
  if (ended) return '已结束'
  if (hasWaiting) return '等待中'
  switch (status) {
    case 'active': return '运行中'
    case 'inProgress': return '执行中'
    case 'notLoaded': return '未接管'
    case 'idle': return '空闲'
    default: return status
  }
}

export function lifecycleLabel(stage: string): string {
  switch (stage) {
    case 'managed': return '已接管'
    case 'ended': return '已结束'
    case 'runtime_available': return '可接管'
    case 'history_only': return '历史'
    case 'discovered': return '已发现'
    default: return stage
  }
}

export function lifecycleTagType(stage: string): string {
  switch (stage) {
    case 'managed': return 'success'
    case 'ended': return 'info'
    case 'runtime_available': return 'warning'
    case 'history_only': return ''
    case 'discovered': return ''
    default: return 'info'
  }
}

export function truncateText(text: string | undefined | null, maxLen: number = 120): string {
  if (!text) return ''
  const cleaned = text.replace(/\s+/g, ' ').trim()
  if (cleaned.length <= maxLen) return cleaned
  return cleaned.substring(0, maxLen) + '…'
}

interface SessionLike {
  name?: string
  agentNickname?: string
  cwd?: string
  preview?: string
  id?: string
}

export function sessionDisplayName(session: SessionLike): string {
  if (session.name?.trim()) return session.name.trim()
  if (session.agentNickname?.trim()) return session.agentNickname.trim()
  const cwd = session.cwd?.trim() || ''
  const dirName = cwd.replace(/\\/g, '/').split('/').pop()
  if (dirName) return dirName
  const preview = session.preview?.replace(/\s+/g, ' ').trim()
  if (preview && preview.length <= 32) return preview
  if (preview) return preview.substring(0, 32) + '…'
  return `Session ${session.id?.substring(0, 8) || 'unknown'}`
}
