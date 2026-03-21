<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '../stores/settings'
import { openInBrowser } from '../api/helper'

const router = useRouter()
const { t, tm } = useI18n()
const settings = useSettingsStore()

const introSteps = tm('welcome.introSteps') as string[]

function startInstall() {
  router.push('/check')
}

function learnMore() {
  openInBrowser('https://openclaw.ai/')
}
</script>

<template>
  <div class="welcome">
    <div class="welcome__hero">
      <div class="welcome__logo">&#x1F99E;</div>
      <h1 class="welcome__title">{{ t('app.title') }}</h1>
      <p class="welcome__subtitle">{{ t('app.subtitle') }}</p>

      <button class="btn btn--primary btn--lg welcome__cta" @click="startInstall">
        {{ t('welcome.oneClick') }}
      </button>
    </div>

    <div class="welcome__intro">
      <h3 class="intro__title">{{ t('welcome.introTitle') }}</h3>
      <div class="intro__steps">
        <div
          v-for="(step, i) in introSteps"
          :key="i"
          class="intro__step"
        >
          <span class="intro__step-num">{{ i + 1 }}</span>
          <span class="intro__step-text">{{ step }}</span>
        </div>
      </div>
      <p class="intro__note">{{ t('welcome.introNote') }}</p>
    </div>

    <div class="welcome__links">
      <button class="btn btn--text" @click="router.push('/config')">
        {{ t('welcome.customInstall') }}
      </button>
      <span class="welcome__divider">|</span>
      <button class="btn btn--text" @click="learnMore">
        {{ t('welcome.learnMore') }}
      </button>
    </div>

    <div class="welcome__locale">
      <button class="btn btn--text btn--sm" @click="settings.toggleLocale">
        {{ t('common.language') }} &#x25BC;
      </button>
    </div>
  </div>
</template>

<style scoped>
.welcome {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-height: 100vh;
  padding: 40px 20px;
  text-align: center;
}
.welcome__hero {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: 32px;
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
  margin-bottom: 24px;
  font-size: 16px;
}
.welcome__cta {
  margin-bottom: 24px;
  min-width: 260px;
}

/* Intro section */
.welcome__intro {
  width: 100%;
  max-width: 400px;
  margin-bottom: 24px;
  text-align: left;
}
.intro__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin-bottom: 12px;
  text-align: center;
}
.intro__steps {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.intro__step {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  background: var(--color-surface);
  border-radius: 10px;
  font-size: 14px;
}
.intro__step-num {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--color-primary);
  color: white;
  font-size: 12px;
  font-weight: 700;
  flex-shrink: 0;
}
.intro__step-text {
  color: var(--color-text);
  line-height: 1.4;
}
.intro__note {
  margin-top: 12px;
  font-size: 13px;
  color: var(--color-text-secondary);
  text-align: center;
  line-height: 1.5;
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
