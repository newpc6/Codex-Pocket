<template>
  <div class="main-layout" :class="{ 'is-mobile': isMobile }">
    <header class="header">
      <div class="brand">
        <div v-if="isMobile" class="mobile-menu-btn" @click="mobileMenuOpen = !mobileMenuOpen">
          <el-icon :size="20"><Operation /></el-icon>
        </div>
        <img class="brand-mark" src="/favicon.svg?v=3" alt="CodexPocket" />
        <div v-if="!isMobile" class="brand-text">
          <div class="brand-title">CodexPocket</div>
          <div class="brand-subtitle">Session Console</div>
        </div>
      </div>
      <div class="header-right">
        <div class="agent-status" :class="{ online: online, connected: app.sseConnected, reconnecting: app.sseStatus === 'reconnecting' || app.sseStatus === 'connecting' }">
          <span class="status-dot"></span>
          <span v-if="!isMobile">{{ statusText }}</span>
        </div>
        <div v-if="!isMobile" class="user-meta">
          <div class="user-name">{{ auth.username }}</div>
        </div>
        <el-avatar class="avatar" :size="isMobile ? 28 : 32" style="background-color: #3388ff">
          {{ (auth.username || 'U')[0].toUpperCase() }}
        </el-avatar>
        <el-dropdown @command="onHeaderCommand">
          <span class="el-dropdown-link">
            <el-icon><Operation /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="logout">
                <el-icon><SwitchButton /></el-icon> 退出登录
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </header>

    <!-- Mobile drawer overlay -->
    <div v-if="isMobile && mobileMenuOpen" class="mobile-overlay" @click="mobileMenuOpen = false"></div>

    <div class="menu-area">
      <el-aside
        :width="isMobile ? '220px' : (collapsed ? '64px' : '220px')"
        :class="{ 'is-collapsed': !isMobile && collapsed, 'is-mobile-drawer': isMobile, 'is-drawer-open': isMobile && mobileMenuOpen }"
      >
        <el-menu :default-active="currentRoute" :collapse="!isMobile && collapsed" @select="handleMenuClick">
          <el-menu-item index="/">
            <span class="menu-icon-box is-soft">
              <el-icon><Monitor /></el-icon>
            </span>
            <span class="text">会话</span>
          </el-menu-item>
          <el-menu-item index="/approvals">
            <span class="menu-icon-box is-soft">
              <el-icon><Checked /></el-icon>
            </span>
            <span class="text">审批</span>
            <el-badge v-if="approvalCount > 0" :value="approvalCount" :max="99" style="margin-left: 8px" />
          </el-menu-item>
          <el-menu-item index="/settings">
            <span class="menu-icon-box is-soft">
              <el-icon><Setting /></el-icon>
            </span>
            <span class="text">设置</span>
          </el-menu-item>
        </el-menu>
        <div v-if="!isMobile" class="collapse-trigger" @click="collapsed = !collapsed">
          <el-icon v-if="!collapsed"><Fold /></el-icon>
          <el-icon v-else><Expand /></el-icon>
        </div>
      </el-aside>
      <div class="menu-main">
        <div class="tabs-container">
          <el-tabs v-model="tabsStore.activeKey" type="card" closable @tab-remove="onTabRemove"
            @tab-click="onTabClick" class="tabs-bar" :class="{ 'hide-close-btn': tabsStore.tabs.length === 1 }">
            <el-tab-pane v-for="tab in tabsStore.tabs" :key="tab.key" :name="tab.key" :closable="tab.closable">
              <template #label>
                <el-dropdown :trigger="['contextmenu']" placement="top-start">
                  <span class="tab-label">{{ isMobile ? truncateTabTitle(tab.title) : tab.title }}</span>
                  <template #dropdown>
                    <el-dropdown-menu @command="onTabMenuClick($event, tab.key)">
                      <el-dropdown-item command="closeOther">
                        <el-icon><Close /></el-icon> 关闭其他
                      </el-dropdown-item>
                      <el-dropdown-item command="closeAll">
                        <el-icon><Close /></el-icon> 关闭全部
                      </el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </template>
            </el-tab-pane>
            <template #extra>
              <div class="tabs-extra" @click="onRefresh">
                <el-icon><Refresh /></el-icon>
                <span v-if="!isMobile">刷新</span>
              </div>
            </template>
          </el-tabs>
        </div>
        <main class="page-content" :class="{ 'is-session-detail-route': route.path.startsWith('/session/') }">
          <router-view v-slot="{ Component, route }">
            <transition name="fade-transform" mode="out-in">
              <component :is="Component" :key="route.path" />
            </transition>
          </router-view>
        </main>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore } from '../stores/app'
import { useAuthStore } from '../stores/auth'
import { useTabsStore } from '../stores/tabs'
import { sseService } from '../utils/sse'
import { ElNotification } from 'element-plus'

const route = useRoute()
const router = useRouter()
const app = useAppStore()
const auth = useAuthStore()
const tabsStore = useTabsStore()
const collapsed = ref(false)
const mobileMenuOpen = ref(false)
const isMobile = ref(window.innerWidth <= 768)

function onResize() { isMobile.value = window.innerWidth <= 768 }
window.addEventListener('resize', onResize)

function truncateTabTitle(title: string) {
  return title.length > 6 ? title.slice(0, 6) + '…' : title
}

const currentRoute = computed(() => {
  if (route.path.startsWith('/session/')) return '/'
  return route.path
})
const online = computed(() => app.isAgentOnline)
const approvalCount = computed(() => app.filteredApprovals.length)
const statusText = computed(() => {
  if (!online.value) return 'Agent 离线'
  if (app.sseStatus === 'connected') return '实时已连接'
  if (app.sseStatus === 'connecting') return '实时连接中'
  if (app.sseStatus === 'reconnecting') return '实时重连中'
  return 'Agent 在线'
})

watch(() => route.path, () => {
  tabsStore.addTab(route)
}, { immediate: true })

function handleMenuClick(index: string) {
  router.push(index)
  if (isMobile.value) mobileMenuOpen.value = false
}

function onHeaderCommand(cmd: string) {
  if (cmd === 'logout') {
    app.disconnectSSE()
    auth.logout()
    tabsStore.closeAllTabs()
    router.push('/login')
  }
}

function onTabRemove(name: string | number) {
  const key = String(name)
  tabsStore.removeTab(key)
  const current = tabsStore.currentTab
  if (current) router.push(current.path)
}

function onTabClick(pane: any) {
  const tab = tabsStore.tabs.find((t) => t.key === pane.paneName)
  if (tab) router.push(tab.path)
}

function onTabMenuClick(command: string, key: string) {
  if (command === 'closeOther') tabsStore.closeOtherTabs(key)
  else if (command === 'closeAll') tabsStore.closeAllTabs()
  const current = tabsStore.currentTab
  if (current) router.push(current.path)
}

function onRefresh() {
  app.refreshDashboard()
}

// SSE: connect on mount, listen for approval notifications
onMounted(async () => {
  await app.refreshDashboard()
  app.connectSSE()

  // Listen for new approval requests and show notification
  sseService.on('approval.created', (event) => {
    const payload = event.payload
    const kind = payload?.kind || '审批请求'
    const reason = payload?.reason || payload?.summary || '有新的审批请求需要处理'
    ElNotification({
      title: `新审批: ${kind}`,
      message: reason,
      type: 'warning',
      duration: 8000,
      position: 'top-right',
      onClick: () => {
        router.push('/approvals')
      },
    })
  })
})

onUnmounted(() => {
  app.disconnectSSE()
  window.removeEventListener('resize', onResize)
})
</script>

<style scoped>
.main-layout {
  height: 100vh;
  background: var(--cf-bg);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.header {
  background-image: linear-gradient(90deg, #2167d9 0%, #3388ff 100%);
  height: var(--cf-header-height);
  line-height: var(--cf-header-height);
  padding: 0 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.header .brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.brand-mark {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: block;
  box-shadow: 0 8px 18px rgba(37, 99, 235, 0.18);
}

.brand-text {
  color: #fff;
  line-height: 1.2;
}

.brand-title {
  font-size: 18px;
  font-weight: 700;
}

.brand-subtitle {
  font-size: 12px;
  opacity: 0.8;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
  color: #fff;
}

.agent-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  opacity: 0.9;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #ef4444;
}

.agent-status.online .status-dot {
  background: #10b981;
  box-shadow: 0 0 6px rgba(16, 185, 129, 0.5);
}

.agent-status.reconnecting .status-dot {
  background: #f59e0b;
  box-shadow: 0 0 6px rgba(245, 158, 11, 0.6);
}

.agent-status.connected .status-dot {
  animation: status-pulse 1.8s ease-in-out infinite;
}

@keyframes status-pulse {
  0%, 100% { transform: scale(1); opacity: 1; }
  50% { transform: scale(0.72); opacity: 0.65; }
}

.user-meta {
  text-align: right;
  line-height: 1.2;
}

.user-name {
  font-size: 14px;
  font-weight: 600;
}

.avatar {
  border: 2px solid rgba(255, 255, 255, 0.3);
}

.el-dropdown-link {
  cursor: pointer;
  color: #fff;
  display: flex;
  align-items: center;
}

.menu-area {
  flex: 1;
  display: flex;
  min-height: 0;
  overflow: hidden;
}

.el-aside {
  width: var(--cf-aside-width);
  border-right: 1px solid var(--cf-border-light);
  background: #fff;
  display: flex;
  flex-direction: column;
  transition: width var(--cf-transition);
  overflow: hidden;
  flex-shrink: 0;
  height: 100%;
}

.el-aside.is-collapsed {
  width: var(--cf-aside-collapsed-width);
}

.el-aside :deep(.el-menu) {
  border-right: none;
  flex: 1;
  overflow-y: auto;
}

.el-aside :deep(.el-menu-item) {
  color: var(--cf-text);
  font-size: 14px;
  height: 44px;
  line-height: 44px;
  margin: 4px 8px;
  border-radius: 6px;
}

.el-aside :deep(.el-menu-item.is-active) {
  background-color: var(--cf-primary-light);
  color: var(--cf-primary);
}

.menu-icon-box {
  width: 26px;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  margin-right: 8px;
  flex-shrink: 0;
}

.menu-icon-box.is-soft {
  background: #eef6ff;
  color: var(--cf-primary);
  box-shadow: 0 4px 10px rgba(37, 99, 235, 0.08);
}

.menu-icon-box .el-icon {
  margin-right: 0;
  font-size: 16px;
}

.collapse-trigger {
  background-color: #fff;
  color: var(--cf-text);
  text-align: center;
  height: 40px;
  line-height: 40px;
  cursor: pointer;
  font-size: 18px;
  border-top: 1px solid var(--cf-border-light);
  transition: all var(--cf-transition);
}

.collapse-trigger:hover {
  color: var(--cf-primary);
  background: var(--cf-primary-light);
}

.menu-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.tabs-container {
  padding: 0 16px;
  height: var(--cf-tabs-height);
  line-height: var(--cf-tabs-height);
  background: linear-gradient(0deg, #ffffff 11%, #e9ecf2 100%);
  box-shadow: 0 2px 8px rgba(0, 114, 255, 0.06);
  flex-shrink: 0;
}

.tabs-bar :deep(.el-tabs__header) {
  margin: 0;
}

.tabs-bar :deep(.el-tabs__item) {
  height: 34px;
  line-height: 34px;
  color: #6b7a99;
  font-size: 14px;
  border: 1px solid #e9ecf2;
  border-bottom: none;
  background: #e9ecf2;
  margin-right: 2px;
  border-radius: 8px 8px 0 0;
  transition: all var(--cf-transition);
  padding: 0 12px !important;
}

.tabs-bar :deep(.el-tabs__item.is-active) {
  background-image: linear-gradient(180deg, #ffffff 0%, #f1f6fa 68%);
  box-shadow: -2px 1px 4px 0 rgba(16, 0, 0, 0.1), 2px 1px 4px 0 rgba(0, 0, 0, 0.08);
  position: relative;
  z-index: 1;
  color: var(--cf-primary);
  font-weight: 500;
}

.tabs-bar :deep(.el-tabs__content) {
  display: none;
}

.tabs-bar.hide-close-btn :deep(.el-tabs__item .el-icon.is-icon-close) {
  display: none;
}

.tabs-extra {
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  color: #6b7a99;
  font-size: 13px;
}

.tabs-extra:hover {
  color: var(--cf-primary);
}

.page-content {
  flex: 1;
  padding: 20px;
  background-color: var(--cf-bg);
  overflow: auto;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.page-content.is-session-detail-route {
  overflow: hidden;
}

/* ---- Mobile ---- */
.mobile-menu-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: #fff;
  border-radius: 6px;
  transition: background 0.15s ease;
}

.mobile-menu-btn:hover {
  background: rgba(255, 255, 255, 0.15);
}

.mobile-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  z-index: 99;
}

.main-layout.is-mobile .header {
  padding: 0 12px;
  height: 48px;
  line-height: 48px;
}

.main-layout.is-mobile .brand-mark {
  width: 30px;
  height: 30px;
}

.main-layout.is-mobile .header-right {
  gap: 10px;
}

.el-aside.is-mobile-drawer {
  position: fixed;
  top: 48px;
  left: 0;
  bottom: 0;
  z-index: 100;
  transform: translateX(-100%);
  transition: transform 0.25s ease;
  box-shadow: none;
}

.el-aside.is-mobile-drawer.is-drawer-open {
  transform: translateX(0);
  box-shadow: 4px 0 16px rgba(0, 0, 0, 0.1);
}

.main-layout.is-mobile .tabs-container {
  padding: 0 8px;
  height: 40px;
  line-height: 40px;
}

.main-layout.is-mobile .page-content {
  padding: 10px;
}

.main-layout.is-mobile .page-content.is-session-detail-route {
  overflow: auto;
}

.tab-label {
  display: inline-block;
  max-width: 80px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
