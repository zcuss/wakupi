<script setup lang="ts">
import { X, Download } from '@lucide/vue'
import { useChatStore } from '../stores/chat'
import { computed } from 'vue'

const store = useChatStore()
const msg = computed(() => store.previewMessage)

function close() {
  store.setPreview(null)
}

function download() {
  if (!msg.value?.mediaUrl) return
  const a = document.createElement('a')
  a.href = msg.value.mediaUrl
  a.download = msg.value.fileName || `wakupi-${msg.value.id}`
  a.click()
}
</script>

<template>
  <div
    v-if="msg"
    class="fixed inset-0 z-40 bg-black/90 flex flex-col"
    @click.self="close"
  >
    <header class="flex items-center justify-end gap-2 p-4 text-white">
      <button @click="download" class="w-10 h-10 rounded-full hover:bg-white/10 flex items-center justify-center">
        <Download :size="22" />
      </button>
      <button @click="close" class="w-10 h-10 rounded-full hover:bg-white/10 flex items-center justify-center">
        <X :size="22" />
      </button>
    </header>
    <div class="flex-1 flex items-center justify-center p-4 overflow-auto">
      <img v-if="msg.mediaType === 'image' && msg.mediaUrl" :src="msg.mediaUrl" class="max-w-full max-h-full object-contain" />
      <video v-else-if="msg.mediaType === 'video' && msg.mediaUrl" :src="msg.mediaUrl" controls autoplay class="max-w-full max-h-full" />
    </div>
  </div>
</template>
