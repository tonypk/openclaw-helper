<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ReportDialog from './ReportDialog.vue'

defineProps<{
  message: string
  phase?: string
}>()

defineEmits<{
  retry: []
}>()

const { t } = useI18n()
const showReport = ref(false)
</script>

<template>
  <div class="error-card">
    <div class="error-card__icon">⚠️</div>
    <div class="error-card__body">
      <div class="error-card__title">{{ phase || 'Error' }}</div>
      <div class="error-card__msg">{{ message }}</div>
    </div>
    <div class="error-card__actions">
      <button class="btn btn--secondary" @click="showReport = true">
        📋 {{ t('report.title') }}
      </button>
      <button class="btn btn--primary" @click="$emit('retry')">
        🔄 {{ t('common.retry') }}
      </button>
    </div>
  </div>

  <ReportDialog
    v-if="showReport"
    :error-phase="phase"
    :error-message="message"
    @close="showReport = false"
  />
</template>

<style scoped>
.error-card {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 12px;
}
:root.dark .error-card {
  background: #451a1a;
  border-color: #7f1d1d;
}
.error-card__icon {
  font-size: 24px;
}
.error-card__body {
  flex: 1;
}
.error-card__title {
  font-weight: 600;
  margin-bottom: 4px;
}
.error-card__msg {
  font-size: 14px;
  color: var(--color-error);
}
.error-card__actions {
  display: flex;
  gap: 8px;
}
</style>
