<script setup lang="ts">
import { useHealingStore } from '../../stores/healing'
import { useI18n } from 'vue-i18n'

const healing = useHealingStore()
const { t } = useI18n()

const statusIcon = (status: string) => {
  switch (status) {
    case 'success': return '\u2713'
    case 'running': return '\u25CF'
    case 'failed': return '\u2717'
    default: return '\u25CB'
  }
}

const statusColor = (status: string) => {
  switch (status) {
    case 'success': return '#4ade80'
    case 'running': return '#60a5fa'
    case 'failed': return '#f87171'
    default: return '#475569'
  }
}
</script>

<template>
  <div v-if="healing.isHealing" class="healing-panel">
    <div class="healing-header">
      <span class="healing-icon">&#x1F527;</span>
      <span class="healing-title">{{ t('healing.autoRepairing') }}</span>
    </div>

    <div class="healing-issue">
      {{ healing.currentIssue }}
    </div>

    <div class="healing-strategies">
      <div
        v-for="s in healing.strategies"
        :key="s.name"
        class="strategy-item"
      >
        <span class="strategy-icon" :style="{ color: statusColor(s.status) }">
          {{ statusIcon(s.status) }}
        </span>
        <span class="strategy-name" :class="{ active: s.status === 'running' }">
          {{ s.name }}
        </span>
      </div>
    </div>

    <div v-if="healing.repairLog.length > 0" class="repair-log">
      <div v-for="(line, i) in healing.repairLog.slice(-5)" :key="i" class="log-line">
        &gt; {{ line }}
      </div>
    </div>
  </div>

  <div v-if="healing.healedCount > 0 && !healing.isHealing && !healing.escalated" class="healing-badge">
    <span class="badge-text">
      {{ t('healing.resolved', { n: healing.healedCount }) }}
    </span>
    <div class="badge-details">
      <div v-for="h in healing.history" :key="h.issue" class="badge-detail-item">
        {{ h.issue }} &rarr; {{ h.resolvedBy }}
      </div>
    </div>
  </div>

  <div v-if="healing.escalated" class="healing-escalated">
    {{ t('healing.escalateAI') }}
  </div>
</template>

<style scoped>
.healing-panel {
  background: #1a2332;
  border: 1px solid #2a3a4a;
  border-radius: 8px;
  padding: 12px;
  margin: 8px 0;
  font-size: 13px;
}

.healing-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.healing-icon { font-size: 16px; }
.healing-title { color: #fbbf24; font-weight: 600; }
.healing-issue { color: #64748b; font-size: 12px; margin-bottom: 8px; }

.healing-strategies {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.strategy-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.strategy-icon { font-size: 12px; width: 14px; text-align: center; }
.strategy-name { color: #94a3b8; font-size: 12px; }
.strategy-name.active { color: #60a5fa; }

.repair-log {
  background: #0f172a;
  border-radius: 4px;
  padding: 8px;
  margin-top: 8px;
  font-family: monospace;
  font-size: 10px;
  color: #94a3b8;
  max-height: 80px;
  overflow-y: auto;
}

.log-line { line-height: 1.6; }

.healing-badge {
  background: #1a2332;
  color: #f59e0b;
  font-size: 11px;
  padding: 4px 8px;
  border-radius: 4px;
  display: inline-block;
  margin-left: 8px;
}

.badge-details {
  color: #4ade80;
  font-size: 10px;
  margin-top: 4px;
}

.badge-detail-item { line-height: 1.6; }

.healing-escalated {
  color: #f87171;
  font-size: 12px;
  margin-top: 4px;
}
</style>
