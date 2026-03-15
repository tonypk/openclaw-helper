<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '../stores/settings'
import { openInBrowser } from '../api/helper'

const router = useRouter()
const { t } = useI18n()
const settings = useSettingsStore()

function startInstall() {
  router.push('/check')
}

function learnMore() {
  openInBrowser('https://openclaw.ai/')
}
</script>

<template>
  <div class="welcome">
    <div class="welcome__logo">🦞</div>
    <h1 class="welcome__title">{{ t('app.title') }}</h1>
    <p class="welcome__subtitle">{{ t('app.subtitle') }}</p>

    <button class="btn btn--primary btn--lg welcome__cta" @click="startInstall">
      ✨ {{ t('welcome.oneClick') }}
    </button>

    <div class="welcome__links">
      <button class="btn btn--text" @click="router.push('/config')">
        ⚙️ {{ t('welcome.customInstall') }}
      </button>
      <span class="welcome__divider">|</span>
      <button class="btn btn--text" @click="learnMore">
        📖 {{ t('welcome.learnMore') }}
      </button>
    </div>

    <div class="welcome__locale">
      <button class="btn btn--text btn--sm" @click="settings.toggleLocale">
        🌐 {{ t('common.language') }} ▼
      </button>
    </div>
  </div>
</template>

<style scoped>
.welcome {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 40px 20px;
  text-align: center;
}
.welcome__logo {
  font-size: 64px;
  margin-bottom: 16px;
}
.welcome__title {
  font-size: 28px;
  font-weight: 700;
  margin-bottom: 8px;
}
.welcome__subtitle {
  color: var(--color-text-secondary);
  margin-bottom: 32px;
  font-size: 16px;
}
.welcome__cta {
  margin-bottom: 24px;
  min-width: 260px;
}
.welcome__links {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 32px;
}
.welcome__divider {
  color: var(--color-text-secondary);
}
.welcome__locale {
  position: absolute;
  bottom: 24px;
  left: 24px;
}
</style>
