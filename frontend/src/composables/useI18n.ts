import { ref } from 'vue'
import { messages, type Locale } from '../i18n/messages'

const STORAGE_KEY = 'disapp-locale'

function browserLang(): Locale {
  if (typeof navigator === 'undefined') return 'en'
  const lang = navigator.language || ''
  if (lang.startsWith('zh')) return 'zh'
  return 'en'
}

function readStored(): Locale {
  if (typeof window === 'undefined') return 'en'
  const v = localStorage.getItem(STORAGE_KEY)
  if (v === 'en' || v === 'zh') return v
  return browserLang()
}

// Module-level state: every component sees the same locale.
const locale = ref<Locale>(readStored())

// Escape user-controlled values before they get substituted into a
// translation string. Messages themselves may contain HTML tags like
// <code>...</code>, but substituted {name}/{value} placeholders are
// treated as plain text.
function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

export function useI18n() {
  function t(key: string, params?: Record<string, string | number>): string {
    const dict = messages[locale.value] ?? messages.en
    let s = dict[key] ?? messages.en[key] ?? key
    if (params) {
      for (const [k, v] of Object.entries(params)) {
        s = s.replace(new RegExp(`\\{${k}\\}`, 'g'), escapeHtml(String(v)))
      }
    }
    return s
  }

  function setLocale(l: Locale) {
    locale.value = l
    localStorage.setItem(STORAGE_KEY, l)
  }

  return { locale, t, setLocale }
}