import { ref } from 'vue'
import { api } from '../api/client'
import type { AppDetail, AppItem } from '../api/types'

// Module-level shared state so the app bar (AppShell) and the download page
// (AppDetail) both reflect the currently-viewed app without duplicate fetches.
const app = ref<AppItem | null>(null)

export function useDownloadApp() {
  return { app }
}

// Fetch the public app detail by name (fallback: numeric id) and publish the
// app to the shared ref. Resets `app` to null on failure so the app bar never
// shows a stale identity.
export async function loadDownloadApp(key: string | number): Promise<AppDetail> {
  try {
    const d = await api.appDetail(key)
    app.value = d.app
    return d
  } catch (e) {
    app.value = null
    throw e
  }
}
