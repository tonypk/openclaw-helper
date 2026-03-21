import { defineStore } from "pinia";
import { ref } from "vue";
import type { InstallStatus, ProgressEvent, HealingEvent } from "../api/helper";
import { useHealingStore } from "./healing";
import {
  installStart,
  installStatus,
  installRetry,
  installCancel,
  installReset,
  installEvents,
} from "../api/helper";

function makeLogEvent(message: string, detail?: string): ProgressEvent {
  return {
    phase: "_system",
    status: "info",
    message,
    detail,
    progress: 0,
    overall: 0,
    timestamp: new Date().toISOString(),
  };
}

export const useInstallStore = defineStore("install", () => {
  const status = ref<InstallStatus | null>(null);
  const events = ref<ProgressEvent[]>([]);
  const polling = ref(false);
  const stuck = ref(false);
  const startError = ref("");
  const backendError = ref("");
  let pollTimer: ReturnType<typeof setInterval> | null = null;
  let lastEventCount = 0;
  let stuckTicks = 0;
  let consecutivePollFailures = 0;
  const STUCK_THRESHOLD = 10; // seconds with no progress
  const POLL_FAILURE_THRESHOLD = 5; // consecutive failures before declaring backend dead

  function log(message: string, detail?: string) {
    events.value.push(makeLogEvent(message, detail));
  }

  async function start() {
    stuck.value = false;
    startError.value = "";
    log("[frontend] Sending install.start request...");
    try {
      const result = await installStart();
      log(`[frontend] install.start response: ${result}`);
      startPolling();
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      log(`[frontend] install.start FAILED: ${msg}`);
      startError.value = msg;
    }
  }

  async function retry() {
    stuck.value = false;
    startError.value = "";
    log("[frontend] Sending install.retry request...");
    try {
      const result = await installRetry();
      log(`[frontend] install.retry response: ${result}`);
      startPolling();
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      log(`[frontend] install.retry FAILED: ${msg}`);
      startError.value = msg;
    }
  }

  async function cancel() {
    log("[frontend] Sending install.cancel...");
    await installCancel();
    stopPolling();
  }

  async function reset() {
    log("[frontend] Sending install.reset...");
    await installReset();
    events.value = [];
    status.value = null;
  }

  async function fetchStatus() {
    status.value = await installStatus();
  }

  async function fetchEvents() {
    const newEvents = await installEvents();
    if (newEvents.length > 0) {
      events.value.push(...newEvents);
      // Keep last 200 events
      if (events.value.length > 200) {
        events.value = events.value.slice(-200);
      }
    }

    // Forward healing events to healing store
    if (newEvents.length > 0) {
      const healingStore = useHealingStore();
      for (const event of newEvents) {
        if (event.message?.startsWith("HEAL:")) {
          try {
            const healData: HealingEvent = JSON.parse(event.detail || "{}");
            healingStore.onHealEvent(healData);
          } catch {
            // Ignore malformed healing events
          }
        }
      }
    }
  }

  function startPolling() {
    if (pollTimer) return;
    polling.value = true;
    lastEventCount = events.value.length;
    stuckTicks = 0;
    consecutivePollFailures = 0;
    backendError.value = "";
    log("[frontend] Polling started (1s interval)");
    pollTimer = setInterval(async () => {
      try {
        await fetchStatus();
        await fetchEvents();
        consecutivePollFailures = 0;
        backendError.value = "";

        // Log status periodically when no events
        const phase = status.value?.current_phase ?? "unknown";
        const running = status.value?.running ?? false;
        const overall = status.value?.overall ?? 0;

        // Stuck detection: no new events for STUCK_THRESHOLD seconds
        if (events.value.length > lastEventCount) {
          lastEventCount = events.value.length;
          stuckTicks = 0;
          stuck.value = false;
        } else {
          stuckTicks++;
          if (stuckTicks === 5) {
            log(
              `[frontend] No new events for 5s (phase=${phase}, running=${running}, overall=${overall}%)`,
            );
          }
          if (stuckTicks >= STUCK_THRESHOLD) {
            log(
              `[frontend] STUCK detected: no events for ${STUCK_THRESHOLD}s (phase=${phase}, running=${running}, overall=${overall}%)`,
            );
            stuck.value = true;
          }
        }

        // Stop polling when done or errored
        if (status.value && !status.value.running) {
          if (
            phase === "done" ||
            phase === "error" ||
            phase === "cancelled"
          ) {
            log(`[frontend] Polling stopped: phase=${phase}`);
            stopPolling();
          }
        }
      } catch (e) {
        consecutivePollFailures++;
        const msg = e instanceof Error ? e.message : "Backend unreachable";
        log(
          `[frontend] Poll failed (${consecutivePollFailures}/${POLL_FAILURE_THRESHOLD}): ${msg}`,
        );
        if (consecutivePollFailures >= POLL_FAILURE_THRESHOLD) {
          backendError.value = msg;
          stuck.value = true;
          log("[frontend] Backend declared unreachable, polling stopped");
          stopPolling();
        }
      }
    }, 1000);
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = null;
    }
    polling.value = false;
  }

  /** Reset and restart installation (for stuck/idle state). */
  async function resetAndStart() {
    stuck.value = false;
    startError.value = "";
    backendError.value = "";
    log("[frontend] Reset and restart...");
    try {
      await installReset();
    } catch {
      log("[frontend] Reset failed (backend may be down), continuing...");
    }
    events.value = [];
    status.value = null;
    await start();
  }

  return {
    status,
    events,
    polling,
    stuck,
    startError,
    backendError,
    start,
    retry,
    cancel,
    reset,
    resetAndStart,
    fetchStatus,
    startPolling,
    stopPolling,
  };
});
