<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { reportCollect, reportSubmit, openInBrowser } from '../../api/helper'

const props = defineProps<{
  errorPhase?: string
  errorMessage?: string
}>()

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()

const title = ref('')
const description = ref('')
const diagInfo = ref('')
const loading = ref(false)
const collecting = ref(true)
const submitted = ref(false)
const telegramSent = ref(false)

const REPO = 'tonypk/openclaw-helper'

onMounted(async () => {
  // Pre-fill title
  if (props.errorPhase) {
    title.value = `Installation failed - ${props.errorPhase}`
  } else {
    title.value = 'Installation issue'
  }

  // Try to collect diagnostic info from Go backend
  try {
    const report = await reportCollect()
    diagInfo.value = report.system_summary || ''
  } catch {
    // Backend unavailable — build basic info client-side
    diagInfo.value = buildClientDiagInfo()
  } finally {
    collecting.value = false
  }
})

function buildClientDiagInfo(): string {
  const lines = [
    `Phase: ${props.errorPhase || 'unknown'}`,
    `Error: ${props.errorMessage || 'unknown'}`,
    `UserAgent: ${navigator.userAgent}`,
    `Time: ${new Date().toISOString()}`,
    `Note: Go helper sidecar was unreachable`,
  ]
  return lines.join('\n')
}

function buildGitHubURL(issueTitle: string, body: string): string {
  const params = new URLSearchParams({
    title: issueTitle,
    labels: 'crash-report',
    body: body,
  })
  let url = `https://github.com/${REPO}/issues/new?${params.toString()}`
  // Truncate if URL too long
  if (url.length > 8000) {
    const truncBody = body.substring(0, 2000) + '\n\n---\n*Report truncated due to URL length limit.*'
    const truncParams = new URLSearchParams({
      title: issueTitle,
      labels: 'crash-report',
      body: truncBody,
    })
    url = `https://github.com/${REPO}/issues/new?${truncParams.toString()}`
  }
  return url
}

function buildIssueBody(): string {
  const lines = [
    props.errorMessage ? `${description.value}\n` : '',
    '## Error',
    `- **Phase**: \`${props.errorPhase || 'unknown'}\``,
    `- **Message**: ${props.errorMessage || 'N/A'}`,
    '',
    '## Diagnostic Info',
    '```',
    diagInfo.value,
    '```',
    '',
    `*Reported at ${new Date().toISOString()}*`,
  ]
  return lines.filter(Boolean).join('\n')
}

async function handleSubmit() {
  if (loading.value) return
  loading.value = true

  try {
    // Try Go backend first
    const res = await reportSubmit(title.value, description.value)
    telegramSent.value = res.telegram_sent
    submitted.value = true
    if (res.github_url) {
      await openInBrowser(res.github_url)
    }
  } catch {
    // Backend unavailable — build URL client-side and open directly
    const body = buildIssueBody()
    const url = buildGitHubURL(title.value, body)
    await openInBrowser(url)
    telegramSent.value = false
    submitted.value = true
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="report-overlay" @click.self="emit('close')">
    <div class="report-dialog">
      <div class="report-dialog__header">
        <h3>{{ t('report.title') }}</h3>
        <button class="report-dialog__close" @click="emit('close')">&#x2715;</button>
      </div>

      <!-- Success state -->
      <div v-if="submitted" class="report-dialog__body">
        <div class="report-success">
          <div class="report-success__text">{{ t('report.success') }}</div>
          <div class="report-success__hint">{{ t('report.githubOpened') }}</div>
          <div v-if="telegramSent" class="report-success__telegram">
            {{ t('report.telegramSent') }}
          </div>
        </div>
        <div class="report-dialog__footer">
          <button class="btn btn--primary" @click="emit('close')">
            {{ t('common.close') }}
          </button>
        </div>
      </div>

      <!-- Form state -->
      <div v-else class="report-dialog__body">
        <div class="report-field">
          <label class="report-label">{{ t('report.titleLabel') }}</label>
          <input
            v-model="title"
            class="report-input"
            type="text"
            :placeholder="t('report.titleLabel')"
          />
        </div>

        <div class="report-field">
          <label class="report-label">{{ t('report.descLabel') }}</label>
          <textarea
            v-model="description"
            class="report-textarea"
            rows="3"
            :placeholder="t('report.descPlaceholder')"
          />
        </div>

        <div class="report-field">
          <details class="report-details">
            <summary class="report-summary">{{ t('report.diagInfo') }}</summary>
            <pre v-if="collecting" class="report-diag">Loading...</pre>
            <pre v-else class="report-diag">{{ diagInfo }}</pre>
          </details>
        </div>

        <div class="report-dialog__footer">
          <button class="btn btn--secondary" @click="emit('close')">
            {{ t('common.cancel') }}
          </button>
          <button
            class="btn btn--primary"
            :disabled="loading || !title"
            @click="handleSubmit"
          >
            {{ loading ? t('report.submitting') : t('report.submit') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.report-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.report-dialog {
  background: var(--color-bg, #fff);
  border-radius: 12px;
  width: 90%;
  max-width: 500px;
  max-height: 80vh;
  overflow-y: auto;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
}

:root.dark .report-dialog {
  background: #1e1e2e;
}

.report-dialog__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--color-border, #e5e7eb);
}

.report-dialog__header h3 {
  margin: 0;
  font-size: 16px;
}

.report-dialog__close {
  background: none;
  border: none;
  font-size: 18px;
  cursor: pointer;
  color: var(--color-text-secondary, #6b7280);
  padding: 4px 8px;
  border-radius: 4px;
}

.report-dialog__close:hover {
  background: var(--color-bg-hover, #f3f4f6);
}

.report-dialog__body {
  padding: 20px;
}

.report-field {
  margin-bottom: 16px;
}

.report-label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 6px;
  color: var(--color-text, #374151);
}

.report-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--color-border, #d1d5db);
  border-radius: 8px;
  font-size: 14px;
  background: var(--color-bg, #fff);
  color: var(--color-text, #374151);
  box-sizing: border-box;
}

.report-textarea {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--color-border, #d1d5db);
  border-radius: 8px;
  font-size: 14px;
  resize: vertical;
  background: var(--color-bg, #fff);
  color: var(--color-text, #374151);
  font-family: inherit;
  box-sizing: border-box;
}

.report-details {
  border: 1px solid var(--color-border, #d1d5db);
  border-radius: 8px;
  overflow: hidden;
}

.report-summary {
  padding: 8px 12px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  background: var(--color-bg-secondary, #f9fafb);
  user-select: none;
}

:root.dark .report-summary {
  background: #2a2a3e;
}

.report-diag {
  padding: 12px;
  font-size: 12px;
  margin: 0;
  max-height: 200px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
  background: var(--color-bg-secondary, #f9fafb);
}

:root.dark .report-diag {
  background: #1a1a2e;
}

.report-dialog__footer {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  margin-top: 16px;
}

.report-success {
  text-align: center;
  padding: 20px 0;
}

.report-success__text {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 8px;
}

.report-success__hint {
  font-size: 14px;
  color: var(--color-text-secondary, #6b7280);
  margin-bottom: 4px;
}

.report-success__telegram {
  font-size: 13px;
  color: var(--color-text-secondary, #6b7280);
  margin-top: 8px;
}
</style>
