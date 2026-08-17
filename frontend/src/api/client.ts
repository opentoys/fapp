import CryptoJS from 'crypto-js'
import axios from 'axios'
import { useAuth } from '../composables/useAuth'
import type { ApiKey, ApiResp, AppDetail, AppGate, AppItem, BotInput, DownloadsTimeSeries, KeyScope, NotificationBot, NotificationLog, Platform, User, Version, VersionMeta, UploadTicket } from './types'

const client = axios.create({ baseURL: '/api/v1', timeout: 60000 })

// Hash router: the app lives under `/#/`. Redirecting a 401 to `/login`
// (a bare path) would reload to a non-SPA route, so jump via the hash
// instead and carry the current path as the post-login redirect target.
function loginURL(): string {
  const cur = location.hash.replace(/^#/, '')
  return `#/login?redirect=${encodeURIComponent(cur || '/admin')}`
}

function isOnLoginPage(): boolean {
  return location.hash.startsWith('#/login')
}

client.interceptors.request.use((cfg) => {
  const token = localStorage.getItem('token')
  if (token) cfg.headers.Authorization = `Bearer ${token}`
  return cfg
})

client.interceptors.response.use((res) => {
  const body = res.data as ApiResp<unknown>
  if (body.code !== 0) {
    if (body.code === 401) {
      useAuth().clearToken()
      if (!isOnLoginPage()) location.href = loginURL()
    }
    return Promise.reject(new Error(body.msg))
  }
  return res
})

export const api = {
  login: (username: string, password: string) =>
    client.post<ApiResp<{ token: string }>>('/auth/login', { username, password }),
  changePassword: (oldPassword: string, newPassword: string) =>
    client.put<ApiResp<{ ok: boolean }>>('/auth/password', { old_password: oldPassword, new_password: newPassword }),

  // The public page is keyed by app name (fallback: numeric id). A
  // password-protected app returns an AppGate until the password unlocks it.
  appDetail: (key: string | number, password?: string) =>
    client
      .get<ApiResp<AppDetail | AppGate>>(`/apps/${encodeURIComponent(key)}`, { params: password ? { password } : {} })
      .then((r) => r.data.data),
  downloadUrl: (id: number, password?: string) =>
    client
      .get<ApiResp<{ url: string }>>(`/versions/${id}/download`, { params: password ? { password } : {} })
      .then((r) => r.data.data.url),
  installUrl: (id: number, password?: string) =>
    client
      .get<ApiResp<{ url: string }>>(`/versions/${id}/install`, { params: password ? { password } : {} })
      .then((r) => r.data.data.url),

  adminApps: () => client.get<ApiResp<AppItem[]>>('/admin/apps').then((r) => r.data.data),
  adminAppDetail: (id: number) => client.get<ApiResp<AppDetail>>(`/admin/apps/${id}`).then((r) => r.data.data),
  createApp: (data: { name: string; description?: string; platform: Platform; icon?: string; appid?: string }) =>
    client.post<ApiResp<AppItem>>('/admin/apps', data).then((r) => r.data.data),
  updateApp: (id: number, data: Partial<AppItem> & { password?: string }) =>
    client.put<ApiResp<AppItem>>(`/admin/apps/${id}`, data),
  // presignFile is the single presigned-upload endpoint for every file kind
  // (version package, icon, screenshot): {app_id, file_name} → {url, key}.
  presignFile: (appId: number, fileName: string) =>
    client.post<ApiResp<UploadTicket>>('/files', { app_id: appId, file_name: fileName }).then((r) => r.data.data),
  deleteAppScreenshot: (id: number, url: string) =>
    client
      .delete<ApiResp<AppItem>>(`/admin/apps/${id}/screenshots`, { params: { url } })
      .then((r) => r.data.data),
  deleteApp: (id: number) => client.delete<ApiResp<unknown>>(`/admin/apps/${id}`),
  appMembers: (id: number) => client.get<ApiResp<number[]>>(`/admin/apps/${id}/members`).then((r) => r.data.data),
  setAppMembers: (id: number, uids: number[]) =>
    client.put<ApiResp<number[]>>(`/admin/apps/${id}/members`, uids).then((r) => r.data.data),
  appDownloads: (id: number, params?: { platform?: string; version_id?: number }) =>
    client
      .get<ApiResp<DownloadsTimeSeries>>(`/admin/apps/${id}/downloads`, { params })
      .then((r) => r.data.data),
  createVersion: (meta: VersionMeta) => client.post<ApiResp<Version>>('/admin/versions', meta).then((r) => r.data.data),
  setCurrentVersion: (appId: number, versionId: number) =>
    client.post<ApiResp<AppItem>>(`/admin/apps/${appId}/current`, { version_id: versionId }).then((r) => r.data.data),
  deleteVersion: (id: number, deleteFile = true) =>
    client.delete<ApiResp<unknown>>(`/admin/versions/${id}`, { params: { delete_file: deleteFile } }),
  versionStats: (id: number) =>
    client
      .get<ApiResp<{ download_count: number; install_count: number; recent: unknown[] }>>(`/admin/versions/${id}/stats`)
      .then((r) => r.data.data),

  adminUsers: () => client.get<ApiResp<User[]>>('/admin/users').then((r) => r.data.data),
  createUser: (data: { username: string; password: string }) =>
    client.post<ApiResp<User>>('/admin/users', data),
  updateUser: (id: number, data: { username?: string; password?: string }) =>
    client.put<ApiResp<User>>(`/admin/users/${id}`, data),
  deleteUser: (id: number) => client.delete<ApiResp<unknown>>(`/admin/users/${id}`),

  manageableApps: () => client.get<ApiResp<AppItem[]>>('/admin/apps/manageable').then((r) => r.data.data),
  adminKeys: () => client.get<ApiResp<ApiKey[]>>('/admin/keys').then((r) => r.data.data),
  createKey: (data: { name: string; scope: KeyScope; expires_at?: string | null }) =>
    client.post<ApiResp<ApiKey>>('/admin/keys', data).then((r) => r.data.data),
  updateKey: (id: number, data: { name?: string; scope?: KeyScope; expires_at?: string | null }) =>
    client.put<ApiResp<ApiKey>>(`/admin/keys/${id}`, data),
  deleteKey: (id: number) => client.delete<ApiResp<unknown>>(`/admin/keys/${id}`),

  subscriptions: () => client.get<ApiResp<NotificationBot[]>>('/admin/subscriptions').then((r) => r.data.data),
  createSubscription: (data: BotInput) =>
    client.post<ApiResp<NotificationBot>>('/admin/subscriptions', data).then((r) => r.data.data),
  updateSubscription: (id: number, data: BotInput) =>
    client.put<ApiResp<NotificationBot>>(`/admin/subscriptions/${id}`, data).then((r) => r.data.data),
  deleteSubscription: (id: number) => client.delete<ApiResp<unknown>>(`/admin/subscriptions/${id}`),
  testSubscription: (id: number) => client.post<ApiResp<unknown>>(`/admin/subscriptions/${id}/test`),
  testSubscriptionConfig: (data: BotInput) => client.post<ApiResp<unknown>>('/admin/subscriptions/test', data),
  subscriptionLogs: (id: number, limit = 20) =>
    client.get<ApiResp<NotificationLog[]>>(`/admin/subscriptions/${id}/logs`, { params: { limit } }).then((r) => r.data.data),
}

// uploadViaURL pushes the file body straight to the presigned url returned by a
// presign endpoint ({key,url}). COS presigns a PUT URL; local signs a server
// upload endpoint (/files/upload) that takes a POST — derive the method from
// the url host/path so neither backend needs a method hint.
export async function uploadViaURL(url: string, file: File): Promise<void> {
  const method = url.includes('/api/v1/files/upload') ? 'POST' : 'PUT'
  const res = await fetch(url, { method, body: file })
  if (!res.ok) throw new Error(`upload failed: ${res.status}`)
}

// fileURL wraps a bare storage key (admin-side responses) into the
// authenticated preview URL that 307s to the actual signed stream. It uses
// the admin variant because dl=1 requires the caller to be authenticated;
// the JWT rides in ?token so <img> tags (no Authorization header) pass auth.
export function fileURL(key: string): string {
  const t = useAuth().token
  const q = new URLSearchParams({ key, dl: '1' })
  if (t) q.set('token', t)
  return `/api/v1/admin/files/preview?${q.toString()}`
}

// sha256Hex computes the hex SHA-256 of a file. It prefers the native Web
// Crypto API (available only in secure contexts) and falls back to crypto-js
// otherwise, so plain-HTTP self-hosted deployments still get a checksum.
export async function sha256Hex(file: File): Promise<string> {
  const buf = await file.arrayBuffer()
  if (typeof crypto !== 'undefined' && crypto.subtle) {
    try {
      const digest = await crypto.subtle.digest('SHA-256', buf)
      return Array.from(new Uint8Array(digest))
        .map((b) => b.toString(16).padStart(2, '0'))
        .join('')
    } catch {
      // fall through to crypto-js
    }
  }
  const wordArray = CryptoJS.lib.WordArray.create(buf)
  return CryptoJS.SHA256(wordArray).toString(CryptoJS.enc.Hex)
}