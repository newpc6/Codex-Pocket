<template>
  <div class="approvals-page">
    <div class="page-title">
      <div class="page-title-heading">审批中心</div>
      <div class="page-title-extra">
        <el-input v-model="searchQuery" placeholder="搜索审批..." prefix-icon="Search" clearable class="search-box" />
        <el-select v-model="filterKind" placeholder="类型" clearable style="width: 130px">
          <el-option label="命令审批" value="command" />
          <el-option label="文件变更" value="fileChange" />
          <el-option label="权限请求" value="permissions" />
          <el-option label="用户输入" value="userInput" />
        </el-select>
        <el-button :icon="Refresh" :loading="app.loading" @click="app.refreshDashboard()" circle />
      </div>
    </div>

    <el-alert v-if="filteredApprovals.length === 0 && !app.loading" type="info" :closable="false"
      description="当前没有待处理的审批。" />

    <div v-for="approval in filteredApprovals" :key="approval.id" class="approval-card">
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
          <div v-if="approval.kind === 'command' && approval.params?.command"
            style="font-size: 12px; font-family: monospace; color: var(--cf-text-secondary); background: #f5f5f5; padding: 6px 10px; border-radius: 6px; margin-top: 6px">
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
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useAppStore, type ApprovalRequest } from '../stores/app'
import { Refresh } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatTimestamp, sessionDisplayName } from '../utils/helpers'

const app = useAppStore()
const searchQuery = ref('')
const filterKind = ref('')

const filteredApprovals = computed(() => {
  let list = app.filteredApprovals
  if (filterKind.value) {
    list = list.filter((a) => a.kind === filterKind.value)
  }
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase()
    list = list.filter((a) =>
      (a.reason || '').toLowerCase().includes(q) ||
      (a.summary || '').toLowerCase().includes(q) ||
      sessionName(a.threadId).toLowerCase().includes(q)
    )
  }
  return list
})

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

onMounted(() => {
  app.refreshDashboard()
})
</script>

<style scoped>
.approvals-page {
  max-width: 1200px;
  margin: 0 auto;
}
</style>
