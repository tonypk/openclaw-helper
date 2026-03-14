<script setup lang="ts">
import { watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from './stores/settings'
import ChatBubble from './components/chat/ChatBubble.vue'

const { locale } = useI18n()
const settings = useSettingsStore()

// Sync i18n locale with settings store
watch(
  () => settings.locale,
  (val) => { locale.value = val },
  { immediate: true }
)

// Apply dark mode on mount
if (settings.darkMode) {
  document.documentElement.classList.add('dark')
}
</script>

<template>
  <router-view />
  <ChatBubble />
</template>
