import type { Architecture, Platform } from '../api/types'

// Supported OS-level platforms (architecture is stored separately in `arch`).
export const PLATFORMS: Platform[] = ['ios', 'android', 'windows', 'macos', 'linux']

// Architecture options per platform. Multiple values are allowed per version
// (e.g. a universal APK supporting arm64/armv7/x86).
export const ARCH_BY_PLATFORM: Record<Platform, Architecture[]> = {
  ios: ['universal'],
  android: ['arm64', 'armv7', 'x86', 'x86_64'],
  windows: ['x86', 'x86_64', 'arm64'],
  macos: ['arm64', 'x86_64'],
  linux: ['x86', 'x86_64', 'armv7', 'arm64'],
}

export const ALL_ARCHS: Architecture[] = ['arm64', 'x86_64', 'armv7', 'x86', 'universal']

// Guess the platform from a file extension; falls back to '' when unknown.
export function detectPlatformFromName(name: string): Platform | '' {
  const ext = name.toLowerCase().split('.').pop() ?? ''
  if (ext === 'apk' || ext === 'aab') return 'android'
  if (ext === 'ipa') return 'ios'
  if (ext === 'exe') return 'windows'
  if (ext === 'dmg') return 'macos'
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
