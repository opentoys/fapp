<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { toast } from 'vue-sonner'
import { Plus } from 'lucide-vue-next'
import { api } from '../../api/client'
import { useAuth } from '../../composables/useAuth'
import { useI18n } from '../../composables/useI18n'
import { fmtDate } from '../../utils/format'
import { Button } from '../../components/ui/button'
import { Card, CardContent } from '../../components/ui/card'
import { Alert } from '../../components/ui/alert'
import { Dialog } from '../../components/ui/dialog'
import { AlertDialog } from '../../components/ui/alert-dialog'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../components/ui/table'
import type { User } from '../../api/types'

const { isAuthed } = useAuth()
const { t } = useI18n()
const users = ref<User[]>([])
const error = ref('')

const newUsername = ref('')
const newPassword = ref('')
const createError = ref('')
const createDialogOpen = ref(false)
const creating = ref(false)

const editTarget = ref<User | null>(null)
const editDialogOpen = ref(false)
const editUsername = ref('')
const editPassword = ref('')
const editError = ref('')
const editLoading = ref(false)

const deleteTarget = ref<User | null>(null)
const deleteDialogOpen = ref(false)

onMounted(load)

async function load() {
  try {
    users.value = await api.adminUsers()
    error.value = ''
  } catch (e) {
    error.value = (e as Error).message
  }
}

function openCreate() {
  newUsername.value = ''
  newPassword.value = ''
  createError.value = ''
  createDialogOpen.value = true
}

async function confirmCreate() {
  if (creating.value) return
  if (!newUsername.value || !newPassword.value) {
    createError.value = t('adminUsers.required')
    return
  }
  createError.value = ''
  creating.value = true
  try {
    await api.createUser({ username: newUsername.value, password: newPassword.value })
    createDialogOpen.value = false
    await load()
    toast(t('adminUsers.userCreated'))
  } catch (e) {
    createError.value = (e as Error).message
  } finally {
    creating.value = false
  }
}

function openEdit(u: User) {
  editTarget.value = u
  editUsername.value = u.username
  editPassword.value = ''
  editError.value = ''
  editDialogOpen.value = true
}

async function confirmEdit() {
  if (editLoading.value) return
  if (!editTarget.value) return
  if (!editUsername.value.trim()) {
    editError.value = t('adminUsers.required')
    return
  }
  editError.value = ''
  editLoading.value = true
  try {
    const data: { username?: string; password?: string } = {}
    if (editUsername.value.trim() !== editTarget.value.username) {
      data.username = editUsername.value.trim()
    }
    if (editPassword.value) {
      data.password = editPassword.value
    }
    await api.updateUser(editTarget.value.id, data)
    editDialogOpen.value = false
    await load()
    toast(t('adminUsers.userUpdated'))
  } catch (e) {
    editError.value = (e as Error).message
  } finally {
    editLoading.value = false
  }
}

function askDelete(u: User) {
  deleteTarget.value = u
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  const id = deleteTarget.value.id
  deleteTarget.value = null
  deleteDialogOpen.value = false
  try {
    await api.deleteUser(id)
    await load()
    toast(t('adminUsers.userDeleted'))
  } catch (e) {
    toast((e as Error).message)
  }
}
</script>

<template>
  <div class="mx-auto max-w-4xl px-4 py-8 sm:px-6">
    <div class="mb-6 flex items-center justify-between">
      <h1 class="text-2xl font-semibold tracking-tight">{{ t('adminUsers.title') }}</h1>
      <Button v-if="isAuthed" @click="openCreate">
        <Plus class="size-4" />
        {{ t('adminUsers.newUser') }}
      </Button>
    </div>

    <Alert v-if="error" variant="destructive" class="mb-4">{{ error }}</Alert>

    <Card class="mb-4">
      <CardContent class="py-4 text-sm">
        <div class="mb-1 text-xs font-semibold uppercase tracking-wider">{{ t('adminUsers.superAdminLabel') }}</div>
        <span class="text-muted-foreground" v-html="t('adminUsers.superAdminNote')" />
      </CardContent>
    </Card>

    <Card>
      <CardContent class="!p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('common.username') }}</TableHead>
              <TableHead>{{ t('common.created') }}</TableHead>
              <TableHead class="text-right"> </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="u in users" :key="u.id">
              <TableCell><code>{{ u.username }}</code></TableCell>
              <TableCell><code class="text-xs">{{ fmtDate(u.created_at) }}</code></TableCell>
              <TableCell class="text-right">
                <Button variant="ghost" size="sm" @click="openEdit(u)">{{ t('common.edit') }}</Button>
                <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="askDelete(u)">{{ t('common.delete') }}</Button>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>

    <Dialog v-model:open="createDialogOpen" :title="t('adminUsers.newUser')" max-width="md">
      <div class="grid gap-4">
        <div class="grid gap-2">
          <Label for="users-create-username">{{ t('common.username') }}</Label>
          <Input id="users-create-username" v-model="newUsername" autofocus />
        </div>
        <div class="grid gap-2">
          <Label for="users-create-password">{{ t('common.password') }}</Label>
          <Input id="users-create-password" v-model="newPassword" type="password" @keyup.enter="confirmCreate" />
        </div>
        <Alert v-if="createError" variant="destructive">{{ createError }}</Alert>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="createDialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button :disabled="!newUsername || !newPassword || creating" @click="confirmCreate">{{ t('common.create') }}</Button>
        </div>
      </div>
    </Dialog>

    <Dialog v-model:open="editDialogOpen" :title="t('adminUsers.editUser')" max-width="md">
      <div class="grid gap-4">
        <div class="grid gap-2">
          <Label for="users-edit-username">{{ t('common.username') }}</Label>
          <Input id="users-edit-username" v-model="editUsername" autofocus />
        </div>
        <div class="grid gap-2">
          <Label for="users-edit-password">{{ t('adminUsers.newPassword') }}</Label>
          <Input id="users-edit-password" v-model="editPassword" type="password" @keyup.enter="confirmEdit" />
        </div>
        <Alert v-if="editError" variant="destructive">{{ editError }}</Alert>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="editDialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button :disabled="!editUsername.trim() || editLoading" @click="confirmEdit">{{ t('common.save') }}</Button>
        </div>
      </div>
    </Dialog>

    <AlertDialog v-model:open="deleteDialogOpen" :title="t('common.confirmDelete')" :description="t('adminUsers.confirmDeleteUser', { name: deleteTarget?.username ?? '' })">
      <template #footer>
        <Button variant="outline" @click="deleteDialogOpen = false">{{ t('common.cancel') }}</Button>
        <Button variant="destructive" @click="confirmDelete">{{ t('common.delete') }}</Button>
      </template>
    </AlertDialog>
  </div>
</template>
