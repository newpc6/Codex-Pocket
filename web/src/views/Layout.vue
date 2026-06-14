<template>
  <el-container style="height: 100vh">
    <el-aside width="220px" style="background: #1d1e22; color: #fff">
      <div style="padding: 20px 16px; font-size: 18px; font-weight: 700; letter-spacing: 0.5px">
        CodexFlow
      </div>
      <el-menu :default-active="currentRoute" background-color="#1d1e22" text-color="#a0aec0"
        active-text-color="#4f6ef7" router>
        <el-menu-item index="/">
          <el-icon><Monitor /></el-icon>
          <span>会话</span>
        </el-menu-item>
        <el-menu-item index="/approvals">
          <el-icon><Checked /></el-icon>
          <span>审批</span>
          <el-badge v-if="approvalCount > 0" :value="approvalCount" :max="99" style="margin-left: 8px" />
        </el-menu-item>
        <el-menu-item index="/settings">
          <el-icon><Setting /></el-icon>
          <span>设置</span>
        </el-menu-item>
      </el-menu>
      <div style="position: absolute; bottom: 20px; left: 16px; right: 16px">
        <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 12px; font-size: 12px; color: #a0aec0">
          <el-icon :color="online ? '#10b981' : '#ef4444'"><VideoCameraFilled /></el-icon>
          <span>{{ online ? 'Agent 在线' : 'Agent 离线' }}</span>
        </div>
        <el-button text style="color: #a0aec0; font-size: 12px" @click="handleLogout">
          <el-icon><SwitchButton /></el-icon> 退出登录
        </el-button>
      </div>
    </el-aside>
    <el-main style="padding: 0; overflow: auto; background: var(--cf-bg)">
      <router-view />
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore } from '../stores/app'
import { useAuthStore } from '../stores/auth'

const route = useRoute()
const router = useRouter()
const app = useAppStore()
const auth = useAuthStore()

const currentRoute = computed(() => route.path)
const online = computed(() => app.isAgentOnline)
const approvalCount = computed(() => app.filteredApprovals.length)

function handleLogout() {
  auth.logout()
  router.push('/login')
}
</script>
