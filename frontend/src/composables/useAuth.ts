import { ref, computed } from 'vue'

const STORAGE_KEY = 'token'

// Module-level state so every component sees the same value. Reading from
// localStorage directly in a computed would not be reactive: the AppShell
// would not re-render when Login.vue stores a new token.
const token = ref<string | null>(localStorage.getItem(STORAGE_KEY))

export function useAuth() {
  const isAuthed = computed(() => !!token.value)

  function setToken(t: string) {
    token.value = t
    localStorage.setItem(STORAGE_KEY, t)
  }

  function clearToken() {
    token.value = null
    localStorage.removeItem(STORAGE_KEY)
  }

  return { token: token.value, isAuthed, setToken, clearToken }
}
