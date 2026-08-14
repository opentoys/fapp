// Type declarations for the UMD `app-info-parser` bundle, loaded via a
// <script> tag in index.html (exposes window.AppInfoParser). The package
// author recommends script-tag loading over Vite module import.
export {}

declare global {
  interface AppInfoParserResult {
    /** APK: Android package id. */
    package?: string
    /** APK: application label (resolved via resources.arsc when possible). */
    appName?: string
    /** APK: manifest versionName. */
    versionName?: string
    /** APK: manifest versionCode. */
    versionCode?: string | number
    /** APK: collapsed AndroidManifest attributes. */
    application?: { label?: string }
    /** IPA: CFBundleIdentifier. */
    CFBundleIdentifier?: string
    /** IPA: CFBundleShortVersionString. */
    CFBundleShortVersionString?: string
    /** IPA: CFBundleVersion (build number). */
    CFBundleVersion?: string | number
    /** IPA: CFBundleDisplayName / CFBundleName. */
    CFBundleDisplayName?: string
    CFBundleName?: string
    /** PNG data URI (data:image/png;base64,...) for both platforms. */
    icon?: string | null
    [key: string]: unknown
  }

  interface AppInfoParser {
    parse(): Promise<AppInfoParserResult>
  }

  interface Window {
    AppInfoParser: new (file: Blob | File) => AppInfoParser
  }
}
