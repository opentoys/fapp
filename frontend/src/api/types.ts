export interface User {
  id: number
  username: string
  created_at: string
}

export interface AppItem {
  id: number
  name: string
  icon: string
  description: string
  created_at: string
  latest_version: Version | null
}

export type ReleaseType = 'production' | 'beta' | 'canary'
export type Platform = 'ios' | 'android' | 'windows' | 'macos' | 'linux'
export type Architecture = 'arm64' | 'x86_64' | 'armv7' | 'x86' | 'universal'

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
  access_mode: 'public' | 'password' | 'expiry'
  expires_at: string | null
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
