import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ChatResponse } from '../api/helper'
import { chatAsk, chatSetContext, chatSuggestions, helperPing } from '../api/helper'

export interface Message {
  role: 'user' | 'assistant'
  content: string
  source?: string
  repairId?: string
  autoRepair?: boolean
  timestamp: Date
}

export const useChatStore = defineStore('chat', () => {
  const messages = ref<Message[]>([])
  const suggestions = ref<{ text: string; text_en: string }[]>([])
  const loading = ref(false)
  const expanded = ref(false)
  const backendOnline = ref<boolean | null>(null) // null = not checked yet

  async function ask(text: string) {
    messages.value.push({
      role: 'user',
      content: text,
      timestamp: new Date(),
    })

    loading.value = true
    try {
      const resp: ChatResponse = await chatAsk(text)
      messages.value.push({
        role: 'assistant',
        content: resp.message,
        source: resp.source,
        repairId: resp.repair_id,
        autoRepair: resp.auto_repair,
        timestamp: new Date(),
      })
      if (resp.suggested) {
        suggestions.value = resp.suggested
      }
    } catch (e) {
      messages.value.push({
        role: 'assistant',
        content: e instanceof Error ? e.message : 'Connection error',
        source: 'offline',
        timestamp: new Date(),
      })
    } finally {
      loading.value = false
    }
  }

  async function setContext(ctx: {
    phase?: string
    error_log?: string
    language?: string
  }) {
    try {
      await chatSetContext(ctx)
    } catch {
      // ignore
    }
  }

  async function fetchSuggestions() {
    try {
      suggestions.value = await chatSuggestions()
    } catch {
      // ignore
    }
  }

  function toggle() {
    expanded.value = !expanded.value
    if (expanded.value && backendOnline.value === null) {
      checkBackend()
    }
  }

  async function checkBackend() {
    try {
      await helperPing()
      backendOnline.value = true
    } catch {
      backendOnline.value = false
    }
  }

  return {
    messages,
    suggestions,
    loading,
    expanded,
    backendOnline,
    ask,
    setContext,
    fetchSuggestions,
    toggle,
    checkBackend,
  }
})
