import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../utils/api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('cf_token') || '')
  const username = ref(localStorage.getItem('cf_user') || '')

  const isLoggedIn = computed(() => !!token.value)

  async function login(user: string, password: string) {
    const res = await api.post('/auth/login', { username: user, password })
    token.value = res.data.token
    username.value = res.data.username
    localStorage.setItem('cf_token', res.data.token)
    localStorage.setItem('cf_user', res.data.username)
  }

  function logout() {
    token.value = ''
    username.value = ''
    localStorage.removeItem('cf_token')
    localStorage.removeItem('cf_user')
  }

  return { token, username, isLoggedIn, login, logout }
})
