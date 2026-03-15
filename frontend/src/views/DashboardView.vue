<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import StatusCard from "../components/dashboard/StatusCard.vue";
import QuickActions from "../components/dashboard/QuickActions.vue";
import { checkUpdate, openInBrowser, OPENCLAW_CONSOLE_URL } from "../api/helper";

const { t } = useI18n();

const running = ref(true);
const uptime = ref("2h 30m");
const gateway = ref("ws://127.0.0.1:18789");
const updateAvailable = ref(false);
const updateVersion = ref("");

function toggleService() {
  running.value = !running.value;
}

function openConsole() {
  if (!running.value) {
    alert(t("dashboard.notRunningHint"));
    return;
  }
  openInBrowser(OPENCLAW_CONSOLE_URL);
}

async function handleAction(id: string) {
  if (id === "console") {
    openConsole();
  } else if (id === "update") {
    try {
      const info = await checkUpdate();
      if (info.available) {
        updateAvailable.value = true;
        updateVersion.value = info.version ?? "";
        alert(`${t("dashboard.updateFound")}: v${info.version}`);
      } else {
        alert(t("dashboard.noUpdate"));
      }
    } catch (e) {
      console.error("update check failed:", e);
    }
  }
}

onMounted(() => {
  // Fetch real status from Go helper
});
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
        @toggle="toggleService"
        @open-console="openConsole"
      />
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
</style>
