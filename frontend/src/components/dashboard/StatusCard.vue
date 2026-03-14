<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  running: boolean
  uptime?: string
  gateway?: string
  memory?: string
  cpu?: string
}>()

defineEmits<{
  toggle: []
}>()

const { t } = useI18n()
</script>

<template>
  <div class="status-card">
    <div class="status-card__header">
      <span class="status-card__dot" :class="{ 'status-card__dot--on': running }" />
      <span class="status-card__label">
        OpenClaw {{ running ? t('dashboard.running') : t('dashboard.stopped') }}
      </span>
    </div>
    <div class="status-card__info">
      <div v-if="uptime">{{ t('dashboard.uptime') }}: {{ uptime }}</div>
      <div v-if="gateway">Gateway: {{ gateway }}</div>
      <div v-if="memory || cpu">
        <span v-if="memory">Memory: {{ memory }}</span>
        <span v-if="cpu"> | CPU: {{ cpu }}</span>
      </div>
    </div>
    <button class="btn" :class="running ? 'btn--danger' : 'btn--primary'" @click="$emit('toggle')">
      {{ running ? `⏹ ${t('dashboard.stop')}` : `▶ ${t('dashboard.start')}` }}
    </button>
  </div>
</template>

<style scoped>
.status-card {
  padding: 20px;
  border-radius: 12px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
}
.status-card__header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 16px;
  font-weight: 600;
}
.status-card__dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #ef4444;
}
.status-card__dot--on {
  background: #10b981;
}
.status-card__info {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-bottom: 12px;
  line-height: 1.6;
}
</style>
