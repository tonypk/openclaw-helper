<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { openInBrowser, OPENCLAW_CONSOLE_URL } from '../api/helper'

const router = useRouter()
const { t } = useI18n()

const actions = [
  { icon: '📖', labelKey: 'success.tutorial', id: 'tutorial' },
  { icon: '🔧', labelKey: 'success.moreConfig', id: 'config' },
]

function handleAction(id: string) {
  if (id === 'config') {
    router.push('/config')
  }
}

function openConsole() {
  openInBrowser(OPENCLAW_CONSOLE_URL)
}

function goToDashboard() {
  router.push('/dashboard')
}
</script>

<template>
  <div class="success-view">
    <div class="success-icon">✅</div>
    <h1>{{ t('success.title') }}</h1>
    <p class="success-subtitle">{{ t('success.subtitle') }}</p>

    <button class="btn btn--primary btn--lg success-main-btn" @click="openConsole">
      🌐 {{ t('success.openConsole') }}
    </button>

    <div class="success-actions">
      <button
        v-for="a in actions"
        :key="a.id"
        class="success-action"
        @click="handleAction(a.id)"
      >
        <span class="success-action__icon">{{ a.icon }}</span>
        <span>{{ t(a.labelKey) }}</span>
      </button>
    </div>

    <button class="btn btn--text" @click="goToDashboard" style="margin-top: 16px;">
      Go to Dashboard →
    </button>

    <p class="success-tray">{{ t('success.trayHint') }}</p>
  </div>
</template>

<style scoped>
.success-view {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 40px 20px;
  text-align: center;
}
.success-icon { font-size: 64px; margin-bottom: 16px; }
.success-view h1 { margin-bottom: 8px; }
.success-subtitle {
  color: var(--color-text-secondary);
  margin-bottom: 24px;
  font-size: 16px;
}
.success-main-btn {
  margin-bottom: 24px;
  font-size: 16px;
  padding: 14px 32px;
}
.success-actions {
  width: 100%;
  max-width: 400px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.success-action {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 20px;
  border-radius: 12px;
  border: 1px solid var(--color-border);
  background: var(--color-surface);
  cursor: pointer;
  font-size: 15px;
  transition: background 0.2s;
  text-align: left;
}
.success-action:hover { background: var(--color-primary-light); }
.success-action__icon { font-size: 20px; }
.success-tray {
  margin-top: 24px;
  color: var(--color-text-secondary);
  font-size: 14px;
}
</style>
