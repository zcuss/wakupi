<script setup lang="ts">
import { computed } from 'vue'
import { Pin, Archive, BellOff, Bell, Ban, ShieldOff, Trash2 } from '@lucide/vue'
import { useChatStore } from '../stores/chat'
import { useUIStore } from '../stores/ui'

const chat = useChatStore()
const ui = useUIStore()

const target = computed(() => {
  if (!ui.showChatMenu) return null
  return chat.chats.find((c) => c.id === ui.showChatMenu!.chatId) || null
})

function close() {
  ui.showChatMenu = null
}

async function doPin() {
  if (target.value) await chat.togglePin(target.value)
  close()
}
async function doArchive() {
  if (target.value) await chat.toggleArchive(target.value)
  close()
}
async function doMute() {
  if (!target.value) return
  const muted = !!target.value.mutedUntil && target.value.mutedUntil > Math.floor(Date.now() / 1000)
  if (muted) await chat.toggleMute(target.value, 0)
  else await chat.toggleMute(target.value, Math.floor(Date.now() / 1000) + 8 * 3600)
  close()
}
async function doBlock() {
  if (target.value) await chat.toggleBlock(target.value)
  close()
}
</script>

<template>
  <div
    v-if="ui.showChatMenu && target"
    class="fixed z-50 bg-white dark:bg-wa-panel-dark shadow-xl rounded-lg py-1 w-56 border border-wa-border dark:border-wa-border-dark"
    :style="{ top: ui.showChatMenu.y + 'px', left: ui.showChatMenu.x + 'px' }"
  >
    <button @click="doPin" class="w-full flex items-center gap-3 px-4 py-2 hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm">
      <Pin :size="16" /> {{ target.pinned ? 'Lepas pin' : 'Pin chat' }}
    </button>
    <button @click="doMute" class="w-full flex items-center gap-3 px-4 py-2 hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm">
      <BellOff v-if="!target.mutedUntil" :size="16" />
      <Bell v-else :size="16" />
      {{ target.mutedUntil && target.mutedUntil > Math.floor(Date.now() / 1000) ? 'Bunyikan' : 'Bisukan 8 jam' }}
    </button>
    <button @click="doArchive" class="w-full flex items-center gap-3 px-4 py-2 hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm">
      <Archive :size="16" /> {{ target.archived ? 'Keluarkan dari arsip' : 'Arsipkan' }}
    </button>
    <div class="border-t border-wa-border dark:border-wa-border-dark my-1" />
    <button @click="doBlock" class="w-full flex items-center gap-3 px-4 py-2 hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm text-red-500">
      <ShieldOff v-if="target.blocked" :size="16" />
      <Ban v-else :size="16" />
      {{ target.blocked ? 'Buka blokir' : 'Blokir' }}
    </button>
  </div>
  <div v-if="ui.showChatMenu" class="fixed inset-0 z-40" @click="close" />
</template>
