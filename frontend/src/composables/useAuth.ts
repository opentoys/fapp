import { ref, computed } from 'vue'

const TOKEN_KEY = 'token'
const USERNAME_KEY = 'disapp-username'

function decodeJWT(token: string): { uid?: number; username?: string } | null {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) return null
    const payload = JSON.parse(atob(parts[1]))
    return payload
  } catch {
    return null
  }
}

// Module-level state so every component sees the same value.
const token = ref<string | null>(localStorage.getItem(TOKEN_KEY))
const username = ref<string | null>(localStorage.getItem(USERNAME_KEY))
const userId = ref<number | null>(null)

// Initialize userId from existing token on load.
if (token.value) {
  const claims = decodeJWT(token.value)
  userId.value = claims?.uid ?? null
}

export function useAuth() {
  const isAuthed = computed(() => !!token.value)
  const isSuperAdmin = computed(() => userId.value === -1)

  function setToken(t: string) {
    token.value = t
    localStorage.setItem(TOKEN_KEY, t)
    const claims = decodeJWT(t)
    userId.value = claims?.uid ?? null
    if (claims?.username) {
      username.value = claims.username
      localStorage.setItem(USERNAME_KEY, claims.username)
    }
  }

  function clearToken() {
    token.value = null
    username.value = null
    userId.value = null
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USERNAME_KEY)
  }

  return { token: token.value, username, userId, isAuthed, isSuperAdmin, setToken, clearToken }
}