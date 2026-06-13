<script setup lang="ts">
import { ref } from 'vue'
import { X, Camera, Save } from '@lucide/vue'
import { SetSelfStatus, SetSelfProfilePicture, PickFile } from '../../wailsjs/go/main/App'
import { useChatStore } from '../stores/chat'
import { useUIStore } from '../stores/ui'

const chat = useChatStore()
const ui = useUIStore()
const status = ref('')
const saving = ref(false)
const message = ref('')

async function saveStatus() {
  if (!chat.activeAccountId) return
  saving.value = true
  try {
    await SetSelfStatus(chat.activeAccountId, status.value)
    message.value = 'Status diperbarui'
    setTimeout(() => (message.value = ''), 2000)
  } catch (e: any) {
    message.value = 'Gagal: ' + (e?.message || e)
  } finally {
    saving.value = false
  }
}

async function changePhoto() {
  if (!chat.activeAccountId) return
  let path = ''
  try { path = await PickFile('image') } catch { return }
  if (!path) return
  saving.value = true
  try {
    await SetSelfProfilePicture(chat.activeAccountId, path)
    message.value = 'Foto profil diperbarui'
    setTimeout(() => (message.value = ''), 2000)
  } catch (e: any) {
    message.value = 'Gagal: ' + (e?.message || e)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div v-if="ui.showProfile" class="fixed inset-0 z-40 bg-black/40 flex items-center justify-center" @click.self="ui.showProfile = false">
    <div class="w-[440px] max-w-[92vw] bg-white dark:bg-wa-panel-dark rounded-2xl shadow-2xl overflow-hidden">
      <header class="flex items-center justify-between px-5 py-3 border-b border-wa-border dark:border-wa-border-dark">
        <h2 class="font-semibold">Profil Saya</h2>
        <button @click="ui.showProfile = false" class="text-wa-muted dark:text-wa-muted-dark"><X :size="18" /></button>
      </header>

      <div class="p-6 space-y-5">
        <div class="text-center">
          <div class="relative w-28 h-28 mx-auto rounded-full bg-wa-green text-white flex items-center justify-center text-2xl font-semibold overflow-hidden">
            {{ (chat.activeAccount?.name || '?')[0]?.toUpperCase() }}
            <button @click="changePhoto" class="absolute bottom-0 right-0 w-9 h-9 rounded-full bg-wa-green-dark text-white flex items-center justify-center border-2 border-white">
              <Camera :size="16" />
            </button>
          </div>
          <div class="mt-3">
            <div class="font-medium">{{ chat.activeAccount?.name }}</div>
            <div class="text-sm text-wa-muted dark:text-wa-muted-dark">{{ chat.activeAccount?.phone }}</div>
          </div>
        </div>

        <div>
          <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark">Tentang / Status</label>
          <div class="mt-1 flex gap-2">
            <input
              v-model="status"
              placeholder="Hai, saya menggunakan WhatsApp"
              class="flex-1 bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none"
            />
            <button @click="saveStatus" :disabled="saving" class="px-3 py-2 rounded-lg bg-wa-green text-white text-sm flex items-center gap-1 disabled:opacity-50">
              <Save :size="14" /> Simpan
            </button>
          </div>
        </div>

        <div v-if="message" class="text-center text-sm text-wa-green">{{ message }}</div>
      </div>
    </div>
  </div>
</template>
