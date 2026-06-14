<template>
  <div class="turn-card" :class="{ 'is-active': turn.status === 'inProgress' }">
    <!-- 折叠头部 -->
    <div class="turn-collapse-header" @click="expanded = !expanded">
      <div class="turn-collapse-left">
        <el-icon class="collapse-arrow" :class="{ 'is-expanded': expanded }">
          <ArrowRight />
        </el-icon>
        <el-tag :type="turnStatusType" size="small" effect="dark" round>{{ turnStatusLabel }}</el-tag>
        <span class="turn-index">Turn #{{ index + 1 }}</span>
        <span v-if="firstUserMsg" class="turn-preview-text">{{ truncateText(firstUserMsg, 60) }}</span>
      </div>
      <div class="turn-collapse-right">
        <span v-if="turn.durationMs > 0" class="turn-duration">{{ (turn.durationMs / 1000).toFixed(1) }}s</span>
        <span class="turn-time">{{ formatTimestamp(turn.startedAt) }}</span>
        <el-tag size="small" type="info" round>{{ itemCount }}</el-tag>
      </div>
    </div>

    <!-- 展开内容 -->
    <transition name="turn-expand">
      <div v-if="expanded" class="turn-body">
        <!-- Plan -->
        <div v-if="turn.planExplanation" class="turn-section turn-section-plan">
          <div class="section-label">
            <el-icon><List /></el-icon>
            <span>计划</span>
          </div>
          <div class="section-content">
            <div class="plan-explanation">{{ turn.planExplanation }}</div>
            <div v-if="turn.plan?.length" class="plan-steps">
              <div v-for="(step, i) in turn.plan" :key="i" class="plan-step" :class="{ 'is-done': step.status === 'completed' }">
                <el-icon :size="14">
                  <CircleCheckFilled v-if="step.status === 'completed'" />
                  <Clock v-else />
                </el-icon>
                <span>{{ step.step }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 消息列表 -->
        <div v-for="item in turn.items" :key="item.id" class="turn-item" :class="itemClass(item.type)">
          <div class="item-header">
            <div class="item-type-badge" :class="itemClass(item.type)">
              <el-icon :size="13">
                <component :is="itemIcon(item.type)" />
              </el-icon>
              <span>{{ itemLabel(item.type) }}</span>
            </div>
            <el-tag v-if="item.status" size="small" :type="itemStatusType(item.status)" round>{{ item.status }}</el-tag>
          </div>
          <div v-if="item.title" class="item-title">{{ item.title }}</div>
          <div v-if="item.body" class="item-body" :class="{ 'is-code': isCodeType(item.type) }">
            <pre v-if="isCodeType(item.type)">{{ item.body }}</pre>
            <div v-else class="markdown-body">
              <VueMarkdown :source="item.body" :options="markdownOptions" />
              <span v-if="isStreamingItem(item)" class="typing-cursor">|</span>
            </div>
          </div>
          <div v-if="item.auxiliary" class="item-auxiliary">
            <el-collapse>
              <el-collapse-item title="详细输出">
                <pre class="aux-pre">{{ item.auxiliary }}</pre>
              </el-collapse-item>
            </el-collapse>
          </div>
        </div>

        <!-- Diff -->
        <div v-if="turn.diff" class="turn-section turn-section-diff">
          <div class="section-label">
            <el-icon><Document /></el-icon>
            <span>文件变更</span>
          </div>
          <div class="section-content">
            <pre class="diff-block">{{ turn.diff }}</pre>
          </div>
        </div>

        <!-- Error -->
        <div v-if="turn.error" class="turn-section turn-section-error">
          <el-alert :title="turn.error" type="error" :closable="false" show-icon />
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { formatTimestamp, truncateText } from '../utils/helpers'
import type { Turn } from '../stores/app'
import VueMarkdown from 'vue-markdown-render'
import {
  ArrowRight, List, Document, CircleCheckFilled, Clock,
  User, Monitor, VideoPlay, Edit, Link, SetUp, Share, InfoFilled, Cpu,
} from '@element-plus/icons-vue'

const props = defineProps<{ turn: Turn; index: number }>()
const expanded = ref(props.turn.status === 'inProgress')
const markdownOptions = {
  html: false,
  breaks: true,
  linkify: true,
  typographer: true,
}

watch(() => props.turn.status, (status) => {
  if (status === 'inProgress') expanded.value = true
})

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

const itemCount = computed(() => props.turn.items?.length || 0)

const firstUserMsg = computed(() => {
  const userItem = props.turn.items?.find((i) => i.type === 'userMessage')
  return userItem?.body || userItem?.title || ''
})

function itemIcon(type: string) {
  switch (type) {
    case 'userMessage': return User
    case 'agentMessage': return Monitor
    case 'commandExecution': return VideoPlay
    case 'fileChange': return Edit
    case 'plan': return List
    case 'reasoning': return Cpu
    case 'mcpToolCall': return Link
    case 'dynamicToolCall': return SetUp
    case 'collabAgentToolCall': return Share
    default: return InfoFilled
  }
}

function itemClass(type: string): string {
  switch (type) {
    case 'userMessage': return 'type-user'
    case 'agentMessage': return 'type-agent'
    case 'commandExecution': return 'type-command'
    case 'fileChange': return 'type-file'
    case 'reasoning': return 'type-reasoning'
    case 'plan': return 'type-plan'
    case 'mcpToolCall': return 'type-mcp'
    case 'dynamicToolCall': return 'type-tool'
    case 'collabAgentToolCall': return 'type-collab'
    default: return 'type-other'
  }
}

function itemLabel(type: string): string {
  switch (type) {
    case 'userMessage': return '用户消息'
    case 'agentMessage': return '助手回复'
    case 'commandExecution': return '命令执行'
    case 'fileChange': return '文件变更'
    case 'reasoning': return '推理过程'
    case 'plan': return '计划'
    case 'mcpToolCall': return 'MCP 工具'
    case 'dynamicToolCall': return '动态工具'
    case 'collabAgentToolCall': return '协作代理'
    default: return type
  }
}

function itemStatusType(status: string) {
  switch (status) {
    case 'completed': return 'success'
    case 'running': return 'warning'
    case 'failed': return 'danger'
    default: return 'info'
  }
}

function isCodeType(type: string): boolean {
  return ['commandExecution', 'fileChange', 'mcpToolCall', 'dynamicToolCall'].includes(type)
}

function isStreamingItem(item: any): boolean {
  // Show typing cursor for agent messages in an in-progress turn
  return props.turn.status === 'inProgress' && item.type === 'agentMessage'
}
</script>

<style scoped>
.turn-card {
  background: var(--cf-card);
  border-radius: var(--cf-radius);
  border: 1px solid var(--cf-border);
  margin-bottom: 10px;
  box-shadow: var(--cf-shadow-sm);
  overflow: hidden;
  transition: border-color 0.2s ease;
}

.turn-card:hover {
  border-color: #cddfff;
}

.turn-card.is-active {
  border-color: var(--cf-warning);
  box-shadow: 0 0 0 1px var(--cf-warning), var(--cf-shadow-sm);
}

/* ---- 折叠头部 ---- */
.turn-collapse-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  cursor: pointer;
  user-select: none;
  gap: 12px;
  transition: background 0.15s ease;
}

.turn-collapse-header:hover {
  background: #f8fafd;
}

.turn-collapse-left {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
}

.turn-collapse-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.collapse-arrow {
  transition: transform 0.25s ease;
  color: var(--cf-text-secondary);
  font-size: 14px;
  flex-shrink: 0;
}

.collapse-arrow.is-expanded {
  transform: rotate(90deg);
}

.turn-index {
  font-size: 13px;
  font-weight: 600;
  color: var(--cf-text-heavy);
  flex-shrink: 0;
}

.turn-preview-text {
  font-size: 13px;
  color: var(--cf-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.turn-duration {
  font-size: 12px;
  color: var(--cf-text-lighter);
  font-variant-numeric: tabular-nums;
}

.turn-time {
  font-size: 12px;
  color: var(--cf-text-lighter);
}

/* ---- 展开内容 ---- */
.turn-body {
  padding: 0 16px 16px;
  border-top: 1px solid var(--cf-border-light);
}

.turn-section {
  margin-bottom: 14px;
}

.turn-section:last-child {
  margin-bottom: 0;
}

.section-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--cf-text-heavy);
  margin-bottom: 8px;
}

.section-content {
  padding-left: 4px;
}

/* ---- Plan ---- */
.turn-section-plan .plan-explanation {
  font-size: 13px;
  color: var(--cf-text-secondary);
  margin-bottom: 8px;
  line-height: 1.6;
}

.plan-steps {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.plan-step {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--cf-text);
}

.plan-step.is-done {
  color: var(--cf-success);
}

.plan-step .el-icon {
  flex-shrink: 0;
}

/* ---- 消息项 ---- */
.turn-item {
  padding: 12px 14px;
  border-radius: var(--cf-radius-sm);
  margin-bottom: 8px;
  border: 1px solid transparent;
  transition: border-color 0.15s ease;
}

.turn-item:last-child {
  margin-bottom: 0;
}

.item-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.item-type-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 10px;
}

/* 用户消息 */
.turn-item.type-user {
  background: #f0f5ff;
  border-color: #d6e4ff;
}

.type-user .item-type-badge {
  background: #d6e4ff;
  color: #1a56db;
}

/* 助手回复 */
.turn-item.type-agent {
  background: #f0fdf4;
  border-color: #bbf7d0;
}

.type-agent .item-type-badge {
  background: #bbf7d0;
  color: #15803d;
}

/* 命令执行 */
.turn-item.type-command {
  background: #1e1e2e;
  border-color: #333;
}

.type-command .item-type-badge {
  background: #333;
  color: #4ade80;
}

.type-command .item-title {
  color: #a5b4fc;
}

.type-command .item-body {
  color: #d4d4d4;
}

.type-command .item-body pre {
  color: #d4d4d4;
}

/* 文件变更 */
.turn-item.type-file {
  background: #fefce8;
  border-color: #fde68a;
}

.type-file .item-type-badge {
  background: #fde68a;
  color: #a16207;
}

/* 推理过程 */
.turn-item.type-reasoning {
  background: #faf5ff;
  border-color: #e9d5ff;
}

.type-reasoning .item-type-badge {
  background: #e9d5ff;
  color: #7c3aed;
}

/* 计划 */
.turn-item.type-plan {
  background: #f0f9ff;
  border-color: #bae6fd;
}

.type-plan .item-type-badge {
  background: #bae6fd;
  color: #0369a1;
}

/* MCP 工具 */
.turn-item.type-mcp {
  background: #fff7ed;
  border-color: #fed7aa;
}

.type-mcp .item-type-badge {
  background: #fed7aa;
  color: #c2410c;
}

/* 动态工具 */
.turn-item.type-tool {
  background: #fdf2f8;
  border-color: #fbcfe8;
}

.type-tool .item-type-badge {
  background: #fbcfe8;
  color: #be185d;
}

/* 协作代理 */
.turn-item.type-collab {
  background: #ecfdf5;
  border-color: #a7f3d0;
}

.type-collab .item-type-badge {
  background: #a7f3d0;
  color: #047857;
}

/* 其他 */
.turn-item.type-other {
  background: #f9fafb;
  border-color: #e5e7eb;
}

.type-other .item-type-badge {
  background: #e5e7eb;
  color: #6b7280;
}

.item-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--cf-text-heavy);
  margin-bottom: 6px;
  word-break: break-all;
}

.item-body {
  font-size: 13px;
  color: var(--cf-text-secondary);
  word-break: break-all;
  line-height: 1.6;
  max-height: 400px;
  overflow: auto;
}

.markdown-body {
  white-space: normal;
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

.markdown-body :deep(li + li) {
  margin-top: 4px;
}

.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3),
.markdown-body :deep(h4),
.markdown-body :deep(h5),
.markdown-body :deep(h6) {
  margin: 0 0 8px;
  color: var(--cf-text-heavy);
  line-height: 1.4;
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

.markdown-body :deep(pre code) {
  padding: 0;
  background: transparent;
}

.markdown-body :deep(blockquote) {
  padding-left: 12px;
  border-left: 3px solid rgba(51, 136, 255, 0.35);
  color: var(--cf-text-secondary);
}

.markdown-body :deep(a) {
  color: var(--cf-primary);
  text-decoration: none;
}

.markdown-body :deep(a:hover) {
  text-decoration: underline;
}

.item-body.is-code pre {
  margin: 0;
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
  font-size: 12px;
  line-height: 1.5;
}

.typing-cursor {
  display: inline;
  color: var(--cf-primary);
  font-weight: 400;
  animation: blink-cursor 0.8s step-end infinite;
}

@keyframes blink-cursor {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

.item-auxiliary {
  margin-top: 8px;
}

.aux-pre {
  font-size: 12px;
  white-space: pre-wrap;
  max-height: 300px;
  overflow: auto;
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
}

/* ---- Diff ---- */
.turn-section-diff .diff-block {
  font-size: 12px;
  background: #1e1e2e;
  color: #d4d4d4;
  padding: 12px;
  border-radius: var(--cf-radius-sm);
  overflow-x: auto;
  max-height: 400px;
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace;
  line-height: 1.5;
  margin: 0;
}

/* ---- Error ---- */
.turn-section-error {
  margin-top: 8px;
}

/* ---- 折叠动画 ---- */
.turn-expand-enter-active,
.turn-expand-leave-active {
  transition: all 0.25s ease;
  overflow: hidden;
}

.turn-expand-enter-from,
.turn-expand-leave-to {
  opacity: 0;
  max-height: 0;
  padding-top: 0;
  padding-bottom: 0;
}
</style>
