<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const { t } = useI18n()

const step = ref(1)
const apiKey = ref('')
const showTip = ref(false)

function next() {
  if (step.value < 3) {
    step.value++
  } else {
    router.push('/success')
  }
}

function skip() {
  router.push('/success')
}
</script>

<template>
  <div class="config-view">
    <h2>⚙️ {{ t('config.title') }}</h2>
    <p class="config-step">{{ t('config.step') }} {{ step }}/3: AI 模型设置</p>

    <div v-if="step === 1" class="config-section">
      <label class="config-label">🔑 {{ t('config.apiKey') }}</label>
      <input
        v-model="apiKey"
        type="password"
        class="config-input"
        :placeholder="t('config.apiKeyPlaceholder')"
      />

      <div class="config-links">
        <button class="btn btn--text btn--sm" @click="showTip = !showTip">
          ❓ {{ t('config.whatIsApiKey') }}
        </button>
        <button class="btn btn--text btn--sm">
          📖 {{ t('config.getKeyGuide') }}
        </button>
      </div>

      <div v-if="showTip" class="tip-box">
        <p>{{ t('config.apiKeyTip') }}</p>
        <p style="margin-top: 12px; font-weight: 600;">推荐服务商：</p>
        <ul>
          <li>🟢 {{ t('config.providers.openai') }}</li>
          <li>🔵 {{ t('config.providers.deepseek') }}</li>
          <li>🟣 {{ t('config.providers.anthropic') }}</li>
        </ul>
      </div>
    </div>

    <div class="config-actions">
      <button class="btn btn--text" @click="skip">
        ⏭ {{ t('config.skip') }}
      </button>
      <button class="btn btn--primary" @click="next">
        ➡️ {{ t('config.next') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.config-view {
  max-width: 480px;
  margin: 0 auto;
  padding: 40px 20px;
}
.config-view h2 { margin-bottom: 8px; }
.config-step {
  color: var(--color-text-secondary);
  margin-bottom: 24px;
  font-size: 14px;
}
.config-section { margin-bottom: 24px; }
.config-label {
  display: block;
  font-weight: 600;
  margin-bottom: 8px;
}
.config-input {
  width: 100%;
  padding: 12px 16px;
  border: 1px solid var(--color-border);
  border-radius: 10px;
  font-size: 15px;
  outline: none;
  box-sizing: border-box;
}
.config-input:focus { border-color: var(--color-primary); }
.config-links {
  display: flex;
  gap: 12px;
  margin-top: 8px;
}
.tip-box {
  margin-top: 12px;
  padding: 16px;
  background: #eff6ff;
  border-radius: 10px;
  font-size: 14px;
  line-height: 1.6;
}
:root.dark .tip-box { background: #1e3a5f; }
.tip-box ul { padding-left: 8px; list-style: none; margin-top: 4px; }
.tip-box li { margin-bottom: 4px; }
.config-actions {
  display: flex;
  justify-content: space-between;
  margin-top: 32px;
}
</style>
