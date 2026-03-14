<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useInstallStore } from '../stores/install'
import { useChatStore } from '../stores/chat'
import StepProgress from '../components/install/StepProgress.vue'
import LogViewer from '../components/install/LogViewer.vue'
import ErrorCard from '../components/install/ErrorCard.vue'

const router = useRouter()
const { t } = useI18n()
const install = useInstallStore()
const chat = useChatStore()

const isDone = computed(() => install.status?.current_phase === 'done')
const isError = computed(() => install.status?.current_phase === 'error')
const overall = computed(() => install.status?.overall ?? 0)

onMounted(async () => {
  await install.start()
  chat.setContext({ phase: 'install' })
})

onUnmounted(() => {
  install.stopPolling()
})

function handleRetry() {
  install.retry()
}

function goSuccess() {
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

    <ErrorCard
      v-if="isError && install.status"
      :message="install.status.error_message || 'Unknown error'"
      :phase="install.status.error_phase"
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
  padding: 40px 20px;
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
</style>
