import { ref, watch, onMounted } from 'vue'
import { useTheme as useVuetifyTheme } from 'vuetify'

export type ThemeChoice = 'system' | 'light' | 'dark'

const STORAGE_KEY = 'disapp-theme'

// Module-level state so all callers share the same value.
const choice = ref<ThemeChoice>('system')

function readStored(): ThemeChoice {
  const v = localStorage.getItem(STORAGE_KEY)
  if (v === 'light' || v === 'dark' || v === 'system') return v
  return 'system'
}

function systemPrefersDark(): boolean {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

function nextChoice(c: ThemeChoice): ThemeChoice {
  return c === 'system' ? 'light' : c === 'light' ? 'dark' : 'system'
}

export function useTheme() {
  const vuetifyTheme = useVuetifyTheme()

  function applyTheme(c: ThemeChoice) {
    const resolved = c === 'system' ? (systemPrefersDark() ? 'dark' : 'light') : c
    vuetifyTheme.global.name.value = resolved
    localStorage.setItem(STORAGE_KEY, c)
  }

  onMounted(() => {
    choice.value = readStored()
    applyTheme(choice.value)

    // React to system changes when in 'system' mode.
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    mq.addEventListener('change', () => {
      if (choice.value === 'system') applyTheme('system')
    })
  })

  watch(choice, (c) => applyTheme(c))

  function cycle() {
    choice.value = nextChoice(choice.value)
  }

  return { choice, cycle }
}
