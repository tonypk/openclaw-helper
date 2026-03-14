<script setup lang="ts">
import { computed } from 'vue'
import type { Message } from '../../stores/chat'

const props = defineProps<{
  message: Message
}>()

const isUser = computed(() => props.message.role === 'user')
</script>

<template>
  <div class="chat-message" :class="{ 'chat-message--user': isUser }">
    <div class="chat-message__bubble">
      <div class="chat-message__content" v-html="message.content.replace(/\n/g, '<br>')"></div>
      <div v-if="message.repairId" class="chat-message__repair">
        <button class="btn btn--sm btn--primary" @click="$emit('repair', message.repairId)">
          {{ message.autoRepair ? '🔧 一键修复' : '📖 查看方案' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.chat-message {
  display: flex;
  margin-bottom: 12px;
}
.chat-message--user {
  justify-content: flex-end;
}
.chat-message__bubble {
  max-width: 85%;
  padding: 10px 14px;
  border-radius: 12px;
  background: var(--color-surface);
  color: var(--color-text);
  font-size: 14px;
  line-height: 1.5;
}
.chat-message--user .chat-message__bubble {
  background: var(--color-primary);
  color: #fff;
}
.chat-message__repair {
  margin-top: 8px;
}
</style>
