<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../../api/client'
import type { AppDetail, Channel, Version } from '../../api/types'
import StatusDot from '../../components/StatusDot.vue'
import MonoText from '../../components/MonoText.vue'

const route = useRoute()
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

function fmtSize(n: number): string {
  if (n > 1024 * 1024 * 1024) return (n / 1024 / 1024 / 1024).toFixed(2) + ' GB'
  if (n > 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MB'
  if (n > 1024) return (n / 1024).toFixed(1) + ' KB'
  return n + ' B'
}

function fmtDate(s: string): string {
  return new Date(s).toISOString().replace('T', ' ').slice(0, 19)
}
</script>

<template>
  <div class="admin-app">
    <div v-if="data" class="page-header">
      <div class="eyebrow">▌ ADMIN</div>
      <h1 class="title">{{ data.app.name }}</h1>
    </div>

    <v-alert v-if="error" type="error" variant="outlined" class="mb-4">
      {{ error }}
    </v-alert>

    <v-tabs v-model="tab" class="mt-4">
      <v-tab value="versions">Versions</v-tab>
      <v-tab value="channels">Channels</v-tab>
      <v-tab value="stats">Stats</v-tab>
    </v-tabs>

    <v-divider />

    <!-- Versions tab -->
    <div v-if="tab === 'versions'" class="tab-body">
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
        hide-default-footer
        :items-per-page="-1"
      >
        <template #item.version_name="{ item }">
          <MonoText>{{ item.version_name }}</MonoText>
          <MonoText muted> · code {{ item.version_code }}</MonoText>
          <v-btn variant="text" size="small" class="ms-2" @click="loadStats(item)">Stats</v-btn>
        </template>
        <template #item.file_size="{ item }">
          <MonoText muted>{{ fmtSize(item.file_size) }}</MonoText>
        </template>
        <template #item.access_mode="{ item }">
          <StatusDot :mode="item.enabled ? item.access_mode : 'taken_down'" />
        </template>
        <template #item.download_count="{ item }">
          <MonoText>{{ item.download_count }}</MonoText>
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
    </div>

    <!-- Channels tab -->
    <div v-else-if="tab === 'channels'" class="tab-body">
      <div class="create-row">
        <v-text-field
          v-model="newChannelName"
          label="New channel name"
          density="comfortable"
          hide-details
          @keyup.enter="createChannel"
        />
        <v-btn color="primary" :disabled="!newChannelName" @click="createChannel">
          Create
        </v-btn>
      </div>
      <v-data-table
        :items="channels"
        :headers="[
          { title: 'Name', key: 'name' },
          { title: 'ID', key: 'id' },
        ]"
        class="mt-6"
        hide-default-footer
        :items-per-page="-1"
      >
        <template #item.name="{ item }">
          <MonoText>{{ item.name }}</MonoText>
        </template>
        <template #item.id="{ item }">
          <MonoText muted>{{ item.id }}</MonoText>
        </template>
      </v-data-table>
    </div>

    <!-- Stats tab -->
    <div v-else-if="tab === 'stats'" class="tab-body">
      <div v-if="!stats" class="empty">
        <p>Click "Stats" next to a version in the Versions tab to view stats.</p>
      </div>
      <div v-else>
        <div class="stat-strip">
          <div class="stat-cell">
            <div class="stat-label">Downloads</div>
            <div class="stat-value">{{ stats.download_count }}</div>
          </div>
          <div class="stat-cell">
            <div class="stat-label">Installs</div>
            <div class="stat-value">{{ stats.install_count }}</div>
          </div>
        </div>
        <div class="eyebrow mt-6">▌ RECENT</div>
        <v-data-table
          :items="stats.recent"
          :headers="[
            { title: 'Time', key: 'created_at' },
            { title: 'IP', key: 'ip' },
            { title: 'User Agent', key: 'user_agent' },
          ]"
          class="mt-2"
          hide-default-footer
          :items-per-page="-1"
        >
          <template #item.created_at="{ item }">
            <MonoText muted>{{ fmtDate(item.created_at) }}</MonoText>
          </template>
          <template #item.ip="{ item }">
            <MonoText>{{ item.ip }}</MonoText>
          </template>
          <template #item.user_agent="{ item }">
            <MonoText muted class="ua-cell">{{ item.user_agent }}</MonoText>
          </template>
        </v-data-table>
        <v-btn class="mt-4" variant="text" @click="stats = null">Clear</v-btn>
      </div>
    </div>

    <v-dialog v-model="dialogOpen" max-width="400">
      <v-card class="pa-5">
        <div class="eyebrow">▌ CONFIRM DELETE</div>
        <p class="dialog-body">
          Delete version <MonoText>{{ deleteTarget?.version_name }}</MonoText>
          and its storage file?
        </p>
        <div class="dialog-actions">
          <v-btn variant="text" @click="cancelDelete">Cancel</v-btn>
          <v-btn color="error" @click="deleteVersion">Delete</v-btn>
        </div>
      </v-card>
    </v-dialog>

    <v-snackbar v-model="snackbarOpen" :timeout="2000">
      {{ snackbar }}
    </v-snackbar>
  </div>
</template>

<style scoped>
.admin-app {
  max-width: var(--max-w);
  margin: 0 auto;
  padding: var(--sp-8) var(--sp-6);
}
.page-header {
  margin-bottom: var(--sp-4);
}
.eyebrow {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: var(--text-mute);
  margin-bottom: var(--sp-2);
}
.title {
  font-size: 2.25rem;
  font-weight: 500;
  letter-spacing: -0.02em;
  margin: 0;
}
.tab-body {
  padding: var(--sp-6) 0;
}
.create-row {
  display: flex;
  gap: var(--sp-2);
  align-items: start;
  max-width: 500px;
}
.stat-strip {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--sp-2);
  max-width: 400px;
}
.stat-cell {
  border: 1px solid var(--border);
  background: var(--surface);
  padding: var(--sp-3) var(--sp-4);
}
.stat-label {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  letter-spacing: 0.15em;
  text-transform: uppercase;
  color: var(--text-mute);
  margin-bottom: var(--sp-1);
}
.stat-value {
  font-size: 1.75rem;
  font-weight: 500;
  color: var(--accent);
}
.ua-cell {
  max-width: 400px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: inline-block;
}
.empty {
  padding: var(--sp-8) 0;
  color: var(--text-mute);
  text-align: center;
}
.dialog-body {
  margin: var(--sp-3) 0;
  color: var(--text-mute);
}
.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--sp-2);
}
</style>
