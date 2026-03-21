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
const { t } = useI18n()
const install = useInstallStore()
const chat = useChatStore()
const starting = ref(true)

const isDone = computed(() => install.status?.current_phase === 'done')
const isError = computed(() => install.status?.current_phase === 'error')
const isStuck = computed(() => install.stuck || !!install.startError)
const overall = computed(() => install.status?.overall ?? 0)
const isWaiting = computed(() =>
  starting.value || (install.polling && !isStuck.value && !isError.value && !isDone.value && overall.value === 0 && install.events.length === 0)
)

const stuckMessage = computed(() => {
  if (install.startError) return install.startError
  if (install.backendError) return t('install.backendError') + ': ' + install.backendError
  return t('install.stuckHint')
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
  // If stuck (not in error state), do a full reset+start instead of just retry
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
    <h2>📦 {{ t('install.title') }}</h2>

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

    <div v-if="isWaiting" class="install-waiting">
      <div class="waiting-spinner" />
      <span>{{ t('install.connecting') }}</span>
    </div>

    <HealingProgress />

    <ErrorCard
      v-if="isError && install.status"
      :message="install.status.error_message || 'Unknown error'"
      :phase="install.status.error_phase"
      @retry="handleRetry"
    />

    <ErrorCard
      v-else-if="isStuck"
      :message="stuckMessage"
      :phase="install.status?.current_phase"
      @retry="handleRetry"
    />

    <div class="install-log">
      <h3>📋 {{ t('install.log') }}</h3>
      <LogViewer :events="install.events" />
    </div>

    <div v-if="isDone" class="install-done">
      <button class="btn btn--primary btn--lg" @click="goSuccess">
        {{ t('config.next') }} →
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
.install-log { margin-top: 24px; }
.install-log h3 { margin-bottom: 8px; font-size: 15px; }
.install-done {
  text-align: center;
  margin-top: 24px;
}
.install-waiting {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background: #eff6ff;
  border-radius: 10px;
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-bottom: 16px;
}
:root.dark .install-waiting { background: #1e3a5f; }
.waiting-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
