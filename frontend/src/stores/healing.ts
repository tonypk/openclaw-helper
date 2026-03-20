import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface HealingStrategy {
  name: string
  status: 'pending' | 'running' | 'success' | 'failed'
}

export interface HealingRecord {
  phase: string
  issue: string
  resolvedBy: string
}

export const useHealingStore = defineStore('healing', () => {
  const activePhase = ref<string | null>(null)
  const currentIssue = ref<string | null>(null)
  const strategies = ref<HealingStrategy[]>([])
  const repairLog = ref<string[]>([])
  const history = ref<HealingRecord[]>([])
  const escalated = ref(false)

  const isHealing = computed(() => activePhase.value !== null && !escalated.value)
  const healedCount = computed(() => history.value.length)

  function onHealEvent(event: { type: string; issue?: string; strategy?: string; detail?: string; attempt?: number; max_retry?: number }) {
    switch (event.type) {
      case 'heal_start':
        activePhase.value = event.issue ?? null
        currentIssue.value = event.issue ?? null
        strategies.value = []
        escalated.value = false
        break

      case 'heal_strategy':
        strategies.value = strategies.value.map(s =>
          s.status === 'running' ? { ...s, status: 'failed' as const } : s
        )
        strategies.value = [...strategies.value, { name: event.strategy ?? '', status: 'running' }]
        break

      case 'heal_repair':
        if (event.detail) {
          repairLog.value = [...repairLog.value, event.detail]
        }
        break

      case 'heal_retry':
        strategies.value = strategies.value.map(s =>
          s.status === 'running' ? { ...s, status: 'success' as const } : s
        )
        break

      case 'heal_resolved': {
        const resolvedIssue = currentIssue.value ?? ''
        const resolvedBy = strategies.value.find(s => s.status === 'success')?.name ?? ''
        history.value = [...history.value, {
          phase: activePhase.value ?? '',
          issue: resolvedIssue,
          resolvedBy,
        }]
        activePhase.value = null
        currentIssue.value = null
        strategies.value = []
        break
      }

      case 'heal_escalate':
        escalated.value = true
        strategies.value = strategies.value.map(s =>
          s.status === 'pending' || s.status === 'running'
            ? { ...s, status: 'failed' as const }
            : s
        )
        break
    }
  }

  function reset() {
    activePhase.value = null
    currentIssue.value = null
    strategies.value = []
    repairLog.value = []
    history.value = []
    escalated.value = false
  }

  return {
    activePhase,
    currentIssue,
    strategies,
    repairLog,
    history,
    escalated,
    isHealing,
    healedCount,
    onHealEvent,
    reset,
  }
})
