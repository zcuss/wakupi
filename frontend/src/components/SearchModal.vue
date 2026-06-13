<script setup lang="ts">
import { ref, computed } from 'vue'
import { Search, X, MessageSquare } from '@lucide/vue'
import { useChatStore } from '../stores/chat'
import { useUIStore } from '../stores/ui'

const chat = useChatStore()
const ui = useUIStore()
const query = ref('')
const results = ref<any[]>([])
const loading = ref(false)

let timer: any = null
function onInput() {
  if (timer) clearTimeout(timer)
  timer = setTimeout(async () => {
    if (!query.value.trim()) {
      results.value = []
      return
    }
    loading.value = true
    results.value = await chat.searchAll(query.value.trim())
    loading.value = false
  }, 250)
}

function chatNameForJID(jid: string) {
  const id = `${chat.activeAccountId}::${jid}`
  return chat.chats.find((c) => c.id === id)?.name || jid.split('@')[0]
}

function jumpTo(r: any) {
  const id = `${chat.activeAccountId}::${r.jid}`
  chat.selectChat(id)
  ui.showSearch = false
}

function fmtTime(ts: number) {
  if (!ts) return ''
  const d = new Date(ts * 1000)
  return d.toLocaleString('id-ID', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <div v-if="ui.showSearch" class="fixed inset-0 z-40 bg-black/40 flex items-start justify-center pt-20" @click.self="ui.showSearch = false">
    <div class="w-[640px] max-w-[92vw] bg-white dark:bg-wa-panel-dark rounded-2xl shadow-2xl overflow-hidden">
      <header class="flex items-center gap-3 px-5 py-3 border-b border-wa-border dark:border-wa-border-dark">
        <Search :size="18" class="text-wa-muted dark:text-wa-muted-dark" />
        <input
          v-model="query"
          @input="onInput"
          autofocus
          placeholder="Cari di semua pesan..."
          class="flex-1 bg-transparent outline-none text-sm text-wa-text dark:text-wa-text-dark"
        />
        <button @click="ui.showSearch = false" class="text-wa-muted dark:text-wa-muted-dark">
          <X :size="18" />
        </button>
      </header>
      <div class="max-h-[60vh] overflow-y-auto">
        <div v-if="loading" class="px-5 py-8 text-center text-sm text-wa-muted dark:text-wa-muted-dark">
          Mencari...
        </div>
        <div v-else-if="!query" class="px-5 py-8 text-center text-sm text-wa-muted dark:text-wa-muted-dark">
          Ketik untuk mencari pesan
        </div>
        <div v-else-if="results.length === 0" class="px-5 py-8 text-center text-sm text-wa-muted dark:text-wa-muted-dark">
          Tidak ada hasil
        </div>
        <ul v-else class="py-2">
          <li
            v-for="r in results"
            :key="r.id + r.jid"
            @click="jumpTo(r)"
            class="px-5 py-3 hover:bg-wa-hover dark:hover:bg-wa-hover-dark cursor-pointer flex items-start gap-3"
          >
            <MessageSquare :size="18" class="mt-0.5 text-wa-muted dark:text-wa-muted-dark" />
            <div class="flex-1 min-w-0">
              <div class="flex items-baseline justify-between">
                <span class="font-medium text-sm truncate">{{ chatNameForJID(r.jid) }}</span>
                <span class="text-xs text-wa-muted dark:text-wa-muted-dark">{{ fmtTime(r.timestamp) }}</span>
              </div>
              <div class="text-sm text-wa-muted dark:text-wa-muted-dark truncate">{{ r.text || r.caption || r.fileName || '' }}</div>
            </div>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>
