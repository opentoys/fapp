import { ref } from 'vue'
import { api } from '../api/client'
import type { AppDetail, AppGate, AppItem } from '../api/types'

// Module-level shared state so the app bar (AppShell) and the download page
// (AppDetail) both reflect the currently-viewed app without duplicate fetches.
const app = ref<AppItem | null>(null)

export function useDownloadApp() {
  return { app }
}

function isDetail(d: AppDetail | AppGate): d is AppDetail {
  return 'versions' in d
}

// Fetch the public app detail by name (fallback: numeric id) and publish the
// app to the shared ref. Resets `app` to null on failure so the app bar never
// shows a stale identity. A locked app (AppGate) keeps the bar on the fallback
// title — there is no identity to show.
export async function loadDownloadApp(key: string | number): Promise<AppDetail | AppGate> {
  try {
    const d = await api.appDetail(key)
    app.value = isDetail(d) ? d.app : null
    return d
  } catch (e) {
    app.value = null
    throw e
  }
}
