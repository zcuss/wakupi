<script setup lang="ts">
import { ref, watch } from 'vue'
import { X, ArrowLeft, UserMinus, UserPlus, Crown, LogOut, Edit3 } from '@lucide/vue'
import { GetGroupInfo, LeaveGroup, UpdateGroupParticipants, SetGroupName } from '../../wailsjs/go/main/App'
import { useChatStore } from '../stores/chat'
import { useUIStore } from '../stores/ui'

const chat = useChatStore()
const ui = useUIStore()
const info = ref<any>(null)
const loading = ref(false)
const editingName = ref(false)
const newName = ref('')

watch(() => ui.showGroupInfo, async (open) => {
  if (!open || !chat.activeChat || !chat.activeChat.isGroup) return
  loading.value = true
  try {
    info.value = await GetGroupInfo(chat.activeChat.accountId, chat.activeChat.jid)
    newName.value = info.value?.name || ''
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
})

async function leave() {
  if (!chat.activeChat) return
  if (!confirm('Keluar dari grup ini?')) return
  try { await LeaveGroup(chat.activeChat.accountId, chat.activeChat.jid) } catch (e) { console.error(e) }
  ui.showGroupInfo = false
}

async function saveName() {
  if (!chat.activeChat || !newName.value.trim()) return
  try {
    await SetGroupName(chat.activeChat.accountId, chat.activeChat.jid, newName.value.trim())
    info.value.name = newName.value.trim()
    editingName.value = false
  } catch (e) { console.error(e) }
}

async function removeMember(jid: string) {
  if (!chat.activeChat) return
  if (!confirm('Hapus anggota ini?')) return
  try {
    await UpdateGroupParticipants(chat.activeChat.accountId, chat.activeChat.jid, [jid], 'remove')
    info.value.participants = info.value.participants.filter((p: any) => p.jid !== jid)
  } catch (e) { console.error(e) }
}

async function promote(jid: string) {
  if (!chat.activeChat) return
  try {
    await UpdateGroupParticipants(chat.activeChat.accountId, chat.activeChat.jid, [jid], 'promote')
    const p = info.value.participants.find((x: any) => x.jid === jid)
    if (p) p.isAdmin = true
  } catch (e) { console.error(e) }
}

const initials = (n: string) => (n || '').split(' ').map((s: string) => s[0]).slice(0, 2).join('').toUpperCase()
</script>

<template>
  <div v-if="ui.showGroupInfo && chat.activeChat?.isGroup" class="fixed inset-0 z-40 bg-black/40 flex items-center justify-center" @click.self="ui.showGroupInfo = false">
    <div class="w-[480px] max-w-[92vw] max-h-[80vh] bg-white dark:bg-wa-panel-dark rounded-2xl shadow-2xl flex flex-col overflow-hidden">
      <header class="flex items-center justify-between px-5 py-3 border-b border-wa-border dark:border-wa-border-dark">
        <div class="flex items-center gap-2">
          <button @click="ui.showGroupInfo = false" class="text-wa-muted dark:text-wa-muted-dark"><ArrowLeft :size="18" /></button>
          <h2 class="font-semibold">Info Grup</h2>
        </div>
        <button @click="ui.showGroupInfo = false" class="text-wa-muted dark:text-wa-muted-dark"><X :size="18" /></button>
      </header>

      <div v-if="loading" class="p-8 text-center text-sm text-wa-muted dark:text-wa-muted-dark">Memuat...</div>

      <div v-else-if="info" class="flex-1 overflow-y-auto scrollbar-thin">
        <div class="p-6 text-center">
          <div class="w-24 h-24 mx-auto rounded-full bg-slate-400 text-white flex items-center justify-center text-2xl font-semibold overflow-hidden">
            <img v-if="chat.activeChat?.avatarUrl" :src="chat.activeChat.avatarUrl" class="w-full h-full object-cover" />
            <span v-else>{{ initials(info.name) }}</span>
          </div>
          <div class="mt-3 flex items-center justify-center gap-2">
            <input
              v-if="editingName"
              v-model="newName"
              @keydown.enter="saveName"
              class="text-lg font-semibold text-center bg-wa-panel dark:bg-wa-hover-dark rounded px-2 outline-none"
            />
            <h3 v-else class="text-lg font-semibold">{{ info.name }}</h3>
            <button @click="editingName = !editingName" class="text-wa-muted dark:text-wa-muted-dark"><Edit3 :size="14" /></button>
          </div>
          <p v-if="info.topic" class="text-sm text-wa-muted dark:text-wa-muted-dark mt-1">{{ info.topic }}</p>
          <p class="text-xs text-wa-muted dark:text-wa-muted-dark mt-2">{{ info.participants?.length || 0 }} anggota</p>
        </div>

        <div class="px-5 py-2 text-xs font-medium text-wa-muted dark:text-wa-muted-dark uppercase">Anggota</div>
        <ul>
          <li v-for="p in info.participants" :key="p.jid" class="flex items-center gap-3 px-5 py-2.5 hover:bg-wa-hover dark:hover:bg-wa-hover-dark">
            <div class="w-9 h-9 rounded-full bg-slate-400 text-white flex items-center justify-center text-sm font-semibold">
              {{ initials(p.pushName || p.jid.split('@')[0]) }}
            </div>
            <div class="flex-1 min-w-0">
              <div class="text-sm font-medium truncate">{{ p.pushName || p.jid.split('@')[0] }}</div>
              <div class="text-xs text-wa-muted dark:text-wa-muted-dark truncate">{{ p.jid.split('@')[0] }}</div>
            </div>
            <span v-if="p.isSuperAdmin" class="text-xs px-2 py-0.5 rounded-full bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300 flex items-center gap-1">
              <Crown :size="12" /> Pembuat
            </span>
            <span v-else-if="p.isAdmin" class="text-xs px-2 py-0.5 rounded-full bg-wa-green/20 text-wa-green">Admin</span>
            <button v-else @click="promote(p.jid)" title="Jadikan admin" class="text-wa-muted dark:text-wa-muted-dark hover:text-wa-green">
              <Crown :size="14" />
            </button>
            <button @click="removeMember(p.jid)" title="Hapus" class="text-red-500 hover:text-red-600">
              <UserMinus :size="14" />
            </button>
          </li>
        </ul>

        <div class="p-5">
          <button @click="leave" class="w-full flex items-center justify-center gap-2 bg-red-500 hover:bg-red-600 text-white rounded-lg py-2 text-sm">
            <LogOut :size="16" /> Keluar dari grup
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
