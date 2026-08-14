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
  package_name: string | null
  icon: string
  description: string
  screenshots: string[]
  access_mode: AccessMode
  expires_at: string | null
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
  package_name: string
  app_name: string
  icon_url: string
  sha256: string
  changelog: string
  published: boolean
  enabled: boolean
  download_count: number
  install_count: number
  created_at: string
}

export interface AppDetail {
  app: AppItem
  versions: Version[]
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
