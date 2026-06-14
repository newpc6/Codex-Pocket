<template>
  <div class="turn-card">
    <div class="turn-header">
      <div>
        <el-tag :type="turnStatusType" size="small">{{ turnStatusLabel }}</el-tag>
        <span style="font-size: 12px; color: var(--cf-text-secondary); margin-left: 8px">
          {{ formatTimestamp(turn.startedAt) }}
        </span>
      </div>
      <span v-if="turn.durationMs > 0" style="font-size: 12px; color: var(--cf-text-secondary)">
        {{ (turn.durationMs / 1000).toFixed(1) }}s
      </span>
    </div>

    <div v-if="turn.planExplanation" style="margin-bottom: 12px">
      <div style="font-size: 13px; font-weight: 600; margin-bottom: 4px">Plan</div>
      <div style="font-size: 13px; color: var(--cf-text-secondary)">{{ turn.planExplanation }}</div>
      <div v-if="turn.plan?.length" style="margin-top: 8px">
        <div v-for="(step, i) in turn.plan" :key="i" style="display: flex; align-items: center; gap: 6px; margin-bottom: 4px">
          <el-icon :color="step.status === 'completed' ? '#10b981' : '#f59e0b'" :size="14">
            <component :is="step.status === 'completed' ? 'CircleCheckFilled' : 'Clock'" />
          </el-icon>
          <span style="font-size: 13px">{{ step.step }}</span>
        </div>
      </div>
    </div>

    <div v-if="turn.diff" style="margin-bottom: 12px">
      <el-collapse>
        <el-collapse-item title="Diff">
          <pre style="font-size: 12px; background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 8px; overflow-x: auto; max-height: 400px">{{ turn.diff }}</pre>
        </el-collapse-item>
      </el-collapse>
    </div>

    <div v-if="turn.error" style="margin-bottom: 12px">
      <el-alert :title="turn.error" type="error" :closable="false" />
    </div>

    <div v-for="item in turn.items" :key="item.id" class="turn-item">
      <div class="item-title">
        <el-icon :size="14" style="vertical-align: middle; margin-right: 4px">
          <component :is="itemIcon(item.type)" />
        </el-icon>
        {{ item.title }}
        <el-tag v-if="item.status" size="small" style="margin-left: 6px">{{ item.status }}</el-tag>
      </div>
      <div v-if="item.body" class="item-body">{{ item.body }}</div>
      <div v-if="item.auxiliary" style="margin-top: 6px">
        <el-collapse>
          <el-collapse-item title="输出">
            <pre style="font-size: 12px; white-space: pre-wrap; max-height: 300px; overflow: auto">{{ item.auxiliary }}</pre>
          </el-collapse-item>
        </el-collapse>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { formatTimestamp } from '../utils/helpers'
import type { Turn } from '../stores/app'

const props = defineProps<{ turn: Turn }>()

const turnStatusType = computed(() => {
  switch (props.turn.status) {
    case 'completed': return 'success'
    case 'inProgress': return 'warning'
    case 'error': return 'danger'
    default: return 'info'
  }
})

const turnStatusLabel = computed(() => {
  switch (props.turn.status) {
    case 'completed': return '已完成'
    case 'inProgress': return '执行中'
    case 'error': return '出错'
    default: return props.turn.status
  }
})

function itemIcon(type: string): string {
  switch (type) {
    case 'userMessage': return 'User'
    case 'agentMessage': return 'Monitor'
    case 'commandExecution': return 'Terminal'
    case 'fileChange': return 'Document'
    case 'plan': return 'List'
    case 'reasoning': return 'Cpu'
    case 'mcpToolCall': return 'Link'
    case 'dynamicToolCall': return 'SetUp'
    case 'collabAgentToolCall': return 'Share'
    default: return 'InfoFilled'
  }
}
</script>
