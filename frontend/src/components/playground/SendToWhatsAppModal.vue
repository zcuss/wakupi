<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { X, Send, Search, Check, Loader2 } from '@lucide/vue'
import { useChatStore } from '../../stores/chat'
import { useUIStore } from '../../stores/ui'
import { markdownToWhatsApp } from '../../lib/whatsappFormat'

const chat = useChatStore()
const ui = useUIStore()

const query = ref('')
const selected = ref<string>('')
const sending = ref(false)
const done = ref(false)
const error = ref('')
const convert = ref(true)

watch(
  () => ui.sendToWhatsApp,
  (open) => {
    if (open !== null) {
      query.value = ''
      selected.value = chat.activeChatId || ''
      sending.value = false
      done.value = false
      error.value = ''
      convert.value = true
    }
  }
)

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  const list = chat.visibleChats
  if (!q) return list
  return list.filter((c) => c.name.toLowerCase().includes(q))
})

const rawText = computed(() => ui.sendToWhatsApp || '')
const outgoing = computed(() => (convert.value ? markdownToWhatsApp(rawText.value) : rawText.value))

function close() {
  ui.sendToWhatsApp = null
}

const initials = (n: string) =>
  n.split(' ').map((s) => s[0]).slice(0, 2).join('').toUpperCase()

async function send() {
  if (!selected.value || !ui.sendToWhatsApp || sending.value) return
  sending.value = true
  error.value = ''
  try {
    await chat.sendTextToChat(selected.value, outgoing.value)
    done.value = true
    setTimeout(close, 700)
  } catch (e: any) {
    error.value = e?.message || String(e)
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <div
    v-if="ui.sendToWhatsApp !== null"
    class="fixed inset-0 z-50 bg-black/50 flex items-center justify-center"
    @click.self="close"
  >
    <div class="w-[440px] max-w-[92vw] max-h-[82vh] bg-white dark:bg-wa-panel-dark rounded-2xl shadow-2xl flex flex-col overflow-hidden">
      <header class="flex items-center justify-between px-5 py-3 border-b border-wa-border dark:border-wa-border-dark">
        <h2 class="font-semibold text-wa-text dark:text-wa-text-dark">Kirim ke WhatsApp</h2>
        <button @click="close" class="text-wa-muted dark:text-wa-muted-dark"><X :size="18" /></button>
      </header>

      <!-- Message preview -->
      <div class="px-5 py-3 border-b border-wa-border dark:border-wa-border-dark">
        <div class="flex items-center justify-between mb-1.5">
          <p class="text-xs text-wa-muted dark:text-wa-muted-dark">Pesan</p>
          <label class="flex items-center gap-1.5 text-xs text-wa-muted dark:text-wa-muted-dark cursor-pointer select-none">
            <input v-model="convert" type="checkbox" class="w-3.5 h-3.5 accent-wa-green" />
            Format WhatsApp
          </label>
        </div>
        <div class="bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm max-h-28 overflow-y-auto whitespace-pre-wrap break-words text-wa-text dark:text-wa-text-dark">
          {{ outgoing }}
        </div>
      </div>

      <!-- Search -->
      <div class="px-5 pt-3">
        <div class="flex items-center gap-2 bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2">
          <Search :size="16" class="text-wa-muted dark:text-wa-muted-dark" />
          <input
            v-model="query"
            placeholder="Cari chat…"
            class="flex-1 bg-transparent outline-none text-sm text-wa-text dark:text-wa-text-dark"
          />
        </div>
      </div>

      <!-- Chat list -->
      <ul class="flex-1 overflow-y-auto scrollbar-thin px-2 py-2">
        <li
          v-for="c in filtered"
          :key="c.id"
          @click="selected = c.id"
          class="flex items-center gap-3 px-3 py-2.5 rounded-lg cursor-pointer transition"
          :class="selected === c.id ? 'bg-wa-green/10' : 'hover:bg-wa-hover dark:hover:bg-wa-hover-dark'"
        >
          <div class="w-9 h-9 rounded-full bg-slate-400 text-white flex items-center justify-center font-semibold overflow-hidden text-sm shrink-0">
            <img v-if="c.avatarUrl" :src="c.avatarUrl" class="w-full h-full object-cover" />
            <span v-else>{{ initials(c.name) }}</span>
          </div>
          <span class="flex-1 min-w-0 font-medium text-sm truncate text-wa-text dark:text-wa-text-dark">{{ c.name }}</span>
          <Check v-if="selected === c.id" :size="18" class="text-wa-green shrink-0" />
        </li>
        <li v-if="filtered.length === 0" class="px-3 py-6 text-center text-sm text-wa-muted dark:text-wa-muted-dark">
          Tidak ada chat ditemukan
        </li>
      </ul>

      <footer class="flex items-center gap-2 px-5 py-3 border-t border-wa-border dark:border-wa-border-dark">
        <span v-if="error" class="text-xs text-red-500 mr-auto truncate">{{ error }}</span>
        <span v-else-if="done" class="text-xs text-wa-green mr-auto flex items-center gap-1"><Check :size="14" /> Terkirim</span>
        <span v-else class="text-xs text-wa-muted dark:text-wa-muted-dark mr-auto">Pilih satu chat tujuan</span>
        <button @click="close" class="text-sm px-3 py-1.5 rounded text-wa-muted dark:text-wa-muted-dark">Batal</button>
        <button
          @click="send"
          :disabled="!selected || sending || done"
          class="text-sm px-4 py-1.5 rounded-lg bg-wa-green hover:bg-wa-green-dark text-white disabled:opacity-50 flex items-center gap-1.5"
        >
          <Loader2 v-if="sending" :size="14" class="animate-spin" />
          <Send v-else :size="14" />
          Kirim
        </button>
      </footer>
    </div>
  </div>
</template>
