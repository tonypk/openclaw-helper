import { defineStore } from "pinia";
import { ref, watch } from "vue";

export const useSettingsStore = defineStore("settings", () => {
  const locale = ref<"zh-CN" | "en-US">(
    (localStorage.getItem("locale") as "zh-CN" | "en-US") || "zh-CN",
  );
  const darkMode = ref(
    localStorage.getItem("darkMode") === "true" ||
      window.matchMedia("(prefers-color-scheme: dark)").matches,
  );

  watch(locale, (val) => {
    localStorage.setItem("locale", val);
  });

  watch(darkMode, (val) => {
    localStorage.setItem("darkMode", String(val));
    document.documentElement.classList.toggle("dark", val);
  });

  function toggleLocale() {
    locale.value = locale.value === "zh-CN" ? "en-US" : "zh-CN";
  }

  function toggleDarkMode() {
    darkMode.value = !darkMode.value;
  }

  return { locale, darkMode, toggleLocale, toggleDarkMode };
});
