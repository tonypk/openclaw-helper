<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSystemStore } from '../stores/system'

const router = useRouter()
const { t, te } = useI18n()
const system = useSystemStore()

const checkNames: Record<string, string> = {
  os: 'check.os',
  memory: 'check.memory',
  disk: 'check.disk',
  virtualization: 'check.virtualization',
  wsl: 'check.wsl',
  node: 'check.node',
  openclaw: 'check.openclaw',
}

const statusIcon: Record<string, string> = {
  pass: '\u2705',
  fail: '\u274C',
  warn: '\u26A0\uFE0F',
  checking: '\u23F3',
  skipped: '\u23ED\uFE0F',
}

const allPassed = computed(() => system.report?.overall_ready ?? false)

function helpText(name: string): string | null {
  const key = `check.checkHelp.${name}`
  return te(key) ? t(key) : null
}

onMounted(() => {
  system.runCheck()
})

function proceed() {
  router.push('/install')
}
</script>

<template>
  <div class="check-view">
    <h2>{{ t('check.title') }}</h2>

    <div class="check-list">
      <div v-for="check in system.getChecks()" :key="check.name" class="check-item">
        <div class="check-item__main">
          <span class="check-item__icon">{{ statusIcon[check.status] || '\u2B1C' }}</span>
          <span class="check-item__name">{{ t(checkNames[check.name] || check.name) }}</span>
          <span class="check-item__msg">{{ check.message }}</span>
        </div>
        <div
          v-if="check.status === 'fail' && helpText(check.name)"
          class="check-item__help"
        >
          &#x1F4A1; {{ helpText(check.name) }}
        </div>
      </div>
    </div>

    <div v-if="system.loading" class="check-tip">
      <div class="tip-box">
        <span class="tip-spinner" />
        {{ t('check.tip') }}
      </div>
    </div>

    <div v-if="!system.loading && system.report" class="check-actions">
      <div v-if="allPassed" class="check-result check-result--pass">
        <div class="result-icon">&#x2705;</div>
        <div class="result-text">{{ t('check.allPassed') }}</div>
      </div>
      <div v-else class="check-result check-result--fail">
        <div class="result-title">&#x26A0;&#xFE0F; {{ t('check.hasFailed') }}</div>
        <div class="result-desc">{{ t('check.hasFailedDesc') }}</div>
      </div>

      <div class="check-buttons">
        <button class="btn btn--text" @click="router.push('/')">
          {{ t('common.back') }}
        </button>
        <button class="btn btn--primary" @click="proceed">
          {{ allPassed ? t('check.allPassedAction') : t('check.autoFix') }} &#x2192;
        </button>
      </div>

      <div v-if="!allPassed" class="autofix-hint">
        &#x1F4A1; {{ t('check.autoFixDesc') }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.check-view {
  max-width: 520px;
  margin: 0 auto;
  padding: 40px 20px 80px;
}
.check-view h2 {
  margin-bottom: 24px;
}
.check-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 24px;
}
.check-item {
  background: var(--color-surface);
  border-radius: 10px;
  overflow: hidden;
}
.check-item__main {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
}
.check-item__icon { font-size: 18px; width: 24px; flex-shrink: 0; }
.check-item__name { font-weight: 500; min-width: 100px; }
.check-item__msg { color: var(--color-text-secondary); font-size: 14px; }
.check-item__help {
  padding: 8px 16px 12px 48px;
  font-size: 13px;
  color: var(--color-text-secondary);
  line-height: 1.5;
  background: #fffbeb;
}
:root.dark .check-item__help {
  background: #42200644;
}

.tip-box {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background: #eff6ff;
  border-radius: 10px;
  font-size: 14px;
  margin-bottom: 24px;
}
:root.dark .tip-box { background: #1e3a5f; }

.tip-spinner {
  display: inline-block;
  width: 16px;
  height: 16px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  flex-shrink: 0;
}
@keyframes spin { to { transform: rotate(360deg); } }

.check-result {
  padding: 16px;
  border-radius: 10px;
  text-align: center;
  margin-bottom: 16px;
}
.check-result--pass {
  background: #ecfdf5;
  color: #059669;
}
.check-result--fail {
  background: #fff7ed;
  color: #92400e;
}
:root.dark .check-result--pass { background: #064e3b; color: #34d399; }
:root.dark .check-result--fail { background: #78350f; color: #fbbf24; }

.result-icon { font-size: 32px; margin-bottom: 8px; }
.result-text { font-weight: 600; font-size: 16px; }
.result-title { font-weight: 600; font-size: 15px; margin-bottom: 6px; }
.result-desc { font-size: 14px; line-height: 1.5; opacity: 0.85; }

.check-buttons {
  display: flex;
  justify-content: space-between;
  position: relative;
  z-index: 1001;
}

.autofix-hint {
  text-align: center;
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-top: 12px;
  line-height: 1.5;
}
</style>
