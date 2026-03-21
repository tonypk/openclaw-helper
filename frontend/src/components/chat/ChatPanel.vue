<script setup lang="ts">
import { ref, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useChatStore } from '../../stores/chat'
import { diagnosisRepair } from '../../api/helper'
import ChatMessage from './ChatMessage.vue'

const { t, locale } = useI18n()
const chat = useChatStore()
const input = ref('')
const messagesEl = ref<HTMLElement>()

async function send() {
  const text = input.value.trim()
  if (!text) return
  input.value = ''
  await chat.ask(text)
  scrollToBottom()
}

function askSuggestion(text: string) {
  input.value = ''
  chat.ask(text)
  scrollToBottom()
}

async function handleRepair(repairId: string) {
  try {
    const result = await diagnosisRepair(repairId)
    const msg = locale.value === 'zh-CN' && result.msg_zh ? result.msg_zh : result.message
    chat.messages.push({
      role: 'assistant',
      content: result.success ? `✅ ${msg}` : `❌ ${msg}`,
      source: 'diagnosis',
      timestamp: new Date(),
    })
  } catch (e) {
    chat.messages.push({
      role: 'assistant',
      content: `❌ ${e instanceof Error ? e.message : 'Repair failed'}`,
      source: 'offline',
      timestamp: new Date(),
    })
  }
}

function scrollToBottom() {
  nextTick(() => {
    if (messagesEl.value) {
      messagesEl.value.scrollTop = messagesEl.value.scrollHeight
    }
  })
}

watch(() => chat.messages.length, scrollToBottom)
</script>

<template>
  <div class="chat-panel">
    <div class="chat-panel__header">
      <span>💬 {{ t('app.title') }}</span>
      <button class="btn btn--icon" @click="chat.toggle()">✕</button>
    </div>

    <div v-if="chat.backendOnline === false" class="chat-panel__offline">
      {{ t('chat.offline') }}
    </div>

    <div ref="messagesEl" class="chat-panel__messages">
      <ChatMessage
        v-for="(msg, i) in chat.messages"
        :key="i"
        :message="msg"
        @repair="handleRepair"
      />
      <div v-if="chat.loading" class="chat-panel__typing">
        {{ t('chat.thinking') }}
      </div>
    </div>

    <div v-if="chat.suggestions.length" class="chat-panel__suggestions">
      <button
        v-for="s in chat.suggestions"
        :key="s.text"
        class="chip"
        @click="askSuggestion(locale === 'zh-CN' ? s.text : s.text_en)"
      >
        {{ locale === 'zh-CN' ? s.text : s.text_en }}
      </button>
    </div>

    <div class="chat-panel__input">
      <input
        v-model="input"
        :placeholder="t('chat.placeholder')"
        @keyup.enter="send"
      />
      <button class="btn btn--primary btn--sm" @click="send" :disabled="!input.trim()">
        {{ t('chat.send') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.chat-panel {
  display: flex;
  flex-direction: column;
  width: 360px;
  height: 480px;
  border-radius: 16px;
  background: var(--color-bg);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
  overflow: hidden;
}
.chat-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: var(--color-primary);
  color: #fff;
  font-weight: 600;
}
.chat-panel__messages {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}
.chat-panel__typing {
  color: var(--color-text-secondary);
  font-size: 13px;
  padding: 4px 0;
}
.chat-panel__offline {
  padding: 8px 12px;
  background: #fef3c7;
  color: #92400e;
  font-size: 12px;
  text-align: center;
}
:root.dark .chat-panel__offline {
  background: #78350f;
  color: #fbbf24;
}
.chat-panel__suggestions {
  display: flex;
  gap: 6px;
  padding: 8px 12px;
  flex-wrap: wrap;
}
.chip {
  padding: 4px 10px;
  border-radius: 12px;
  border: 1px solid var(--color-border);
  background: var(--color-surface);
  font-size: 12px;
  cursor: pointer;
}
.chip:hover {
  background: var(--color-primary-light);
}
.chat-panel__input {
  display: flex;
  gap: 8px;
  padding: 10px 12px;
  border-top: 1px solid var(--color-border);
}
.chat-panel__input input {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  outline: none;
  font-size: 14px;
}
</style>
