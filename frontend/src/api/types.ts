export interface User {
  id: number
  username: string
  created_at: string
}

export type AccessMode = 'public' | 'password' | 'expiry'

export interface AppItem {
  id: number
  name: string
  platform: Platform
  appid: string | null
  icon: string
  description: string
  screenshots: string[]
  access_mode: AccessMode
  expires_at: string | null
  published: boolean
  current_version_id: number
  created_at: string
  latest_version: Version | null
}

export type ReleaseType = 'production' | 'beta' | 'canary'
export type Platform = 'ios' | 'android'

export interface Version {
  id: number
  app_id: number
  release_type: ReleaseType
  platform: Platform | ''
  arch: string
  version_name: string
  version_code: number
  file_type: string
  file_name: string
  file_size: number
  appid: string
  app_name: string
  icon_url: string
  sha256: string
  changelog: string
  download_count: number
  install_count: number
  created_at: string
}

export interface AppDetail {
  app: AppItem
  versions: Version[]
}

// Password-protected apps return only this minimal shape until unlocked.
export interface AppGate {
  app: {
    id: number
    access_mode: 'password'
  }
}

export interface UploadTicket {
  key: string
  url: string
}

// VersionMeta is the JSON body sent to create a version whose bytes were
// already pushed to the returned key.
export interface VersionMeta {
  app_id: number
  version_code: number
  version_name: string
  release_type: ReleaseType
  arch?: string
  appid?: string
  app_name?: string
  changelog?: string
  file_name: string
  content_type?: string
  sha256: string
  file_size: number
  key: string
}

export type NotifyEvent = 'version_uploaded' | 'version_current' | 'app_publish' | 'app_expire'

export interface NotificationBot {
  id: number
  name: string
  app_id: number
  method: 'POST' | 'GET' | 'PUT'
  url: string
  headers: string[]
  body_template: string
  events: NotifyEvent[]
  created_at: string
  updated_at: string
}

export interface BotInput {
  name: string
  app_id: number
  method: 'POST' | 'GET' | 'PUT'
  url: string
  headers: string[]
  body_template: string
  events: NotifyEvent[]
}

export interface NotificationLog {
  id: number
  bot_id: number
  app_id: number
  event: string
  url: string
  body: string
  status: number
  error: string
  created_at: string
}

export type KeyScope = 'read' | 'run'

export interface ApiKey {
  id: number
  name: string
  key: string
  user_id: number
  scope: KeyScope
  expires_at: string | null
  last_used_at: string | null
  created_at: string
}

export interface ApiResp<T> {
  code: number
  msg: string
  data: T
}

export interface DownloadsTimeSeries {
  dates: string[]
  total: number[]
  // Parallel to dates; null when no platform/version filter is active.
  selected: number[] | null
}
