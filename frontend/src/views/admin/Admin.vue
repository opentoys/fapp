<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../../api/client'
import { useI18n } from '../../composables/useI18n'
import { fmtDate } from '../../utils/format'
import type { AppItem } from '../../api/types'

const router = useRouter()
const { t } = useI18n()
const apps = ref<AppItem[]>([])
const error = ref('')
const newName = ref('')
const createIcon = ref<File | null>(null)
const createIconPreview = ref('')
const createError = ref('')
const creating = ref(false)
const createDialogOpen = ref(false)
const editTarget = ref<AppItem | null>(null)
const editDialogOpen = ref(false)
const editName = ref('')
const editIcon = ref<File | null>(null)
const editIconPreview = ref('')
const editError = ref('')
const editing = ref(false)
const deleteTarget = ref<AppItem | null>(null)
const deleteDialogOpen = ref(false)
const snackbar = ref('')
const snackbarOpen = ref(false)

onMounted(load)
async function load() {
  try {
    apps.value = await api.adminApps()
    error.value = ''
  } catch (e) {
    error.value = (e as Error).message
  }
}

function openCreate() {
  newName.value = ''
  createIcon.value = null
  createIconPreview.value = ''
  createError.value = ''
  createDialogOpen.value = true
}

function onCreateIconChange(f: File | File[] | null) {
  const file = Array.isArray(f) ? f[0] ?? null : f
  createIcon.value = file
  createIconPreview.value = file ? URL.createObjectURL(file) : ''
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
    // Best-effort: set the icon after creating the app; a failure to upload
    // the icon should not block app creation.
    if (createIcon.value) {
      try {
        await api.uploadAppIcon(app.id, createIcon.value)
      } catch (e) {
        showSnack((e as Error).message)
      }
    }
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

function openEdit(a: AppItem) {
  editTarget.value = a
  editName.value = a.name
  editIcon.value = null
  editIconPreview.value = a.icon || ''
  editError.value = ''
  editDialogOpen.value = true
}

function closeEdit() {
  editDialogOpen.value = false
  editTarget.value = null
}

function onEditIconChange(f: File | File[] | null) {
  const file = Array.isArray(f) ? f[0] ?? null : f
  editIcon.value = file
  editIconPreview.value = file
    ? URL.createObjectURL(file)
    : editTarget.value?.icon || ''
}

async function confirmEdit() {
  if (!editTarget.value) return
  if (!editName.value.trim()) {
    editError.value = t('admin.nameRequired')
    return
  }
  editing.value = true
  editError.value = ''
  const id = editTarget.value.id
  try {
    await api.updateApp(id, { name: editName.value.trim() })
    if (editIcon.value) await api.uploadAppIcon(id, editIcon.value)
    await load()
    closeEdit()
    showSnack(t('admin.appUpdated'))
  } catch (e) {
    editError.value = (e as Error).message
  } finally {
    editing.value = false
  }
}

function showSnack(msg: string) {
  snackbar.value = msg
  snackbarOpen.value = true
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

    <v-row v-if="apps.length">
      <v-col
        v-for="a in apps"
        :key="a.id"
        cols="12"
        sm="6"
        md="4"
        lg="3"
      >
        <v-card hover class="d-flex flex-column h-100">
          <!-- The content area is the click target for opening the detail;
               the action buttons below sit OUTSIDE it, so they never conflict
               with the navigation click. -->
          <div
            class="flex-grow-1"
            style="cursor: pointer;"
            @click="router.push(`/admin/app/${a.id}`)"
          >
            <v-card-text>
              <div class="d-flex align-center mb-2" style="gap: 12px;">
                <v-avatar v-if="a.icon" :image="a.icon" size="48" />
                <v-avatar v-else color="primary" size="48">
                  <span class="text-h6">{{ a.name.charAt(0).toUpperCase() }}</span>
                </v-avatar>
                <span class="text-h6 text-truncate">{{ a.name }}</span>
              </div>
              <p v-if="a.description" class="text-body-2 text-medium-emphasis mb-0">
                {{ a.description }}
              </p>
            </v-card-text>
          </div>
          <v-divider />
          <v-card-actions>
            <span class="text-caption text-medium-emphasis ml-2">
              {{ fmtDate(a.created_at) }}
            </span>
            <v-spacer />
            <v-btn variant="text" size="small" @click="openEdit(a)">
              {{ t('common.edit') }}
            </v-btn>
            <v-btn
              variant="text"
              size="small"
              color="error"
              @click="askDelete(a)"
            >
              {{ t('common.delete') }}
            </v-btn>
          </v-card-actions>
        </v-card>
      </v-col>
    </v-row>

    <v-alert v-if="error" type="error" variant="tonal" class="mb-4">
      {{ error }}
    </v-alert>

    <v-card v-else-if="!apps.length" variant="tonal" class="text-center pa-8">
      <v-card-text>{{ t('admin.empty') }}</v-card-text>
    </v-card>

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
          <div class="d-flex align-center" style="gap: 12px;">
            <v-avatar v-if="createIconPreview" :image="createIconPreview" size="48" />
            <v-avatar v-else color="primary" size="48">
              <v-icon>mdi-image-outline</v-icon>
            </v-avatar>
            <v-file-input
              :model-value="createIcon"
              :label="t('admin.appIcon')"
              accept="image/*"
              density="compact"
              hide-details
              class="flex-grow-1"
              @update:model-value="onCreateIconChange"
            />
          </div>
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

    <v-dialog v-model="editDialogOpen" max-width="480">
      <v-card>
        <v-card-title>{{ t('admin.editApp') }}</v-card-title>
        <v-card-text>
          <v-text-field
            v-model="editName"
            :label="t('admin.appName')"
            autofocus
            :error="!!editError"
            @keyup.enter="confirmEdit"
          />
          <div class="d-flex align-center" style="gap: 12px;">
            <v-avatar v-if="editIconPreview" :image="editIconPreview" size="48" />
            <v-avatar v-else color="primary" size="48">
              <v-icon>mdi-image-outline</v-icon>
            </v-avatar>
            <v-file-input
              :model-value="editIcon"
              :label="t('admin.appIcon')"
              accept="image/*"
              density="compact"
              hide-details
              class="flex-grow-1"
              @update:model-value="onEditIconChange"
            />
          </div>
          <v-alert v-if="editError" type="error" variant="tonal" density="compact" class="mt-2">
            {{ editError }}
          </v-alert>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="closeEdit">{{ t('common.cancel') }}</v-btn>
          <v-btn
            color="primary"
            variant="flat"
            :loading="editing"
            :disabled="!editName.trim()"
            @click="confirmEdit"
          >
            {{ t('common.save') }}
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
