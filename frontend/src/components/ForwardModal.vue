<script setup lang="ts">
import { ref, watch } from 'vue'
import { X, Send } from '@lucide/vue'
import { useChatStore } from '../stores/chat'
import { useUIStore } from '../stores/ui'

const chat = useChatStore()
const ui = useUIStore()
const selected = ref<Set<string>>(new Set())

watch(() => ui.showForward, () => { selected.value = new Set() })

function toggle(id: string) {
  if (selected.value.has(id)) selected.value.delete(id)
  else selected.value.add(id)
  selected.value = new Set(selected.value)
}

async function send() {
  if (!ui.showForward || selected.value.size === 0) return
  await chat.forwardTo(ui.showForward as any, Array.from(selected.value))
  ui.showForward = null
}

const initials = (n: string) => n.split(' ').map((s) => s[0]).slice(0, 2).join('').toUpperCase()
</script>

<template>
  <div v-if="ui.showForward" class="fixed inset-0 z-50 bg-black/50 flex items-center justify-center" @click.self="ui.showForward = null">
    <div class="w-[420px] max-w-[92vw] max-h-[80vh] bg-white dark:bg-wa-panel-dark rounded-2xl shadow-2xl flex flex-col overflow-hidden">
      <header class="flex items-center justify-between px-5 py-3 border-b border-wa-border dark:border-wa-border-dark">
        <h2 class="font-semibold">Teruskan ke...</h2>
        <button @click="ui.showForward = null" class="text-wa-muted dark:text-wa-muted-dark"><X :size="18" /></button>
      </header>
      <ul class="flex-1 overflow-y-auto scrollbar-thin">
        <li
          v-for="c in chat.visibleChats"
          :key="c.id"
          @click="toggle(c.id)"
          class="flex items-center gap-3 px-5 py-3 cursor-pointer hover:bg-wa-hover dark:hover:bg-wa-hover-dark"
        >
          <input type="checkbox" :checked="selected.has(c.id)" class="w-4 h-4 accent-wa-green" />
          <div class="w-9 h-9 rounded-full bg-slate-400 text-white flex items-center justify-center font-semibold overflow-hidden text-sm">
            <img v-if="c.avatarUrl" :src="c.avatarUrl" class="w-full h-full object-cover" />
            <span v-else>{{ initials(c.name) }}</span>
          </div>
          <div class="flex-1 min-w-0">
            <div class="font-medium text-sm truncate">{{ c.name }}</div>
          </div>
        </li>
      </ul>
      <footer class="flex items-center justify-end gap-2 px-5 py-3 border-t border-wa-border dark:border-wa-border-dark">
        <span class="text-xs text-wa-muted dark:text-wa-muted-dark mr-auto">{{ selected.size }} dipilih</span>
        <button @click="ui.showForward = null" class="text-sm px-3 py-1.5 rounded text-wa-muted dark:text-wa-muted-dark">Batal</button>
        <button @click="send" :disabled="selected.size === 0" class="text-sm px-3 py-1.5 rounded bg-wa-green text-white disabled:opacity-50 flex items-center gap-1">
          <Send :size="14" /> Kirim
        </button>
      </footer>
    </div>
  </div>
</template>
