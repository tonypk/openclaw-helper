import { defineStore } from "pinia";
import { ref } from "vue";
import type { InstallStatus, ProgressEvent } from "../api/helper";
import {
  installStart,
  installStatus,
  installRetry,
  installCancel,
  installReset,
  installEvents,
} from "../api/helper";

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
  const STUCK_THRESHOLD = 30; // seconds with no progress
  const POLL_FAILURE_THRESHOLD = 5; // consecutive failures before declaring backend dead

  async function start() {
    stuck.value = false;
    startError.value = "";
    try {
      await installStart();
      startPolling();
    } catch (e) {
      startError.value = e instanceof Error ? e.message : String(e);
    }
  }

  async function retry() {
    stuck.value = false;
    startError.value = "";
    try {
      await installRetry();
      startPolling();
    } catch (e) {
      startError.value = e instanceof Error ? e.message : String(e);
    }
  }

  async function cancel() {
    await installCancel();
    stopPolling();
  }

  async function reset() {
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
  }

  function startPolling() {
    if (pollTimer) return;
    polling.value = true;
    lastEventCount = events.value.length;
    stuckTicks = 0;
    consecutivePollFailures = 0;
    backendError.value = "";
    pollTimer = setInterval(async () => {
      try {
        await fetchStatus();
        await fetchEvents();
        consecutivePollFailures = 0;
        backendError.value = "";

        // Stuck detection: no new events for STUCK_THRESHOLD seconds
        if (events.value.length > lastEventCount) {
          lastEventCount = events.value.length;
          stuckTicks = 0;
          stuck.value = false;
        } else {
          stuckTicks++;
          if (stuckTicks >= STUCK_THRESHOLD) {
            stuck.value = true;
          }
        }

        // Stop polling when done or errored
        if (status.value && !status.value.running) {
          const phase = status.value.current_phase;
          if (phase === "done" || phase === "error" || phase === "cancelled") {
            stopPolling();
          }
        }
      } catch (e) {
        consecutivePollFailures++;
        if (consecutivePollFailures >= POLL_FAILURE_THRESHOLD) {
          backendError.value =
            e instanceof Error ? e.message : "Backend unreachable";
          stuck.value = true;
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
    try {
      await installReset();
    } catch {
      // Reset may fail if backend is down, continue anyway
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
