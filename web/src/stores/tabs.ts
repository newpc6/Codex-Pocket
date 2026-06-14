import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { RouteLocationNormalized } from 'vue-router'

export interface TabItem {
  key: string
  name?: string
  title: string
  path: string
  closable: boolean
  query?: Record<string, any>
  params?: Record<string, any>
}

export const useTabsStore = defineStore('tabs', () => {
  const tabs = ref<TabItem[]>([
    { key: '/', title: '会话', path: '/', closable: false },
  ])
  const activeKey = ref('/')

  function addTab(route: RouteLocationNormalized) {
    const { path, name, query, params, meta } = route
    const key = path
    const title = (meta?.title as string) || (name as string) || '未命名页面'

    const existing = tabs.value.find((t) => t.key === key)
    if (existing) {
      activeKey.value = key
      return
    }

    tabs.value.push({
      key,
      title,
      path,
      closable: true,
      query: query as Record<string, any>,
      params: params as Record<string, any>,
      name: name as string,
    })
    activeKey.value = key
  }

  function removeTab(targetKey: string) {
    const idx = tabs.value.findIndex((t) => t.key === targetKey)
    if (idx === -1) return
    tabs.value.splice(idx, 1)
    if (activeKey.value === targetKey && tabs.value.length > 0) {
      const newIdx = idx < tabs.value.length ? idx : idx - 1
      activeKey.value = tabs.value[newIdx].key
    }
  }

  function closeOtherTabs(targetKey: string) {
    tabs.value = tabs.value.filter((t) => t.key === targetKey || !t.closable)
    activeKey.value = targetKey
  }

  function closeAllTabs() {
    tabs.value = tabs.value.filter((t) => !t.closable)
    if (tabs.value.length > 0) activeKey.value = tabs.value[0].key
  }

  function setActiveKey(key: string) {
    activeKey.value = key
  }

  const currentTab = computed(() => tabs.value.find((t) => t.key === activeKey.value))

  return {
    tabs, activeKey, currentTab,
    addTab, removeTab, closeOtherTabs, closeAllTabs, setActiveKey,
  }
})
