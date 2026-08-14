<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Copy, Upload as UploadIcon } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { api } from '../../api/client'
import { useI18n } from '../../composables/useI18n'
import { PLATFORMS, formatArch } from '../../constants/platform'
import { fmtDate } from '../../utils/format'
import LineChart from '../../components/LineChart.vue'
import { Button } from '../../components/ui/button'
import { Avatar } from '../../components/ui/avatar'
import { Badge } from '../../components/ui/badge'
import { Card, CardContent, CardTitle } from '../../components/ui/card'
import { Alert } from '../../components/ui/alert'
import { Dialog } from '../../components/ui/dialog'
import { AlertDialog } from '../../components/ui/alert-dialog'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Textarea } from '../../components/ui/textarea'
import { RadioGroup, RadioGroupItem } from '../../components/ui/radio-group'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/tabs'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, TableEmpty } from '../../components/ui/table'
import { Separator } from '../../components/ui/separator'
import { Skeleton } from '../../components/ui/skeleton'
import AppSelect from '../../components/AppSelect.vue'
import FileUpload from '../../components/FileUpload.vue'
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
    { name: t('adminApp.chartTotal'), color: 'var(--color-primary)', values: s.total },
  ]
  if (s.selected) {
    series.push({ name: t('adminApp.chartSelected'), color: 'var(--color-warning)', values: s.selected })
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
  if (infoSaving.value) return
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
  if (accessSaving.value) return
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
  if (publishLoading.value) return
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
    error.value = ''
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

function statusBadge(v: Version): 'secondary' | 'success' | 'destructive' {
  if (!v.published) return 'secondary'
  return v.enabled ? 'success' : 'destructive'
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

function releaseBadge(rt: string): 'default' | 'info' | 'warning' {
  if (rt === 'beta') return 'info'
  if (rt === 'canary') return 'warning'
  return 'default'
}

function accessBadge(appAccess: string, v: Version): 'secondary' | 'success' | 'warning' | 'destructive' {
  if (!v.published) return 'secondary'
  if (!v.enabled) return 'destructive'
  if (appAccess === 'public') return 'success'
  if (appAccess === 'password' || appAccess === 'expiry') return 'warning'
  return 'secondary'
}

function accessLabel(mode: string, published: boolean, enabled: boolean): string {
  if (!published) return t('adminApp.statusDraft')
  if (!enabled) return t('detail.takenDown')
  return t(`access.${mode}`)
}

function showSnack(msg: string) {
  toast(msg)
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
</script>

<template>
  <div class="mx-auto max-w-6xl px-4 py-8 sm:px-6">
    <div v-if="data" class="mb-6 flex items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <Avatar :src="data.app.icon" :fallback="data.app.name.charAt(0).toUpperCase()" class="size-10" />
        <h1 class="text-2xl font-semibold tracking-tight">{{ data.app.name }}</h1>
      </div>
      <Button @click="goUpload">
        <UploadIcon class="size-4" />
        {{ t('adminApp.upload') }}
      </Button>
    </div>

    <Alert v-if="error" variant="destructive" class="mb-4">{{ error }}</Alert>

    <Tabs v-model="tab" class="mt-4">
      <TabsList>
        <TabsTrigger value="overview">{{ t('adminApp.tabOverview') }}</TabsTrigger>
        <TabsTrigger value="versions">{{ t('adminApp.tabVersions') }}</TabsTrigger>
        <TabsTrigger value="stats">{{ t('adminApp.tabStats') }}</TabsTrigger>
      </TabsList>

      <TabsContent value="overview" class="mt-6 grid grid-cols-1 gap-4 md:grid-cols-2">
        <!-- App info -->
        <Card class="p-5">
          <CardTitle class="text-base mb-4">{{ t('adminApp.overviewInfo') }}</CardTitle>
          <div class="mb-4 flex items-center gap-3">
            <Avatar :src="infoIconPreview" :fallback="(infoName || data?.app.name || '?').charAt(0).toUpperCase()" class="size-14" />
            <div class="flex-1">
              <FileUpload :label="t('admin.appIcon')" accept="image/*" @change="onInfoIconChange" />
            </div>
          </div>
          <div class="grid gap-3">
            <div class="grid gap-2">
              <Label for="info-name">{{ t('admin.appName') }}</Label>
              <Input id="info-name" v-model="infoName" />
            </div>
            <div class="grid gap-2">
              <Label for="info-description">{{ t('adminApp.appDescription') }}</Label>
              <Textarea id="info-description" v-model="infoDescription" rows="2" />
            </div>
            <Alert v-if="infoError" variant="destructive">{{ infoError }}</Alert>
            <div class="flex justify-end">
              <Button :disabled="!infoName.trim() || infoSaving" @click="saveInfo">{{ t('common.save') }}</Button>
            </div>
          </div>
        </Card>

        <!-- Download link + access permission -->
        <Card class="p-5">
          <CardTitle class="text-base mb-4">{{ t('adminApp.downloadLink') }}</CardTitle>
          <div class="flex gap-2">
            <Input :model-value="downloadLink" readonly class="flex-1" />
            <Button @click="copyLink">
              <Copy class="size-4" />
              {{ t('adminApp.copyLink') }}
            </Button>
          </div>
          <Separator class="my-4" />
          <CardTitle class="text-base mb-4">{{ t('upload.access') }}</CardTitle>
          <RadioGroup v-model="accessMode">
            <div class="flex items-center gap-2 text-sm">
              <RadioGroupItem value="public" id="r-public" />
              <Label for="r-public">{{ t('upload.accessPublic') }}</Label>
            </div>
            <div class="flex items-center gap-2 text-sm">
              <RadioGroupItem value="password" id="r-password" />
              <Label for="r-password">{{ t('upload.accessPassword') }}</Label>
            </div>
            <div class="flex items-center gap-2 text-sm">
              <RadioGroupItem value="expiry" id="r-expiry" />
              <Label for="r-expiry">{{ t('upload.accessExpiry') }}</Label>
            </div>
          </RadioGroup>
          <div v-if="accessMode === 'password'" class="mt-3 grid gap-2">
            <Label for="access-password">{{ t('upload.downloadPassword') }}</Label>
            <Input id="access-password" v-model="accessPassword" type="password" />
          </div>
          <div v-if="accessMode === 'expiry'" class="mt-3 grid gap-2">
            <Label for="access-expires-at">{{ t('upload.expiresAt') }}</Label>
            <Input id="access-expires-at" v-model="accessExpiresAt" type="datetime-local" />
          </div>
          <Alert v-if="accessError" variant="destructive" class="mt-2">{{ accessError }}</Alert>
          <div class="mt-3 flex justify-end">
            <Button :disabled="accessSaving" @click="saveAccess">{{ t('common.save') }}</Button>
          </div>
        </Card>

        <!-- Screenshots -->
        <Card class="p-5 md:col-span-2">
          <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
            <CardTitle class="text-base">{{ t('adminApp.overviewScreenshots') }}</CardTitle>
            <FileUpload :label="t('adminApp.uploadScreenshots')" accept="image/*" multiple @change="onScreenshotsChange" />
          </div>
          <Alert v-if="shotsError" variant="destructive" class="mb-2">{{ shotsError }}</Alert>
          <div v-if="data && data.app.screenshots.length" class="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4">
            <div v-for="url in data.app.screenshots" :key="url" class="overflow-hidden rounded-lg border">
              <img :src="url" class="aspect-[9/16] w-full object-cover" />
              <div class="flex justify-end p-1">
                <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="deleteScreenshot(url)">{{ t('common.delete') }}</Button>
              </div>
            </div>
          </div>
          <p v-else class="text-muted-foreground py-4 text-center text-sm">{{ t('adminApp.noScreenshots') }}</p>
        </Card>
      </TabsContent>

      <TabsContent value="versions" class="mt-6">
        <div class="mb-4 flex flex-wrap items-center gap-3">
          <AppSelect v-model="statusFilter" :items="statusFilterItems" class="w-40" />
          <AppSelect v-model="releaseFilter" :items="releaseFilterItems" class="w-40" />
          <AppSelect v-model="platformFilter" :items="platformFilterItems" class="w-48" />
          <Button variant="ghost" size="sm" @click="statusFilter = 'all'; releaseFilter = 'all'; platformFilter = 'all'">{{ t('adminApp.filterReset') }}</Button>
        </div>

        <Card>
          <CardContent class="p-0!">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{{ t('adminApp.colVersion') }}</TableHead>
                  <TableHead>{{ t('adminApp.colAppName') }}</TableHead>
                  <TableHead>{{ t('adminApp.colPackage') }}</TableHead>
                  <TableHead>{{ t('adminApp.colPlatform') }}</TableHead>
                  <TableHead>{{ t('adminApp.colRelease') }}</TableHead>
                  <TableHead>{{ t('adminApp.colSize') }}</TableHead>
                  <TableHead>{{ t('adminApp.colAccess') }}</TableHead>
                  <TableHead>{{ t('adminApp.colDownloads') }}</TableHead>
                  <TableHead>{{ t('adminApp.colStatus') }}</TableHead>
                  <TableHead class="text-right"> </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="v in filteredVersions" :key="v.id">
                  <TableCell>
                    <div class="flex items-center gap-2">
                      <Avatar :src="v.icon_url" :fallback="(v.app_name || v.version_name || '?').charAt(0).toUpperCase()" class="size-7" />
                      <code>{{ v.version_name }}</code>
                      <span class="text-muted-foreground text-xs">{{ t('detail.code') }} {{ v.version_code }}</span>
                    </div>
                  </TableCell>
                  <TableCell>{{ v.app_name || '—' }}</TableCell>
                  <TableCell><code v-if="v.package_name" class="text-xs">{{ v.package_name }}</code><span v-else class="text-muted-foreground">—</span></TableCell>
                  <TableCell>
                    <span v-if="v.platform" class="mr-1">
                      <Badge variant="outline">{{ t('platform.' + v.platform) }}</Badge>
                    </span>
                    <span v-if="v.arch" class="text-muted-foreground text-xs">{{ archLabel(v.arch) }}</span>
                    <span v-if="!v.platform && !v.arch" class="text-muted-foreground">—</span>
                  </TableCell>
                  <TableCell>
                    <Badge v-if="v.release_type" :variant="releaseBadge(v.release_type)">{{ t('release.' + v.release_type) }}</Badge>
                  </TableCell>
                  <TableCell><code class="text-xs">{{ fmtSize(v.file_size) }}</code></TableCell>
                  <TableCell>
                    <Badge :variant="accessBadge(appAccessMode, v)">{{ accessLabel(appAccessMode, v.published, v.enabled) }}</Badge>
                  </TableCell>
                  <TableCell>{{ v.download_count }}</TableCell>
                  <TableCell><Badge :variant="statusBadge(v)">{{ statusLabel(v) }}</Badge></TableCell>
                  <TableCell class="text-right">
                    <Button variant="ghost" size="sm" :class="v.published && v.enabled ? '' : 'text-primary'" @click="onMainAction(v)">{{ actionLabel(v) }}</Button>
                    <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="askDelete(v)">{{ t('common.delete') }}</Button>
                  </TableCell>
                </TableRow>
                <TableEmpty v-if="!filteredVersions.length" :colspan="10">{{ t('adminApp.statsEmpty') }}</TableEmpty>
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="stats" class="mt-6">
        <Card class="mb-4 p-5">
          <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
            <CardTitle class="text-base">{{ t('adminApp.chartTitle') }}</CardTitle>
            <div class="flex flex-wrap items-center gap-3">
              <AppSelect v-model="chartFilterPlatform" :items="platformFilterItems" class="w-40" />
              <AppSelect v-model="chartFilterVersion" :items="chartVersionItems" class="w-52" />
              <AppSelect v-model="chartRange" :items="rangeItems" class="w-40" />
            </div>
          </div>
          <Skeleton v-if="chartLoading" class="mb-2 h-8 w-full" />
          <Alert v-else-if="chartError" variant="destructive" class="mb-2">{{ chartError }}</Alert>
          <LineChart :dates="sliced.dates" :series="chartSeries" :empty-text="t('adminApp.chartEmpty')" />
          <div v-if="chartSeries.length" class="mt-2 flex items-center gap-4">
            <div v-for="s in chartSeries" :key="s.name" class="flex items-center gap-1.5 text-xs">
              <span class="inline-block size-2.5 rounded-full" :style="{ background: s.color }" />
              <span>{{ s.name }}</span>
            </div>
          </div>
        </Card>

        <Card v-if="!stats" class="text-center">
          <CardContent class="py-12 text-muted-foreground">{{ t('adminApp.statsEmpty') }}</CardContent>
        </Card>
        <template v-else>
          <div class="mb-4 grid grid-cols-2 gap-4 md:grid-cols-3">
            <Card>
              <CardContent class="py-4">
                <div class="text-muted-foreground text-xs uppercase tracking-wider">{{ t('adminApp.statDownloads') }}</div>
                <div class="text-3xl font-semibold">{{ stats.download_count }}</div>
              </CardContent>
            </Card>
            <Card>
              <CardContent class="py-4">
                <div class="text-muted-foreground text-xs uppercase tracking-wider">{{ t('adminApp.statInstalls') }}</div>
                <div class="text-3xl font-semibold">{{ stats.install_count }}</div>
              </CardContent>
            </Card>
          </div>
          <Card>
            <CardContent class="p-0!">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{{ t('adminApp.colTime') }}</TableHead>
                    <TableHead>{{ t('adminApp.colIp') }}</TableHead>
                    <TableHead>{{ t('adminApp.colUserAgent') }}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow v-for="(row, i) in stats.recent" :key="i">
                    <TableCell><code class="text-xs">{{ fmtDate(row.created_at) }}</code></TableCell>
                    <TableCell><code class="text-xs">{{ row.ip }}</code></TableCell>
                    <TableCell><code class="text-xs block max-w-[400px] truncate">{{ row.user_agent }}</code></TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </CardContent>
          </Card>
          <Button variant="ghost" class="mt-4" @click="chartFilterVersion = 'all'">{{ t('adminApp.statsClear') }}</Button>
        </template>
      </TabsContent>
    </Tabs>

    <!-- Publish dialog -->
    <Dialog v-model:open="publishDialogOpen" :title="t('adminApp.publishTitle')" max-width="md">
      <div class="grid gap-4">
        <div class="text-sm">
          <code>{{ publishTarget?.version_name }}</code>
          <span class="text-muted-foreground"> · {{ t('detail.code') }} {{ publishTarget?.version_code }}</span>
        </div>
        <p class="text-muted-foreground text-sm">{{ t('adminApp.publishHint') }}</p>
        <Alert v-if="publishError" variant="destructive">{{ publishError }}</Alert>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="closePublish">{{ t('common.cancel') }}</Button>
          <Button :disabled="publishLoading" @click="submitPublish">{{ t('adminApp.publish') }}</Button>
        </div>
      </div>
    </Dialog>

    <!-- Delete version dialog -->
    <AlertDialog v-model:open="dialogOpen" :title="t('common.confirmDelete')" :description="t('adminApp.confirmDeleteVersion', { name: deleteTarget?.version_name ?? '' })">
      <template #footer>
        <Button variant="outline" @click="cancelDelete">{{ t('common.cancel') }}</Button>
        <Button variant="destructive" @click="deleteVersion">{{ t('common.delete') }}</Button>
      </template>
    </AlertDialog>
  </div>
</template>
