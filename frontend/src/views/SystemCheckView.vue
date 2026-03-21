<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSystemStore } from '../stores/system'

const router = useRouter()
const { t } = useI18n()
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
  pass: '✅',
  fail: '❌',
  warn: '⚠️',
  checking: '⏳',
  skipped: '⏭️',
}

const allPassed = computed(() => system.report?.overall_ready ?? false)

onMounted(() => {
  system.runCheck()
})

function proceed() {
  router.push('/install')
}
</script>

<template>
  <div class="check-view">
    <h2>🔍 {{ t('check.title') }}</h2>

    <div class="check-list">
      <div v-for="check in system.getChecks()" :key="check.name" class="check-item">
        <span class="check-item__icon">{{ statusIcon[check.status] || '⬜' }}</span>
        <span class="check-item__name">{{ t(checkNames[check.name] || check.name) }}</span>
        <span class="check-item__msg">{{ check.message }}</span>
      </div>
    </div>

    <div v-if="system.loading" class="check-tip">
      <div class="tip-box">💡 {{ t('check.tip') }}</div>
    </div>

    <div v-if="!system.loading && system.report" class="check-actions">
      <div v-if="allPassed" class="check-result check-result--pass">
        ✅ {{ t('check.allPassed') }}
      </div>
      <div v-else class="check-result check-result--fail">
        ⚠️ {{ t('check.hasFailed') }}
      </div>

      <div class="check-buttons">
        <button class="btn btn--text" @click="router.push('/')">
          {{ t('common.back') }}
        </button>
        <button class="btn btn--primary" @click="proceed">
          {{ allPassed ? t('config.next') : t('check.autoFix') }} →
        </button>
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
  gap: 12px;
  margin-bottom: 24px;
}
.check-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background: var(--color-surface);
  border-radius: 10px;
}
.check-item__icon { font-size: 18px; width: 24px; }
.check-item__name { font-weight: 500; min-width: 100px; }
.check-item__msg { color: var(--color-text-secondary); font-size: 14px; }

.tip-box {
  padding: 12px 16px;
  background: #eff6ff;
  border-radius: 10px;
  font-size: 14px;
  margin-bottom: 24px;
}
:root.dark .tip-box { background: #1e3a5f; }

.check-result { padding: 12px; border-radius: 8px; text-align: center; margin-bottom: 16px; font-weight: 600; }
.check-result--pass { background: #ecfdf5; color: #059669; }
.check-result--fail { background: #fff7ed; color: #d97706; }
:root.dark .check-result--pass { background: #064e3b; }
:root.dark .check-result--fail { background: #78350f; }

.check-buttons { display: flex; justify-content: space-between; position: relative; z-index: 1001; }
</style>
