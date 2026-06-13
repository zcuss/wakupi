<script setup lang="ts">
import { ref, onUnmounted } from 'vue'
import { Mic, Square, Send, X } from '@lucide/vue'
import { useChatStore } from '../stores/chat'

const store = useChatStore()
const recording = ref(false)
const elapsed = ref(0)
let timer: any = null
let recorder: MediaRecorder | null = null
let stream: MediaStream | null = null
const chunks: Blob[] = []
const previewBlob = ref<Blob | null>(null)
const previewUrl = ref<string>('')

async function start() {
  try {
    stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    const mime = MediaRecorder.isTypeSupported('audio/ogg;codecs=opus')
      ? 'audio/ogg;codecs=opus'
      : MediaRecorder.isTypeSupported('audio/webm;codecs=opus')
        ? 'audio/webm;codecs=opus'
        : ''
    recorder = new MediaRecorder(stream, mime ? { mimeType: mime } : undefined)
    chunks.length = 0
    recorder.ondataavailable = (e) => {
      if (e.data.size > 0) chunks.push(e.data)
    }
    recorder.onstop = () => {
      const blob = new Blob(chunks, { type: chunks[0]?.type || 'audio/ogg' })
      previewBlob.value = blob
      previewUrl.value = URL.createObjectURL(blob)
      stream?.getTracks().forEach((t) => t.stop())
      stream = null
    }
    recorder.start()
    recording.value = true
    elapsed.value = 0
    timer = setInterval(() => elapsed.value++, 1000)
  } catch (e) {
    console.error('mic error', e)
    alert('Tidak bisa akses mikrofon')
  }
}

function stop() {
  if (timer) clearInterval(timer)
  timer = null
  recording.value = false
  if (recorder && recorder.state !== 'inactive') recorder.stop()
}

function cancel() {
  if (timer) clearInterval(timer)
  timer = null
  recording.value = false
  if (recorder && recorder.state !== 'inactive') {
    recorder.onstop = null
    recorder.stop()
  }
  stream?.getTracks().forEach((t) => t.stop())
  stream = null
  previewBlob.value = null
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
  previewUrl.value = ''
  emit('done')
}

async function send() {
  if (!previewBlob.value) return
  const reader = new FileReader()
  reader.onload = async () => {
    const b64 = (reader.result as string).split(',')[1]
    await store.sendVoiceBlob(b64)
    if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
    previewUrl.value = ''
    previewBlob.value = null
    emit('done')
  }
  reader.readAsDataURL(previewBlob.value)
}

const emit = defineEmits<{ (e: 'done'): void }>()

onUnmounted(() => {
  if (timer) clearInterval(timer)
  stream?.getTracks().forEach((t) => t.stop())
})

function fmt(sec: number) {
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return `${m}:${s.toString().padStart(2, '0')}`
}
</script>

<template>
  <div class="flex items-center gap-2 flex-1 px-4">
    <template v-if="!recording && !previewBlob">
      <button @click="start" class="w-10 h-10 rounded-full bg-red-500 text-white flex items-center justify-center">
        <Mic :size="20" />
      </button>
      <span class="text-sm text-wa-muted dark:text-wa-muted-dark">Tekan untuk mulai merekam</span>
      <button @click="emit('done')" class="ml-auto text-wa-muted dark:text-wa-muted-dark hover:text-red-500">
        <X :size="20" />
      </button>
    </template>

    <template v-else-if="recording">
      <button @click="cancel" class="w-9 h-9 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center text-red-500">
        <X :size="20" />
      </button>
      <span class="w-2 h-2 rounded-full bg-red-500 animate-pulse" />
      <span class="text-sm font-medium tabular-nums">{{ fmt(elapsed) }}</span>
      <span class="flex-1 text-sm text-wa-muted dark:text-wa-muted-dark">Merekam...</span>
      <button @click="stop" class="w-10 h-10 rounded-full bg-wa-green text-white flex items-center justify-center">
        <Square :size="18" />
      </button>
    </template>

    <template v-else-if="previewBlob">
      <button @click="cancel" class="w-9 h-9 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center text-red-500">
        <X :size="20" />
      </button>
      <audio :src="previewUrl" controls class="flex-1 h-10" />
      <button @click="send" class="w-10 h-10 rounded-full bg-wa-green text-white flex items-center justify-center">
        <Send :size="18" />
      </button>
    </template>
  </div>
</template>
