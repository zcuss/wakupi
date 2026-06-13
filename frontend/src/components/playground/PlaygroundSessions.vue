<script setup lang="ts">
import { Plus, MessageSquare, Trash2, PanelLeftClose } from '@lucide/vue'
import { usePlaygroundStore } from '../../stores/playground'
import { useUIStore } from '../../stores/ui'

const pg = usePlaygroundStore()
const ui = useUIStore()

function fmt(ts: number): string {
  const d = new Date(ts)
  const today = new Date()
  if (d.toDateString() === today.toDateString()) {
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }
  return d.toLocaleDateString([], { day: '2-digit', month: 'short' })
}

function remove(e: Event, id: string) {
  e.stopPropagation()
  pg.deleteSession(id)
}
</script>

<template>
  <div class="h-full flex flex-col bg-wa-panel dark:bg-[#111b21] border-r border-wa-border dark:border-wa-border-dark">
    <header class="flex items-center justify-between px-3 py-3 border-b border-wa-border dark:border-wa-border-dark">
      <span class="text-sm font-semibold text-wa-text dark:text-wa-text-dark">Percakapan</span>
      <button
        @click="ui.pgLeftCollapsed = true"
        class="p-1.5 rounded-lg text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark"
        title="Sembunyikan panel"
      >
        <PanelLeftClose :size="16" />
      </button>
    </header>

    <div class="px-3 py-2">
      <button
        @click="pg.createSession()"
        class="w-full flex items-center justify-center gap-2 bg-wa-green hover:bg-wa-green-dark text-white py-2 rounded-lg text-sm font-medium transition"
      >
        <Plus :size="16" /> Chat baru
      </button>
    </div>

    <div class="flex-1 overflow-y-auto px-2 pb-2 space-y-0.5">
      <button
        v-for="s in pg.sortedSessions"
        :key="s.id"
        @click="pg.selectSession(s.id)"
        class="group w-full text-left px-3 py-2.5 rounded-lg flex items-start gap-2.5 transition"
        :class="s.id === pg.activeId
          ? 'bg-wa-green/10 text-wa-text dark:text-wa-text-dark'
          : 'hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-wa-text dark:text-wa-text-dark'"
      >
        <MessageSquare :size="16" class="mt-0.5 shrink-0 text-wa-muted dark:text-wa-muted-dark" />
        <span class="flex-1 min-w-0">
          <span class="block text-sm truncate">{{ s.title }}</span>
          <span class="block text-xs text-wa-muted dark:text-wa-muted-dark">{{ fmt(s.updatedAt) }}</span>
        </span>
        <span
          @click="remove($event, s.id)"
          class="opacity-0 group-hover:opacity-100 p-1 rounded text-wa-muted hover:text-red-500 transition"
          title="Hapus"
        >
          <Trash2 :size="14" />
        </span>
      </button>
    </div>
  </div>
</template>
