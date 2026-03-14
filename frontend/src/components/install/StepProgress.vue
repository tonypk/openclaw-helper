<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { PhaseProgress } from '../../api/helper'

defineProps<{
  phases: PhaseProgress[]
  currentPhase: string
}>()

const { locale } = useI18n()

function phaseLabel(p: PhaseProgress): string {
  return locale.value === 'zh-CN' ? p.label_zh : p.label
}

function statusIcon(status: string): string {
  switch (status) {
    case 'completed': return '✅'
    case 'running': return '⏳'
    case 'failed': return '❌'
    case 'skipped': return '⏭️'
    default: return '⬜'
  }
}
</script>

<template>
  <div class="step-progress">
    <div
      v-for="(phase, i) in phases"
      :key="phase.phase"
      class="step"
      :class="{
        'step--active': phase.phase === currentPhase,
        'step--done': phase.status === 'completed',
        'step--failed': phase.status === 'failed',
      }"
    >
      <div class="step__icon">{{ statusIcon(phase.status) }}</div>
      <div class="step__label">{{ phaseLabel(phase) }}</div>
      <div v-if="i < phases.length - 1" class="step__line" />
    </div>
  </div>
</template>

<style scoped>
.step-progress {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 16px 0;
}
.step {
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  flex: 1;
}
.step__icon {
  font-size: 20px;
  margin-bottom: 6px;
}
.step__label {
  font-size: 12px;
  color: var(--color-text-secondary);
  text-align: center;
}
.step--active .step__label {
  color: var(--color-primary);
  font-weight: 600;
}
.step__line {
  position: absolute;
  top: 12px;
  left: 55%;
  width: 90%;
  height: 2px;
  background: var(--color-border);
}
.step--done .step__line {
  background: var(--color-success);
}
</style>
