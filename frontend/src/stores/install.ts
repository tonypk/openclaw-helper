import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { InstallStatus, ProgressEvent } from '../api/helper'
import {
  installStart,
  installStatus,
  installRetry,
  installCancel,
  installReset,
  installEvents,
} from '../api/helper'

export const useInstallStore = defineStore('install', () => {
  const status = ref<InstallStatus | null>(null)
  const events = ref<ProgressEvent[]>([])
  const polling = ref(false)
  let pollTimer: ReturnType<typeof setInterval> | null = null

  async function start() {
    await installStart()
    startPolling()
  }

  async function retry() {
    await installRetry()
    startPolling()
  }

  async function cancel() {
    await installCancel()
    stopPolling()
  }

  async function reset() {
    await installReset()
    events.value = []
    status.value = null
  }

  async function fetchStatus() {
    status.value = await installStatus()
  }

  async function fetchEvents() {
    const newEvents = await installEvents()
    if (newEvents.length > 0) {
      events.value.push(...newEvents)
      // Keep last 200 events
      if (events.value.length > 200) {
        events.value = events.value.slice(-200)
      }
    }
  }

  function startPolling() {
    if (pollTimer) return
    polling.value = true
    pollTimer = setInterval(async () => {
      try {
        await fetchStatus()
        await fetchEvents()
        // Stop polling when done or errored
        if (status.value && !status.value.running) {
          const phase = status.value.current_phase
          if (phase === 'done' || phase === 'error' || phase === 'cancelled') {
            stopPolling()
          }
        }
      } catch {
        // Ignore polling errors
      }
    }, 1000)
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
    polling.value = false
  }

  return {
    status,
    events,
    polling,
    start,
    retry,
    cancel,
    reset,
    fetchStatus,
    startPolling,
    stopPolling,
  }
})
