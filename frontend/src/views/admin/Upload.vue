<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../../api/client'
import { useI18n } from '../../composables/useI18n'
import { ARCH_BY_PLATFORM, PLATFORMS, detectPlatformFromName } from '../../constants/platform'
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

// Parsed metadata (browser-side, via window.AppInfoParser).
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

const archItems = computed(() =>
  (platform.value ? ARCH_BY_PLATFORM[platform.value] : []).map((a) => ({
    title: t('arch.' + a),
    value: a,
  }))
)

// Architecture options depend on the platform; clear stale selections on change.
watch(platform, () => {
  arch.value = []
})

// Normalize the two result shapes into a single shape.
//  - APK: { package, versionName, versionCode, application.label, icon }
//  - IPA: { CFBundleIdentifier, CFBundleShortVersionString, CFBundleVersion,
//           CFBundleDisplayName/CFBundleName, icon }
function normalizeResult(res: AppInfoParserResult, ext: string) {
  if (ext === 'apk') {
    let appName = res.appName || ''
    // Unresolved resource references (e.g. @string/app_name) are useless.
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
    // Auto-fill the editable fields; the user can override before upload.
    versionName.value = info.versionName
    versionCode.value = info.versionCode || null
    platform.value = info.platform
  } catch (e) {
    parseError.value = (e as Error).message || String(e)
  } finally {
    parsing.value = false
  }
}

function onFileChange(f: File | File[] | null) {
  if (Array.isArray(f)) file.value = f[0] ?? null
  else file.value = f
  parsed.value = null
  parseError.value = ''
  if (!file.value) return
  const ext = (file.value.name.split('.').pop() ?? '').toLowerCase()
  if (!platform.value) platform.value = detectPlatformFromName(file.value.name)
  if (ext === 'apk' || ext === 'ipa') {
    parseApp(file.value, ext)
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
  <v-container class="pa-6" max-width="720">
    <h1 class="text-h4 mb-6">{{ t('upload.title') }}</h1>

    <v-alert v-if="error" type="error" variant="tonal" class="mb-4" closable>
      {{ error }}
    </v-alert>

    <v-form @submit.prevent="submit">
      <v-card variant="outlined" class="mb-4">
        <v-card-text>
          <v-file-input
            :model-value="file"
            :label="t('upload.file')"
            accept=".apk,.aab,.ipa,.exe,.dmg"
            prepend-icon=""
            show-size
            :loading="parsing"
            @update:model-value="onFileChange"
          />

          <v-alert
            v-if="parsing"
            type="info"
            variant="tonal"
            density="compact"
            class="mt-2"
          >
            {{ t('upload.parsing') }}
          </v-alert>
          <v-alert
            v-else-if="parseError"
            type="warning"
            variant="tonal"
            density="compact"
            class="mt-2"
          >
            {{ t('upload.parseFailed') }}
          </v-alert>

          <v-card v-if="parsed" variant="tonal" class="mt-3 pa-3">
            <div class="d-flex align-center" style="gap: 12px;">
              <v-avatar v-if="parsed.iconDataUri" :image="parsed.iconDataUri" size="40" />
              <v-avatar v-else color="primary" size="40">
                <span class="text-h6">{{ (parsed.appName || '?').charAt(0).toUpperCase() }}</span>
              </v-avatar>
              <div class="text-body-2" style="min-width: 0;">
                <div v-if="parsed.appName" class="font-weight-medium">{{ parsed.appName }}</div>
                <code v-if="parsed.package" class="text-caption">{{ parsed.package }}</code>
              </div>
            </div>
          </v-card>
        </v-card-text>
      </v-card>

      <v-card variant="outlined" class="mb-4">
        <v-card-text>
          <v-select
            v-model="appId"
            :items="appItems"
            :label="t('upload.app')"
          />
          <v-row>
            <v-col cols="12" sm="6">
              <v-select
                v-model="releaseType"
                :items="releaseItems"
                :label="t('upload.releaseType')"
              />
            </v-col>
            <v-col cols="12" sm="6">
              <v-select
                v-model="platform"
                :items="platformItems"
                :label="t('upload.platform')"
                clearable
              />
            </v-col>
          </v-row>
          <v-row>
            <v-col cols="12">
              <v-select
                v-model="arch"
                :items="archItems"
                :label="t('upload.arch')"
                multiple
                chips
                clearable
                :disabled="!platform"
              />
            </v-col>
          </v-row>
          <v-row>
            <v-col cols="12" sm="6">
              <v-text-field v-model="versionName" :label="t('upload.versionName')" placeholder="1.0.0" />
            </v-col>
            <v-col cols="12" sm="6">
              <v-text-field
                v-model.number="versionCode"
                :label="t('upload.versionCode')"
                type="number"
                placeholder="1"
              />
            </v-col>
          </v-row>
          <div class="text-caption text-medium-emphasis mt-1">
            {{ t('upload.parseHint') }}
          </div>
          <v-textarea
            v-model="changelog"
            :label="t('upload.changelog')"
            rows="3"
            auto-grow
          />
        </v-card-text>
      </v-card>

      <v-alert type="info" variant="tonal" class="mb-4">
        {{ t('upload.publishHint') }}
      </v-alert>

      <div class="d-flex justify-end" style="gap: 8px;">
        <v-btn @click="router.back()">{{ t('common.cancel') }}</v-btn>
        <v-btn
          color="primary"
          variant="flat"
          :loading="loading"
          :disabled="!file || !appId"
          @click="submit"
        >
          {{ t('upload.submit') }}
        </v-btn>
      </div>
    </v-form>
  </v-container>
</template>
