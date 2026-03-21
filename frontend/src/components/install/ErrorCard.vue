<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ReportDialog from './ReportDialog.vue'

defineProps<{
  message: string
  phase?: string
  guidance?: string
}>()

defineEmits<{
  retry: []
}>()

const { t } = useI18n()
const showReport = ref(false)
</script>

<template>
  <div class="error-card">
    <div class="error-card__header">
      <span class="error-card__icon">&#x26A0;&#xFE0F;</span>
      <span class="error-card__title">{{ phase || 'Error' }}</span>
    </div>

    <div class="error-card__msg">{{ message }}</div>

    <div v-if="guidance" class="error-card__guidance">
      <span class="guidance-icon">&#x1F4DD;</span>
      <span>{{ guidance }}</span>
    </div>

    <div class="error-card__actions">
      <button class="btn btn--secondary btn--sm" @click="showReport = true">
        &#x1F4CB; {{ t('report.title') }}
      </button>
      <button class="btn btn--primary btn--sm" @click="$emit('retry')">
        &#x1F504; {{ t('common.retry') }}
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
  padding: 16px;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 12px;
}
:root.dark .error-card {
  background: #451a1a;
  border-color: #7f1d1d;
}
.error-card__header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.error-card__icon {
  font-size: 20px;
}
.error-card__title {
  font-weight: 600;
  font-size: 15px;
}
.error-card__msg {
  font-size: 14px;
  color: var(--color-error);
  margin-bottom: 12px;
  line-height: 1.5;
}
.error-card__guidance {
  display: flex;
  gap: 8px;
  padding: 12px;
  background: #fff7ed;
  border-radius: 8px;
  font-size: 13px;
  color: var(--color-text-secondary);
  line-height: 1.6;
  margin-bottom: 12px;
}
:root.dark .error-card__guidance {
  background: #422006;
}
.guidance-icon {
  flex-shrink: 0;
  font-size: 14px;
}
.error-card__actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
</style>
