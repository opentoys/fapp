import axios from 'axios'
import { useAuth } from '../composables/useAuth'
import type { ApiKey, ApiResp, AppDetail, AppItem, DownloadsTimeSeries, KeyScope, Platform, User, Version, VersionMeta, UploadTicket } from './types'

const client = axios.create({ baseURL: '/api/v1', timeout: 60000 })

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
      if (!location.pathname.startsWith('/login')) location.href = '/login'
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

  apps: () => client.get<ApiResp<AppItem[]>>('/apps').then((r) => r.data.data),
  // The public page is keyed by app name (fallback: numeric id).
  appDetail: (key: string | number) =>
    client.get<ApiResp<AppDetail>>(`/apps/${encodeURIComponent(key)}`).then((r) => r.data.data),
  verify: (id: number, password: string) =>
    client.post<ApiResp<{ ok: boolean }>>(`/versions/${id}/verify`, { password }),
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

// sha256Hex computes the hex SHA-256 of a file in the browser.
export async function sha256Hex(file: File): Promise<string> {
  const digest = await crypto.subtle.digest('SHA-256', await file.arrayBuffer())
  return Array.from(new Uint8Array(digest))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('')
}