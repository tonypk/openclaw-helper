<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineEmits<{
  action: [name: string]
}>()

const actions = [
  { id: 'console', icon: '🌐', labelKey: 'dashboard.openConsole' },
  { id: 'logs', icon: '📋', labelKey: 'dashboard.viewLogs' },
  { id: 'update', icon: '🔄', labelKey: 'dashboard.checkUpdate' },
  { id: 'reinstall', icon: '🔧', labelKey: 'dashboard.reinstall' },
]
</script>

<template>
  <div class="quick-actions">
    <button
      v-for="a in actions"
      :key="a.id"
      class="quick-actions__btn"
      @click="$emit('action', a.id)"
    >
      <span class="quick-actions__icon">{{ a.icon }}</span>
      <span>{{ t(a.labelKey) }}</span>
    </button>
  </div>
</template>

<style scoped>
.quick-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
.quick-actions__btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-radius: 10px;
  border: 1px solid var(--color-border);
  background: var(--color-surface);
  cursor: pointer;
  font-size: 14px;
  transition: background 0.2s;
}
.quick-actions__btn:hover {
  background: var(--color-primary-light);
}
.quick-actions__icon {
  font-size: 18px;
}
</style>
