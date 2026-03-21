<script setup lang="ts">
import { onMounted, onUnmounted, computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useInstallStore } from '../stores/install'
import { useChatStore } from '../stores/chat'
import StepProgress from '../components/install/StepProgress.vue'
import LogViewer from '../components/install/LogViewer.vue'
import ErrorCard from '../components/install/ErrorCard.vue'
import HealingProgress from '../components/install/HealingProgress.vue'

const router = useRouter()
const { t, te } = useI18n()
const install = useInstallStore()
const chat = useChatStore()
const starting = ref(true)
const showLogs = ref(false)

const isDone = computed(() => install.status?.current_phase === 'done')
const isError = computed(() => install.status?.current_phase === 'error')
const isStuck = computed(() => install.stuck || !!install.startError)
const overall = computed(() => install.status?.overall ?? 0)
const currentPhase = computed(() => install.status?.current_phase ?? 'idle')
const isWaiting = computed(() =>
  starting.value || (install.polling && !isStuck.value && !isError.value && !isDone.value && overall.value === 0 && install.events.length === 0)
)

const stuckMessage = computed(() => {
  if (install.startError) return install.startError
  if (install.backendError) return t('install.backendError') + ': ' + install.backendError
  return t('install.stuckHint')
})

// Phase-specific tutorial tip
const phaseTip = computed(() => {
  const phase = currentPhase.value
  if (phase === 'idle' || phase === 'done' || phase === 'error' || phase === 'cancelled') return null
  const key = `install.phaseTips.${phase}`
  if (!te(`${key}.title`)) return null
  return {
    title: t(`${key}.title`),
    desc: t(`${key}.desc`),
    time: te(`${key}.time`) ? t(`${key}.time`) : null,
    warn: te(`${key}.warn`) ? t(`${key}.warn`) : null,
  }
})

// Phase-specific error guidance
const phaseErrorTip = computed(() => {
  const phase = install.status?.error_phase || currentPhase.value
  const key = `install.errorTips.${phase}`
  return te(key) ? t(key) : null
})

onMounted(async () => {
  starting.value = true
  await install.start()
  starting.value = false
  chat.setContext({ phase: 'install' })
})

onUnmounted(() => {
  install.stopPolling()
})

function handleRetry() {
  const phase = install.status?.current_phase
  if (phase === 'idle' || phase === 'cancelled' || install.backendError) {
    install.resetAndStart()
  } else {
    install.retry()
  }
}

function goSuccess() {
  localStorage.setItem('openclaw_installed', 'true')
  router.push('/success')
}
</script>

<template>
  <div class="install-view">
    <h2>{{ t('install.title') }}</h2>

    <StepProgress
      v-if="install.status"
      :phases="install.status.phases"
      :current-phase="install.status.current_phase"
    />

    <div class="install-progress">
      <div class="progress-bar">
        <div class="progress-bar__fill" :style="{ width: overall + '%' }" />
      </div>
      <span class="progress-text">{{ overall }}%</span>
    </div>

    <!-- Connecting / waiting state -->
    <div v-if="isWaiting" class="tip-card tip-card--info">
      <div class="tip-card__header">
        <span class="tip-card__icon waiting-spinner-icon" />
        <span class="tip-card__title">{{ t('install.connecting') }}</span>
      </div>
      <p class="tip-card__desc">{{ t('install.connectingDetail') }}</p>
    </div>

    <!-- Auto-retrying state -->
    <div v-else-if="install.autoRetrying" class="tip-card tip-card--warn">
      <div class="tip-card__header">
        <span class="tip-card__icon">&#x1F504;</span>
        <span class="tip-card__title">{{ t('install.stuckAutoRetry') }}</span>
      </div>
    </div>

    <!-- Phase tutorial tip -->
    <div v-else-if="phaseTip && !isError && !isStuck" class="tip-card tip-card--tutorial">
      <div class="tip-card__header">
        <span class="tip-card__icon">&#x1F4A1;</span>
        <span class="tip-card__title">{{ phaseTip.title }}</span>
      </div>
      <p class="tip-card__desc">{{ phaseTip.desc }}</p>
      <div class="tip-card__meta">
        <span v-if="phaseTip.time" class="tip-meta__item">
          &#x23F1;&#xFE0F; {{ phaseTip.time }}
        </span>
        <span v-if="phaseTip.warn" class="tip-meta__item tip-meta__item--warn">
          &#x26A0;&#xFE0F; {{ phaseTip.warn }}
        </span>
      </div>
    </div>

    <HealingProgress />

    <!-- Error state with friendly guidance -->
    <div v-if="isError && install.status" class="error-section">
      <ErrorCard
        :message="install.status.error_message || 'Unknown error'"
        :phase="install.status.error_phase"
        :guidance="phaseErrorTip ?? undefined"
        @retry="handleRetry"
      />
    </div>

    <!-- Stuck state with friendly guidance -->
    <div v-else-if="isStuck" class="error-section">
      <ErrorCard
        :message="stuckMessage"
        :phase="install.status?.current_phase"
        :guidance="install.backendError ? t('install.backendErrorHelp') : undefined"
        @retry="handleRetry"
      />
    </div>

    <!-- Collapsible log viewer -->
    <div class="install-log">
      <button class="log-toggle" @click="showLogs = !showLogs">
        <span class="log-toggle__arrow" :class="{ 'log-toggle__arrow--open': showLogs }">&#x25B6;</span>
        {{ showLogs ? t('install.logHide') : t('install.logToggle') }}
        <span class="log-toggle__count">({{ install.events.length }})</span>
      </button>
      <div v-show="showLogs">
        <LogViewer :events="install.events" />
      </div>
    </div>

    <div v-if="isDone" class="install-done">
      <div class="done-celebration">&#x1F389;</div>
      <button class="btn btn--primary btn--lg" @click="goSuccess">
        {{ t('config.next') }} &#x2192;
      </button>
    </div>
  </div>
</template>

<style scoped>
.install-view {
  max-width: 560px;
  margin: 0 auto;
  padding: 40px 20px 80px;
}
.install-view h2 { margin-bottom: 16px; }
.install-progress {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 16px 0 24px;
}
.progress-bar {
  flex: 1;
  height: 10px;
  background: var(--color-border);
  border-radius: 5px;
  overflow: hidden;
}
.progress-bar__fill {
  height: 100%;
  background: var(--color-primary);
  border-radius: 5px;
  transition: width 0.5s ease;
}
.progress-text {
  font-weight: 600;
  font-size: 14px;
  min-width: 40px;
}

/* Tip cards */
.tip-card {
  border-radius: 12px;
  padding: 16px;
  margin-bottom: 16px;
  line-height: 1.6;
}
.tip-card--info {
  background: #eff6ff;
  border: 1px solid #bfdbfe;
}
:root.dark .tip-card--info {
  background: #1e3a5f;
  border-color: #2563eb44;
}
.tip-card--warn {
  background: #fffbeb;
  border: 1px solid #fde68a;
}
:root.dark .tip-card--warn {
  background: #422006;
  border-color: #92400e;
}
.tip-card--tutorial {
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
}
:root.dark .tip-card--tutorial {
  background: #052e16;
  border-color: #166534;
}
.tip-card__header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.tip-card__icon {
  font-size: 18px;
  flex-shrink: 0;
}
.tip-card__title {
  font-weight: 600;
  font-size: 15px;
}
.tip-card__desc {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin: 0;
}
.tip-card__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 10px;
}
.tip-meta__item {
  font-size: 13px;
  color: var(--color-text-secondary);
  display: flex;
  align-items: center;
  gap: 4px;
}
.tip-meta__item--warn {
  color: #d97706;
}
:root.dark .tip-meta__item--warn {
  color: #fbbf24;
}

/* Waiting spinner icon */
.waiting-spinner-icon {
  display: inline-block;
  width: 18px;
  height: 18px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

/* Collapsible log */
.install-log { margin-top: 24px; }
.log-toggle {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 0;
  background: none;
  border: none;
  cursor: pointer;
  font-size: 13px;
  color: var(--color-text-secondary);
  transition: color 0.2s;
}
.log-toggle:hover {
  color: var(--color-text);
}
.log-toggle__arrow {
  font-size: 10px;
  transition: transform 0.2s;
  display: inline-block;
}
.log-toggle__arrow--open {
  transform: rotate(90deg);
}
.log-toggle__count {
  color: var(--color-text-secondary);
  font-size: 12px;
}

/* Error section */
.error-section {
  margin-bottom: 16px;
}

/* Done */
.install-done {
  text-align: center;
  margin-top: 24px;
}
.done-celebration {
  font-size: 48px;
  margin-bottom: 16px;
  animation: bounce 0.6s ease-in-out;
}
@keyframes bounce {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-12px); }
}
</style>
