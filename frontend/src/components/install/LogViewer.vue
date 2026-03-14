<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import type { ProgressEvent } from '../../api/helper'

const props = defineProps<{
  events: ProgressEvent[]
}>()

const el = ref<HTMLElement>()

watch(() => props.events.length, () => {
  nextTick(() => {
    if (el.value) el.value.scrollTop = el.value.scrollHeight
  })
})

function formatTime(ts: string): string {
  try {
    return new Date(ts).toLocaleTimeString()
  } catch {
    return ''
  }
}
</script>

<template>
  <div ref="el" class="log-viewer">
    <div
      v-for="(evt, i) in events"
      :key="i"
      class="log-entry"
      :class="`log-entry--${evt.status}`"
    >
      <span class="log-entry__time">[{{ formatTime(evt.timestamp) }}]</span>
      <span class="log-entry__msg">{{ evt.message }}</span>
    </div>
    <div v-if="!events.length" class="log-viewer__empty">
      Waiting for logs...
    </div>
  </div>
</template>

<style scoped>
.log-viewer {
  background: var(--color-surface-dark, #1e293b);
  color: #e2e8f0;
  border-radius: 8px;
  padding: 12px;
  font-family: 'Consolas', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  max-height: 200px;
  overflow-y: auto;
}
.log-entry {
  white-space: pre-wrap;
}
.log-entry__time {
  color: #64748b;
  margin-right: 6px;
}
.log-entry--completed .log-entry__msg {
  color: #34d399;
}
.log-entry--failed .log-entry__msg {
  color: #f87171;
}
.log-viewer__empty {
  color: #64748b;
  font-style: italic;
}
</style>
