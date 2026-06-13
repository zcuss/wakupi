<script setup lang="ts">
import { ref } from 'vue'
import { X, Send } from '@lucide/vue'
import { useStatusStore } from '../stores/status'
import { useChatStore } from '../stores/chat'

const status = useStatusStore()
const chat = useChatStore()
const text = ref('')
const sending = ref(false)

const colors = ['#00a884', '#1f2937', '#7c3aed', '#dc2626', '#f59e0b', '#0284c7', '#be185d']
const bgIdx = ref(0)

function close() {
  status.showComposer = false
  text.value = ''
}

async function send() {
  if (!text.value.trim() || !chat.activeAccountId) return
  sending.value = true
  try {
    await status.postText(chat.activeAccountId, text.value)
    close()
  } catch (e) {
    console.error('post status failed', e)
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <div class="fixed inset-0 z-50 bg-black/95 flex flex-col">
    <header class="flex items-center justify-between px-6 py-4 text-white">
      <button @click="close" class="w-10 h-10 rounded-full hover:bg-white/10 flex items-center justify-center">
        <X :size="22" />
      </button>
      <div class="flex gap-2">
        <button
          v-for="(c, i) in colors"
          :key="c"
          @click="bgIdx = i"
          class="w-8 h-8 rounded-full border-2"
          :class="bgIdx === i ? 'border-white' : 'border-transparent'"
          :style="{ backgroundColor: c }"
        />
      </div>
      <button
        @click="send"
        :disabled="!text.trim() || sending"
        class="w-12 h-12 rounded-full bg-wa-green text-white flex items-center justify-center disabled:opacity-50"
      >
        <Send :size="22" />
      </button>
    </header>

    <div class="flex-1 flex items-center justify-center p-12" :style="{ backgroundColor: colors[bgIdx] }">
      <textarea
        v-model="text"
        placeholder="Ketik status kamu..."
        autofocus
        class="w-full max-w-2xl bg-transparent text-white text-4xl font-medium text-center outline-none resize-none placeholder:text-white/40"
        rows="6"
      />
    </div>
  </div>
</template>
