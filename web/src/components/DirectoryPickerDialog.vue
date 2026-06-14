<template>
  <el-dialog
    :model-value="modelValue"
    title="选择工作目录"
    width="720px"
    :close-on-click-modal="false"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="picker-toolbar">
      <el-button :icon="House" @click="openHome" text>Home</el-button>
      <el-button :icon="Top" @click="openParent" text :disabled="!browser.parentPath">上级</el-button>
      <el-button :icon="Refresh" @click="refreshCurrent" text :loading="loading">刷新</el-button>
      <el-input
        v-model="pathInput"
        placeholder="输入绝对路径"
        class="path-input"
        @keyup.enter="openPath(pathInput)"
      >
        <template #append>
          <el-button @click="openPath(pathInput)">打开</el-button>
        </template>
      </el-input>
    </div>

    <div class="picker-current">
      <span class="picker-label">当前目录</span>
      <code class="picker-path">{{ browser.currentPath || '-' }}</code>
    </div>

    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" style="margin-bottom: 12px" />

    <div class="picker-grid">
      <div class="picker-roots">
        <div class="pane-title">快捷位置</div>
        <div
          v-for="entry in quickEntries"
          :key="entry.path"
          class="location-item"
          :class="{ active: entry.path === browser.currentPath }"
          @click="openPath(entry.path)"
        >
          <el-icon><Folder /></el-icon>
          <span>{{ entry.name }}</span>
        </div>
      </div>

      <div class="picker-list">
        <div class="pane-title">目录</div>
        <div v-if="browser.entries.length === 0 && !loading" class="empty-state">当前目录下没有可浏览的子目录</div>
        <div
          v-for="entry in browser.entries"
          :key="entry.path"
          class="directory-item"
          :class="{ selected: selectedPath === entry.path, disabled: !entry.isReadable }"
          @click="selectPath(entry.path)"
          @dblclick="entry.isReadable && openPath(entry.path)"
        >
          <div class="directory-main">
            <el-icon><Folder /></el-icon>
            <span>{{ entry.name }}</span>
          </div>
          <el-tag v-if="!entry.isReadable" type="info" size="small">不可读</el-tag>
        </div>
      </div>
    </div>

    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :disabled="!selectedPath" @click="confirmSelection">选择当前目录</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Folder, House, Refresh, Top } from '@element-plus/icons-vue'
import api from '../utils/api'

interface DirectoryEntry {
  name: string
  path: string
  isDir: boolean
  isParent: boolean
  isHome: boolean
  isRoot: boolean
  isReadable: boolean
}

interface DirectoryBrowseResult {
  currentPath: string
  parentPath: string
  homePath: string
  roots: DirectoryEntry[]
  entries: DirectoryEntry[]
}

const props = defineProps<{
  modelValue: boolean
  initialPath?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [boolean]
  select: [string]
}>()

const loading = ref(false)
const error = ref('')
const selectedPath = ref('')
const pathInput = ref('')
const browser = ref<DirectoryBrowseResult>({
  currentPath: '',
  parentPath: '',
  homePath: '',
  roots: [],
  entries: [],
})

const quickEntries = computed(() => {
  const items: DirectoryEntry[] = []
  if (browser.value.homePath) {
    items.push({
      name: 'Home',
      path: browser.value.homePath,
      isDir: true,
      isParent: false,
      isHome: true,
      isRoot: false,
      isReadable: true,
    })
  }
  return [...items, ...browser.value.roots]
})

watch(() => props.modelValue, async (open) => {
  if (!open) return
  const initial = props.initialPath?.trim() || browser.value.currentPath || ''
  await openPath(initial)
})

async function openPath(path: string) {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<DirectoryBrowseResult>('/directories', {
      params: path ? { path } : {},
    })
    browser.value = res.data
    selectedPath.value = res.data.currentPath
    pathInput.value = res.data.currentPath
  } catch (e: any) {
    error.value = e.response?.data?.error || '目录加载失败'
  } finally {
    loading.value = false
  }
}

function selectPath(path: string) {
  selectedPath.value = path
}

function openParent() {
  if (!browser.value.parentPath) return
  void openPath(browser.value.parentPath)
}

function openHome() {
  if (!browser.value.homePath) {
    ElMessage.warning('未找到 Home 目录')
    return
  }
  void openPath(browser.value.homePath)
}

function refreshCurrent() {
  void openPath(selectedPath.value || browser.value.currentPath)
}

function confirmSelection() {
  if (!selectedPath.value) return
  emit('select', selectedPath.value)
  emit('update:modelValue', false)
}
</script>

<style scoped>
.picker-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.path-input {
  flex: 1;
  min-width: 260px;
}

.picker-current {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  background: #f8fafc;
}

.picker-label {
  font-size: 12px;
  color: var(--cf-text-secondary);
}

.picker-path {
  font-size: 12px;
  color: var(--cf-text-heavy);
  word-break: break-all;
}

.picker-grid {
  display: grid;
  grid-template-columns: 180px minmax(0, 1fr);
  gap: 12px;
  min-height: 360px;
}

.picker-roots,
.picker-list {
  border: 1px solid var(--cf-border-light);
  border-radius: 8px;
  background: #fff;
  overflow: auto;
}

.pane-title {
  position: sticky;
  top: 0;
  z-index: 1;
  padding: 10px 12px;
  background: #f8fafc;
  border-bottom: 1px solid var(--cf-border-light);
  font-size: 12px;
  font-weight: 700;
  color: var(--cf-text-secondary);
}

.location-item,
.directory-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 10px 12px;
  cursor: pointer;
  border-bottom: 1px solid #f3f4f6;
}

.location-item:last-child,
.directory-item:last-child {
  border-bottom: none;
}

.location-item:hover,
.directory-item:hover {
  background: #f8fbff;
}

.location-item.active,
.directory-item.selected {
  background: #eef6ff;
  color: var(--cf-primary);
}

.directory-item.disabled {
  opacity: 0.6;
}

.directory-main {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.directory-main span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-state {
  padding: 24px 12px;
  font-size: 13px;
  color: var(--cf-text-secondary);
}

@media (max-width: 768px) {
  .picker-grid {
    grid-template-columns: 1fr;
    min-height: 0;
  }
}
</style>
