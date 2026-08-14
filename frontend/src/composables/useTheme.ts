import { ref, watch, onMounted } from 'vue'

export type ThemeChoice = 'system' | 'light' | 'dark'

const STORAGE_KEY = 'disapp-theme'

function systemPrefersDark(): boolean {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

function applyTheme(c: ThemeChoice) {
  const resolved = c === 'system' ? (systemPrefersDark() ? 'dark' : 'light') : c
  document.documentElement.classList.toggle('dark', resolved === 'dark')
  document.documentElement.style.colorScheme = resolved
  localStorage.setItem(STORAGE_KEY, c)
}

function readStored(): ThemeChoice {
  const v = localStorage.getItem(STORAGE_KEY)
  if (v === 'light' || v === 'dark' || v === 'system') return v
  return 'system'
}

// Module-level state so all callers share the same value.
const choice = ref<ThemeChoice>(readStored())

// Apply immediately (avoid a flash before mount), then keep listening to the OS.
applyTheme(choice.value)
let listenerBound = false

export function useTheme() {
  onMounted(() => {
    if (listenerBound) return
    listenerBound = true
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    mq.addEventListener('change', () => {
      if (choice.value === 'system') applyTheme('system')
    })
  })

  watch(choice, (c) => applyTheme(c))

  function setChoice(c: ThemeChoice) {
    choice.value = c
  }

  return { choice, setChoice }
}
