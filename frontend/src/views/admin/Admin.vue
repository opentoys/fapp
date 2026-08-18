<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { Plus, Upload } from 'lucide-vue-next'
import { api, fileURL, sha256Hex, uploadViaURL } from '../../api/client'
import { useI18n } from '../../composables/useI18n'
import { PLATFORMS } from '../../constants/platform'
import { fmtDate } from '../../utils/format'
import { Button } from '../../components/ui/button'
import { Avatar } from '../../components/ui/avatar'
import { Badge } from '../../components/ui/badge'
import { Card, CardContent, CardFooter } from '../../components/ui/card'
import { Alert } from '../../components/ui/alert'
import { Dialog } from '../../components/ui/dialog'
import { AlertDialog } from '../../components/ui/alert-dialog'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Separator } from '../../components/ui/separator'
import AppSelect from '../../components/AppSelect.vue'
import FileUpload from '../../components/FileUpload.vue'
import type { AppItem, Platform } from '../../api/types'

const router = useRouter()
const { t } = useI18n()
const apps = ref<AppItem[]>([])
const error = ref('')

const newName = ref('')
const newPlatform = ref<Platform | ''>('')
const createIcon = ref<File | null>(null)
const createIconPreview = ref('')
const createError = ref('')
const creating = ref(false)
const createDialogOpen = ref(false)

const platformItems = computed(() =>
  PLATFORMS.map((p) => ({ title: t('platform.' + p), value: p }))
)

const editTarget = ref<AppItem | null>(null)
const editDialogOpen = ref(false)
const editName = ref('')
const editIcon = ref<File | null>(null)
const editIconPreview = ref('')
const editError = ref('')
const editing = ref(false)

const deleteTarget = ref<AppItem | null>(null)
const deleteDialogOpen = ref(false)

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
  newPlatform.value = ''
  createIcon.value = null
  createIconPreview.value = ''
  createError.value = ''
  createDialogOpen.value = true
}

function onCreateIcon(file: File | File[]) {
  const f = Array.isArray(file) ? file[0] : file
  createIcon.value = f
  createIconPreview.value = URL.createObjectURL(f)
}

async function confirmCreate() {
  if (creating.value) return
  if (!newName.value.trim()) {
    createError.value = t('admin.nameRequired')
    return
  }
  if (!newPlatform.value) {
    createError.value = t('admin.platformRequired')
    return
  }
  creating.value = true
  createError.value = ''
  try {
    const app = await api.createApp({ name: newName.value.trim(), platform: newPlatform.value })
    if (createIcon.value) {
      try {
        const sha = await sha256Hex(createIcon.value)
        const ticket = await api.presignFile(app.id, createIcon.value.name, sha, createIcon.value.size)
        await uploadViaURL(ticket.url, createIcon.value)
        await api.updateApp(app.id, { icon: ticket.key })
      } catch (e) {
        toast((e as Error).message)
      }
    }
    createDialogOpen.value = false
    router.push(`/admin/app/${app.id}`)
  } catch (e) {
    createError.value = (e as Error).message
  } finally {
    creating.value = false
  }
}

function openEdit(a: AppItem) {
  editTarget.value = a
  editName.value = a.name
  editIcon.value = null
  editIconPreview.value = a.icon ? fileURL(a.icon) : ''
  editError.value = ''
  editDialogOpen.value = true
}

function onEditIcon(file: File | File[]) {
  const f = Array.isArray(file) ? file[0] : file
  editIcon.value = f
  editIconPreview.value = URL.createObjectURL(f)
}

async function confirmEdit() {
  if (editing.value) return
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
    if (editIcon.value) {
      const sha = await sha256Hex(editIcon.value)
      const ticket = await api.presignFile(id, editIcon.value.name, sha, editIcon.value.size)
      await uploadViaURL(ticket.url, editIcon.value)
      await api.updateApp(id, { icon: ticket.key })
    }
    await load()
    editDialogOpen.value = false
    toast(t('admin.appUpdated'))
  } catch (e) {
    editError.value = (e as Error).message
  } finally {
    editing.value = false
  }
}

function askDelete(item: AppItem) {
  deleteTarget.value = item
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  const id = deleteTarget.value.id
  deleteTarget.value = null
  deleteDialogOpen.value = false
  try {
    await api.deleteApp(id)
    await load()
    toast(t('admin.appDeleted'))
  } catch (e) {
    toast((e as Error).message)
  }
}
</script>

<template>
  <div class="mx-auto max-w-6xl px-4 py-8 sm:px-6">
    <div class="mb-6 flex items-center justify-between">
      <h1 class="text-2xl font-semibold tracking-tight">{{ t('admin.title') }}</h1>
      <div class="flex gap-2">
        <Button variant="outline" @click="router.push('/admin/upload')">
          <Upload class="size-4" />
          {{ t('admin.uploadApp') }}
        </Button>
        <Button @click="openCreate">
          <Plus class="size-4" />
          {{ t('admin.newApp') }}
        </Button>
      </div>
    </div>

    <Alert v-if="error" variant="destructive" class="mb-4">{{ error }}</Alert>

    <div v-if="apps.length" class="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
      <Card v-for="a in apps" :key="a.id" class="gap-0">
        <div class="flex flex-1 cursor-pointer flex-col gap-3 p-5" @click="router.push(`/admin/app/${a.id}`)">
          <div class="flex items-center gap-3">
            <Avatar :src="fileURL(a.icon)" :fallback="a.name.charAt(0).toUpperCase()" class="size-12" />
            <span class="truncate font-semibold">{{ a.name }}</span>
          </div>
          <div v-if="a.platform" class="flex items-center gap-2">
            <Badge variant="outline" class="text-xs">{{ t('platform.' + a.platform) }}</Badge>
          </div>
          <p v-if="a.description" class="text-muted-foreground text-sm">{{ a.description }}</p>
        </div>
        <Separator />
        <CardFooter class="justify-between py-3">
          <span class="text-muted-foreground text-xs">{{ fmtDate(a.created_at) }}</span>
          <div class="flex gap-1">
            <Button variant="ghost" size="sm" @click="openEdit(a)">{{ t('common.edit') }}</Button>
            <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="askDelete(a)">{{ t('common.delete') }}</Button>
          </div>
        </CardFooter>
      </Card>
    </div>

    <Card v-else-if="!error" class="text-center">
      <CardContent class="py-12 text-muted-foreground">{{ t('admin.empty') }}</CardContent>
    </Card>

    <!-- Create dialog -->
    <Dialog v-model:open="createDialogOpen" :title="t('admin.newApp')" max-width="md">
      <div class="grid gap-4">
        <div class="grid gap-2">
          <Label for="app-name">{{ t('admin.appName') }}</Label>
          <Input id="app-name" v-model="newName" autofocus @keyup.enter="confirmCreate" />
        </div>
        <div class="grid gap-2">
          <Label for="app-platform">{{ t('admin.platform') }}</Label>
          <AppSelect id="app-platform" v-model="newPlatform" :items="platformItems" :placeholder="t('admin.platform')" />
        </div>
        <div class="flex items-center gap-3">
          <Avatar :src="createIconPreview" :fallback="(newName || '?').charAt(0).toUpperCase()" class="size-12" />
          <div class="flex-1">
            <FileUpload :label="t('admin.appIcon')" accept="image/*" @change="onCreateIcon" />
          </div>
        </div>
        <Alert v-if="createError" variant="destructive">{{ createError }}</Alert>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="createDialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button :disabled="!newName.trim() || !newPlatform || creating" @click="confirmCreate">{{ t('common.create') }}</Button>
        </div>
      </div>
    </Dialog>

    <!-- Edit dialog -->
    <Dialog v-model:open="editDialogOpen" :title="t('admin.editApp')" max-width="md">
      <div class="grid gap-4">
        <div class="grid gap-2">
          <Label for="edit-name">{{ t('admin.appName') }}</Label>
          <Input id="edit-name" v-model="editName" autofocus @keyup.enter="confirmEdit" />
        </div>
        <div class="grid gap-2">
          <Label>{{ t('admin.platform') }}</Label>
          <div class="text-sm">
            <Badge v-if="editTarget?.platform" variant="outline">{{ t('platform.' + editTarget.platform) }}</Badge>
            <span class="text-muted-foreground text-xs">{{ t('admin.platformImmutable') }}</span>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <Avatar :src="editIconPreview" :fallback="(editName || '?').charAt(0).toUpperCase()" class="size-12" />
          <div class="flex-1">
            <FileUpload :label="t('admin.appIcon')" accept="image/*" @change="onEditIcon" />
          </div>
        </div>
        <Alert v-if="editError" variant="destructive">{{ editError }}</Alert>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="editDialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button :disabled="!editName.trim() || editing" @click="confirmEdit">{{ t('common.save') }}</Button>
        </div>
      </div>
    </Dialog>

    <!-- Delete dialog -->
    <AlertDialog v-model:open="deleteDialogOpen" :title="t('common.confirmDelete')" :description="t('admin.confirmDeleteApp', { name: deleteTarget?.name ?? '' })">
      <template #footer>
        <Button variant="outline" @click="deleteDialogOpen = false">{{ t('common.cancel') }}</Button>
        <Button variant="destructive" @click="confirmDelete">{{ t('common.delete') }}</Button>
      </template>
    </AlertDialog>
  </div>
</template>
