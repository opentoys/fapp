<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../../api/client'
import type { AppItem } from '../../api/types'

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
  <v-container class="pa-6" max-width="1200">
    <h1 class="text-h4 mb-6">Applications</h1>

    <v-alert v-if="error" type="error" variant="tonal" class="mb-4" closable>
      {{ error }}
    </v-alert>

    <div class="d-flex align-start mb-6" style="gap: 8px; max-width: 600px;">
      <v-text-field
        v-model="name"
        label="New application name"
        density="comfortable"
        hide-details
        @keyup.enter="create"
      />
      <v-btn color="primary" variant="flat" :disabled="!name" @click="create">Create</v-btn>
    </div>

    <v-data-table
      :items="apps"
      :headers="[
        { title: 'Name', key: 'name' },
        { title: 'Created', key: 'created_at' },
        { title: '', key: 'actions', sortable: false, align: 'end' },
      ]"
      :items-per-page="-1"
    >
      <template #item.name="{ item }">
        <router-link :to="`/admin/app/${item.id}`" class="text-primary font-weight-medium">
          {{ item.name }}
        </router-link>
      </template>
      <template #item.created_at="{ item }">
        <code class="text-caption">{{ fmtDate(item.created_at) }}</code>
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
      <v-card>
        <v-card-title>Confirm delete</v-card-title>
        <v-card-text>
          Delete <b>{{ deleteTarget?.name }}</b>? Associated versions and channels will be removed.
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="cancelDelete">Cancel</v-btn>
          <v-btn color="error" variant="flat" @click="confirmDelete">Delete</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-snackbar v-model="snackbarOpen" :timeout="2000">
      {{ snackbar }}
    </v-snackbar>
  </v-container>
</template>
