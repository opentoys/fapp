<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../../api/client'
import type { AppDetail, Channel, Version } from '../../api/types'

const route = useRoute()
const router = useRouter()
const data = ref<AppDetail | null>(null)
const channels = ref<Channel[]>([])
const newChannelName = ref('')
const stats = ref<{ download_count: number; install_count: number; recent: Array<{ ip: string; user_agent: string; created_at: string }> } | null>(null)
const error = ref('')
const tab = ref<'versions' | 'channels' | 'stats'>('versions')
const deleteTarget = ref<Version | null>(null)
const dialogOpen = ref(false)
const snackbar = ref('')
const snackbarOpen = ref(false)

onMounted(load)
async function load() {
  const id = Number(route.params.id)
  try {
    data.value = await api.appDetail(id)
    channels.value = await api.channels(id)
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function createChannel() {
  if (!newChannelName.value || !data.value) return
  try {
    await api.createChannel(data.value.app.id, newChannelName.value)
    newChannelName.value = ''
    channels.value = await api.channels(data.value.app.id)
    showSnack('Channel created')
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
    showSnack('Version deleted')
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function toggleEnabled(v: Version) {
  try {
    await api.updateVersion(v.id, { enabled: !v.enabled })
    await load()
  } catch (e) {
    error.value = (e as Error).message
  }
}

const versions = computed(() => data.value?.versions ?? [])

function showSnack(msg: string) {
  snackbar.value = msg
  snackbarOpen.value = true
}

function goUpload() {
  if (!data.value) return
  router.push(`/admin/upload?app_id=${data.value.app.id}`)
}

function fmtSize(n: number): string {
  if (n > 1024 * 1024 * 1024) return (n / 1024 / 1024 / 1024).toFixed(2) + ' GB'
  if (n > 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MB'
  if (n > 1024) return (n / 1024).toFixed(1) + ' KB'
  return n + ' B'
}

function fmtDate(s: string): string {
  return new Date(s).toISOString().replace('T', ' ').slice(0, 19)
}

function accessColor(mode: string, enabled: boolean): string {
  if (!enabled) return 'error'
  if (mode === 'public') return 'success'
  if (mode === 'password' || mode === 'expiry') return 'warning'
  return 'grey'
}

function accessLabel(mode: string, enabled: boolean): string {
  if (!enabled) return 'taken down'
  return mode
}
</script>

<template>
  <v-container class="pa-6" max-width="1200">
    <div v-if="data" class="d-flex align-center justify-space-between mb-2">
      <h1 class="text-h4">{{ data.app.name }}</h1>
      <v-btn color="primary" variant="flat" @click="goUpload">
        Upload version
      </v-btn>
    </div>

    <v-alert v-if="error" type="error" variant="tonal" class="mb-4" closable>
      {{ error }}
    </v-alert>

    <v-tabs v-model="tab" class="mt-4">
      <v-tab value="versions">Versions</v-tab>
      <v-tab value="channels">Channels</v-tab>
      <v-tab value="stats">Stats</v-tab>
    </v-tabs>

    <v-divider />

    <v-window v-model="tab" class="mt-6">
      <v-window-item value="versions">
        <v-data-table
          :items="versions"
          :headers="[
            { title: 'Version', key: 'version_name' },
            { title: 'Size', key: 'file_size' },
            { title: 'Access', key: 'access_mode' },
            { title: 'Downloads', key: 'download_count' },
            { title: 'Status', key: 'enabled' },
            { title: '', key: 'actions', sortable: false, align: 'end' },
          ]"
          :items-per-page="-1"
        >
          <template #item.version_name="{ item }">
            <code>{{ item.version_name }}</code>
            <span class="text-caption text-medium-emphasis ml-2">code {{ item.version_code }}</span>
            <v-btn variant="text" size="small" class="ms-2" @click="loadStats(item)">Stats</v-btn>
          </template>
          <template #item.file_size="{ item }">
            <code class="text-caption">{{ fmtSize(item.file_size) }}</code>
          </template>
          <template #item.access_mode="{ item }">
            <v-chip
              :color="accessColor(item.access_mode, item.enabled)"
              size="small"
              variant="tonal"
            >
              {{ accessLabel(item.access_mode, item.enabled) }}
            </v-chip>
          </template>
          <template #item.download_count="{ item }">
            {{ item.download_count }}
          </template>
          <template #item.enabled="{ item }">
            <v-btn variant="text" size="small" @click="toggleEnabled(item)">
              {{ item.enabled ? 'Take down' : 'Re-enable' }}
            </v-btn>
          </template>
          <template #item.actions="{ item }">
            <v-btn
              variant="text"
              size="small"
              color="error"
              @click="askDelete(item)"
            >
              Delete
            </v-btn>
          </template>
        </v-data-table>
      </v-window-item>

      <v-window-item value="channels">
        <div class="d-flex align-start mb-6" style="gap: 8px; max-width: 500px;">
          <v-text-field
            v-model="newChannelName"
            label="New channel name"
            density="comfortable"
            hide-details
            @keyup.enter="createChannel"
          />
          <v-btn color="primary" variant="flat" :disabled="!newChannelName" @click="createChannel">
            Create
          </v-btn>
        </div>
        <v-data-table
          :items="channels"
          :headers="[
            { title: 'Name', key: 'name' },
            { title: 'ID', key: 'id' },
          ]"
          :items-per-page="-1"
        >
          <template #item.id="{ item }">
            <code class="text-caption">{{ item.id }}</code>
          </template>
        </v-data-table>
      </v-window-item>

      <v-window-item value="stats">
        <v-card v-if="!stats" variant="tonal" class="text-center pa-8">
          <v-card-text>Click "Stats" next to a version in the Versions tab to view stats.</v-card-text>
        </v-card>
        <template v-else>
          <v-row class="mb-4">
            <v-col cols="6" md="3">
              <v-card variant="tonal" color="primary">
                <v-card-text>
                  <div class="text-overline">Downloads</div>
                  <div class="text-h4">{{ stats.download_count }}</div>
                </v-card-text>
              </v-card>
            </v-col>
            <v-col cols="6" md="3">
              <v-card variant="tonal" color="primary">
                <v-card-text>
                  <div class="text-overline">Installs</div>
                  <div class="text-h4">{{ stats.install_count }}</div>
                </v-card-text>
              </v-card>
            </v-col>
          </v-row>
          <v-data-table
            :items="stats.recent"
            :headers="[
              { title: 'Time', key: 'created_at' },
              { title: 'IP', key: 'ip' },
              { title: 'User Agent', key: 'user_agent' },
            ]"
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
          <v-btn class="mt-4" variant="text" @click="stats = null">Clear</v-btn>
        </template>
      </v-window-item>
    </v-window>

    <v-dialog v-model="dialogOpen" max-width="400">
      <v-card>
        <v-card-title>Confirm delete</v-card-title>
        <v-card-text>
          Delete version <code>{{ deleteTarget?.version_name }}</code>
          and its storage file?
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="cancelDelete">Cancel</v-btn>
          <v-btn color="error" variant="flat" @click="deleteVersion">Delete</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-snackbar v-model="snackbarOpen" :timeout="2000">
      {{ snackbar }}
    </v-snackbar>
  </v-container>
</template>
