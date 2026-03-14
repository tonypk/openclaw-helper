<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useChatStore } from '../../stores/chat'
import ChatPanel from './ChatPanel.vue'

const { t } = useI18n()
const chat = useChatStore()
</script>

<template>
  <div class="chat-bubble-wrapper">
    <Transition name="slide-up">
      <ChatPanel v-if="chat.expanded" />
    </Transition>
    <button class="chat-bubble" @click="chat.toggle()">
      <span v-if="!chat.expanded">💬 {{ t('welcome.chatHint') }}</span>
      <span v-else>✕</span>
    </button>
  </div>
</template>

<style scoped>
.chat-bubble-wrapper {
  position: fixed;
  bottom: 24px;
  right: 24px;
  z-index: 1000;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 12px;
}
.chat-bubble {
  padding: 12px 20px;
  border-radius: 24px;
  background: var(--color-primary);
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  border: none;
  cursor: pointer;
  box-shadow: 0 4px 16px rgba(37, 99, 235, 0.3);
  transition: transform 0.2s;
}
.chat-bubble:hover {
  transform: scale(1.05);
}
.slide-up-enter-active,
.slide-up-leave-active {
  transition: all 0.3s ease;
}
.slide-up-enter-from,
.slide-up-leave-to {
  opacity: 0;
  transform: translateY(20px);
}
</style>
