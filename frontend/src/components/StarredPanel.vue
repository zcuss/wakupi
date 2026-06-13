<script setup lang="ts">
import { ref, watch } from 'vue'
import { X, Star } from '@lucide/vue'
import { useChatStore } from '../stores/chat'
import { useUIStore } from '../stores/ui'

const chat = useChatStore()
const ui = useUIStore()
const items = ref<any[]>([])
const loading = ref(false)

watch(() => ui.showStarred, async (v) => {
  if (!v) return
  loading.value = true
  items.value = await chat.getStarredList()
  loading.value = false
})

function chatNameForJID(jid: string) {
  const id = `${chat.activeAccountId}::${jid}`
  return chat.chats.find((c) => c.id === id)?.name || jid.split('@')[0]
}

function jumpTo(it: any) {
  const id = `${chat.activeAccountId}::${it.jid}`
  chat.selectChat(id)
  ui.showStarred = false
}

function fmtTime(ts: number) {
  if (!ts) return ''
  return new Date(ts * 1000).toLocaleString('id-ID', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <div v-if="ui.showStarred" class="fixed inset-0 z-40 bg-black/40 flex items-start justify-center pt-20" @click.self="ui.showStarred = false">
    <div class="w-[560px] max-w-[92vw] max-h-[70vh] bg-white dark:bg-wa-panel-dark rounded-2xl shadow-2xl flex flex-col overflow-hidden">
      <header class="flex items-center justify-between px-5 py-3 border-b border-wa-border dark:border-wa-border-dark">
        <div class="flex items-center gap-2">
          <Star :size="18" class="text-amber-500" />
          <h2 class="font-semibold">Pesan berbintang</h2>
        </div>
        <button @click="ui.showStarred = false" class="text-wa-muted dark:text-wa-muted-dark"><X :size="18" /></button>
      </header>
      <div class="flex-1 overflow-y-auto scrollbar-thin">
        <div v-if="loading" class="px-5 py-8 text-center text-sm text-wa-muted dark:text-wa-muted-dark">Memuat...</div>
        <div v-else-if="items.length === 0" class="px-5 py-8 text-center text-sm text-wa-muted dark:text-wa-muted-dark">
          Belum ada pesan berbintang
        </div>
        <ul v-else>
          <li v-for="it in items" :key="it.id + it.jid" @click="jumpTo(it)" class="px-5 py-3 cursor-pointer hover:bg-wa-hover dark:hover:bg-wa-hover-dark">
            <div class="flex items-center justify-between text-xs text-wa-muted dark:text-wa-muted-dark">
              <span class="font-medium text-wa-text dark:text-wa-text-dark">{{ chatNameForJID(it.jid) }}</span>
              <span>{{ fmtTime(it.timestamp) }}</span>
            </div>
            <div class="text-sm mt-1 text-wa-text dark:text-wa-text-dark truncate">{{ it.text || it.caption || it.fileName || '' }}</div>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>
