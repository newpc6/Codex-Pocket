<template>
  <el-dialog
    :model-value="modelValue"
    title="选择工作目录"
    :width="dialogWidth"
    class="directory-picker-dialog"
    :close-on-click-modal="false"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="picker-shell">
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
            @click="handleDirectoryClick(entry)"
            @dblclick="entry.isReadable && openPath(entry.path)"
          >
            <div class="directory-main">
              <el-icon><Folder /></el-icon>
              <span>{{ entry.name }}</span>
            </div>
            <el-button v-if="entry.isReadable && !isMobile" link type="primary" @click.stop="openPath(entry.path)">进入</el-button>
            <el-tag v-else-if="!entry.isReadable" type="info" size="small">不可读</el-tag>
          </div>
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
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
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
const isMobile = ref(window.innerWidth <= 768)
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

const dialogWidth = computed(() => isMobile.value ? 'calc(100vw - 24px)' : '720px')

function onResize() {
  isMobile.value = window.innerWidth <= 768
}

onMounted(() => window.addEventListener('resize', onResize))
onUnmounted(() => window.removeEventListener('resize', onResize))

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

function handleDirectoryClick(entry: DirectoryEntry) {
  if (!entry.isReadable) {
    selectPath(entry.path)
    return
  }
  if (isMobile.value) {
    void openPath(entry.path)
    return
  }
  selectPath(entry.path)
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
.directory-picker-dialog :deep(.el-dialog) {
  max-height: min(78vh, 680px);
  display: flex;
  flex-direction: column;
}

.directory-picker-dialog :deep(.el-dialog__body) {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.directory-picker-dialog :deep(.el-dialog__footer) {
  flex-shrink: 0;
}

.picker-shell {
  display: flex;
  flex-direction: column;
  min-height: 0;
  max-height: min(64vh, 560px);
}

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
  min-height: 0;
  flex: 1;
  overflow: hidden;
}

.picker-roots,
.picker-list {
  border: 1px solid var(--cf-border-light);
  border-radius: 8px;
  background: #fff;
  overflow-y: auto;
  min-height: 0;
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
  .directory-picker-dialog :deep(.el-dialog) {
    width: calc(100vw - 24px) !important;
    max-width: calc(100vw - 24px);
    max-height: calc(100dvh - 32px);
    margin: 16px auto !important;
  }

  .directory-picker-dialog :deep(.el-dialog__header) {
    padding: 14px 16px 8px;
  }

  .directory-picker-dialog :deep(.el-dialog__body) {
    padding: 10px 16px;
  }

  .directory-picker-dialog :deep(.el-dialog__footer) {
    padding: 10px 16px 14px;
  }

  .directory-picker-dialog :deep(.dialog-footer),
  .directory-picker-dialog :deep(.el-dialog__footer) {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }

  .picker-shell {
    max-height: calc(100dvh - 150px);
  }

  .picker-toolbar {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 6px;
  }

  .picker-toolbar :deep(.el-button) {
    justify-content: center;
    min-width: 0;
    padding: 8px 6px;
  }

  .path-input {
    grid-column: 1 / -1;
    min-width: 0;
    width: 100%;
  }

  .picker-current {
    align-items: flex-start;
    flex-direction: column;
    gap: 4px;
    padding: 8px 10px;
  }

  .picker-path {
    max-width: 100%;
  }

  .picker-grid {
    grid-template-columns: 1fr;
    gap: 10px;
    overflow: auto;
  }

  .picker-roots,
  .picker-list {
    max-height: none;
    overflow: visible;
  }

  .location-item,
  .directory-item {
    min-height: 42px;
    padding: 11px 10px;
  }

  .directory-main span {
    white-space: normal;
    word-break: break-all;
  }
}
</style>
