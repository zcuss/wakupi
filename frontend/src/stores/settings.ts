import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type Theme = 'light' | 'dark' | 'auto'

const STORAGE_KEY = 'wakupi.settings'

interface Settings {
  theme: Theme
  notifications: boolean
  notificationSound: boolean
  enterToSend: boolean
}

const defaults: Settings = {
  theme: 'auto',
  notifications: true,
  notificationSound: true,
  enterToSend: true,
}

function load(): Settings {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return { ...defaults }
    return { ...defaults, ...JSON.parse(raw) }
  } catch {
    return { ...defaults }
  }
}

function persist(s: Settings) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(s))
  } catch {}
}

export const useSettingsStore = defineStore('settings', () => {
  const initial = load()
  const theme = ref<Theme>(initial.theme)
  const notifications = ref<boolean>(initial.notifications)
  const notificationSound = ref<boolean>(initial.notificationSound)
  const enterToSend = ref<boolean>(initial.enterToSend)

  function applyTheme(t: Theme) {
    const html = document.documentElement
    let dark = false
    if (t === 'dark') dark = true
    else if (t === 'light') dark = false
    else dark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches
    if (dark) html.classList.add('dark')
    else html.classList.remove('dark')
  }

  applyTheme(theme.value)

  if (window.matchMedia) {
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    mq.addEventListener('change', () => {
      if (theme.value === 'auto') applyTheme('auto')
    })
  }

  watch([theme, notifications, notificationSound, enterToSend], () => {
    persist({
      theme: theme.value,
      notifications: notifications.value,
      notificationSound: notificationSound.value,
      enterToSend: enterToSend.value,
    })
    applyTheme(theme.value)
  })

  function setTheme(t: Theme) {
    theme.value = t
  }

  return {
    theme,
    notifications,
    notificationSound,
    enterToSend,
    setTheme,
  }
})
