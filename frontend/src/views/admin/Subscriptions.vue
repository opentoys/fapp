<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { toast } from 'vue-sonner'
import { Plus, Send, Loader2, Info } from 'lucide-vue-next'
import { api } from '../../api/client'
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
import { Textarea } from '../../components/ui/textarea'
import { Select } from '../../components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, TableEmpty } from '../../components/ui/table'
import type { AppItem, NotificationBot, NotificationLog, NotifyEvent } from '../../api/types'

const { t } = useI18n()
const bots = ref<NotificationBot[]>([])
const apps = ref<AppItem[]>([])
const error = ref('')

// --- create / edit dialog ---
const dialogOpen = ref(false)
const editing = ref<NotificationBot | null>(null)
const saving = ref(false)
const dialogError = ref('')

const fName = ref('')
const fApp = ref<number | null>(null)
const fMethod = ref<'POST' | 'GET' | 'PUT'>('POST')
const fUrl = ref('')
const fHeaders = ref('')
const fBody = ref('')
const fEvents = ref<NotifyEvent[]>([])

const previewParams = computed(() => {
  const app = apps.value.find((a) => a.id === fApp.value)
  return {
    event: app?.name ? t('notify.events.version_uploaded') : '—',
    event_key: 'version_uploaded',
    app_name: app?.name ?? '—',
    app_id: app?.id ? String(app.id) : '—',
    time: fmtDate(new Date().toISOString()),
    version_name: '1.2.3',
    version_code: '123',
    version_id: '42',
    file_name: 'app.apk',
    file_size: '10485760',
    published: 'true',
    expires_at: '2026-12-31 23:59:59',
  }
})

const previewRendered = computed(() => fill(fBody.value, previewParams.value))

const paramLabel = (k: string) => `{{.${k}}}`

function fill(s: string, p: Record<string, string>): string {
  return s.replace(/\{\{\s*\.?(\w+)\s*\}\}/g, (_, k: string) => (k in p ? p[k] : `{{.${k}}}`))
}

function openCreate() {
  editing.value = null
  fName.value = ''
  fApp.value = null
  fMethod.value = 'POST'
  fUrl.value = ''
  fHeaders.value = ''
  fBody.value = JSON.stringify({ event: '{{.event}}', app: '{{.app_name}}', version: '{{.version_name}}', time: '{{.time}}' })
  fEvents.value = ['version_uploaded']
  dialogError.value = ''
  dialogOpen.value = true
}

function openEdit(b: NotificationBot) {
  editing.value = b
  fName.value = b.name
  fApp.value = b.app_id
  fMethod.value = b.method
  fUrl.value = b.url
  fHeaders.value = b.headers.join('\n')
  fBody.value = b.body_template
  fEvents.value = [...b.events]
  dialogError.value = ''
  dialogOpen.value = true
}

const methodItems = [
  { value: 'POST', title: 'POST' },
  { value: 'GET', title: 'GET' },
  { value: 'PUT', title: 'PUT' },
]

const eventItems: { value: NotifyEvent; title: string }[] = [
  { value: 'version_uploaded', title: t('notify.events.version_uploaded') },
  { value: 'version_current', title: t('notify.events.version_current') },
  { value: 'app_publish', title: t('notify.events.app_publish') },
  { value: 'app_expire', title: t('notify.events.app_expire') },
]

const appItems = computed(() => apps.value.map((a) => ({ value: a.id, title: a.name })))

function buildPayload() {
  return {
    name: fName.value.trim(),
    app_id: fApp.value!,
    method: fMethod.value,
    url: fUrl.value.trim(),
    headers: fHeaders.value.split('\n').map((s) => s.trim()).filter(Boolean),
    body_template: fBody.value,
    events: fEvents.value,
  }
}

async function save() {
  if (saving.value) return
  if (!fName.value.trim() || !fApp.value || !fUrl.value.trim() || !fEvents.value.length) {
    dialogError.value = t('notify.required')
    return
  }
  dialogError.value = ''
  saving.value = true
  try {
    if (editing.value) await api.updateSubscription(editing.value.id, buildPayload())
    else await api.createSubscription(buildPayload())
    dialogOpen.value = false
    await load()
    toast(t('notify.saved'))
  } catch (e) {
    dialogError.value = (e as Error).message
  } finally {
    saving.value = false
  }
}

// Test the unsaved form config by firing a sample webhook.
const testing = ref(false)
async function testDraft() {
  if (testing.value) return
  if (!fName.value.trim() || !fApp.value || !fUrl.value.trim() || !fEvents.value.length) {
    dialogError.value = t('notify.required')
    return
  }
  dialogError.value = ''
  testing.value = true
  try {
    await api.testSubscriptionConfig(buildPayload())
    toast(t('notify.testSent'))
  } catch (e) {
    dialogError.value = (e as Error).message
  } finally {
    testing.value = false
  }
}

// --- test ---
const testId = ref<number | null>(null)
async function testBot(b: NotificationBot) {
  testId.value = b.id
  try {
    await api.testSubscription(b.id)
    toast(t('notify.testSent'))
  } catch (e) {
    toast((e as Error).message)
  } finally {
    testId.value = null
  }
}

// --- delete ---
const deleteTarget = ref<NotificationBot | null>(null)
const deleteOpen = ref(false)
async function confirmDelete() {
  if (!deleteTarget.value) return
  const id = deleteTarget.value.id
  deleteTarget.value = null
  deleteOpen.value = false
  try {
    await api.deleteSubscription(id)
    await load()
    toast(t('notify.deleted'))
  } catch (e) {
    toast((e as Error).message)
  }
}

// --- logs ---
const logBot = ref<NotificationBot | null>(null)
const logs = ref<NotificationLog[]>([])
const logsOpen = ref(false)
const logsLoading = ref(false)
async function openLogs(b: NotificationBot) {
  logBot.value = b
  logsOpen.value = true
  logsLoading.value = true
  logs.value = []
  try {
    logs.value = await api.subscriptionLogs(b.id)
  } catch (e) {
    logs.value = []
    toast((e as Error).message)
  } finally {
    logsLoading.value = false
  }
}

const appName = (id: number) => apps.value.find((a) => a.id === id)?.name ?? String(id)
const methodBadge = (m: string): 'success' | 'secondary' => (m === 'POST' ? 'success' : 'secondary')
const eventCount = (b: NotificationBot) => b.events.map((e) => t('notify.events.' + e)).join(', ')

onMounted(load)
async function load() {
  error.value = ''
  try {
    const [b, a] = await Promise.all([api.subscriptions(), api.manageableApps()])
    bots.value = b
    apps.value = a
  } catch (e) {
    error.value = (e as Error).message
  }
}
</script>

<template>
  <div class="mx-auto max-w-6xl px-4 py-8 sm:px-6">
    <div class="mb-6 flex items-center justify-between">
      <h1 class="text-2xl font-semibold tracking-tight">{{ t('notify.title') }}</h1>
      <Button @click="openCreate">
        <Plus class="size-4" />
        {{ t('notify.new') }}
      </Button>
    </div>

    <Alert v-if="error" variant="destructive" class="mb-4">{{ error }}</Alert>

    <Card>
      <CardContent class="p-0!">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('notify.colName') }}</TableHead>
              <TableHead>{{ t('notify.colApp') }}</TableHead>
              <TableHead>{{ t('notify.colMethod') }}</TableHead>
              <TableHead>{{ t('notify.colUrl') }}</TableHead>
              <TableHead>{{ t('notify.colEvents') }}</TableHead>
              <TableHead>{{ t('notify.colCreated') }}</TableHead>
              <TableHead class="text-right"> </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="b in bots" :key="b.id">
              <TableCell class="font-medium">{{ b.name }}</TableCell>
              <TableCell>{{ appName(b.app_id) }}</TableCell>
              <TableCell><Badge :variant="methodBadge(b.method)">{{ b.method }}</Badge></TableCell>
              <TableCell><code class="text-xs block max-w-[260px] truncate">{{ b.url }}</code></TableCell>
              <TableCell><code class="text-xs">{{ eventCount(b) }}</code></TableCell>
              <TableCell><code class="text-xs">{{ fmtDate(b.created_at) }}</code></TableCell>
              <TableCell class="text-right">
                <Button variant="ghost" size="sm" :disabled="testId === b.id" @click="testBot(b)">
                  <Loader2 v-if="testId === b.id" class="size-3 animate-spin" />
                  <Send v-else class="size-3" />
                  {{ t('notify.test') }}
                </Button>
                <Button variant="ghost" size="sm" @click="openLogs(b)">{{ t('notify.logs') }}</Button>
                <Button variant="ghost" size="sm" @click="openEdit(b)">{{ t('common.edit') }}</Button>
                <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="deleteTarget = b; deleteOpen = true">{{ t('common.delete') }}</Button>
              </TableCell>
            </TableRow>
            <TableEmpty v-if="!bots.length" :colspan="7">{{ t('notify.empty') }}</TableEmpty>
          </TableBody>
        </Table>
      </CardContent>
    </Card>

    <!-- Create / edit dialog -->
    <Dialog v-model:open="dialogOpen" :title="editing ? t('notify.edit') : t('notify.new')">
      <div class="grid gap-4">
        <div class="grid gap-2 sm:grid-cols-2">
          <div class="grid gap-2">
            <Label>{{ t('notify.fName') }}</Label>
            <Input v-model="fName" autofocus />
          </div>
          <div class="grid gap-2">
            <Label>{{ t('notify.fApp') }}</Label>
            <Select v-model="fApp" :items="appItems" :placeholder="t('notify.fAppPlaceholder')" />
          </div>
        </div>
        <div class="grid gap-2 sm:grid-cols-2">
          <div class="grid gap-2">
            <Label>{{ t('notify.fMethod') }}</Label>
            <Select v-model="fMethod" :items="methodItems" />
          </div>
          <div class="grid gap-2">
            <Label>{{ t('notify.fUrl') }}</Label>
            <Input v-model="fUrl" placeholder="https://example.com/webhook" />
          </div>
        </div>
        <div class="grid gap-2">
          <Label>{{ t('notify.fHeaders') }}</Label>
          <Textarea v-model="fHeaders" :placeholder="`Authorization: token xxx\nX-App: {{.app_name}}`" rows="2" />
        </div>
        <div class="grid gap-2">
          <div class="flex items-center gap-1">
            <Label>{{ t('notify.fBody') }}</Label>
            <!-- Common parameters reference in a hover tooltip -->
            <div class="group relative">
              <span class="text-muted-foreground cursor-help">
                <Info class="size-3.5" />
              </span>
              <div class="pointer-events-auto absolute bottom-full left-0 z-10 mb-2 hidden w-72 rounded-md border bg-popover p-3 text-xs shadow-md group-hover:block after:absolute after:inset-x-0 after:top-full after:h-2">
                <div class="mb-1 font-semibold">{{ t('notify.paramsTitle') }}</div>
                <div class="grid gap-y-1 text-muted-foreground">
                  <code v-for="(v, k) in previewParams" :key="k" class="truncate">
                    {{ paramLabel(k) }} → <span class="text-foreground">{{ v }}</span>
                  </code>
                </div>
              </div>
            </div>
          </div>
          <Textarea v-model="fBody" rows="6" class="font-mono text-xs" />
          <div v-if="fBody.trim()">
            <div class="text-xs text-muted-foreground mb-1">{{ t('notify.preview') }}</div>
            <pre class="bg-muted rounded-md p-2 text-xs whitespace-pre-wrap break-all">{{ previewRendered }}</pre>
          </div>
        </div>
        <div class="grid gap-2">
          <Label>{{ t('notify.fEvents') }}</Label>
          <div class="grid gap-2">
            <div v-for="e in eventItems" :key="e.value" class="flex items-center gap-2 text-sm">
              <input
                :id="'ev-' + e.value"
                type="checkbox"
                class="size-4 rounded border-input"
                :checked="fEvents.includes(e.value)"
                @change="fEvents = fEvents.includes(e.value) ? fEvents.filter((x) => x !== e.value) : [...fEvents, e.value]"
              />
              <label :for="'ev-' + e.value">{{ e.title }}</label>
            </div>
          </div>
        </div>
        <Alert v-if="dialogError" variant="destructive">{{ dialogError }}</Alert>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="dialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button variant="outline" :disabled="testing || saving" @click="testDraft">
            <Loader2 v-if="testing" class="size-4 animate-spin" />
            <Send v-else class="size-4" />
            {{ t('notify.testDraft') }}
          </Button>
          <Button :disabled="saving" @click="save">{{ t('common.save') }}</Button>
        </div>
      </div>
    </Dialog>

    <!-- Logs dialog -->
    <Dialog v-model:open="logsOpen" :title="t('notify.logsTitle', { name: logBot?.name ?? '' })">
      <div class="grid gap-3">
        <Alert v-if="!logsLoading && !logs.length">{{ t('notify.logsEmpty') }}</Alert>
        <div v-for="(g, i) in logs" :key="i" class="rounded-md border p-3 text-xs">
          <div class="mb-1 flex items-center justify-between gap-2">
            <span class="font-mono">{{ g.event }} · {{ fmtDate(g.created_at) }}</span>
            <Badge :variant="g.status && g.status < 400 ? 'success' : 'destructive'">
              {{ g.status ? g.status : (g.error || '—') }}
            </Badge>
          </div>
          <div class="text-muted-foreground truncate">{{ g.url }}</div>
          <pre class="bg-muted mt-1 rounded p-1 whitespace-pre-wrap break-all">{{ g.body }}</pre>
        </div>
      </div>
    </Dialog>

    <AlertDialog v-model:open="deleteOpen" :title="t('common.confirmDelete')" :description="t('notify.confirmDelete', { name: deleteTarget?.name ?? '' })">
      <template #footer>
        <Button variant="outline" @click="deleteOpen = false">{{ t('common.cancel') }}</Button>
        <Button variant="destructive" @click="confirmDelete">{{ t('common.delete') }}</Button>
      </template>
    </AlertDialog>

    <div v-if="editing" class="text-xs text-muted-foreground mt-2">{{ t('notify.editHint') }}</div>
  </div>
</template>