<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../../api/client'
import { useI18n } from '../../composables/useI18n'
import { ARCH_BY_PLATFORM, PLATFORMS, detectPlatformFromName } from '../../constants/platform'
import { Alert } from '../../components/ui/alert'
import { Avatar } from '../../components/ui/avatar'
import { Button } from '../../components/ui/button'
import { Card, CardContent } from '../../components/ui/card'
import { Checkbox } from '../../components/ui/checkbox'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Textarea } from '../../components/ui/textarea'
import AppSelect from '../../components/AppSelect.vue'
import FileUpload from '../../components/FileUpload.vue'
import type { AppItem, Architecture, Platform, ReleaseType } from '../../api/types'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const file = ref<File | null>(null)
const initialAppId = Number(route.query.app_id)
const appId = ref<number | null>(Number.isFinite(initialAppId) && initialAppId > 0 ? initialAppId : null)
const releaseType = ref<ReleaseType>('production')
const platform = ref<Platform | ''>('')
const arch = ref<Architecture[]>([])
const versionName = ref('')
const versionCode = ref<number | null>(null)
const changelog = ref('')
const error = ref('')
const loading = ref(false)

const parsing = ref(false)
const parseError = ref('')
const parsed = ref<{
  platform: Platform
  package: string
  appName: string
  iconDataUri: string
} | null>(null)

const apps = ref<AppItem[]>([])

onMounted(async () => {
  try {
    apps.value = await api.adminApps()
  } catch (e) {
    error.value = (e as Error).message
  }
})

const appItems = computed(() =>
  apps.value.map((a) => ({ title: a.name, value: a.id }))
)

const releaseItems = computed(() => [
  { title: t('release.production'), value: 'production' },
  { title: t('release.beta'), value: 'beta' },
  { title: t('release.canary'), value: 'canary' },
])

const platformItems = computed(() =>
  PLATFORMS.map((p) => ({ title: t('platform.' + p), value: p }))
)

const archOptions = computed(() =>
  platform.value ? ARCH_BY_PLATFORM[platform.value] : []
)

watch(platform, () => { arch.value = [] })

function toggleArch(a: Architecture, checked: boolean) {
  arch.value = checked ? [...arch.value, a] : arch.value.filter((x) => x !== a)
}

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
    platform.value = info.platform
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
  if (!platform.value) platform.value = detectPlatformFromName(resolved.name)
  if (ext === 'apk' || ext === 'ipa') {
    parseApp(resolved, ext)
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
  if (!file.value || !appId.value) {
    error.value = t('upload.required')
    return
  }
  if (!versionName.value) {
    error.value = t('upload.versionNameRequired')
    return
  }
  error.value = ''
  loading.value = true

  const form = new FormData()
  form.append('file', file.value)
  form.append('app_id', String(appId.value))
  form.append('release_type', releaseType.value)
  if (platform.value) form.append('platform', platform.value)
  if (arch.value.length) form.append('arch', arch.value.join(','))
  if (versionName.value) form.append('version_name', versionName.value)
  if (versionCode.value) form.append('version_code', String(versionCode.value))
  form.append('changelog', changelog.value)
  if (parsed.value) {
    if (parsed.value.package) form.append('package_name', parsed.value.package)
    if (parsed.value.appName) form.append('app_name', parsed.value.appName)
    if (parsed.value.iconDataUri) {
      form.append('icon', dataUriToBlob(parsed.value.iconDataUri), 'icon.png')
    }
  }

  try {
    await api.uploadVersion(form)
    router.push(`/admin/app/${appId.value}`)
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="mx-auto max-w-2xl px-4 py-8 sm:px-6">
    <h1 class="text-2xl font-semibold tracking-tight mb-6">{{ t('upload.title') }}</h1>

    <Alert v-if="error" variant="destructive" class="mb-4">{{ error }}</Alert>

    <form class="grid gap-4" @submit.prevent="submit">
      <Card>
        <CardContent class="grid gap-4">
          <FileUpload
            :label="t('upload.file')"
            accept=".apk,.aab,.ipa,.exe,.dmg"
            drop-zone
            :disabled="parsing"
            @change="onFile"
          />
          <Alert v-if="parsing" variant="info">{{ t('upload.parsing') }}</Alert>
          <Alert v-else-if="parseError" variant="warning">{{ t('upload.parseFailed') }}</Alert>

          <div v-if="parsed" class="flex items-center gap-3 rounded-lg border bg-muted/30 p-3">
            <Avatar :src="parsed.iconDataUri" :fallback="(parsed.appName || '?').charAt(0).toUpperCase()" class="size-10" />
            <div class="min-w-0 text-sm">
              <div v-if="parsed.appName" class="font-medium">{{ parsed.appName }}</div>
              <code v-if="parsed.package" class="text-muted-foreground text-xs">{{ parsed.package }}</code>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent class="grid gap-4">
          <div class="grid gap-2">
            <Label>{{ t('upload.app') }}</Label>
            <AppSelect v-model="appId" :items="appItems" :placeholder="t('upload.app')" />
          </div>

          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div class="grid gap-2">
              <Label>{{ t('upload.releaseType') }}</Label>
              <AppSelect v-model="releaseType" :items="releaseItems" />
            </div>
            <div class="grid gap-2">
              <Label>{{ t('upload.platform') }}</Label>
              <AppSelect v-model="platform" :items="platformItems" :placeholder="t('upload.platform')" />
            </div>
          </div>

          <div class="grid gap-2">
            <Label>{{ t('upload.arch') }}</Label>
            <div v-if="archOptions.length" class="flex flex-wrap gap-4">
              <label v-for="a in archOptions" :key="a" class="flex cursor-pointer items-center gap-2 text-sm">
                <Checkbox :model-value="arch.includes(a)" @update:model-value="(c) => toggleArch(a, !!c)" />
                {{ t('arch.' + a) }}
              </label>
            </div>
            <p v-else class="text-muted-foreground text-sm">—</p>
          </div>

          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div class="grid gap-2">
              <Label>{{ t('upload.versionName') }}</Label>
              <Input v-model="versionName" :placeholder="'1.0.0'" />
            </div>
            <div class="grid gap-2">
              <Label>{{ t('upload.versionCode') }}</Label>
              <Input v-model.number="versionCode" type="number" :placeholder="'1'" />
            </div>
          </div>

          <p class="text-muted-foreground text-xs">{{ t('upload.parseHint') }}</p>

          <div class="grid gap-2">
            <Label>{{ t('upload.changelog') }}</Label>
            <Textarea v-model="changelog" rows="3" />
          </div>
        </CardContent>
      </Card>

      <Alert variant="info">{{ t('upload.publishHint') }}</Alert>

      <div class="flex justify-end gap-2">
        <Button variant="outline" @click="router.back()">{{ t('common.cancel') }}</Button>
        <Button type="submit" :disabled="!file || !appId || loading">
          {{ t('upload.submit') }}
        </Button>
      </div>
    </form>
  </div>
</template>
