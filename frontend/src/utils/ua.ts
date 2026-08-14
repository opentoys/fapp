import { UAParser } from 'ua-parser-js'
import type { Platform } from '../api/types'

export interface DetectedUA {
  /** True for desktop/unknown OSes, false for android/ios. */
  isDesktop: boolean
  platform: Platform | ''
}

// Platform detection via ua-parser-js. Android/iOS render mobile styles; every
// other OS (or an unknown OS) renders desktop styles.
export function detectUA(): DetectedUA {
  // Pass the UA string explicitly so parsing reflects exactly what the browser
  // (or DevTools device emulation) is sending.
  const { os } = new UAParser(navigator.userAgent).getResult()

  const osName = (os.name ?? '').toLowerCase()
  let platform: Platform | '' = ''
  if (osName.includes('android')) platform = 'android'
  else if (osName.includes('ios') || osName.includes('ipados')) platform = 'ios'

  // Android/iOS are mobile; every other OS is desktop.
  const isDesktop = platform === ''

  return { isDesktop, platform }
}
