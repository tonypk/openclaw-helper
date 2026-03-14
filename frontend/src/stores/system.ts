import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { SystemReport, CheckResult } from '../api/helper'
import { systemCheck } from '../api/helper'

export const useSystemStore = defineStore('system', () => {
  const report = ref<SystemReport | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function runCheck() {
    loading.value = true
    error.value = null
    try {
      report.value = await systemCheck()
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      loading.value = false
    }
  }

  function getChecks(): CheckResult[] {
    if (!report.value) return []
    return [
      report.value.os,
      report.value.memory,
      report.value.disk,
      report.value.virtualization,
      report.value.wsl,
      report.value.node,
      report.value.openclaw,
    ]
  }

  return { report, loading, error, runCheck, getChecks }
})
