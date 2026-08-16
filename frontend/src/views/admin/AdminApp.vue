<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Copy, Loader2, Upload as UploadIcon, X } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { api, fileURL, uploadViaURL } from '../../api/client'
import { useAuth } from '../../composables/useAuth'
import { useI18n } from '../../composables/useI18n'
import { formatArch } from '../../constants/platform'
import { fmtDate } from '../../utils/format'
import LineChart from '../../components/LineChart.vue'
import { Button } from '../../components/ui/button'
import { Avatar } from '../../components/ui/avatar'
import { Badge } from '../../components/ui/badge'
import { Card, CardContent, CardTitle } from '../../components/ui/card'
import { Alert } from '../../components/ui/alert'
import { AlertDialog } from '../../components/ui/alert-dialog'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Textarea } from '../../components/ui/textarea'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/tabs'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, TableEmpty } from '../../components/ui/table'
import { Separator } from '../../components/ui/separator'
import { Skeleton } from '../../components/ui/skeleton'
import AppSelect from '../../components/AppSelect.vue'
import FileUpload from '../../components/FileUpload.vue'
import type { AppDetail, AppItem, DownloadsTimeSeries, ReleaseType, User, Version } from '../../api/types'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const { isSuperAdmin } = useAuth()

// --- Members: which users can manage this app (tag list + picker) ---
const allUsers = ref<User[]>([])
const memberIds = ref<number[]>([])
const memberDirty = ref(false)
const membersError = ref('')
const membersSaving = ref(false)

// Users not yet added, shown in the trailing picker.
const pickableUsers = computed(() =>
  allUsers.value.filter((u) => !memberIds.value.includes(u.id))
)
const memberName = (id: number) =>
  allUsers.value.find((u) => u.id === id)?.username ?? String(id)
const data = ref<AppDetail | null>(null)
const stats = ref<{ download_count: number; install_count: number; recent: Array<{ ip: string; user_agent: string; created_at: string }> } | null>(null)
const error = ref('')

// --- Stats: app-level download trend chart, aggregated by day ---
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
  const params: { version_id?: number } = {}
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

const tab = ref<'overview' | 'versions' | 'stats' | 'members'>('overview')
const deleteTarget = ref<Version | null>(null)

// Reload the trend chart when entering the stats tab or changing a filter.
watch([chartFilterVersion, chartRange, () => data.value?.app.id], () => {
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

// --- Overview: access permission (independent password + expiry) ---
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
  infoIconPreview.value = app.icon ? fileURL(app.icon) : ''
  accessPassword.value = ''
  accessExpiresAt.value = toLocalInput(app.expires_at ?? null)
}

function onInfoIconChange(f: File | File[] | null) {
  const file = Array.isArray(f) ? f[0] ?? null : f
  infoIcon.value = file
  infoIconPreview.value = file ? URL.createObjectURL(file) : (data.value?.app.icon ? fileURL(data.value.app.icon) : '')
}

// presignUpload presigns, pushes bytes to the url, and returns the key.
async function presignUpload(presign: () => Promise<{ key: string; url: string }>, file: File): Promise<string> {
  const ticket = await presign()
  await uploadViaURL(ticket.url, file)
  return ticket.key
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
    const icon = infoIcon.value
    if (icon) {
      const key = await presignUpload(
        () => api.presignFile(id, icon.name),
        icon,
      )
      await api.updateApp(id, { icon: key })
      if (data.value) data.value.app.icon = key
    }
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
    const keys: string[] = []
    for (const f of list) {
      const key = await presignUpload(
        () => api.presignFile(id, f.name),
        f,
      )
      keys.push(key)
    }
    await api.updateApp(id, { screenshots: [...(data.value.app.screenshots), ...keys] })
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
  accessError.value = ''
  accessSaving.value = true
  const id = data.value.app.id
  // Password and expiry are independent. Empty password input clears the
  // stored password; empty expiry input clears the download-link expiry.
  const payload: Partial<AppItem> & { password?: string; clear_password?: boolean; clear_expiry?: boolean } = {}
  if (accessPassword.value) {
    payload.password = accessPassword.value
  } else {
    payload.clear_password = true
  }
  if (accessExpiresAt.value) {
    payload.expires_at = new Date(accessExpiresAt.value).toISOString()
  } else {
    payload.clear_expiry = true
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

// --- Overview: publish / take down the whole app ---
const publishSaving = ref(false)

async function togglePublish() {
  if (!data.value || publishSaving.value) return
  publishSaving.value = true
  const app = data.value.app
  const next = !app.published
  try {
    await api.updateApp(app.id, { published: next })
    await load()
    syncOverview()
    showSnack(next ? t('adminApp.appPublished') : t('adminApp.appUnpublished'))
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    publishSaving.value = false
  }
}

onMounted(load)

// Router reuses this component when only the :id param changes (browser
// back/forward across apps), so reload and reset per-app state.
watch(
  () => route.params.id,
  () => {
    chartData.value = null
    chartError.value = ''
    stats.value = null
    error.value = ''
    load()
  }
)

// Load member list when entering the members tab.
watch(
  () => tab.value,
  (v) => {
    if (v === 'members') loadMembers()
  }
)

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

// The app's single current version; only it is publicly downloadable.
function isCurrent(v: Version): boolean {
  return !!data.value && data.value.app.current_version_id === v.id
}

async function setCurrent(v: Version) {
  if (!data.value) return
  try {
    await api.setCurrentVersion(data.value.app.id, v.id)
    await load()
    showSnack(t('adminApp.currentVersionSaved'))
  } catch (e) {
    error.value = (e as Error).message
  }
}

// Newest first by creation time.
const versions = computed(() =>
  [...(data.value?.versions ?? [])].sort(
    (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  )
)

// --- Version filters ---
const releaseFilter = ref<'all' | ReleaseType>('all')

const releaseFilterItems = computed(() => [
  { title: t('adminApp.filterAll'), value: 'all' },
  { title: t('release.production'), value: 'production' },
  { title: t('release.beta'), value: 'beta' },
  { title: t('release.canary'), value: 'canary' },
])

function archLabel(arch: string): string {
  return formatArch(t, arch)
}

const filteredVersions = computed(() =>
  versions.value.filter((v) => {
    if (releaseFilter.value !== 'all' && v.release_type !== releaseFilter.value) return false
    return true
  })
)

function releaseBadge(rt: string): 'default' | 'info' | 'warning' {
  if (rt === 'beta') return 'info'
  if (rt === 'canary') return 'warning'
  return 'default'
}

async function loadMembers() {
  if (!data.value) return
  const id = data.value.app.id
  membersError.value = ''
  try {
    const [users, members] = await Promise.all([api.adminUsers(), api.appMembers(id)])
    allUsers.value = users
    memberIds.value = members
    memberDirty.value = false
  } catch (e) {
    membersError.value = (e as Error).message
  }
}

function addMember(id: string | number | null) {
  if (id === null) return
  const n = Number(id)
  if (memberIds.value.includes(n)) return
  memberIds.value.push(n)
  memberDirty.value = true
}

function removeMember(id: number) {
  const i = memberIds.value.indexOf(id)
  if (i >= 0) memberIds.value.splice(i, 1)
  memberDirty.value = true
}

async function saveMembers() {
  if (!data.value || membersSaving.value) return
  membersSaving.value = true
  membersError.value = ''
  try {
    memberIds.value = await api.setAppMembers(data.value.app.id, memberIds.value)
    memberDirty.value = false
    showSnack(t('adminApp.membersSaved'))
  } catch (e) {
    membersError.value = (e as Error).message
  } finally {
    membersSaving.value = false
  }
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
        <Avatar :src="fileURL(data.app.icon)" :fallback="data.app.name.charAt(0).toUpperCase()" class="size-10" />
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
        <TabsTrigger value="members">{{ t('adminApp.tabManage') }}</TabsTrigger>
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
            <div class="grid gap-1">
              <Label>{{ t('adminApp.appid') }}</Label>
              <div class="text-sm">
                <code v-if="data?.app.appid" class="text-xs">{{ data?.app.appid }}</code>
                <span v-else class="text-muted-foreground text-xs">{{ t('adminApp.appidUnlocked') }}</span>
              </div>
            </div>
            <div class="grid gap-2">
              <Label for="info-description">{{ t('adminApp.appDescription') }}</Label>
              <Textarea id="info-description" v-model="infoDescription" rows="2" />
            </div>
            <Alert v-if="infoError" variant="destructive">{{ infoError }}</Alert>
            <div class="flex justify-end">
              <Button :disabled="!infoName.trim() || infoSaving" @click="saveInfo">
            <Loader2 v-if="infoSaving" class="size-4 animate-spin" />
            {{ t('common.save') }}
          </Button>
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
          <!-- Download password (independent of expiry). Empty = removed. -->
          <div class="mb-4 grid gap-2">
            <Label for="access-password">{{ t('upload.downloadPassword') }}</Label>
            <Input id="access-password" v-model="accessPassword" type="password" />
            <p class="text-muted-foreground text-xs">{{ t('access.passwordHint') }}</p>
          </div>
          <!-- Download-link expiry (independent of password). Past expiry hides the app. -->
          <div class="grid gap-2">
            <Label for="access-expires-at">{{ t('upload.expiresAt') }}</Label>
            <Input id="access-expires-at" v-model="accessExpiresAt" type="datetime-local" />
            <p class="text-muted-foreground text-xs">{{ t('access.expiryHint') }}</p>
          </div>
          <Alert v-if="accessError" variant="destructive" class="mt-2">{{ accessError }}</Alert>
          <div class="mt-3 flex justify-end">
            <Button :disabled="accessSaving" @click="saveAccess">{{ t('common.save') }}</Button>
          </div>
        </Card>

        <!-- Publish status -->
        <Card class="p-5 md:col-span-2">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <CardTitle class="text-base mb-1">{{ t('adminApp.overviewPublish') }}</CardTitle>
              <div class="text-muted-foreground flex items-center gap-2 text-sm">
                <Badge :variant="data?.app.published ? 'success' : 'secondary'">
                  {{ data?.app.published ? t('adminApp.appPublished') : t('adminApp.appUnpublished') }}
                </Badge>
                <span>{{ t('adminApp.publishHint') }}</span>
              </div>
            </div>
            <Button v-if="!data?.app.published" :disabled="publishSaving" @click="togglePublish">
              {{ t('adminApp.publishApp') }}
            </Button>
            <Button v-else variant="destructive" :disabled="publishSaving" @click="togglePublish">
              {{ t('adminApp.takeDownApp') }}
            </Button>
          </div>
        </Card>

        <!-- Screenshots -->
        <Card class="p-5 md:col-span-2">
          <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
            <CardTitle class="text-base">{{ t('adminApp.overviewScreenshots') }}</CardTitle>
            <FileUpload :label="t('adminApp.uploadScreenshots')" accept="image/*" multiple :loading="shotsUploading" :disabled="shotsUploading" @change="onScreenshotsChange" />
          </div>
          <Alert v-if="shotsError" variant="destructive" class="mb-2">{{ shotsError }}</Alert>
          <div v-if="data && data.app.screenshots.length" class="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4">
            <div v-for="url in data.app.screenshots" :key="url" class="overflow-hidden rounded-lg border">
              <img :src="fileURL(url)" class="aspect-[9/16] w-full object-cover" />
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
          <AppSelect v-model="releaseFilter" :items="releaseFilterItems" class="w-40" />
          <Button variant="ghost" size="sm" @click="releaseFilter = 'all'">{{ t('adminApp.filterReset') }}</Button>
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
                  <TableHead>{{ t('adminApp.colDownloads') }}</TableHead>
                  <TableHead class="text-right"> </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="v in filteredVersions" :key="v.id">
                  <TableCell>
                    <div class="flex items-center gap-2">
                      <Avatar :src="fileURL(v.icon_url)" :fallback="(v.app_name || v.version_name || '?').charAt(0).toUpperCase()" class="size-7" />
                      <code>{{ v.version_name }}</code>
                      <Badge v-if="isCurrent(v)" variant="success">{{ t('adminApp.currentVersion') }}</Badge>
                      <span class="text-muted-foreground text-xs">{{ t('detail.code') }} {{ v.version_code }}</span>
                    </div>
                  </TableCell>
                  <TableCell>{{ v.app_name || '—' }}</TableCell>
                  <TableCell><code v-if="v.appid" class="text-xs">{{ v.appid }}</code><span v-else class="text-muted-foreground">—</span></TableCell>
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
                  <TableCell>{{ v.download_count }}</TableCell>
                  <TableCell class="text-right">
                    <Button v-if="!isCurrent(v)" variant="ghost" size="sm" class="text-primary" @click="setCurrent(v)">{{ t('adminApp.setCurrentVersion') }}</Button>
                    <Button v-else variant="ghost" size="sm" disabled>{{ t('adminApp.currentVersion') }}</Button>
                    <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="askDelete(v)">{{ t('common.delete') }}</Button>
                  </TableCell>
                </TableRow>
                <TableEmpty v-if="!filteredVersions.length" :colspan="8">{{ t('adminApp.versionsEmpty') }}</TableEmpty>
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

      <TabsContent value="members" class="mt-6">
        <Card class="p-5">
          <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
            <CardTitle class="text-base">{{ t('adminApp.membersTitle') }}</CardTitle>
            <span v-if="!isSuperAdmin" class="text-muted-foreground text-sm">{{ t('adminApp.membersSuperHint') }}</span>
          </div>
          <Alert v-if="membersError" variant="destructive" class="mb-2">{{ membersError }}</Alert>
          <div class="flex flex-wrap items-center gap-2">
            <template v-for="id in memberIds" :key="id">
              <Badge variant="secondary" class="gap-1 pr-1">
                <code class="text-xs">{{ memberName(id) }}</code>
                <Button
                  v-if="isSuperAdmin"
                  variant="ghost"
                  size="icon"
                  class="text-muted-foreground hover:text-destructive size-5"
                  :title="t('common.remove')"
                  @click="removeMember(id)"
                >
                  <X class="size-3" />
                </Button>
              </Badge>
            </template>
            <AppSelect
              v-if="isSuperAdmin && pickableUsers.length"
              :items="pickableUsers.map((u) => ({ title: u.username, value: u.id }))"
              :placeholder="t('adminApp.membersPick')"
              class="w-40"
              @update:model-value="addMember"
            />
          </div>
          <p v-if="!memberIds.length && !membersError" class="text-muted-foreground py-2 text-sm">
            {{ t('adminApp.membersEmpty') }}
          </p>
          <div class="mt-4 flex justify-end">
            <Button :disabled="!isSuperAdmin || !memberDirty || membersSaving" @click="saveMembers">{{ t('common.save') }}</Button>
          </div>
        </Card>
      </TabsContent>
    </Tabs>

    <!-- Delete version dialog -->
    <AlertDialog v-model:open="dialogOpen" :title="t('common.confirmDelete')" :description="t('adminApp.confirmDeleteVersion', { name: deleteTarget?.version_name ?? '' })">
      <template #footer>
        <Button variant="outline" @click="cancelDelete">{{ t('common.cancel') }}</Button>
        <Button variant="destructive" @click="deleteVersion">{{ t('common.delete') }}</Button>
      </template>
    </AlertDialog>
  </div>
</template>
