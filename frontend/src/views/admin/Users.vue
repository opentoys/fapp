<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../../api/client'
import { useAuth } from '../../composables/useAuth'
import type { User } from '../../api/types'

const { isAuthed } = useAuth()
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
    error.value = 'Username and password are required.'
    return
  }
  error.value = ''
  try {
    await api.createUser({ username: newUsername.value, password: newPassword.value })
    closeCreate()
    await load()
    showSnack('User created')
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
    showSnack('User deleted')
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
      <h1 class="text-h4">Users</h1>
      <v-btn
        v-if="isAuthed"
        color="primary"
        variant="flat"
        @click="openCreate"
      >
        New user
      </v-btn>
    </div>

    <v-alert v-if="error" type="error" variant="tonal" class="mb-4" closable>
      {{ error }}
    </v-alert>

    <v-card variant="outlined" class="mb-4">
      <v-card-text>
        <div class="text-overline mb-1">Super-admin</div>
        <div class="text-caption text-medium-emphasis">
          The super-admin is configured in <code>config.json</code> and is
          not stored in the database. Actions performed by the super-admin
          are recorded with operator id <code>-1</code>.
        </div>
      </v-card-text>
    </v-card>

    <v-data-table
      :items="users"
      :headers="[
        { title: 'Username', key: 'username' },
        { title: 'Created', key: 'created_at' },
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
          Delete
        </v-btn>
      </template>
    </v-data-table>

    <v-dialog v-model="createDialogOpen" max-width="480">
      <v-card>
        <v-card-title>New user</v-card-title>
        <v-card-text>
          <v-text-field
            v-model="newUsername"
            label="Username"
            autofocus
            :error="!!error"
          />
          <v-text-field
            v-model="newPassword"
            label="Password"
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
          <v-btn @click="closeCreate">Cancel</v-btn>
          <v-btn
            color="primary"
            variant="flat"
            :disabled="!newUsername || !newPassword"
            @click="confirmCreate"
          >
            Create
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="dialogOpen" max-width="400">
      <v-card>
        <v-card-title>Confirm delete</v-card-title>
        <v-card-text>
          Delete user <code>{{ deleteTarget?.username }}</code>?
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
