<template>
  <article class="v-login">
    <section class="login-left">
      <div class="left-content">
        <div class="brand-lockup">
          <img class="logo-icon" src="/favicon.svg?v=2" alt="CodexPocket" />
          <div>
            <div class="brand-label">Session Console</div>
            <h1 class="left-title">CodexPocket</h1>
          </div>
        </div>
        <p class="left-desc">Codex 会话管理控制台，支持会话监控、审批处理、指令发送与实时交互。</p>
        <div class="platform-grid">
          <div class="platform-item" v-for="item in platformItems" :key="item">{{ item }}</div>
        </div>
        <div class="security-panel">
          <div class="security-title">安全机制</div>
          <div class="security-list">
            <span v-for="item in securityItems" :key="item">{{ item }}</span>
          </div>
        </div>
      </div>
    </section>
    <section class="login-right">
      <div class="login-shell">
        <div class="form-header">
          <div class="welcome-text">欢迎登录</div>
          <div class="project-name">CodexPocket</div>
          <p class="form-desc">请输入账号信息完成身份验证。</p>
        </div>
        <el-form :model="form" :rules="rules" ref="formRef" label-width="0" :disabled="submitting">
          <el-form-item prop="username">
            <el-input v-model.trim="form.username" placeholder="请输入用户名" @keyup.enter="handleLogin"
              @input="showError = false" clearable size="large">
              <template #prefix>
                <el-icon><User /></el-icon>
              </template>
            </el-input>
          </el-form-item>
          <el-form-item prop="password">
            <el-input v-model.trim="form.password" type="password" show-password placeholder="请输入密码"
              @keyup.enter="handleLogin" @input="showError = false" clearable size="large">
              <template #prefix>
                <el-icon><Lock /></el-icon>
              </template>
            </el-input>
          </el-form-item>
        </el-form>
        <el-button type="primary" :loading="submitting" :disabled="submitting" @click="handleLogin" size="large">
          登录
        </el-button>
        <div class="login-footer">
          <span>CodexPocket v0.1.0</span>
          <span>安全登录入口</span>
        </div>
      </div>
    </section>
  </article>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'

const router = useRouter()
const auth = useAuthStore()
const formRef = ref<FormInstance>()
const submitting = ref(false)
const showError = ref(false)

const platformItems = ['会话监控', '审批处理', '指令发送', '实时交互', '多 Agent', '安全认证']
const securityItems = ['JWT Token', '密码验证', '会话隔离', '操作审计']

const form = reactive({ username: '', password: '' })
const rules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function handleLogin() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    await auth.login(form.username, form.password)
    ElMessage.success('登录成功')
    router.push('/')
  } catch (e: any) {
    showError.value = true
    ElMessage.error(e.response?.data?.error || '登录失败')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.v-login {
  min-height: 100vh;
  overflow: hidden;
  position: relative;
  display: flex;
  background: #f5f7fb;
}

.login-left {
  flex: 1;
  min-width: 0;
  position: relative;
  background: linear-gradient(90deg, #2167d9 0%, #3388ff 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
  color: #ffffff;
}

.login-left::after {
  content: "";
  position: absolute;
  bottom: 0;
  right: 0;
  width: 260px;
  height: 260px;
  border-radius: 50%;
  border: 1px solid rgba(255, 255, 255, 0.16);
  background: rgba(255, 255, 255, 0.1);
  transform: translate(28%, 32%);
}

.left-content {
  position: relative;
  z-index: 1;
  width: min(560px, 100%);
}

.brand-lockup {
  display: flex;
  align-items: center;
  gap: 18px;
  margin-bottom: 34px;
}

.logo-icon {
  width: 72px;
  height: 72px;
  flex: 0 0 auto;
  display: block;
  border-radius: 18px;
  box-shadow: 0 18px 36px rgba(8, 28, 72, 0.26);
}

.brand-label {
  margin-bottom: 6px;
  color: rgba(255, 255, 255, 0.74);
  font-size: 13px;
  letter-spacing: 4px;
  text-transform: uppercase;
}

.left-title {
  margin: 0;
  color: #ffffff;
  font-size: 40px;
  font-weight: 700;
  line-height: 1.15;
}

.left-desc {
  max-width: 500px;
  margin: 0 0 42px 0;
  color: rgba(255, 255, 255, 0.86);
  font-size: 17px;
  line-height: 1.9;
}

.platform-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 32px;
}

.platform-item {
  min-height: 46px;
  display: flex;
  align-items: center;
  padding: 0 16px;
  border: 1px solid rgba(255, 255, 255, 0.26);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.14);
  color: #ffffff;
  font-size: 15px;
}

.security-panel {
  border: 1px solid rgba(255, 255, 255, 0.24);
  border-radius: 8px;
  padding: 18px;
  background: rgba(8, 28, 72, 0.18);
}

.security-title {
  margin-bottom: 14px;
  color: rgba(255, 255, 255, 0.74);
  font-size: 13px;
}

.security-list {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.security-list span {
  display: inline-flex;
  align-items: center;
  min-height: 30px;
  padding: 0 12px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.16);
  color: #ffffff;
  font-size: 13px;
}

.login-right {
  width: 535px;
  min-width: 470px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px;
  background: #ffffff;
  border-left: 1px solid #e7edf7;
}

.login-shell {
  width: 372px;
}

.form-header {
  width: 100%;
  margin-bottom: 34px;
}

.welcome-text {
  margin-bottom: 8px;
  color: #5f6f8f;
  font-size: 15px;
}

.project-name {
  color: #0f1f3d;
  font-weight: 700;
  font-size: 29px;
  line-height: 38px;
}

.form-desc {
  margin: 12px 0 0;
  color: #7a89a6;
  font-size: 14px;
  line-height: 1.6;
}

.login-shell :deep(.el-form) {
  width: 100%;
  position: relative;
}

.login-shell :deep(.el-form-item) {
  margin-bottom: 28px;
}

.login-shell :deep(.el-input__wrapper) {
  height: 52px;
  padding: 0 14px;
  border-radius: 8px;
  box-shadow: 0 0 0 1px #dce5f3 inset;
  background: #fbfdff;
}

.login-shell :deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px #b7c9e8 inset;
}

.login-shell :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px #2563eb inset;
}

.login-shell :deep(.el-input__prefix) {
  margin-right: 8px;
}

.login-shell :deep(.el-input__prefix .el-icon) {
  color: #8393b2;
  font-size: 18px;
}

.login-shell :deep(.el-input__inner) {
  height: 36px;
  color: #223355;
  font-size: 15px;
}

.login-shell :deep(.el-button) {
  width: 100%;
  height: 52px;
  border: none;
  border-radius: 8px;
  background: #1d4ed8;
  box-shadow: 0 8px 18px rgba(29, 78, 216, 0.22);
  font-size: 17px;
  font-weight: 600;
}

.login-shell :deep(.el-button:hover),
.login-shell :deep(.el-button:focus) {
  background: #2563eb;
}

.login-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 20px;
  color: #8a98b3;
  font-size: 12px;
}

@media (max-width: 980px) {
  .login-left { display: none; }
  .login-right { width: 100%; min-width: 0; border-left: none; }
}

@media (max-width: 560px) {
  .login-right { padding: 28px; }
  .login-shell { width: 100%; }
}
</style>
