<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { toast } from 'vue-sonner'
import { Plus, Copy } from 'lucide-vue-next'
import { api } from '../../api/client'
import { useAuth } from '../../composables/useAuth'
import { useI18n } from '../../composables/useI18n'
import { fmtDate } from '../../utils/format'
import { Button } from '../../components/ui/button'
import { Badge } from '../../components/ui/badge'
import { Card, CardContent } from '../../components/ui/card'
import { Alert } from '../../components/ui/alert'
import { Dialog } from '../../components/ui/dialog'
import { AlertDialog } from '../../components/ui/alert-dialog'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { RadioGroup, RadioGroupItem } from '../../components/ui/radio-group'
import { Select } from '../../components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, TableEmpty } from '../../components/ui/table'
import type { ApiKey, AppItem, KeyScope } from '../../api/types'

interface KeyRow extends ApiKey {
  owner_username?: string
}

const { isSuperAdmin } = useAuth()
const { t } = useI18n()
const keys = ref<KeyRow[]>([])
const managedApps = ref<AppItem[]>([])
const error = ref('')

const createDialogOpen = ref(false)
const createName = ref('')
const createScope = ref<KeyScope>('run')
const createExpires = ref('never')
const createError = ref('')
const creating = ref(false)

const editTarget = ref<KeyRow | null>(null)
const editDialogOpen = ref(false)
const editName = ref('')
const editScope = ref<KeyScope>('run')
const editExpires = ref('never')
const editError = ref('')
const editing = ref(false)

const deleteTarget = ref<KeyRow | null>(null)
const deleteDialogOpen = ref(false)

onMounted(load)

async function load() {
  error.value = ''
  try {
    const [k, apps] = await Promise.all([api.adminKeys(), api.manageableApps()])
    keys.value = k as KeyRow[]
    managedApps.value = apps
  } catch (e) {
    error.value = (e as Error).message
  }
}

// Expiry presets. Value 'never' means no expiry; others are relative offsets
// from "now" at submit time. The description column previews the concrete date.
const expiryPresets: { value: string; label: string; ms?: number }[] = [
  { value: 'never', label: 'adminKeys.expiryNever' },
  { value: '1d', label: 'adminKeys.expiry1d', ms: 1 * 24 * 3600 * 1000 },
  { value: '3d', label: 'adminKeys.expiry3d', ms: 3 * 24 * 3600 * 1000 },
  { value: '7d', label: 'adminKeys.expiry7d', ms: 7 * 24 * 3600 * 1000 },
  { value: '1m', label: 'adminKeys.expiry1m', ms: 30 * 24 * 3600 * 1000 },
  { value: '6m', label: 'adminKeys.expiry6m', ms: 6 * 30 * 24 * 3600 * 1000 },
  { value: '1y', label: 'adminKeys.expiry1y', ms: 365 * 24 * 3600 * 1000 },
]

// resolveExpiry converts a preset value to an ISO date string for the API
// (null → never). For edit, an existing expires_at is mapped back to the
// nearest preset so the dropdown shows the current choice.
function presetToIso(v: string): string | null {
  const p = expiryPresets.find((x) => x.value === v)
  if (!p?.ms) return null
  return new Date(Date.now() + p.ms).toISOString()
}

function isoToPreset(iso: string | null): string {
  if (!iso) return 'never'
  const ts = new Date(iso).getTime()
  const now = Date.now()
  // Map existing expiry to the preset whose offset is closest to it.
  let best = 'never'
  let bestDiff = Infinity
  for (const p of expiryPresets) {
    if (!p.ms) continue
    const target = now + p.ms
    const diff = Math.abs(target - ts)
    if (diff < bestDiff) {
      bestDiff = diff
      best = p.value
    }
  }
  return best
}

const expiryItems = expiryPresets.map((p) => ({ value: p.value, title: t(p.label) }))

function expiryPreview(v: string): string {
  const iso = presetToIso(v)
  if (!iso) return t('adminKeys.noExpiry')
  return fmtDate(iso)
}

function openCreate() {
  createName.value = ''
  createScope.value = 'run'
  createExpires.value = ''
  createError.value = ''
  createDialogOpen.value = true
}

async function confirmCreate() {
  if (creating.value) return
  if (!createName.value.trim()) {
    createError.value = t('adminKeys.nameRequired')
    return
  }
  createError.value = ''
  creating.value = true
  try {
    await api.createKey({
      name: createName.value.trim(),
      scope: createScope.value,
      expires_at: presetToIso(createExpires.value),
    })
    createDialogOpen.value = false
    await load()
    toast(t('adminKeys.saved'))
  } catch (e) {
    createError.value = (e as Error).message
  } finally {
    creating.value = false
  }
}

async function copyKey(k: KeyRow) {
  try {
    await navigator.clipboard.writeText(k.key)
    toast(t('adminApp.linkCopied'))
  } catch {
    toast(k.key)
  }
}

function openEdit(k: KeyRow) {
  editTarget.value = k
  editName.value = k.name
  editScope.value = k.scope
  editExpires.value = isoToPreset(k.expires_at)
  editError.value = ''
  editDialogOpen.value = true
}

async function confirmEdit() {
  if (editing.value) return
  if (!editTarget.value) return
  if (!editName.value.trim()) {
    editError.value = t('adminKeys.nameRequired')
    return
  }
  editError.value = ''
  editing.value = true
  try {
    await api.updateKey(editTarget.value.id, {
      name: editName.value.trim(),
      scope: editScope.value,
      expires_at: presetToIso(editExpires.value),
    })
    editDialogOpen.value = false
    await load()
    toast(t('adminKeys.saved'))
  } catch (e) {
    editError.value = (e as Error).message
  } finally {
    editing.value = false
  }
}

function askDelete(k: KeyRow) {
  deleteTarget.value = k
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  const id = deleteTarget.value.id
  deleteTarget.value = null
  deleteDialogOpen.value = false
  try {
    await api.deleteKey(id)
    await load()
    toast(t('adminKeys.deleted'))
  } catch (e) {
    toast((e as Error).message)
  }
}

function scopeBadge(s: KeyScope): 'success' | 'secondary' {
  return s === 'run' ? 'success' : 'secondary'
}

function fmtExpiry(k: KeyRow): string {
  if (!k.expires_at) return t('adminKeys.noExpiry')
  return fmtDate(k.expires_at)
}

function fmtLastUsed(k: KeyRow): string {
  if (!k.last_used_at) return t('adminKeys.never')
  return fmtDate(k.last_used_at)
}
</script>

<template>
  <div class="mx-auto max-w-6xl px-4 py-8 sm:px-6">
    <div class="mb-6 flex items-center justify-between">
      <h1 class="text-2xl font-semibold tracking-tight">{{ t('adminKeys.title') }}</h1>
      <Button @click="openCreate">
        <Plus class="size-4" />
        {{ t('adminKeys.newKey') }}
      </Button>
    </div>

    <Alert v-if="error" variant="destructive" class="mb-4">{{ error }}</Alert>

    <Card class="mb-4">
      <CardContent class="py-4 text-sm">
        <div class="mb-1 text-xs font-semibold uppercase tracking-wider">{{ t('adminKeys.colScope') }}</div>
        <p class="text-muted-foreground">{{ t('adminKeys.scopeHint') }}</p>
        <div class="mt-2 text-muted-foreground">
          {{ isSuperAdmin ? t('adminKeys.scopeRangeAll') : t('adminKeys.scopeRange') }}
          <span v-if="!isSuperAdmin">
            <Badge v-for="a in managedApps" :key="a.id" variant="outline" class="mr-1">{{ a.name }}</Badge>
            <span v-if="!managedApps.length" class="text-muted-foreground">—</span>
          </span>
        </div>
      </CardContent>
    </Card>

    <Card>
      <CardContent class="p-0!">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('adminKeys.colName') }}</TableHead>
              <TableHead>{{ t('adminKeys.colKey') }}</TableHead>
              <TableHead>{{ t('adminKeys.colScope') }}</TableHead>
              <TableHead>{{ t('adminKeys.colExpires') }}</TableHead>
              <TableHead>{{ t('adminKeys.colLastUsed') }}</TableHead>
              <TableHead>{{ t('adminKeys.colCreated') }}</TableHead>
              <TableHead v-if="isSuperAdmin">{{ t('adminKeys.colOwner') }}</TableHead>
              <TableHead class="text-right"> </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="k in keys" :key="k.id">
              <TableCell>{{ k.name }}</TableCell>
              <TableCell>
                <div class="flex items-center gap-1">
                  <code class="text-xs">{{ k.key }}</code>
                  <Button variant="ghost" size="icon" class="size-6" @click="copyKey(k)">
                    <Copy class="size-3" />
                  </Button>
                </div>
              </TableCell>
              <TableCell>
                <Badge :variant="scopeBadge(k.scope)">{{ t('adminKeys.scope' + (k.scope === 'run' ? 'Run' : 'Read')) }}</Badge>
              </TableCell>
              <TableCell><code class="text-xs">{{ fmtExpiry(k) }}</code></TableCell>
              <TableCell><code class="text-xs">{{ fmtLastUsed(k) }}</code></TableCell>
              <TableCell><code class="text-xs">{{ fmtDate(k.created_at) }}</code></TableCell>
              <TableCell v-if="isSuperAdmin"><code class="text-xs">{{ k.owner_username || '—' }}</code></TableCell>
              <TableCell class="text-right">
                <Button variant="ghost" size="sm" @click="openEdit(k)">{{ t('common.edit') }}</Button>
                <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="askDelete(k)">{{ t('common.delete') }}</Button>
              </TableCell>
            </TableRow>
            <TableEmpty v-if="!keys.length" :colspan="isSuperAdmin ? 8 : 7">{{ t('adminKeys.empty') }}</TableEmpty>
          </TableBody>
        </Table>
      </CardContent>
    </Card>

    <Dialog v-model:open="createDialogOpen" :title="t('adminKeys.newKey')" max-width="md">
      <div class="grid gap-4">
        <div class="grid gap-2">
          <Label>{{ t('adminKeys.nameLabel') }}</Label>
          <Input v-model="createName" autofocus />
        </div>
        <div class="grid gap-2">
          <Label>{{ t('adminKeys.scopeLabel') }}</Label>
          <RadioGroup v-model="createScope">
            <div class="flex items-center gap-2 text-sm">
              <RadioGroupItem value="run" id="k-create-run" />
              <Label for="k-create-run">{{ t('adminKeys.scopeRun') }}</Label>
            </div>
            <div class="flex items-center gap-2 text-sm">
              <RadioGroupItem value="read" id="k-create-read" />
              <Label for="k-create-read">{{ t('adminKeys.scopeRead') }}</Label>
            </div>
          </RadioGroup>
        </div>
        <div class="grid gap-2">
          <Label>{{ t('adminKeys.expiresLabel') }}</Label>
          <Select v-model="createExpires" :items="expiryItems" />
          <p class="text-xs text-muted-foreground">{{ expiryPreview(createExpires) }}</p>
        </div>
        <Alert v-if="createError" variant="destructive">{{ createError }}</Alert>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="createDialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button :disabled="!createName.trim() || creating" @click="confirmCreate">{{ t('common.create') }}</Button>
        </div>
      </div>
    </Dialog>

    <Dialog v-model:open="editDialogOpen" :title="t('adminKeys.editKey')" max-width="md">
      <div class="grid gap-4">
        <div class="grid gap-2">
          <Label>{{ t('adminKeys.nameLabel') }}</Label>
          <Input v-model="editName" autofocus />
        </div>
        <div class="grid gap-2">
          <Label>{{ t('adminKeys.scopeLabel') }}</Label>
          <RadioGroup v-model="editScope">
            <div class="flex items-center gap-2 text-sm">
              <RadioGroupItem value="run" id="k-edit-run" />
              <Label for="k-edit-run">{{ t('adminKeys.scopeRun') }}</Label>
            </div>
            <div class="flex items-center gap-2 text-sm">
              <RadioGroupItem value="read" id="k-edit-read" />
              <Label for="k-edit-read">{{ t('adminKeys.scopeRead') }}</Label>
            </div>
          </RadioGroup>
        </div>
        <div class="grid gap-2">
          <Label>{{ t('adminKeys.expiresLabel') }}</Label>
          <Select v-model="editExpires" :items="expiryItems" />
          <p class="text-xs text-muted-foreground">{{ expiryPreview(editExpires) }}</p>
        </div>
        <Alert v-if="editError" variant="destructive">{{ editError }}</Alert>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="editDialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button :disabled="!editName.trim() || editing" @click="confirmEdit">{{ t('common.save') }}</Button>
        </div>
      </div>
    </Dialog>

    <AlertDialog v-model:open="deleteDialogOpen" :title="t('common.confirmDelete')" :description="t('adminKeys.confirmDelete', { name: deleteTarget?.name ?? '' })">
      <template #footer>
        <Button variant="outline" @click="deleteDialogOpen = false">{{ t('common.cancel') }}</Button>
        <Button variant="destructive" @click="confirmDelete">{{ t('common.delete') }}</Button>
      </template>
    </AlertDialog>
  </div>
</template>