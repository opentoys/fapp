<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { mdiContentCopy } from '@mdi/js'
import { api } from '../../api/client'
import { useI18n } from '../../composables/useI18n'
import { PLATFORMS, formatArch } from '../../constants/platform'
import { fmtDate } from '../../utils/format'
import type { AppDetail, Platform, ReleaseType, Version } from '../../api/types'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const data = ref<AppDetail | null>(null)
const stats = ref<{ download_count: number; install_count: number; recent: Array<{ ip: string; user_agent: string; created_at: string }> } | null>(null)
const error = ref('')
const tab = ref<'versions' | 'stats'>('versions')
const deleteTarget = ref<Version | null>(null)
const dialogOpen = ref(false)
const snackbar = ref('')
const snackbarOpen = ref(false)

// --- Publish dialog ---
const publishDialogOpen = ref(false)
const publishTarget = ref<Version | null>(null)
const publishMode = ref<'public' | 'password' | 'expiry'>('public')
const publishPassword = ref('')
const publishExpiresAt = ref('')
const publishError = ref('')
const publishLoading = ref(false)

function openPublish(v: Version) {
  publishTarget.value = v
  publishMode.value = v.access_mode
  publishPassword.value = ''
  publishExpiresAt.value = ''
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
  if (publishMode.value === 'password' && !publishPassword.value) {
    publishError.value = t('adminApp.publishRequired')
    return
  }
  publishError.value = ''
  publishLoading.value = true
  const data: Partial<Version> & { password?: string } = {
    published: true,
    enabled: true,
    access_mode: publishMode.value,
  }
  if (publishMode.value === 'password') data.password = publishPassword.value
  if (publishMode.value === 'expiry' && publishExpiresAt.value) {
    data.expires_at = new Date(publishExpiresAt.value).toISOString()
  }
  try {
    await api.updateVersion(v.id, data)
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
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function loadStats(v: Version) {
  try {
    const raw = await api.versionStats(v.id)
    stats.value = raw as typeof stats.value
    tab.value = 'stats'
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
// published+disabled → re-enable (reuses its existing access scope).
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

function showSnack(msg: string) {
  snackbar.value = msg
  snackbarOpen.value = true
}

function goUpload() {
  if (!data.value) return
  router.push(`/admin/upload?app_id=${data.value.app.id}`)
}

async function copyLink() {
  if (!data.value) return
  const url = `${location.origin}/app/${data.value.app.id}`
  try {
    await navigator.clipboard.writeText(url)
    showSnack(t('adminApp.linkCopied'))
  } catch {
    showSnack(url)
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
      <v-tab value="versions">{{ t('adminApp.tabVersions') }}</v-tab>
      <v-tab value="stats">{{ t('adminApp.tabStats') }}</v-tab>
    </v-tabs>

    <v-divider />

    <v-window v-model="tab" class="mt-6">
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
            <v-tooltip :text="t('adminApp.copyLink')">
              <template #activator="{ props }">
                <v-btn icon variant="text" size="small" class="ms-1" v-bind="props" @click="copyLink">
                  <v-icon :icon="mdiContentCopy" size="small" />
                </v-btn>
              </template>
            </v-tooltip>
            <v-btn variant="text" size="small" class="ms-1" @click="loadStats(item)">{{ t('adminApp.statsBtn') }}</v-btn>
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
              :color="accessColor(item.access_mode, item.published, item.enabled)"
              size="small"
              variant="tonal"
            >
              {{ accessLabel(item.access_mode, item.published, item.enabled) }}
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
          <v-btn class="mt-4" variant="text" @click="stats = null">{{ t('adminApp.statsClear') }}</v-btn>
        </template>
      </v-window-item>
    </v-window>

    <v-dialog v-model="publishDialogOpen" max-width="480">
      <v-card>
        <v-card-title>{{ t('adminApp.publishTitle') }}</v-card-title>
        <v-card-text>
          <div class="text-body-2 mb-2">
            <code>{{ publishTarget?.version_name }}</code>
            <span class="text-medium-emphasis"> · {{ t('detail.code') }} {{ publishTarget?.version_code }}</span>
          </div>
          <div class="text-overline mb-2">{{ t('upload.access') }}</div>
          <v-radio-group v-model="publishMode">
            <v-radio :label="t('upload.accessPublic')" value="public" />
            <v-radio :label="t('upload.accessPassword')" value="password" />
            <v-radio :label="t('upload.accessExpiry')" value="expiry" />
          </v-radio-group>
          <v-text-field
            v-if="publishMode === 'password'"
            v-model="publishPassword"
            :label="t('upload.downloadPassword')"
            type="password"
          />
          <v-text-field
            v-if="publishMode === 'expiry'"
            v-model="publishExpiresAt"
            :label="t('upload.expiresAt')"
            type="datetime-local"
          />
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
