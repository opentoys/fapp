import { UAParser } from 'ua-parser-js'
import type { Architecture, Platform } from '../api/types'

export interface DetectedUA {
  /** True for desktop OSes (windows/macos/linux), false for android/ios. */
  isDesktop: boolean
  platform: Platform | ''
  arch: Architecture | ''
}

// Default architecture per platform when the UA carries no CPU info.
const FALLBACK_ARCH: Record<Platform, Architecture> = {
  ios: 'universal',
  macos: 'arm64',
  windows: 'x86_64',
  linux: 'x86_64',
  android: 'arm64',
}

// Platform + architecture detection via ua-parser-js. Android/iOS always render
// mobile styles; windows/macos/linux (and unknown OSes) render desktop styles.
export function detectUA(): DetectedUA {
  // Pass the UA string explicitly so parsing reflects exactly what the browser
  // (or DevTools device emulation) is sending.
  const { os, cpu } = new UAParser(navigator.userAgent).getResult()

  const osName = (os.name ?? '').toLowerCase()
  let platform: Platform | '' = ''
  if (osName.includes('android')) platform = 'android'
  else if (osName.includes('ios') || osName.includes('ipados')) platform = 'ios'
  else if (osName.includes('windows')) platform = 'windows'
  else if (osName.includes('mac')) platform = 'macos'
  else if (osName.includes('linux')) platform = 'linux'

  // Android/iOS are always mobile; everything else is desktop.
  const isDesktop = !(platform === 'android' || platform === 'ios')

  const cpuArch = (cpu.architecture ?? '').toLowerCase()
  let arch: Architecture | '' = ''
  if (cpuArch === 'amd64') arch = 'x86_64'
  else if (cpuArch === 'arm64') arch = 'arm64'
  else if (cpuArch === 'arm' || cpuArch === 'armhf') arch = 'armv7'
  else if (cpuArch === 'ia32') arch = 'x86'
  if (!arch && platform) arch = FALLBACK_ARCH[platform]

  return { isDesktop, platform, arch }
}
