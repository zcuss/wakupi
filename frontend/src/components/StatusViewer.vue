<script setup lang="ts">
import { computed, watch, ref, onUnmounted } from 'vue'
import { X, ChevronLeft, ChevronRight } from '@lucide/vue'
import { useStatusStore } from '../stores/status'

const status = useStatusStore()

const group = computed(() => {
  if (!status.viewer) return null
  return status.grouped.find((g) => g.sender === status.viewer!.groupSender) || null
})

const item = computed(() => {
  if (!group.value || !status.viewer) return null
  return group.value.items[status.viewer.index] || null
})

const progress = ref(0)
let timer: any = null

function startTimer() {
  stopTimer()
  if (item.value?.mediaType === 'video') return
  progress.value = 0
  timer = setInterval(() => {
    progress.value += 1
    if (progress.value >= 100) next()
  }, 60)
}

function stopTimer() {
  if (timer) clearInterval(timer)
  timer = null
}

function next() {
  if (!group.value || !status.viewer) return
  if (status.viewer.index < group.value.items.length - 1) {
    status.viewer.index++
  } else {
    const idx = status.grouped.findIndex((g) => g.sender === group.value!.sender)
    if (idx >= 0 && idx < status.grouped.length - 1) {
      status.viewer.groupSender = status.grouped[idx + 1].sender
      status.viewer.index = 0
    } else {
      status.closeViewer()
    }
  }
}

function prev() {
  if (!status.viewer) return
  if (status.viewer.index > 0) {
    status.viewer.index--
  } else {
    const idx = status.grouped.findIndex((g) => g.sender === status.viewer!.groupSender)
    if (idx > 0) {
      const prevGroup = status.grouped[idx - 1]
      status.viewer.groupSender = prevGroup.sender
      status.viewer.index = prevGroup.items.length - 1
    }
  }
}

watch(item, () => startTimer())
onUnmounted(() => stopTimer())

const initials = (n: string) =>
  n.split(' ').map((s) => s[0]).slice(0, 2).join('').toUpperCase()

function fmtTime(ts: number) {
  return new Date(ts * 1000).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <div
    v-if="status.viewer && group && item"
    class="fixed inset-0 z-50 bg-black flex flex-col"
    @click.self="status.closeViewer()"
  >
    <div class="flex gap-1 px-4 pt-3">
      <div
        v-for="(_, i) in group.items"
        :key="i"
        class="flex-1 h-1 bg-white/30 rounded-full overflow-hidden"
      >
        <div
          class="h-full bg-white transition-all"
          :style="{ width: i < status.viewer.index ? '100%' : i === status.viewer.index ? progress + '%' : '0%' }"
        />
      </div>
    </div>

    <header class="flex items-center gap-3 px-4 py-3 text-white">
      <div class="w-10 h-10 rounded-full bg-slate-400 flex items-center justify-center font-semibold">
        {{ initials(group.name) }}
      </div>
      <div class="flex-1">
        <div class="font-medium">{{ group.name }}</div>
        <div class="text-xs text-white/70">{{ fmtTime(item.timestamp) }}</div>
      </div>
      <button @click="status.closeViewer()" class="w-10 h-10 rounded-full hover:bg-white/10 flex items-center justify-center">
        <X :size="22" />
      </button>
    </header>

    <div class="flex-1 flex items-center justify-center px-4 pb-8 relative">
      <button
        @click="prev"
        class="absolute left-2 top-1/2 -translate-y-1/2 w-10 h-10 rounded-full bg-black/30 text-white flex items-center justify-center hover:bg-black/50"
      >
        <ChevronLeft :size="22" />
      </button>
      <button
        @click="next"
        class="absolute right-2 top-1/2 -translate-y-1/2 w-10 h-10 rounded-full bg-black/30 text-white flex items-center justify-center hover:bg-black/50"
      >
        <ChevronRight :size="22" />
      </button>

      <div v-if="item.mediaType === 'image' && item.mediaUrl" class="max-w-full max-h-full flex flex-col items-center gap-4">
        <img :src="item.mediaUrl" class="max-w-full max-h-[75vh] object-contain rounded" />
        <div v-if="item.caption" class="text-white text-center max-w-md">{{ item.caption }}</div>
      </div>

      <video
        v-else-if="item.mediaType === 'video' && item.mediaUrl"
        :src="item.mediaUrl"
        autoplay
        controls
        class="max-w-full max-h-[80vh] rounded"
        @ended="next"
      />

      <div
        v-else-if="!item.mediaType && item.text"
        class="w-full max-w-2xl rounded-lg p-12 text-center text-white text-3xl font-medium"
        style="background: linear-gradient(135deg, #00a884 0%, #008f72 100%);"
      >
        {{ item.text }}
      </div>
    </div>
  </div>
</template>
