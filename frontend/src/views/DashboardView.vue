<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import StatusCard from '../components/dashboard/StatusCard.vue'
import QuickActions from '../components/dashboard/QuickActions.vue'

const { t } = useI18n()

const running = ref(true)
const uptime = ref('2h 30m')
const gateway = ref('ws://127.0.0.1:18789')

const channels = ref([
  { icon: '📱', name: 'Telegram', account: '@my_bot', online: true },
  { icon: '💬', name: 'WhatsApp', account: '+86...', online: true },
  { icon: '🎮', name: 'Discord', account: 'MyServer', online: false },
])

function toggleService() {
  running.value = !running.value
}

function handleAction(id: string) {
  console.log('action:', id)
}

onMounted(() => {
  // Fetch real status from Go helper
})
</script>

<template>
  <div class="dashboard">
    <h2>{{ t('dashboard.title') }}</h2>

    <section class="dashboard__section">
      <h3>{{ t('dashboard.status') }}</h3>
      <StatusCard
        :running="running"
        :uptime="uptime"
        :gateway="gateway"
        memory="156 MB"
        cpu="2.1%"
        @toggle="toggleService"
      />
    </section>

    <section class="dashboard__section">
      <h3>{{ t('dashboard.channels') }}</h3>
      <div class="channel-list">
        <div v-for="ch in channels" :key="ch.name" class="channel-item">
          <span class="channel-item__icon">{{ ch.icon }}</span>
          <span class="channel-item__name">{{ ch.name }}</span>
          <span class="channel-item__account">{{ ch.account }}</span>
          <span
            class="channel-item__status"
            :class="{ 'channel-item__status--on': ch.online }"
          >
            {{ ch.online ? t('dashboard.online') : t('dashboard.offline') }}
          </span>
        </div>
        <button class="btn btn--text btn--sm">
          + {{ t('dashboard.addChannel') }}
        </button>
      </div>
    </section>

    <section class="dashboard__section">
      <h3>{{ t('dashboard.quickActions') }}</h3>
      <QuickActions @action="handleAction" />
    </section>
  </div>
</template>

<style scoped>
.dashboard {
  max-width: 560px;
  margin: 0 auto;
  padding: 24px 20px;
}
.dashboard h2 { margin-bottom: 24px; }
.dashboard__section {
  margin-bottom: 24px;
}
.dashboard__section h3 {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-bottom: 10px;
}
.channel-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.channel-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background: var(--color-surface);
  border-radius: 10px;
  border: 1px solid var(--color-border);
}
.channel-item__icon { font-size: 18px; }
.channel-item__name { font-weight: 500; min-width: 80px; }
.channel-item__account { flex: 1; color: var(--color-text-secondary); font-size: 13px; }
.channel-item__status {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 10px;
  background: #fee2e2;
  color: #ef4444;
}
.channel-item__status--on {
  background: #d1fae5;
  color: #059669;
}
</style>
