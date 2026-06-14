import { defineStore } from 'pinia'
import { ref } from 'vue'
import { GetAIConfig, SetAIConfig, AISuggestReplies, AISummarize, AICompose, AITestConnection, AIListModels } from '../../wailsjs/go/main/App'

export interface AIConfig {
  provider: 'openai' | 'anthropic' | 'gemini' | 'ollama' | 'hermes' | ''
  apiKey: string
  baseUrl: string
  model: string
  enabled: boolean
}

export type ConnStatus = 'off' | 'unknown' | 'ok' | 'error'

// Hermes is the default AI provider. hermes.zcuss.xyz is the public
// Hermes Agent (Nous Research's gateway), which handles auth and routes to
// whatever upstream model is configured there. No API key required, so a
// fresh install has a working AI Assistant out of the box.
//
// Wakupi can also be registered as a Hermes Agent platform (in addition to
// Telegram / WhatsApp / Discord) via the wakupi platform plugin, but the
// AI Assistant panel itself talks directly to the public Hermes endpoint.
const defaults: AIConfig = {
  provider: 'hermes',
  apiKey: '',
  baseUrl: 'https://hermes.zcuss.xyz/v1/chat/completions',
  model: 'hermes',
  enabled: true,
}

export const useAIStore = defineStore('ai', () => {
  const config = ref<AIConfig>({ ...defaults })
  const loaded = ref(false)
  const suggestions = ref<string[]>([])
  const suggesting = ref(false)
  const composing = ref(false)
  const connStatus = ref<ConnStatus>('off')
  const connMessage = ref('')
  const testing = ref(false)

  async function load() {
    try {
      const cfg = (await GetAIConfig()) as any
      config.value = { ...defaults, ...cfg }
    } catch (e) {
      console.error('load AI config', e)
    }
    loaded.value = true
    connStatus.value = config.value.enabled ? 'unknown' : 'off'
  }

  async function save(cfg: AIConfig) {
    await SetAIConfig(cfg as any)
    config.value = { ...cfg }
    if (!cfg.enabled) connStatus.value = 'off'
  }

  async function testConnection(cfg: AIConfig): Promise<boolean> {
    testing.value = true
    connMessage.value = ''
    try {
      await AITestConnection(cfg as any)
      connStatus.value = cfg.enabled ? 'ok' : 'off'
      connMessage.value = 'Terhubung'
      return true
    } catch (e: any) {
      connStatus.value = 'error'
      connMessage.value = e?.message || String(e)
      return false
    } finally {
      testing.value = false
    }
  }

  async function listModels(cfg: AIConfig): Promise<string[]> {
    return ((await AIListModels(cfg as any)) as string[]) || []
  }

  async function suggest(contactName: string, lastMessages: string) {
    if (!config.value.enabled) {
      suggestions.value = []
      return
    }
    suggesting.value = true
    try {
      const res = (await AISuggestReplies(contactName, lastMessages)) || []
      suggestions.value = res
    } catch (e) {
      console.error(e)
      suggestions.value = []
    } finally {
      suggesting.value = false
    }
  }

  function clearSuggestions() {
    suggestions.value = []
  }

  async function summarize(text: string) {
    return (await AISummarize(text)) || ''
  }

  async function compose(prompt: string, tone: string = 'friendly') {
    composing.value = true
    try {
      return (await AICompose(prompt, tone)) || ''
    } finally {
      composing.value = false
    }
  }

  return {
    config,
    loaded,
    suggestions,
    suggesting,
    composing,
    connStatus,
    connMessage,
    testing,
    load,
    save,
    testConnection,
    listModels,
    suggest,
    clearSuggestions,
    summarize,
    compose,
  }
})
