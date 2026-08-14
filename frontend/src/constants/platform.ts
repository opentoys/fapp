import type { Platform } from '../api/types'

// Supported OS-level platforms. Initial release ships Android/iOS only.
export const PLATFORMS: Platform[] = ['ios', 'android']

// Guess the platform from a file extension; falls back to '' when unknown.
export function detectPlatformFromName(name: string): Platform | '' {
  const ext = name.toLowerCase().split('.').pop() ?? ''
  if (ext === 'apk' || ext === 'aab') return 'android'
  if (ext === 'ipa') return 'ios'
  return ''
}

// Render a comma-separated arch string through i18n labels.
export function formatArch(t: (key: string) => string, arch: string): string {
  return arch
    .split(',')
    .filter(Boolean)
    .map((a) => t('arch.' + a))
    .join(' / ')
}
