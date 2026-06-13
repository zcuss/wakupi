<script setup lang="ts">
import { computed } from 'vue'
import { ArrowLeft, Plus, Camera, Type } from '@lucide/vue'
import { useStatusStore } from '../stores/status'
import { useChatStore } from '../stores/chat'

const status = useStatusStore()
const chat = useChatStore()

const initials = (n: string) =>
  n.split(' ').map((s) => s[0]).slice(0, 2).join('').toUpperCase()

function rel(ts: number) {
  const diff = Math.floor(Date.now() / 1000) - ts
  if (diff < 60) return 'baru saja'
  if (diff < 3600) return Math.floor(diff / 60) + ' menit lalu'
  if (diff < 86400) return Math.floor(diff / 3600) + ' jam lalu'
  return Math.floor(diff / 86400) + ' hari lalu'
}

const recents = computed(() => status.grouped.filter((g) => !g.items.every((i) => i.fromMe)))
const my = computed(() => status.myStatus)

function close() {
  status.showStatusPanel = false
}

function openViewer(sender: string) {
  status.openViewer(sender, 0)
}

async function postText() {
  status.showComposer = true
}

async function postImage() {
  if (!chat.activeAccountId) return
  await status.postImage(chat.activeAccountId, '')
}
</script>

<template>
  <section class="w-[380px] bg-white dark:bg-[#111b21] border-r border-wa-border dark:border-wa-border-dark flex flex-col">
    <header class="h-14 px-4 flex items-center gap-3 bg-wa-panel dark:bg-wa-panel-dark">
      <button @click="close" class="w-9 h-9 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center text-wa-muted dark:text-wa-muted-dark">
        <ArrowLeft :size="20" />
      </button>
      <h1 class="text-base font-semibold text-wa-text dark:text-wa-text-dark">Status</h1>
    </header>

    <div class="flex-1 overflow-y-auto scrollbar-thin">
      <div
        class="flex items-center gap-3 px-4 py-3 border-b border-wa-border dark:border-wa-border-dark cursor-pointer hover:bg-wa-hover dark:hover:bg-wa-hover-dark"
        @click="my && openViewer(my.sender)"
      >
        <div class="relative">
          <div class="w-12 h-12 rounded-full bg-slate-400 text-white flex items-center justify-center font-semibold">
            {{ initials(chat.activeAccount?.name || 'Saya') }}
          </div>
          <button
            @click.stop="postImage"
            class="absolute -bottom-0.5 -right-0.5 w-5 h-5 rounded-full bg-wa-green text-white flex items-center justify-center"
          >
            <Plus :size="14" />
          </button>
        </div>
        <div class="flex-1">
          <div class="font-medium text-wa-text dark:text-wa-text-dark">Status saya</div>
          <div class="text-sm text-wa-muted dark:text-wa-muted-dark">
            {{ my ? rel(my.latestTime) : 'Ketuk + untuk menambahkan status' }}
          </div>
        </div>
      </div>

      <div class="px-4 py-2 text-xs font-medium text-wa-muted dark:text-wa-muted-dark uppercase">Pembaruan terbaru</div>

      <ul>
        <li
          v-for="g in recents"
          :key="g.sender"
          class="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-wa-hover dark:hover:bg-wa-hover-dark"
          @click="openViewer(g.sender)"
        >
          <div class="w-12 h-12 rounded-full bg-slate-400 text-white flex items-center justify-center font-semibold ring-2 ring-wa-green ring-offset-2 ring-offset-white dark:ring-offset-[#111b21]">
            {{ initials(g.name) }}
          </div>
          <div class="flex-1 min-w-0">
            <div class="font-medium truncate text-wa-text dark:text-wa-text-dark">{{ g.name }}</div>
            <div class="text-sm text-wa-muted dark:text-wa-muted-dark">{{ rel(g.latestTime) }}</div>
          </div>
        </li>
        <li v-if="recents.length === 0" class="text-center text-sm text-wa-muted dark:text-wa-muted-dark py-8">
          Belum ada status terbaru
        </li>
      </ul>
    </div>

    <div class="px-4 py-3 flex gap-2 bg-wa-panel dark:bg-wa-panel-dark border-t border-wa-border dark:border-wa-border-dark">
      <button @click="postText" class="flex-1 flex items-center justify-center gap-2 bg-white dark:bg-wa-hover-dark hover:bg-wa-hover dark:hover:bg-wa-panel-dark rounded-lg py-2 text-sm">
        <Type :size="16" /> Teks
      </button>
      <button @click="postImage" class="flex-1 flex items-center justify-center gap-2 bg-wa-green hover:bg-wa-green-dark text-white rounded-lg py-2 text-sm">
        <Camera :size="16" /> Foto
      </button>
    </div>
  </section>
</template>
