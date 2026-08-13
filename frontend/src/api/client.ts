import axios from 'axios'
import { useAuth } from '../composables/useAuth'
import type { ApiResp, AppDetail, AppItem, Channel, User, Version } from './types'

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

  apps: () => client.get<ApiResp<AppItem[]>>('/apps').then((r) => r.data.data),
  appDetail: (id: number) => client.get<ApiResp<AppDetail>>(`/apps/${id}`).then((r) => r.data.data),
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
  createApp: (data: { name: string; description?: string }) =>
    client.post<ApiResp<AppItem>>('/admin/apps', data).then((r) => r.data.data),
  updateApp: (id: number, data: Partial<AppItem>) => client.put<ApiResp<AppItem>>(`/admin/apps/${id}`, data),
  deleteApp: (id: number) => client.delete<ApiResp<unknown>>(`/admin/apps/${id}`),
  channels: (appId?: number) =>
    client.get<ApiResp<Channel[]>>('/admin/channels', { params: { app_id: appId } }).then((r) => r.data.data),
  createChannel: (appId: number, name: string) =>
    client.post<ApiResp<Channel>>('/admin/channels', { app_id: appId, name }),
  uploadVersion: (form: FormData) => client.post<ApiResp<Version>>('/admin/versions', form),
  updateVersion: (id: number, data: Partial<Version> & { password?: string }) =>
    client.put<ApiResp<Version>>(`/admin/versions/${id}`, data),
  deleteVersion: (id: number, deleteFile = true) =>
    client.delete<ApiResp<unknown>>(`/admin/versions/${id}`, { params: { delete_file: deleteFile } }),
  versionStats: (id: number) =>
    client
      .get<ApiResp<{ download_count: number; install_count: number; recent: unknown[] }>>(`/admin/versions/${id}/stats`)
      .then((r) => r.data.data),

  adminUsers: () => client.get<ApiResp<User[]>>('/admin/users').then((r) => r.data.data),
  createUser: (data: { username: string; password: string }) =>
    client.post<ApiResp<User>>('/admin/users', data),
  updateUser: (id: number, data: { password?: string }) =>
    client.put<ApiResp<User>>(`/admin/users/${id}`, data),
  deleteUser: (id: number) => client.delete<ApiResp<unknown>>(`/admin/users/${id}`),
}