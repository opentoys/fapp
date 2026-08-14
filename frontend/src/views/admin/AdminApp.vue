<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { mdiContentCopy } from '@mdi/js'
import { api } from '../../api/client'
import { useI18n } from '../../composables/useI18n'
import { PLATFORMS, formatArch } from '../../constants/platform'
import { fmtDate } from '../../utils/format'
import LineChart from '../../components/LineChart.vue'
import type { AppDetail, AppItem, DownloadsTimeSeries, Platform, ReleaseType, Version } from '../../api/types'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const data = ref<AppDetail | null>(null)
const stats = ref<{ download_count: number; install_count: number; recent: Array<{ ip: string; user_agent: string; created_at: string }> } | null>(null)
const error = ref('')

// --- Stats: app-level download trend chart, aggregated by day ---
const chartFilterPlatform = ref<'all' | Platform>('all')
const chartFilterVersion = ref<'all' | number>('all')
const chartRange = ref<'7' | '30' | '90' | 'all'>('30')
const chartData = ref<DownloadsTimeSeries | null>(null)
const chartLoading = ref(false)
const chartError = ref('')

const rangeItems = computed(() => [
  { title: t('adminApp.chartRange7'), value: '7' },
  { title: t('adminApp.chartRange30'), value: '30' },
  { title: t('adminApp.chartRange90'), value: '90' },
  { title: t('adminApp.chartRangeAll'), value: 'all' },
])

// Version choices for the chart filter, newest first.
const chartVersionItems = computed(() => {
  const list = data.value?.versions ?? []
  return [
    { title: t('adminApp.filterAll'), value: 'all' },
    ...list
      .slice()
      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
      .map((v) => ({
        title: `${v.version_name}${v.platform ? ` · ${t('platform.' + v.platform)}` : ''}`,
        value: v.id,
      })),
  ]
})

// The backend returns the full series; clip to the chosen time window here.
const sliced = computed(() => {
  const d = chartData.value
  if (!d) return { dates: [] as string[], total: [] as number[], selected: null as number[] | null }
  let start = 0
  if (chartRange.value !== 'all') {
    start = Math.max(0, d.dates.length - Number(chartRange.value))
  }
  return {
    dates: d.dates.slice(start),
    total: d.total.slice(start),
    selected: d.selected ? d.selected.slice(start) : null,
  }
})

const chartSeries = computed(() => {
  const s = sliced.value
  if (!s.dates.length) return []
  const series: { name: string; color: string; values: number[] }[] = [
    { name: t('adminApp.chartTotal'), color: 'rgb(var(--v-theme-primary))', values: s.total },
  ]
  if (s.selected) {
    series.push({ name: t('adminApp.chartSelected'), color: 'rgb(var(--v-theme-warning))', values: s.selected })
  }
  return series
})

async function loadChart() {
  if (!data.value) return
  chartLoading.value = true
  chartError.value = ''
  const id = data.value.app.id
  const params: { platform?: string; version_id?: number } = {}
  if (chartFilterPlatform.value !== 'all') params.platform = chartFilterPlatform.value
  if (chartFilterVersion.value !== 'all') params.version_id = chartFilterVersion.value as number
  try {
    chartData.value = await api.appDownloads(id, params)
  } catch (e) {
    chartError.value = (e as Error).message
    chartData.value = null
  } finally {
    chartLoading.value = false
  }
}

const tab = ref<'overview' | 'versions' | 'stats'>('overview')
const deleteTarget = ref<Version | null>(null)

// Reload the trend chart when entering the stats tab or changing a filter.
// Registered after `tab` is declared (watch evaluates its source eagerly).
watch([chartFilterPlatform, chartFilterVersion, chartRange, () => data.value?.app.id], () => {
  if (tab.value === 'stats') loadChart()
})
watch(
  () => tab.value,
  (v) => {
    if (v === 'stats') loadChart()
  }
)

// Selecting a specific version in the chart filter shows that version's
// download detail (cards + recent logs); resetting to "All" clears it.
watch(chartFilterVersion, async (id) => {
  if (!data.value) return
  stats.value = null
  const v = data.value.versions.find((x) => x.id === id)
  if (!v) return
  try {
    const raw = await api.versionStats(v.id)
    stats.value = {
      download_count: raw.download_count,
      install_count: raw.install_count,
      recent: raw.recent as Array<{ ip: string; user_agent: string; created_at: string }>,
    }
  } catch (e) {
    error.value = (e as Error).message
  }
})

const dialogOpen = ref(false)
const snackbar = ref('')
const snackbarOpen = ref(false)

// --- Overview: app info ---
const infoName = ref('')
const infoDescription = ref('')
const infoIcon = ref<File | null>(null)
const infoIconPreview = ref('')
const infoError = ref('')
const infoSaving = ref(false)

// --- Overview: screenshots ---
const shotsInput = ref<File[]>([])
const shotsUploading = ref(false)
const shotsError = ref('')

// --- Overview: access permission (app-level, applies to all versions) ---
const accessMode = ref<'public' | 'password' | 'expiry'>('public')
const accessPassword = ref('')
const accessExpiresAt = ref('')
const accessError = ref('')
const accessSaving = ref(false)

function toLocalInput(iso: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`
}

function syncOverview() {
  const app = data.value?.app
  if (!app) return
  infoName.value = app.name
  infoDescription.value = app.description ?? ''
  infoIcon.value = null
  infoIconPreview.value = app.icon || ''
  accessMode.value = app.access_mode || 'public'
  accessPassword.value = ''
  accessExpiresAt.value = toLocalInput(app.expires_at ?? null)
}

function onInfoIconChange(f: File | File[] | null) {
  const file = Array.isArray(f) ? f[0] ?? null : f
  infoIcon.value = file
  infoIconPreview.value = file ? URL.createObjectURL(file) : (data.value?.app.icon ?? '')
}

async function saveInfo() {
  if (!data.value) return
  if (!infoName.value.trim()) {
    infoError.value = t('admin.nameRequired')
    return
  }
  infoError.value = ''
  infoSaving.value = true
  const id = data.value.app.id
  try {
    await api.updateApp(id, { name: infoName.value.trim(), description: infoDescription.value })
    if (infoIcon.value) await api.uploadAppIcon(id, infoIcon.value)
    await load()
    syncOverview()
    showSnack(t('adminApp.infoSaved'))
  } catch (e) {
    infoError.value = (e as Error).message
  } finally {
    infoSaving.value = false
  }
}

async function onScreenshotsChange(files: File[] | File | null) {
  const list = Array.isArray(files) ? files : files ? [files] : []
  if (!data.value || !list.length) return
  shotsError.value = ''
  shotsUploading.value = true
  const id = data.value.app.id
  try {
    for (const f of list) {
      await api.uploadAppScreenshot(id, f)
    }
    await load()
    showSnack(t('adminApp.screenshotUploaded'))
  } catch (e) {
    shotsError.value = (e as Error).message
  } finally {
    shotsUploading.value = false
    shotsInput.value = []
  }
}

async function deleteScreenshot(url: string) {
  if (!data.value) return
  try {
    await api.deleteAppScreenshot(data.value.app.id, url)
    await load()
    showSnack(t('adminApp.screenshotDeleted'))
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function saveAccess() {
  if (!data.value) return
  if (accessMode.value === 'password' && !accessPassword.value) {
    accessError.value = t('adminApp.publishRequired')
    return
  }
  accessError.value = ''
  accessSaving.value = true
  const id = data.value.app.id
  const payload: Partial<AppItem> & { password?: string } = { access_mode: accessMode.value }
  if (accessMode.value === 'password') payload.password = accessPassword.value
  if (accessMode.value === 'expiry' && accessExpiresAt.value) {
    payload.expires_at = new Date(accessExpiresAt.value).toISOString()
  }
  try {
    await api.updateApp(id, payload)
    await load()
    syncOverview()
    showSnack(t('adminApp.accessSaved'))
  } catch (e) {
    accessError.value = (e as Error).message
  } finally {
    accessSaving.value = false
  }
}

// --- Publish dialog (visibility only; access is app-level in Overview) ---
const publishDialogOpen = ref(false)
const publishTarget = ref<Version | null>(null)
const publishError = ref('')
const publishLoading = ref(false)

function openPublish(v: Version) {
  publishTarget.value = v
  publishError.value = ''
  publishDialogOpen.value = true
}

function closePublish() {
  publishTarget.value = null
  publishDialogOpen.value = false
}

async function submitPublish() {
  const v = publishTarget.value
  if (!v) return
  publishError.value = ''
  publishLoading.value = true
  try {
    await api.updateVersion(v.id, { published: true, enabled: true })
    closePublish()
    await load()
    showSnack(t('adminApp.statusPublished'))
  } catch (e) {
    publishError.value = (e as Error).message
  } finally {
    publishLoading.value = false
  }
}

onMounted(load)
async function load() {
  const id = Number(route.params.id)
  try {
    data.value = await api.adminAppDetail(id)
    syncOverview()
  } catch (e) {
    error.value = (e as Error).message
  }
}

function askDelete(v: Version) {
  deleteTarget.value = v
  dialogOpen.value = true
}

function cancelDelete() {
  dialogOpen.value = false
  deleteTarget.value = null
}

async function deleteVersion() {
  if (!deleteTarget.value) return
  const id = deleteTarget.value.id
  cancelDelete()
  try {
    await api.deleteVersion(id, true)
    await load()
    showSnack(t('adminApp.versionDeleted'))
  } catch (e) {
    error.value = (e as Error).message
  }
}

// Draft → open publish dialog; published+enabled → take down;
// published+disabled → re-enable.
function onMainAction(v: Version) {
  if (!v.published) { openPublish(v); return }
  if (v.enabled) takeDown(v)
  else reEnable(v)
}

async function takeDown(v: Version) {
  try {
    await api.updateVersion(v.id, { enabled: false })
    await load()
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function reEnable(v: Version) {
  try {
    await api.updateVersion(v.id, { enabled: true })
    await load()
  } catch (e) {
    error.value = (e as Error).message
  }
}

function statusColor(v: Version): string {
  if (!v.published) return 'grey'
  return v.enabled ? 'success' : 'error'
}

function statusLabel(v: Version): string {
  if (!v.published) return t('adminApp.statusDraft')
  return v.enabled ? t('adminApp.statusPublished') : t('adminApp.statusTakenDown')
}

function actionLabel(v: Version): string {
  if (!v.published) return t('adminApp.publish')
  return v.enabled ? t('adminApp.takeDown') : t('adminApp.rePublish')
}

// Newest first by creation time.
const versions = computed(() =>
  [...(data.value?.versions ?? [])].sort(
    (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  )
)

// --- Version filters ---
const statusFilter = ref<'all' | 'draft' | 'published' | 'taken_down'>('all')
const releaseFilter = ref<'all' | ReleaseType>('all')
const platformFilter = ref<'all' | Platform>('all')

const statusFilterItems = computed(() => [
  { title: t('adminApp.filterAll'), value: 'all' },
  { title: t('adminApp.statusDraft'), value: 'draft' },
  { title: t('adminApp.statusPublished'), value: 'published' },
  { title: t('adminApp.statusTakenDown'), value: 'taken_down' },
])

const releaseFilterItems = computed(() => [
  { title: t('adminApp.filterAll'), value: 'all' },
  { title: t('release.production'), value: 'production' },
  { title: t('release.beta'), value: 'beta' },
  { title: t('release.canary'), value: 'canary' },
])

const platformFilterItems = computed(() => [
  { title: t('adminApp.filterAll'), value: 'all' },
  ...PLATFORMS.map((p) => ({ title: t('platform.' + p), value: p })),
])

function archLabel(arch: string): string {
  return formatArch(t, arch)
}

const filteredVersions = computed(() =>
  versions.value.filter((v) => {
    if (statusFilter.value === 'draft' && v.published) return false
    if (statusFilter.value === 'published' && !(v.published && v.enabled)) return false
    if (statusFilter.value === 'taken_down' && v.enabled) return false
    if (releaseFilter.value !== 'all' && v.release_type !== releaseFilter.value) return false
    if (platformFilter.value !== 'all' && v.platform !== platformFilter.value) return false
    return true
  })
)

// Effective download access for every version, set once at the app level.
const appAccessMode = computed(() => data.value?.app.access_mode || 'public')

function showSnack(msg: string) {
  snackbar.value = msg
  snackbarOpen.value = true
}

function goUpload() {
  if (!data.value) return
  router.push(`/admin/upload?app_id=${data.value.app.id}`)
}

// Public download page for the app, shown on the Overview tab. Name-based:
// `/app/{name}` — no numeric id in the shareable link.
const downloadLink = computed(() =>
  data.value ? `${location.origin}/app/${encodeURIComponent(data.value.app.name)}` : ''
)

async function copyLink() {
  if (!downloadLink.value) return
  try {
    await navigator.clipboard.writeText(downloadLink.value)
    showSnack(t('adminApp.linkCopied'))
  } catch {
    showSnack(downloadLink.value)
  }
}

function fmtSize(n: number): string {
  if (n > 1024 * 1024 * 1024) return (n / 1024 / 1024 / 1024).toFixed(2) + ' GB'
  if (n > 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MB'
  if (n > 1024) return (n / 1024).toFixed(1) + ' KB'
  return n + ' B'
}


function releaseColor(rt: string): string {
  if (rt === 'beta') return 'info'
  if (rt === 'canary') return 'warning'
  return 'primary'
}

function accessColor(mode: string, published: boolean, enabled: boolean): string {
  if (!published) return 'grey'
  if (!enabled) return 'error'
  if (mode === 'public') return 'success'
  if (mode === 'password' || mode === 'expiry') return 'warning'
  return 'grey'
}

function accessLabel(mode: string, published: boolean, enabled: boolean): string {
  if (!published) return t('adminApp.statusDraft')
  if (!enabled) return t('detail.takenDown')
  return t(`access.${mode}`)
}

const versionHeaders = computed(() => [
  { title: t('adminApp.colVersion'), key: 'version_name' },
  { title: t('adminApp.colAppName'), key: 'app_name' },
  { title: t('adminApp.colPackage'), key: 'package_name' },
  { title: t('adminApp.colPlatform'), key: 'platform' },
  { title: t('adminApp.colRelease'), key: 'release_type' },
  { title: t('adminApp.colSize'), key: 'file_size' },
  { title: t('adminApp.colAccess'), key: 'access_mode' },
  { title: t('adminApp.colDownloads'), key: 'download_count' },
  { title: t('adminApp.colStatus'), key: 'enabled' },
  { title: '', key: 'actions', sortable: false, align: 'end' as const },
])

const statsHeaders = computed(() => [
  { title: t('adminApp.colTime'), key: 'created_at' },
  { title: t('adminApp.colIp'), key: 'ip' },
  { title: t('adminApp.colUserAgent'), key: 'user_agent' },
])
</script>

<template>
  <v-container class="pa-6" max-width="1200">
    <div v-if="data" class="d-flex align-center justify-space-between mb-2">
      <div class="d-flex align-center" style="gap: 12px;">
        <v-avatar v-if="data.app.icon" :image="data.app.icon" size="40" />
        <v-avatar v-else color="primary" size="40">
          <span class="text-h6">{{ data.app.name.charAt(0).toUpperCase() }}</span>
        </v-avatar>
        <h1 class="text-h4 mb-0">{{ data.app.name }}</h1>
      </div>
      <v-btn color="primary" variant="flat" @click="goUpload">
        {{ t('adminApp.upload') }}
      </v-btn>
    </div>

    <v-alert v-if="error" type="error" variant="tonal" class="mb-4" closable>
      {{ error }}
    </v-alert>

    <v-tabs v-model="tab" class="mt-4">
      <v-tab value="overview">{{ t('adminApp.tabOverview') }}</v-tab>
      <v-tab value="versions">{{ t('adminApp.tabVersions') }}</v-tab>
      <v-tab value="stats">{{ t('adminApp.tabStats') }}</v-tab>
    </v-tabs>

    <v-divider />

    <v-window v-model="tab" class="mt-6">
      <v-window-item value="overview">
        <v-row>
          <!-- App info -->
          <v-col cols="12" md="6">
            <v-card variant="tonal" class="pa-4 h-100">
              <div class="text-subtitle-1 font-weight-medium mb-3">{{ t('adminApp.overviewInfo') }}</div>
              <div class="d-flex align-center mb-3" style="gap: 12px;">
                <v-avatar v-if="infoIconPreview" :image="infoIconPreview" size="56" />
                <v-avatar v-else color="primary" size="56">
                  <span class="text-h5">{{ (infoName || data?.app.name || '?').charAt(0).toUpperCase() }}</span>
                </v-avatar>
                <v-file-input
                  :model-value="infoIcon"
                  :label="t('admin.appIcon')"
                  accept="image/*"
                  density="compact"
                  hide-details
                  class="flex-grow-1"
                  @update:model-value="onInfoIconChange"
                />
              </div>
              <v-text-field v-model="infoName" :label="t('admin.appName')" density="compact" />
              <v-textarea
                v-model="infoDescription"
                :label="t('adminApp.appDescription')"
                density="compact"
                auto-grow
                rows="2"
              />
              <v-alert v-if="infoError" type="error" variant="tonal" density="compact" class="mt-2">
                {{ infoError }}
              </v-alert>
              <div class="d-flex justify-end mt-3">
                <v-btn
                  color="primary"
                  variant="flat"
                  :loading="infoSaving"
                  :disabled="!infoName.trim()"
                  @click="saveInfo"
                >
                  {{ t('common.save') }}
                </v-btn>
              </div>
            </v-card>
          </v-col>

          <!-- Download link + access permission -->
          <v-col cols="12" md="6">
            <v-card variant="tonal" class="pa-4 h-100">
              <div class="text-subtitle-1 font-weight-medium mb-3">{{ t('adminApp.downloadLink') }}</div>
              <div class="d-flex align-center" style="gap: 8px;">
                <v-text-field
                  :model-value="downloadLink"
                  readonly
                  variant="outlined"
                  density="compact"
                  hide-details
                  class="flex-grow-1"
                />
                <v-btn color="primary" variant="flat" @click="copyLink">
                  <v-icon start :icon="mdiContentCopy" />
                  {{ t('adminApp.copyLink') }}
                </v-btn>
              </div>

              <v-divider class="my-4" />
              <div class="text-subtitle-1 font-weight-medium mb-3">{{ t('upload.access') }}</div>
              <v-radio-group v-model="accessMode" hide-details>
                <v-radio :label="t('upload.accessPublic')" value="public" />
                <v-radio :label="t('upload.accessPassword')" value="password" />
                <v-radio :label="t('upload.accessExpiry')" value="expiry" />
              </v-radio-group>
              <v-text-field
                v-if="accessMode === 'password'"
                v-model="accessPassword"
                :label="t('upload.downloadPassword')"
                type="password"
                density="compact"
                class="mt-3"
              />
              <v-text-field
                v-if="accessMode === 'expiry'"
                v-model="accessExpiresAt"
                :label="t('upload.expiresAt')"
                type="datetime-local"
                density="compact"
                class="mt-3"
              />
              <v-alert v-if="accessError" type="error" variant="tonal" density="compact" class="mt-2">
                {{ accessError }}
              </v-alert>
              <div class="d-flex justify-end mt-3">
                <v-btn color="primary" variant="flat" :loading="accessSaving" @click="saveAccess">
                  {{ t('common.save') }}
                </v-btn>
              </div>
            </v-card>
          </v-col>

          <!-- Screenshots -->
          <v-col cols="12">
            <v-card variant="tonal" class="pa-4">
              <div class="d-flex align-center justify-space-between mb-3" style="gap: 12px; flex-wrap: wrap;">
                <div class="text-subtitle-1 font-weight-medium">{{ t('adminApp.overviewScreenshots') }}</div>
                <v-file-input
                  v-model="shotsInput"
                  :label="t('adminApp.uploadScreenshots')"
                  accept="image/*"
                  multiple
                  density="compact"
                  hide-details
                  style="max-width: 320px;"
                  :loading="shotsUploading"
                  @update:model-value="onScreenshotsChange"
                />
              </div>
              <v-alert v-if="shotsError" type="error" variant="tonal" density="compact" class="mb-2">
                {{ shotsError }}
              </v-alert>
              <v-row v-if="data && data.app.screenshots.length">
                <v-col v-for="url in data.app.screenshots" :key="url" cols="6" sm="4" md="3">
                  <v-card>
                    <v-img :src="url" aspect-ratio="9/16" class="rounded" />
                    <v-card-actions>
                      <v-spacer />
                      <v-btn variant="text" size="small" color="error" @click="deleteScreenshot(url)">
                        {{ t('common.delete') }}
                      </v-btn>
                    </v-card-actions>
                  </v-card>
                </v-col>
              </v-row>
              <div v-else class="text-center text-medium-emphasis pa-4">
                {{ t('adminApp.noScreenshots') }}
              </div>
            </v-card>
          </v-col>
        </v-row>
      </v-window-item>

      <v-window-item value="versions">
        <div class="d-flex align-center mb-4" style="gap: 12px; flex-wrap: wrap;">
          <v-select
            v-model="statusFilter"
            :items="statusFilterItems"
            :label="t('adminApp.colStatus')"
            density="compact"
            hide-details
            style="max-width: 160px;"
          />
          <v-select
            v-model="releaseFilter"
            :items="releaseFilterItems"
            :label="t('adminApp.colRelease')"
            density="compact"
            hide-details
            style="max-width: 160px;"
          />
          <v-select
            v-model="platformFilter"
            :items="platformFilterItems"
            :label="t('adminApp.colPlatform')"
            density="compact"
            hide-details
            style="max-width: 200px;"
          />
          <v-btn
            variant="text"
            size="small"
            @click="statusFilter = 'all'; releaseFilter = 'all'; platformFilter = 'all'"
          >
            {{ t('adminApp.filterReset') }}
          </v-btn>
        </div>
        <v-data-table
          :items="filteredVersions"
          :headers="versionHeaders"
          :items-per-page="-1"
        >
          <template #item.version_name="{ item }">
            <div class="d-flex align-center">
              <v-avatar v-if="item.icon_url" :image="item.icon_url" size="28" class="me-2" />
              <v-avatar v-else color="primary" size="28" class="me-2">
                <span class="text-caption">{{ (item.app_name || item.version_name || '?').charAt(0).toUpperCase() }}</span>
              </v-avatar>
              <code>{{ item.version_name }}</code>
              <span class="text-caption text-medium-emphasis ml-2">{{ t('detail.code') }} {{ item.version_code }}</span>
            </div>
          </template>
          <template #item.app_name="{ item }">
            <span v-if="item.app_name">{{ item.app_name }}</span>
            <span v-else class="text-medium-emphasis">—</span>
          </template>
          <template #item.package_name="{ item }">
            <code v-if="item.package_name" class="text-caption">{{ item.package_name }}</code>
            <span v-else class="text-medium-emphasis">—</span>
          </template>
          <template #item.platform="{ item }">
            <v-chip v-if="item.platform" size="x-small" variant="outlined">
              {{ t('platform.' + item.platform) }}
            </v-chip>
            <span v-if="item.arch" class="text-caption text-medium-emphasis ml-1">
              {{ archLabel(item.arch) }}
            </span>
            <span v-if="!item.platform && !item.arch" class="text-medium-emphasis">—</span>
          </template>
          <template #item.release_type="{ item }">
            <v-chip
              v-if="item.release_type"
              size="x-small"
              variant="tonal"
              :color="releaseColor(item.release_type)"
            >
              {{ t('release.' + item.release_type) }}
            </v-chip>
          </template>
          <template #item.file_size="{ item }">
            <code class="text-caption">{{ fmtSize(item.file_size) }}</code>
          </template>
          <template #item.access_mode="{ item }">
            <v-chip
              :color="accessColor(appAccessMode, item.published, item.enabled)"
              size="small"
              variant="tonal"
            >
              {{ accessLabel(appAccessMode, item.published, item.enabled) }}
            </v-chip>
          </template>
          <template #item.download_count="{ item }">
            {{ item.download_count }}
          </template>
          <template #item.enabled="{ item }">
            <v-chip :color="statusColor(item)" size="small" variant="tonal">
              {{ statusLabel(item) }}
            </v-chip>
          </template>
          <template #item.actions="{ item }">
            <v-btn
              variant="text"
              size="small"
              :color="item.published && item.enabled ? '' : 'primary'"
              @click="onMainAction(item)"
            >
              {{ actionLabel(item) }}
            </v-btn>
            <v-btn
              variant="text"
              size="small"
              color="error"
              @click="askDelete(item)"
            >
              {{ t('common.delete') }}
            </v-btn>
          </template>
        </v-data-table>
      </v-window-item>

      <v-window-item value="stats">
        <v-card variant="tonal" class="pa-4 mb-4">
          <div class="d-flex align-center justify-space-between mb-3" style="gap: 12px; flex-wrap: wrap;">
            <div class="text-subtitle-1 font-weight-medium">{{ t('adminApp.chartTitle') }}</div>
            <div class="d-flex align-center" style="gap: 12px; flex-wrap: wrap;">
              <v-select
                v-model="chartFilterPlatform"
                :items="platformFilterItems"
                :label="t('adminApp.colPlatform')"
                density="compact"
                hide-details
                style="max-width: 160px;"
              />
              <v-select
                v-model="chartFilterVersion"
                :items="chartVersionItems"
                :label="t('adminApp.colVersion')"
                density="compact"
                hide-details
                style="max-width: 220px;"
              />
              <v-select
                v-model="chartRange"
                :items="rangeItems"
                :label="t('adminApp.chartRange')"
                density="compact"
                hide-details
                style="max-width: 160px;"
              />
            </div>
          </div>
          <v-progress-linear v-if="chartLoading" indeterminate color="primary" class="mb-2" />
          <v-alert v-else-if="chartError" type="error" variant="tonal" density="compact" class="mb-2">
            {{ chartError }}
          </v-alert>
          <LineChart
            :dates="sliced.dates"
            :series="chartSeries"
            :empty-text="t('adminApp.chartEmpty')"
          />
          <div v-if="chartSeries.length" class="d-flex align-center mt-2" style="gap: 16px;">
            <div v-for="s in chartSeries" :key="s.name" class="d-flex align-center" style="gap: 6px;">
              <span
                class="d-inline-block rounded-circle"
                style="width: 10px; height: 10px;"
                :style="{ background: s.color }"
              />
              <span class="text-caption">{{ s.name }}</span>
            </div>
          </div>
        </v-card>

        <v-card v-if="!stats" variant="tonal" class="text-center pa-8">
          <v-card-text>{{ t('adminApp.statsEmpty') }}</v-card-text>
        </v-card>
        <template v-else>
          <v-row class="mb-4">
            <v-col cols="6" md="3">
              <v-card variant="tonal" color="primary">
                <v-card-text>
                  <div class="text-overline">{{ t('adminApp.statDownloads') }}</div>
                  <div class="text-h4">{{ stats.download_count }}</div>
                </v-card-text>
              </v-card>
            </v-col>
            <v-col cols="6" md="3">
              <v-card variant="tonal" color="primary">
                <v-card-text>
                  <div class="text-overline">{{ t('adminApp.statInstalls') }}</div>
                  <div class="text-h4">{{ stats.install_count }}</div>
                </v-card-text>
              </v-card>
            </v-col>
          </v-row>
          <v-data-table
            :items="stats.recent"
            :headers="statsHeaders"
            :items-per-page="-1"
          >
            <template #item.created_at="{ item }">
              <code class="text-caption">{{ fmtDate(item.created_at) }}</code>
            </template>
            <template #item.ip="{ item }">
              <code class="text-caption">{{ item.ip }}</code>
            </template>
            <template #item.user_agent="{ item }">
              <code class="text-caption" style="max-width: 400px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; display: inline-block;">{{ item.user_agent }}</code>
            </template>
          </v-data-table>
          <v-btn class="mt-4" variant="text" @click="chartFilterVersion = 'all'">{{ t('adminApp.statsClear') }}</v-btn>
        </template>
      </v-window-item>
    </v-window>

    <v-dialog v-model="publishDialogOpen" max-width="440">
      <v-card>
        <v-card-title>{{ t('adminApp.publishTitle') }}</v-card-title>
        <v-card-text>
          <div class="text-body-2">
            <code>{{ publishTarget?.version_name }}</code>
            <span class="text-medium-emphasis"> · {{ t('detail.code') }} {{ publishTarget?.version_code }}</span>
          </div>
          <p class="text-body-2 text-medium-emphasis mt-2 mb-0">{{ t('adminApp.publishHint') }}</p>
          <v-alert v-if="publishError" type="error" variant="tonal" density="compact" class="mt-2">
            {{ publishError }}
          </v-alert>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="closePublish">{{ t('common.cancel') }}</v-btn>
          <v-btn
            color="primary"
            variant="flat"
            :loading="publishLoading"
            @click="submitPublish"
          >
            {{ t('adminApp.publish') }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="dialogOpen" max-width="400">
      <v-card>
        <v-card-title>{{ t('common.confirmDelete') }}</v-card-title>
        <v-card-text>
          <span v-html="t('adminApp.confirmDeleteVersion', { name: deleteTarget?.version_name ?? '' })" />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="cancelDelete">{{ t('common.cancel') }}</v-btn>
          <v-btn color="error" variant="flat" @click="deleteVersion">{{ t('common.delete') }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-snackbar v-model="snackbarOpen" :timeout="2000">
      {{ snackbar }}
    </v-snackbar>
  </v-container>
</template>
