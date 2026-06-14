<template>
  <div class="page-container">
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px">
      <div style="font-size: 18px; font-weight: 600">审批中心</div>
      <el-button :icon="Refresh" :loading="app.loading" @click="app.refreshDashboard()">刷新</el-button>
    </div>

    <el-alert v-if="app.filteredApprovals.length === 0 && !app.loading" type="info" :closable="false"
      description="当前没有待处理的审批。" />

    <div v-for="approval in app.filteredApprovals" :key="approval.id" class="approval-card">
      <div style="display: flex; justify-content: space-between; align-items: flex-start">
        <div style="flex: 1">
          <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 6px">
            <el-tag size="small" :type="kindTagType(approval.kind)">{{ kindLabel(approval.kind) }}</el-tag>
            <span style="font-size: 12px; color: var(--cf-text-secondary)">
              {{ formatTimestamp(Math.floor(new Date(approval.createdAt).getTime() / 1000)) }}
            </span>
          </div>
          <div style="font-size: 14px; font-weight: 500; margin-bottom: 4px">
            {{ approval.reason || approval.summary }}
          </div>
          <div v-if="approval.kind === 'command' && approval.params?.command" style="font-size: 12px; font-family: monospace; color: var(--cf-text-secondary); background: #f5f5f5; padding: 6px 10px; border-radius: 6px; margin-top: 6px">
            {{ approval.params.command }}
          </div>
          <div v-if="approval.kind === 'fileChange' && approval.params?.changes" style="font-size: 12px; margin-top: 6px">
            <div v-for="(change, i) in approval.params.changes" :key="i" style="color: var(--cf-text-secondary)">
              {{ change.path || change.filePath || change.relativePath || JSON.stringify(change) }}
            </div>
          </div>
          <div style="font-size: 12px; color: var(--cf-text-secondary); margin-top: 6px">
            会话: {{ sessionName(approval.threadId) }}
          </div>
        </div>
        <div style="display: flex; gap: 8px; margin-left: 16px">
          <el-button size="small" type="success" @click="handleApproval(approval, true)">批准</el-button>
          <el-button size="small" type="danger" @click="handleApproval(approval, false)">拒绝</el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useAppStore, type ApprovalRequest } from '../stores/app'
import { Refresh } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatTimestamp, sessionDisplayName } from '../utils/helpers'

const app = useAppStore()

function kindTagType(kind: string): string {
  switch (kind) {
    case 'command': return 'warning'
    case 'fileChange': return ''
    case 'permissions': return 'danger'
    case 'userInput': return 'info'
    default: return 'info'
  }
}

function kindLabel(kind: string): string {
  switch (kind) {
    case 'command': return '命令审批'
    case 'fileChange': return '文件变更'
    case 'permissions': return '权限请求'
    case 'userInput': return '用户输入'
    default: return kind
  }
}

function sessionName(threadId: string): string {
  const s = app.dashboard.sessions.find((s) => s.id === threadId)
  return s ? sessionDisplayName(s) : threadId?.substring(0, 8)
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

onMounted(() => {
  app.refreshDashboard()
})
</script>
