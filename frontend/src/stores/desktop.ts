import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  DesktopListApps,
  DesktopOpenApp,
  DesktopCloseApp,
  DesktopMediaPlayPause,
  DesktopMediaNext,
  DesktopMediaPrev,
  DesktopMediaNowPlaying,
  DesktopGetVolume,
  DesktopSetVolume,
  DesktopScreenshot,
  DesktopLockScreen,
} from '../../wailsjs/go/main/App'

export interface AppInfo {
  name: string
  pid: number
  icon?: string
}

export interface MediaInfo {
  title: string
  artist: string
  album: string
  playing: boolean
  player: string
}

function toAppInfo(raw: any): AppInfo {
  return { name: raw?.name || '', pid: raw?.pid || 0, icon: raw?.icon || '' }
}

function toMediaInfo(raw: any): MediaInfo | null {
  if (!raw) return null
  return {
    title: raw.title || '',
    artist: raw.artist || '',
    album: raw.album || '',
    playing: !!raw.playing,
    player: raw.player || '',
  }
}

export const useDesktopStore = defineStore('desktop', () => {
  const runningApps = ref<AppInfo[]>([])
  const mediaInfo = ref<MediaInfo | null>(null)
  const volume = ref(50)
  const loading = ref(false)
  const error = ref('')
  let refreshTimer: ReturnType<typeof setInterval> | null = null

  async function listApps() {
    try {
      const raw = await DesktopListApps()
      const list = Array.isArray(raw) ? raw : []
      runningApps.value = list.map(toAppInfo)
    } catch (e: any) {
      error.value = e?.message || String(e)
    }
  }

  async function openApp(name: string) {
    try {
      await DesktopOpenApp(name)
      await listApps()
    } catch (e: any) {
      error.value = e?.message || String(e)
    }
  }

  async function closeApp(name: string) {
    try {
      await DesktopCloseApp(name)
      await listApps()
    } catch (e: any) {
      error.value = e?.message || String(e)
    }
  }

  async function mediaPlayPause() {
    try {
      await DesktopMediaPlayPause()
      await nowPlaying()
    } catch (e: any) {
      error.value = e?.message || String(e)
    }
  }

  async function mediaNext() {
    try {
      await DesktopMediaNext()
      await nowPlaying()
    } catch (e: any) {
      error.value = e?.message || String(e)
    }
  }

  async function mediaPrev() {
    try {
      await DesktopMediaPrev()
      await nowPlaying()
    } catch (e: any) {
      error.value = e?.message || String(e)
    }
  }

  async function nowPlaying() {
    try {
      const raw = await DesktopMediaNowPlaying()
      mediaInfo.value = toMediaInfo(raw)
    } catch {
      mediaInfo.value = null
    }
  }

  async function getVolume() {
    try {
      volume.value = await DesktopGetVolume()
    } catch {
      // ignore
    }
  }

  async function setVolume(pct: number) {
    try {
      await DesktopSetVolume(pct)
      volume.value = pct
    } catch (e: any) {
      error.value = e?.message || String(e)
    }
  }

  async function screenshot() {
    try {
      return await DesktopScreenshot()
    } catch (e: any) {
      error.value = e?.message || String(e)
      return ''
    }
  }

  async function lockScreen() {
    try {
      await DesktopLockScreen()
    } catch (e: any) {
      error.value = e?.message || String(e)
    }
  }

  async function refreshAll() {
    loading.value = true
    error.value = ''
    await Promise.all([listApps(), nowPlaying(), getVolume()])
    loading.value = false
  }

  function startAutoRefresh() {
    refreshAll()
    if (refreshTimer) clearInterval(refreshTimer)
    refreshTimer = setInterval(() => refreshAll(), 5000)
  }

  function stopAutoRefresh() {
    if (refreshTimer) {
      clearInterval(refreshTimer)
      refreshTimer = null
    }
  }

  return {
    runningApps,
    mediaInfo,
    volume,
    loading,
    error,
    listApps,
    openApp,
    closeApp,
    mediaPlayPause,
    mediaNext,
    mediaPrev,
    nowPlaying,
    getVolume,
    setVolume,
    screenshot,
    lockScreen,
    refreshAll,
    startAutoRefresh,
    stopAutoRefresh,
  }
})
