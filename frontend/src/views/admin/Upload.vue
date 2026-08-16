<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Loader2 } from 'lucide-vue-next'
import { api, sha256Hex, uploadViaURL } from '../../api/client'
import { useI18n } from '../../composables/useI18n'
import { detectPlatformFromName } from '../../constants/platform'
import { Alert } from '../../components/ui/alert'
import { Avatar } from '../../components/ui/avatar'
import { Badge } from '../../components/ui/badge'
import { Button } from '../../components/ui/button'
import { Card, CardContent } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Textarea } from '../../components/ui/textarea'
import { RadioGroup, RadioGroupItem } from '../../components/ui/radio-group'
import AppSelect from '../../components/AppSelect.vue'
import FileUpload from '../../components/FileUpload.vue'
import type { AppItem, Platform, ReleaseType } from '../../api/types'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

// Arriving with ?app_id= (from the app detail page) uploads to that app;
// otherwise the page defaults to creating a brand-new app from the package.
const initialAppId = Number(route.query.app_id)
const mode = ref<'new' | 'existing'>(
  Number.isFinite(initialAppId) && initialAppId > 0 ? 'existing' : 'new'
)

const file = ref<File | null>(null)
const parsed = ref<{
  platform: Platform
  package: string
  appName: string
  iconDataUri: string
} | null>(null)
const parsing = ref(false)
const parseError = ref('')

const apps = ref<AppItem[]>([])
const appId = ref<number | null>(Number.isFinite(initialAppId) && initialAppId > 0 ? initialAppId : null)

const newAppName = ref('')
const releaseType = ref<ReleaseType>('production')
const versionName = ref('')
const versionCode = ref<number | null>(null)
const changelog = ref('')
const error = ref('')
const loading = ref(false)

onMounted(async () => {
  try {
    apps.value = await api.adminApps()
  } catch (e) {
    error.value = (e as Error).message
  }
})

const releaseItems = computed(() => [
  { title: t('release.production'), value: 'production' },
  { title: t('release.beta'), value: 'beta' },
  { title: t('release.canary'), value: 'canary' },
])

const appItems = computed(() =>
  apps.value.map((a) => ({ title: a.name, value: a.id }))
)

const selectedApp = computed(() => apps.value.find((a) => a.id === appId.value) ?? null)

// The platform is never chosen freely: for a new app it comes from parsing the
// package, for an existing app it is locked to the app's own platform.
const lockedPlatform = computed<Platform | ''>(() => {
  if (mode.value === 'existing') return selectedApp.value?.platform ?? ''
  return parsed.value?.platform ?? ''
})

// Uploading a package of one platform to an app of another would create a
// wrong-platform version — the server forces the app's platform anyway, so we
// surface the conflict up front and block.
const mismatch = computed(() => {
  if (mode.value !== 'existing') return false
  if (!parsed.value?.platform || !selectedApp.value?.platform) return false
  return parsed.value.platform !== selectedApp.value.platform
})

// The app's appid is locked on its first version upload. A
// locked app may only receive packages exposing the exact same appid — the
// server enforces this, and we surface it up front like the platform check.
const appidMismatch = computed(() => {
  if (mode.value !== 'existing') return false
  const locked = selectedApp.value?.appid
  if (!locked || !parsed.value) return false
  return (parsed.value.package || '') !== locked
})

function normalizeResult(res: AppInfoParserResult, ext: string) {
  if (ext === 'apk') {
    let appName = res.appName || ''
    if (appName.startsWith('@') || appName.startsWith('resourceId:')) appName = ''
    return {
      platform: 'android' as Platform,
      package: res.package || '',
      versionName: res.versionName || '',
      versionCode: Number(res.versionCode) || 0,
      appName,
      iconDataUri: res.icon || '',
    }
  }
  return {
    platform: 'ios' as Platform,
    package: res.CFBundleIdentifier || '',
    versionName: res.CFBundleShortVersionString || '',
    versionCode: Number(res.CFBundleVersion) || 0,
    appName: (res.CFBundleDisplayName as string) || (res.CFBundleName as string) || '',
    iconDataUri: res.icon || '',
  }
}

async function parseApp(f: File, ext: string) {
  parsing.value = true
  parseError.value = ''
  try {
    const info = normalizeResult(await new window.AppInfoParser(f).parse(), ext)
    parsed.value = info
    versionName.value = info.versionName
    versionCode.value = info.versionCode || null
    if (!newAppName.value) newAppName.value = info.appName || f.name.replace(/\.[^.]+$/, '')
  } catch (e) {
    parseError.value = (e as Error).message || String(e)
  } finally {
    parsing.value = false
  }
}

function onFile(f: File | File[]) {
  const resolved = Array.isArray(f) ? f[0] : f
  file.value = resolved
  parsed.value = null
  parseError.value = ''
  if (!resolved) return
  const ext = (resolved.name.split('.').pop() ?? '').toLowerCase()
  if (ext === 'apk' || ext === 'ipa') {
    parseApp(resolved, ext)
  } else {
    // .aab or unknown: platform known from the extension but no deep parse.
    parsed.value = {
      platform: detectPlatformFromName(resolved.name) as Platform,
      package: '',
      appName: resolved.name.replace(/\.[^.]+$/, ''),
      iconDataUri: '',
    }
  }
}

function dataUriToBlob(uri: string): Blob {
  const comma = uri.indexOf(',')
  const mime = /data:([^;]+)/.exec(uri.slice(0, comma))?.[1] || 'image/png'
  const bin = atob(uri.slice(comma + 1))
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
  return new Blob([bytes], { type: mime })
}

async function submit() {
  if (loading.value) return
  if (!file.value) {
    error.value = t('upload.required')
    return
  }
  if (mode.value === 'existing' && !appId.value) {
    error.value = t('upload.appRequired')
    return
  }
  if (mode.value === 'new' && !parsed.value?.platform) {
    error.value = t('upload.parseRequired')
    return
  }
  if (!versionName.value) {
    error.value = t('upload.versionNameRequired')
    return
  }
  if (mismatch.value) {
    error.value = t('upload.platformMismatch')
    return
  }
  if (appidMismatch.value) {
    error.value = t('upload.appidMismatch')
    return
  }
  error.value = ''
  loading.value = true
  const f = file.value as File

  // New-app mode: create the app first, then attach its parsed icon.
  let targetAppId: number
  try {
    if (mode.value === 'existing') {
      targetAppId = appId.value as number
    } else {
      const name = newAppName.value.trim()
      if (!name) {
        error.value = t('upload.appNameRequired')
        loading.value = false
        return
      }
      const app = await api.createApp({
        name,
        platform: parsed.value!.platform,
        appid: parsed.value?.package || undefined,
      })
      targetAppId = app.id
    }

    // Upload the package bytes to the presigned url, then submit metadata
    // (including the key, size and sha256) to persist the version.
    const ticket = await api.presignFile(targetAppId, f.name)
    await uploadViaURL(ticket.url, f)
    const [sha256, fileSize] = await Promise.all([sha256Hex(f), Promise.resolve(f.size)])
    await api.createVersion({
      app_id: targetAppId,
      version_code: versionCode.value ?? 0,
      version_name: versionName.value,
      release_type: releaseType.value,
      changelog: changelog.value,
      file_name: f.name,
      content_type: f.type,
      sha256,
      file_size: fileSize,
      key: ticket.key,
      ...(parsed.value?.package ? { appid: parsed.value.package } : {}),
      ...(parsed.value?.appName ? { app_name: parsed.value.appName } : {}),
    })

    // New-app mode: push the parsed icon to storage and record its key on the app.
    if (mode.value === 'new' && parsed.value?.iconDataUri) {
      const iconBlob = dataUriToBlob(parsed.value.iconDataUri)
      const iconTicket = await api.presignFile(targetAppId, 'icon.png')
      await uploadViaURL(iconTicket.url, new File([iconBlob], 'icon.png'))
      await api.updateApp(targetAppId, { icon: iconTicket.key })
    }

    router.push(`/admin/app/${targetAppId}`)
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="mx-auto max-w-2xl px-4 py-8 sm:px-6">
    <h1 class="mb-6 text-2xl font-semibold tracking-tight">{{ t('upload.title') }}</h1>

    <Alert v-if="error" variant="destructive" class="mb-4">{{ error }}</Alert>

    <RadioGroup v-model="mode" class="mb-6 flex gap-6">
      <div class="flex items-center gap-2 text-sm">
        <RadioGroupItem value="new" id="mode-new" />
        <Label for="mode-new">{{ t('upload.modeNew') }}</Label>
      </div>
      <div class="flex items-center gap-2 text-sm">
        <RadioGroupItem value="existing" id="mode-existing" />
        <Label for="mode-existing">{{ t('upload.modeExisting') }}</Label>
      </div>
    </RadioGroup>

    <form class="grid gap-4" @submit.prevent="submit">
      <Card>
        <CardContent class="grid gap-4">
          <FileUpload
            :label="t('upload.file')"
            accept=".apk,.aab,.ipa"
            drop-zone
            :disabled="parsing"
            @change="onFile"
          />
          <Alert v-if="parsing" variant="info">{{ t('upload.parsing') }}</Alert>
          <Alert v-else-if="parseError" variant="warning">{{ t('upload.parseFailed') }}</Alert>

          <div v-if="parsed" class="flex items-center gap-3 rounded-lg border bg-muted/30 p-3">
            <Avatar :src="parsed.iconDataUri" :fallback="(parsed.appName || '?').charAt(0).toUpperCase()" class="size-10" />
            <div class="min-w-0 text-sm">
              <div class="flex items-center gap-2">
                <span v-if="parsed.appName" class="font-medium">{{ parsed.appName }}</span>
                <Badge v-if="parsed.platform" variant="outline" class="text-xs">{{ t('platform.' + parsed.platform) }}</Badge>
              </div>
              <code v-if="parsed.package" class="text-muted-foreground text-xs">{{ parsed.package }}</code>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent class="grid gap-4">
          <div v-if="mode === 'new'" class="grid gap-2">
            <Label for="upload-app-name">{{ t('upload.appName') }}</Label>
            <Input id="upload-app-name" v-model="newAppName" :placeholder="parsed?.appName || 'My App'" />
          </div>
          <div v-else class="grid gap-2">
            <Label for="upload-app">{{ t('upload.app') }}</Label>
            <AppSelect id="upload-app" v-model="appId" :items="appItems" :placeholder="t('upload.app')" />
          </div>

          <div v-if="lockedPlatform" class="flex items-center gap-2 text-sm">
            <span class="text-muted-foreground">{{ t('upload.platform') }}:</span>
            <Badge variant="outline">{{ t('platform.' + lockedPlatform) }}</Badge>
          </div>
          <Alert v-if="mismatch" variant="warning">{{ t('upload.platformMismatch') }}</Alert>

          <div v-if="mode === 'existing' && selectedApp" class="flex items-center gap-2 text-sm">
            <span class="text-muted-foreground">{{ t('upload.appid') }}:</span>
            <code v-if="selectedApp.appid" class="text-xs">{{ selectedApp.appid }}</code>
            <span v-else class="text-muted-foreground text-xs">{{ t('upload.appidUnlocked') }}</span>
          </div>
          <Alert v-if="appidMismatch" variant="warning">{{ t('upload.appidMismatch') }}</Alert>

          <div class="grid gap-2">
            <Label for="upload-release-type">{{ t('upload.releaseType') }}</Label>
            <AppSelect id="upload-release-type" v-model="releaseType" :items="releaseItems" />
          </div>

          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div class="grid gap-2">
              <Label for="upload-version-name">{{ t('upload.versionName') }}</Label>
              <Input id="upload-version-name" v-model="versionName" :placeholder="'1.0.0'" />
            </div>
            <div class="grid gap-2">
              <Label for="upload-version-code">{{ t('upload.versionCode') }}</Label>
              <Input id="upload-version-code" v-model.number="versionCode" type="number" :placeholder="'1'" />
            </div>
          </div>

          <p class="text-muted-foreground text-xs">{{ t('upload.parseHint') }}</p>

          <div class="grid gap-2">
            <Label for="upload-changelog">{{ t('upload.changelog') }}</Label>
            <Textarea id="upload-changelog" v-model="changelog" rows="3" />
          </div>
        </CardContent>
      </Card>

      <Alert variant="info">{{ t('upload.publishHint') }}</Alert>

      <div class="flex justify-end gap-2">
        <Button variant="outline" @click="router.back()">{{ t('common.cancel') }}</Button>
        <Button type="submit" :disabled="!file || loading">
          <Loader2 v-if="loading" class="size-4 animate-spin" />
          {{ mode === 'new' ? t('upload.createAndUpload') : t('upload.submit') }}
        </Button>
      </div>
    </form>
  </div>
</template>
