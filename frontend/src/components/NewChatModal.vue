<script setup lang="ts">
import { ref, computed } from 'vue'
import { X, Search, Phone, MessageSquarePlus } from '@lucide/vue'
import { useChatStore } from '../stores/chat'
import { useUIStore } from '../stores/ui'

const chat = useChatStore()
const ui = useUIStore()
const query = ref('')
const phoneInput = ref('')
const checking = ref(false)
const checkResult = ref<{ jid: string; onWA: boolean } | null>(null)

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  const known = chat.chats.filter((c) => c.accountId === chat.activeAccountId && !c.isGroup)
  if (!q) return known
  return known.filter((c) => c.name.toLowerCase().includes(q) || c.jid.includes(q))
})

const initials = (n: string) => n.split(' ').map((s) => s[0]).slice(0, 2).join('').toUpperCase()

async function checkPhone() {
  if (!phoneInput.value.trim()) return
  checking.value = true
  checkResult.value = null
  checkResult.value = await chat.checkIsOnWA(phoneInput.value)
  checking.value = false
}

function startWith(jid: string, name: string) {
  chat.startChatWithJID(jid, name)
  ui.showNewChat = false
}

function close() {
  ui.showNewChat = false
  query.value = ''
  phoneInput.value = ''
  checkResult.value = null
}
</script>

<template>
  <div v-if="ui.showNewChat" class="fixed inset-0 z-40 bg-black/40 flex items-center justify-center" @click.self="close">
    <div class="w-[420px] max-w-[92vw] max-h-[80vh] bg-white dark:bg-wa-panel-dark rounded-2xl shadow-2xl overflow-hidden flex flex-col">
      <header class="flex items-center justify-between px-5 py-3 border-b border-wa-border dark:border-wa-border-dark">
        <h2 class="font-semibold">Chat baru</h2>
        <button @click="close" class="text-wa-muted dark:text-wa-muted-dark"><X :size="18" /></button>
      </header>

      <div class="px-5 py-3 border-b border-wa-border dark:border-wa-border-dark space-y-2">
        <div class="flex items-center gap-2 bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 h-9">
          <Phone :size="16" class="text-wa-muted dark:text-wa-muted-dark" />
          <input
            v-model="phoneInput"
            @keydown.enter="checkPhone"
            placeholder="Nomor telepon (contoh: 6281234567890)"
            class="flex-1 bg-transparent outline-none text-sm"
          />
          <button @click="checkPhone" :disabled="checking" class="text-xs px-2 py-1 rounded bg-wa-green text-white disabled:opacity-50">
            {{ checking ? '...' : 'Cek' }}
          </button>
        </div>
        <div v-if="checkResult && checkResult.onWA" class="flex items-center justify-between bg-emerald-50 dark:bg-emerald-900/20 rounded-lg px-3 py-2 text-sm">
          <span class="text-emerald-700 dark:text-emerald-300">Aktif di WhatsApp</span>
          <button @click="startWith(checkResult.jid, phoneInput)" class="text-xs px-2 py-1 rounded bg-wa-green text-white">
            Mulai chat
          </button>
        </div>
        <div v-else-if="checkResult && !checkResult.onWA" class="text-sm text-red-500 px-3">
          Nomor ini tidak terdaftar di WhatsApp
        </div>
      </div>

      <div class="px-5 py-2">
        <div class="flex items-center gap-2 bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 h-9">
          <Search :size="16" class="text-wa-muted dark:text-wa-muted-dark" />
          <input
            v-model="query"
            placeholder="Cari kontak"
            class="flex-1 bg-transparent outline-none text-sm"
          />
        </div>
      </div>

      <ul class="flex-1 overflow-y-auto scrollbar-thin">
        <li
          v-for="c in filtered"
          :key="c.id"
          @click="startWith(c.jid, c.name)"
          class="flex items-center gap-3 px-5 py-3 cursor-pointer hover:bg-wa-hover dark:hover:bg-wa-hover-dark"
        >
          <div class="w-10 h-10 rounded-full bg-slate-400 text-white flex items-center justify-center font-semibold overflow-hidden">
            <img v-if="c.avatarUrl" :src="c.avatarUrl" class="w-full h-full object-cover" />
            <span v-else>{{ initials(c.name) }}</span>
          </div>
          <div class="flex-1 min-w-0">
            <div class="font-medium truncate">{{ c.name }}</div>
            <div class="text-xs text-wa-muted dark:text-wa-muted-dark truncate">{{ c.jid.split('@')[0] }}</div>
          </div>
        </li>
        <li v-if="filtered.length === 0" class="px-5 py-8 text-center text-sm text-wa-muted dark:text-wa-muted-dark">
          Tidak ada kontak
        </li>
      </ul>
    </div>
  </div>
</template>
