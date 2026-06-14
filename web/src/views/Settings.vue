<template>
  <div class="page-container">
    <div style="font-size: 18px; font-weight: 600; margin-bottom: 16px">设置</div>

    <div class="card">
      <div style="font-size: 15px; font-weight: 600; margin-bottom: 16px">Agent 状态</div>
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
    </div>

    <div class="card">
      <div style="font-size: 15px; font-weight: 600; margin-bottom: 16px">可用 Agent</div>
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
    </div>

    <div class="card">
      <div style="font-size: 15px; font-weight: 600; margin-bottom: 16px">当前用户</div>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="用户名">{{ auth.username }}</el-descriptions-item>
      </el-descriptions>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useAppStore } from '../stores/app'
import { useAuthStore } from '../stores/auth'

const app = useAppStore()
const auth = useAuthStore()

onMounted(() => {
  app.refreshDashboard()
})
</script>
