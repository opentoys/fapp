export interface AppItem {
  id: number
  name: string
  icon: string
  description: string
  created_at: string
  latest_version: Version | null
}

export interface Channel {
  id: number
  app_id: number
  name: string
}

export interface Version {
  id: number
  app_id: number
  channel_id: number
  version_name: string
  version_code: number
  file_type: string
  file_name: string
  file_size: number
  sha256: string
  changelog: string
  access_mode: 'public' | 'password' | 'expiry'
  expires_at: string | null
  enabled: boolean
  download_count: number
  install_count: number
  created_at: string
  channel?: Channel
}

export interface AppDetail {
  app: AppItem
  channels: Channel[]
  versions: Version[]
}

export interface ApiResp<T> {
  code: number
  msg: string
  data: T
}
