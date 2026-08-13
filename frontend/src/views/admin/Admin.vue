<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../../api/client'
import { useI18n } from '../../composables/useI18n'
import type { AppItem } from '../../api/types'

const router = useRouter()
const { t } = useI18n()
const apps = ref<AppItem[]>([])
const newName = ref('')
const createError = ref('')
const creating = ref(false)
const createDialogOpen = ref(false)
const deleteTarget = ref<AppItem | null>(null)
const deleteDialogOpen = ref(false)
const snackbar = ref('')
const snackbarOpen = ref(false)

onMounted(load)
async function load() {
  try {
    apps.value = await api.adminApps()
  } catch (e) {
    showSnack((e as Error).message)
  }
}

function openCreate() {
  newName.value = ''
  createError.value = ''
  createDialogOpen.value = true
}

function closeCreate() {
  createDialogOpen.value = false
}

async function confirmCreate() {
  if (!newName.value.trim()) {
    createError.value = t('admin.nameRequired')
    return
  }
  creating.value = true
  createError.value = ''
  try {
    const app = await api.createApp({ name: newName.value.trim() })
    closeCreate()
    router.push(`/admin/app/${app.id}`)
  } catch (e) {
    createError.value = (e as Error).message
  } finally {
    creating.value = false
  }
}

function askDelete(item: AppItem) {
  deleteTarget.value = item
  deleteDialogOpen.value = true
}

function cancelDelete() {
  deleteDialogOpen.value = false
  deleteTarget.value = null
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  const id = deleteTarget.value.id
  cancelDelete()
  try {
    await api.deleteApp(id)
    await load()
    showSnack(t('admin.appDeleted'))
  } catch (e) {
    showSnack((e as Error).message)
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
    <div class="d-flex align-center justify-space-between mb-6">
      <h1 class="text-h4">{{ t('admin.title') }}</h1>
      <v-btn color="primary" variant="flat" @click="openCreate">
        {{ t('admin.newApp') }}
      </v-btn>
    </div>

    <v-data-table
      :items="apps"
      :headers="[
        { title: t('common.name'), key: 'name' },
        { title: t('common.created'), key: 'created_at' },
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
          {{ t('common.delete') }}
        </v-btn>
      </template>
    </v-data-table>

    <v-dialog v-model="createDialogOpen" max-width="480">
      <v-card>
        <v-card-title>{{ t('admin.newApp') }}</v-card-title>
        <v-card-text>
          <v-text-field
            v-model="newName"
            :label="t('admin.appName')"
            autofocus
            :error="!!createError"
            @keyup.enter="confirmCreate"
          />
          <v-alert v-if="createError" type="error" variant="tonal" density="compact" class="mt-2">
            {{ createError }}
          </v-alert>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="closeCreate">{{ t('common.cancel') }}</v-btn>
          <v-btn
            color="primary"
            variant="flat"
            :loading="creating"
            :disabled="!newName.trim()"
            @click="confirmCreate"
          >
            {{ t('common.create') }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="deleteDialogOpen" max-width="400">
      <v-card>
        <v-card-title>{{ t('common.confirmDelete') }}</v-card-title>
        <v-card-text>
          <span v-html="t('admin.confirmDeleteApp', { name: deleteTarget?.name ?? '' })" />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="cancelDelete">{{ t('common.cancel') }}</v-btn>
          <v-btn color="error" variant="flat" @click="confirmDelete">{{ t('common.delete') }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-snackbar v-model="snackbarOpen" :timeout="2000">
      {{ snackbar }}
    </v-snackbar>
  </v-container>
</template>
