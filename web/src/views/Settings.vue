<template>
  <div class="settings-page">
    <div class="page-title">
      <div class="page-title-heading">设置</div>
    </div>

    <div style="display: grid; grid-template-columns: minmax(0, 1.55fr) minmax(320px, 1fr); gap: 18px;">
      <div>
        <el-card shadow="never" style="border-radius: var(--cf-radius); margin-bottom: 18px;">
          <template #header>
            <span style="font-weight: 700; color: var(--cf-text-heavy)">Agent 状态</span>
          </template>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="连接状态">
              <el-tag :type="app.isAgentOnline ? 'success' : 'danger'" size="small">
                {{ app.isAgentOnline ? '在线' : '离线' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="监听地址">{{ app.dashboard.agent.listenAddr }}</el-descriptions-item>
            <el-descriptions-item label="Codex 路径">{{ app.dashboard.agent.codexBinaryPath }}</el-descriptions-item>
            <el-descriptions-item label="启动时间">{{ app.dashboard.agent.startedAt }}</el-descriptions-item>
          </el-descriptions>
        </el-card>

        <el-card shadow="never" style="border-radius: var(--cf-radius);">
          <template #header>
            <span style="font-weight: 700; color: var(--cf-text-heavy)">可用 Agent</span>
          </template>
          <el-table :data="app.dashboard.agents" stripe>
            <el-table-column prop="name" label="名称" />
            <el-table-column prop="id" label="ID" />
            <el-table-column label="状态">
              <template #default="{ row }">
                <el-tag :type="row.available ? 'success' : 'info'" size="small">
                  {{ row.available ? '可用' : '不可用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="默认">
              <template #default="{ row }">
                <el-icon v-if="row.default" color="#10b981"><Check /></el-icon>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </div>

      <div>
        <el-card shadow="never" style="border-radius: var(--cf-radius); margin-bottom: 18px;">
          <template #header>
            <span style="font-weight: 700; color: var(--cf-text-heavy)">当前用户</span>
          </template>
          <div style="display: flex; align-items: center; gap: 16px; margin-bottom: 16px;">
            <el-avatar :size="56" style="background-color: #3388ff; font-size: 22px; font-weight: 700;">
              {{ (auth.username || 'U')[0].toUpperCase() }}
            </el-avatar>
            <div>
              <div style="font-size: 18px; font-weight: 700; color: var(--cf-text-heavy)">{{ auth.username }}</div>
              <div style="font-size: 13px; color: var(--cf-text-secondary)">系统用户</div>
            </div>
          </div>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="用户名">{{ auth.username }}</el-descriptions-item>
            <el-descriptions-item label="认证方式">JWT Token</el-descriptions-item>
          </el-descriptions>
          <el-button type="danger" style="margin-top: 16px; width: 100%;" @click="handleLogout">退出登录</el-button>
        </el-card>

        <el-card shadow="never" style="border-radius: var(--cf-radius);">
          <template #header>
            <span style="font-weight: 700; color: var(--cf-text-heavy)">系统信息</span>
          </template>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="版本">0.1.0</el-descriptions-item>
            <el-descriptions-item label="前端">Vue3 + TypeScript + Element Plus</el-descriptions-item>
            <el-descriptions-item label="后端">Go + Codex Agent</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAppStore } from '../stores/app'
import { useAuthStore } from '../stores/auth'
import { useTabsStore } from '../stores/tabs'

const app = useAppStore()
const auth = useAuthStore()
const router = useRouter()
const tabsStore = useTabsStore()

function handleLogout() {
  auth.logout()
  tabsStore.closeAllTabs()
  router.push('/login')
}

onMounted(() => {
  app.refreshDashboard()
})
</script>

<style scoped>
.settings-page {
  max-width: 1200px;
  margin: 0 auto;
}

@media (max-width: 1200px) {
  .settings-page > div:last-child {
    grid-template-columns: minmax(0, 1fr) !important;
  }
}
</style>
