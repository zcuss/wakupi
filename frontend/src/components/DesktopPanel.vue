<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import {
  X,
  Monitor,
  Play,
  Pause,
  SkipForward,
  SkipBack,
  Volume2,
  AppWindow,
  Lock,
  Camera,
  Terminal,
  Globe,
  Folder,
  Settings,
  Music,
  HelpCircle,
} from '@lucide/vue'
import { useDesktopStore } from '../stores/desktop'

const emit = defineEmits<{ close: [] }>()
const ds = useDesktopStore()

onMounted(() => ds.startAutoRefresh())
onUnmounted(() => ds.stopAutoRefresh())

const quickApps = [
  { name: 'Terminal', icon: Terminal },
  { name: 'Firefox', icon: Globe },
  { name: 'google-chrome', icon: Globe, label: 'Chrome' },
  { name: 'nautilus', icon: Folder, label: 'Files' },
  { name: 'spotify', icon: Music },
  { name: 'gnome-control-center', icon: Settings, label: 'Settings' },
]

function formatAppIcon(name: string): string {
  const icons: Record<string, string> = {
    firefox: '🌐', 'google-chrome': '🌐', 'google-chrome-stable': '🌐',
    spotify: '🎵', nautilus: '📁', 'thunar': '📁', 'dolphin': '📁',
    code: '💻', 'code-insiders': '💻', 'gnome-terminal': '💻',
    terminator: '💻', alacritty: '💻', kitty: '💻',
    discord: '💬', telegram: '💬', slack: '💬',
    gimp: '🎨', inkscape: '🎨', blender: '🎨',
    vlc: '🎬', mpv: '🎬', 'steam': '🎮',
    'gnome-control-center': '⚙️', 'system-monitor': '📊',
  }
  return icons[name] || '📱'
}
</script>

<template>
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
    <div class="bg-white dark:bg-gray-800 rounded-xl w-full max-w-md shadow-2xl max-h-[90vh] flex flex-col">
      <!-- Header -->
      <div class="flex items-center justify-between p-4 border-b dark:border-gray-700 shrink-0">
        <div class="flex items-center gap-2">
          <Monitor class="w-5 h-5 text-blue-500" />
          <h2 class="font-semibold text-lg">Desktop Controller</h2>
        </div>
        <button @click="emit('close')" class="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded">
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Content -->
      <div class="p-4 space-y-4 overflow-y-auto">
        <!-- Error -->
        <div v-if="ds.error" class="p-3 bg-red-50 dark:bg-red-900/30 text-red-600 text-sm rounded-lg">
          {{ ds.error }}
        </div>

        <!-- Media Control -->
        <div class="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-3">
          <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">🎵 Media</h3>
          <div v-if="ds.mediaInfo" class="mb-2">
            <p class="font-medium text-sm truncate">{{ ds.mediaInfo.title || 'Unknown' }}</p>
            <p class="text-xs text-gray-500 truncate">{{ ds.mediaInfo.artist }}</p>
          </div>
          <div v-else class="mb-2">
            <p class="text-sm text-gray-400">No media playing</p>
          </div>
          <div class="flex items-center gap-2">
            <button @click="ds.mediaPrev()" class="p-2 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-lg transition">
              <SkipBack :size="18" />
            </button>
            <button @click="ds.mediaPlayPause()" class="p-3 bg-blue-500 hover:bg-blue-600 text-white rounded-full transition">
              <Pause v-if="ds.mediaInfo?.playing" :size="20" />
              <Play v-else :size="20" />
            </button>
            <button @click="ds.mediaNext()" class="p-2 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-lg transition">
              <SkipForward :size="18" />
            </button>
          </div>
        </div>

        <!-- Volume -->
        <div class="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-3">
          <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">🔊 Volume</h3>
          <div class="flex items-center gap-3">
            <Volume2 :size="18" class="text-gray-400 shrink-0" />
            <input
              type="range"
              min="0"
              max="100"
              :value="ds.volume"
              @input="ds.setVolume(parseInt(($event.target as HTMLInputElement).value))"
              class="w-full h-2 bg-gray-200 dark:bg-gray-600 rounded-lg appearance-none cursor-pointer accent-blue-500"
            />
            <span class="text-sm font-medium w-10 text-right">{{ ds.volume }}%</span>
          </div>
        </div>

        <!-- Quick Launch -->
        <div>
          <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">🚀 Quick Launch</h3>
          <div class="grid grid-cols-3 gap-2">
            <button
              v-for="app in quickApps"
              :key="app.name"
              @click="ds.openApp(app.name)"
              class="flex flex-col items-center gap-1 p-3 bg-gray-50 dark:bg-gray-700/50 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-lg transition text-sm"
            >
              <component :is="app.icon" :size="20" class="text-blue-500" />
              <span class="text-xs">{{ app.label || app.name }}</span>
            </button>
          </div>
        </div>

        <!-- Running Apps -->
        <div>
          <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">
            📱 Running Apps
            <span class="text-xs text-gray-400">({{ ds.runningApps.length }})</span>
          </h3>
          <div class="max-h-40 overflow-y-auto space-y-1">
            <div
              v-for="app in ds.runningApps"
              :key="app.pid"
              class="flex items-center justify-between p-2 hover:bg-gray-50 dark:hover:bg-gray-700/50 rounded-lg"
            >
              <div class="flex items-center gap-2 min-w-0">
                <span class="text-lg">{{ formatAppIcon(app.name) }}</span>
                <span class="text-sm truncate">{{ app.name }}</span>
                <span class="text-xs text-gray-400">PID {{ app.pid }}</span>
              </div>
              <button
                @click="ds.closeApp(app.name)"
                class="p-1 hover:bg-red-100 dark:hover:bg-red-900/30 text-red-500 rounded transition shrink-0"
                title="Close app"
              >
                <X :size="14" />
              </button>
            </div>
            <p v-if="ds.runningApps.length === 0" class="text-sm text-gray-400 text-center py-2">
              No apps detected
            </p>
          </div>
        </div>

        <!-- System -->
        <div>
          <h3 class="text-sm font-medium text-gray-500 dark:text-gray-400 mb-2">⚙️ System</h3>
          <div class="flex gap-2">
            <button
              @click="ds.screenshot()"
              class="flex-1 flex items-center justify-center gap-2 p-2 bg-gray-50 dark:bg-gray-700/50 hover:bg-gray-100 dark:hover:bg-gray-600 rounded-lg transition text-sm"
            >
              <Camera :size="16" /> Screenshot
            </button>
            <button
              @click="ds.lockScreen()"
              class="flex-1 flex items-center justify-center gap-2 p-2 bg-gray-50 dark:bg-gray-700/50 hover:bg-gray-100 dark:hover:bg-gray-600 rounded-lg transition text-sm"
            >
              <Lock :size="16" /> Lock
            </button>
          </div>
        </div>

        <!-- WA Commands Help -->
        <div class="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-3">
          <h3 class="text-sm font-medium text-blue-600 dark:text-blue-400 mb-1">
            <HelpCircle :size="14" class="inline" /> WhatsApp Commands
          </h3>
          <p class="text-xs text-blue-500/80 dark:text-blue-300/80 leading-relaxed">
            Kirim <code class="bg-blue-100 dark:bg-blue-800 px-1 rounded">!open terminal</code>,
            <code class="bg-blue-100 dark:bg-blue-800 px-1 rounded">!close firefox</code>,
            <code class="bg-blue-100 dark:bg-blue-800 px-1 rounded">!play</code>,
            <code class="bg-blue-100 dark:bg-blue-800 px-1 rounded">!next</code>,
            <code class="bg-blue-100 dark:bg-blue-800 px-1 rounded">!now</code>,
            <code class="bg-blue-100 dark:bg-blue-800 px-1 rounded">!volume 80</code>,
            <code class="bg-blue-100 dark:bg-blue-800 px-1 rounded">!screenshot</code>,
            <code class="bg-blue-100 dark:bg-blue-800 px-1 rounded">!lock</code>,
            <code class="bg-blue-100 dark:bg-blue-800 px-1 rounded">!apps</code>,
            <code class="bg-blue-100 dark:bg-blue-800 px-1 rounded">!help</code>
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
