<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../../api/client'
import type { AppItem } from '../../api/types'
import MonoText from '../../components/MonoText.vue'

const apps = ref<AppItem[]>([])
const name = ref('')
const error = ref('')
const deleteTarget = ref<AppItem | null>(null)
const dialogOpen = ref(false)
const snackbar = ref('')
const snackbarOpen = ref(false)

onMounted(load)
async function load() {
  try {
    apps.value = await api.adminApps()
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function create() {
  if (!name.value) return
  try {
    await api.createApp({ name: name.value })
    name.value = ''
    await load()
    showSnack('App created')
  } catch (e) {
    error.value = (e as Error).message
  }
}

function askDelete(item: AppItem) {
  deleteTarget.value = item
  dialogOpen.value = true
}

function cancelDelete() {
  dialogOpen.value = false
  deleteTarget.value = null
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  const id = deleteTarget.value.id
  cancelDelete()
  try {
    await api.deleteApp(id)
    await load()
    showSnack('App deleted')
  } catch (e) {
    error.value = (e as Error).message
  }
}

function showSnack(msg: string) {
  snackbar.value = msg
  snackbarOpen.value = true
}

function fmtDate(s: string): string {
  return new Date(s).toISOString().replace('T', ' ').slice(0, 19)
}
</script>

<template>
  <div class="admin">
    <div class="page-header">
      <div class="eyebrow">▌ ADMIN</div>
      <h1 class="title">Applications</h1>
    </div>

    <v-alert v-if="error" type="error" variant="outlined" class="mb-4">
      {{ error }}
    </v-alert>

    <div class="create-row">
      <v-text-field
        v-model="name"
        label="New application name"
        density="comfortable"
        hide-details
        @keyup.enter="create"
      />
      <v-btn color="primary" :disabled="!name" @click="create">Create</v-btn>
    </div>

    <v-data-table
      :items="apps"
      :headers="[
        { title: 'Name', key: 'name' },
        { title: 'Created', key: 'created_at' },
        { title: '', key: 'actions', sortable: false, align: 'end' },
      ]"
      class="mt-6"
      hide-default-footer
      :items-per-page="-1"
    >
      <template #item.name="{ item }">
        <router-link :to="`/admin/app/${item.id}`" class="name-link">
          {{ item.name }}
        </router-link>
      </template>
      <template #item.created_at="{ item }">
        <MonoText muted>{{ fmtDate(item.created_at) }}</MonoText>
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

    <v-dialog v-model="dialogOpen" max-width="400">
      <v-card class="pa-5">
        <div class="eyebrow">▌ CONFIRM DELETE</div>
        <p class="dialog-body">
          Delete <b>{{ deleteTarget?.name }}</b>? Associated versions and channels will be removed.
        </p>
        <div class="dialog-actions">
          <v-btn variant="text" @click="cancelDelete">Cancel</v-btn>
          <v-btn color="error" @click="confirmDelete">Delete</v-btn>
        </div>
      </v-card>
    </v-dialog>

    <v-snackbar v-model="snackbarOpen" :timeout="2000">
      {{ snackbar }}
    </v-snackbar>
  </div>
</template>

<style scoped>
.admin {
  max-width: var(--max-w);
  margin: 0 auto;
  padding: var(--sp-8) var(--sp-6);
}
.page-header {
  margin-bottom: var(--sp-6);
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
.create-row {
  display: flex;
  gap: var(--sp-2);
  align-items: start;
  max-width: 600px;
}
.name-link {
  color: var(--accent);
  font-weight: 500;
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
