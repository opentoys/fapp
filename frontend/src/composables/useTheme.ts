import { ref, watch, onMounted } from 'vue'

export type ThemeChoice = 'system' | 'light' | 'dark'

const STORAGE_KEY = 'disapp-theme'

// Module-level state so all callers share the same value.
const choice = ref<ThemeChoice>('system')
const resolved = ref<'light' | 'dark'>('dark')

function readStored(): ThemeChoice {
  const v = localStorage.getItem(STORAGE_KEY)
  if (v === 'light' || v === 'dark' || v === 'system') return v
  return 'system'
}

function systemPrefersDark(): boolean {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

function applyTheme(c: ThemeChoice) {
  const r = c === 'system' ? (systemPrefersDark() ? 'dark' : 'light') : c
  resolved.value = r
  document.documentElement.setAttribute('data-theme', r)
  localStorage.setItem(STORAGE_KEY, c)
}

function nextChoice(c: ThemeChoice): ThemeChoice {
  return c === 'system' ? 'light' : c === 'light' ? 'dark' : 'system'
}

export function useTheme() {
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

  return { choice, resolved, cycle }
}
