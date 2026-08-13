<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../../api/client'
import { useAuth } from '../../composables/useAuth'
import { useI18n } from '../../composables/useI18n'
import type { User } from '../../api/types'

const { isAuthed } = useAuth()
const { t } = useI18n()
const users = ref<User[]>([])
const newUsername = ref('')
const newPassword = ref('')
const error = ref('')
const deleteTarget = ref<User | null>(null)
const dialogOpen = ref(false)
const createDialogOpen = ref(false)
const snackbar = ref('')
const snackbarOpen = ref(false)

onMounted(load)

async function load() {
  try {
    users.value = await api.adminUsers()
  } catch (e) {
    error.value = (e as Error).message
  }
}

function openCreate() {
  newUsername.value = ''
  newPassword.value = ''
  error.value = ''
  createDialogOpen.value = true
}

function closeCreate() {
  createDialogOpen.value = false
}

async function confirmCreate() {
  if (!newUsername.value || !newPassword.value) {
    error.value = t('adminUsers.required')
    return
  }
  error.value = ''
  try {
    await api.createUser({ username: newUsername.value, password: newPassword.value })
    closeCreate()
    await load()
    showSnack(t('adminUsers.userCreated'))
  } catch (e) {
    error.value = (e as Error).message
  }
}

function askDelete(u: User) {
  deleteTarget.value = u
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
    await api.deleteUser(id)
    await load()
    showSnack(t('adminUsers.userDeleted'))
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
  <v-container class="pa-6" max-width="1000">
    <div class="d-flex align-center justify-space-between mb-6">
      <h1 class="text-h4">{{ t('adminUsers.title') }}</h1>
      <v-btn
        v-if="isAuthed"
        color="primary"
        variant="flat"
        @click="openCreate"
      >
        {{ t('adminUsers.newUser') }}
      </v-btn>
    </div>

    <v-alert v-if="error" type="error" variant="tonal" class="mb-4" closable>
      {{ error }}
    </v-alert>

    <v-card variant="outlined" class="mb-4">
      <v-card-text>
        <div class="text-overline mb-1">{{ t('adminUsers.superAdminLabel') }}</div>
        <div class="text-caption text-medium-emphasis" v-html="t('adminUsers.superAdminNote')" />
      </v-card-text>
    </v-card>

    <v-data-table
      :items="users"
      :headers="[
        { title: t('common.username'), key: 'username' },
        { title: t('common.created'), key: 'created_at' },
        { title: '', key: 'actions', sortable: false, align: 'end' },
      ]"
      :items-per-page="-1"
    >
      <template #item.username="{ item }">
        <code>{{ item.username }}</code>
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
        <v-card-title>{{ t('adminUsers.newUser') }}</v-card-title>
        <v-card-text>
          <v-text-field
            v-model="newUsername"
            :label="t('common.username')"
            autofocus
            :error="!!error"
          />
          <v-text-field
            v-model="newPassword"
            :label="t('common.password')"
            type="password"
            :error="!!error"
            @keyup.enter="confirmCreate"
          />
          <v-alert v-if="error" type="error" variant="tonal" density="compact" class="mt-2">
            {{ error }}
          </v-alert>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="closeCreate">{{ t('common.cancel') }}</v-btn>
          <v-btn
            color="primary"
            variant="flat"
            :disabled="!newUsername || !newPassword"
            @click="confirmCreate"
          >
            {{ t('common.create') }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="dialogOpen" max-width="400">
      <v-card>
        <v-card-title>{{ t('common.confirmDelete') }}</v-card-title>
        <v-card-text>
          <span v-html="t('adminUsers.confirmDeleteUser', { name: deleteTarget?.username ?? '' })" />
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
