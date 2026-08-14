<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Download } from 'lucide-vue-next'
import { api } from '../api/client'
import { useI18n } from '../composables/useI18n'
import { loadDownloadApp } from '../composables/useDownloadApp'
import { detectUA } from '../utils/ua'
import { fmtDate } from '../utils/format'
import { Alert } from '../components/ui/alert'
import { Avatar } from '../components/ui/avatar'
import { Badge } from '../components/ui/badge'
import { Button } from '../components/ui/button'
import { Card, CardContent } from '../components/ui/card'
import { Dialog } from '../components/ui/dialog'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'
import VersionPanel from '../components/VersionPanel.vue'
import type { AppDetail, Version } from '../api/types'

const route = useRoute()
const { t } = useI18n()
const detected = detectUA()

const data = ref<AppDetail | null>(null)
const error = ref('')
const passwordPrompt = ref<{ versionId: number; password: string } | null>(null)
const dialogOpen = ref(false)

// Newest first by creation time.
const versions = computed(() =>
  [...(data.value?.versions ?? [])].sort(
    (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  )
)

// A single-platform app shows one latest version, then a list of the rest.
const latest = computed(() => versions.value[0] ?? null)

function openPasswordPrompt(versionId: number) {
  passwordPrompt.value = { versionId, password: '' }
  dialogOpen.value = true
}
function closePasswordPrompt() {
  dialogOpen.value = false
  passwordPrompt.value = null
}

onMounted(load)
watch(() => route.params.name, (name) => { if (name) load() })

async function load() {
  data.value = null
  error.value = ''
  try {
    data.value = await loadDownloadApp(String(route.params.name))
  } catch (e) {
    error.value = (e as Error).message
  }
}

const appAccess = computed(() => data.value?.app.access_mode ?? 'public')
const appExpiresAt = computed(() => data.value?.app.expires_at ?? null)

function isExpired(): boolean {
  return appAccess.value === 'expiry' && !!appExpiresAt.value && new Date(appExpiresAt.value) < new Date()
}

async function download(v: Version | null) {
  if (!v) return
  if (appAccess.value === 'password') {
    openPasswordPrompt(v.id)
    return
  }
  await doDownload(v.id, undefined)
}

async function submitPassword() {
  if (!passwordPrompt.value) return
  const pwd = passwordPrompt.value.password
  const vid = passwordPrompt.value.versionId
  closePasswordPrompt()
  await doDownload(vid, pwd)
}

async function doDownload(versionId: number, password: string | undefined) {
  try {
    const url = await api.downloadUrl(versionId, password)
    window.location.href = url
  } catch (e) {
    error.value = (e as Error).message
  }
}

function releaseVariant(rt: string): 'default' | 'info' | 'warning' {
  if (rt === 'beta') return 'info'
  if (rt === 'canary') return 'warning'
  return 'default'
}

function fmtSize(n: number): string {
  if (n > 1024 * 1024 * 1024) return (n / 1024 / 1024 / 1024).toFixed(2) + ' GB'
  if (n > 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MB'
  if (n > 1024) return (n / 1024).toFixed(1) + ' KB'
  return n + ' B'
}
</script>

<template>
  <div class="mx-auto max-w-6xl px-4 py-8 sm:px-6">
    <Alert v-if="error" variant="destructive" class="mb-4">
      {{ error }}
    </Alert>

    <template v-if="data">
      <div class="mb-2 flex items-center gap-3">
        <Avatar :src="data.app.icon" :fallback="data.app.name.charAt(0).toUpperCase()" class="size-10" />
        <div>
          <h1 class="text-2xl font-semibold tracking-tight">{{ data.app.name }}</h1>
          <Badge v-if="data.app.platform" variant="outline" class="text-xs">
            {{ t('platform.' + data.app.platform) }}
          </Badge>
        </div>
      </div>
      <p v-if="data.app.description" class="text-muted-foreground mb-6 text-sm">{{ data.app.description }}</p>

      <!-- Mobile: latest version hero, no card wrapper -->
      <div v-if="!detected.isDesktop && latest" class="max-w-[560px] pb-24">
        <VersionPanel
          :version="latest"
          :fallback-name="data.app.name"
          :fallback-icon="data.app.icon"
          :access-mode="data.app.access_mode"
          :expires-at="data.app.expires_at"
          :no-download="true"
          @download="download"
        />
      </div>

      <!-- Desktop: latest version card + full version list -->
      <template v-else-if="latest">
        <div class="max-w-[560px]">
          <Card class="p-5">
            <CardContent class="p-0!">
              <VersionPanel
                :version="latest"
                :fallback-name="data.app.name"
                :fallback-icon="data.app.icon"
                :access-mode="data.app.access_mode"
                :expires-at="data.app.expires_at"
                @download="download"
              />
            </CardContent>
          </Card>
        </div>

        <div v-if="versions.length > 1" class="mt-6">
          <div class="mb-3 text-sm font-medium">{{ t('detail.versions') }}</div>
          <div class="grid gap-2">
            <Card v-for="v in versions.slice(1)" :key="v.id" class="p-4">
              <div class="flex items-center justify-between gap-3">
                <div class="flex min-w-0 items-center gap-3">
                  <Avatar :src="v.icon_url || data.app.icon" :fallback="(v.version_name || '?').charAt(0).toUpperCase()" class="size-9" />
                  <div class="min-w-0">
                    <div class="flex flex-wrap items-center gap-2">
                      <span class="font-medium">{{ v.version_name }}</span>
                      <Badge v-if="v.release_type" :variant="releaseVariant(v.release_type)" class="text-xs">
                        {{ t('release.' + v.release_type) }}
                      </Badge>
                    </div>
                    <div class="text-muted-foreground text-xs">{{ fmtSize(v.file_size) }} · {{ fmtDate(v.created_at) }}</div>
                  </div>
                </div>
                <Button :disabled="isExpired()" @click="download(v)">
                  <Download class="size-4" />
                  {{ t('detail.download') }}
                </Button>
              </div>
            </Card>
          </div>
        </div>
      </template>

      <Card v-else-if="!error" class="text-center">
        <CardContent class="py-12 text-muted-foreground">{{ t('detail.empty') }}</CardContent>
      </Card>
    </template>

    <!-- Floating download button: 80% width, pinned to the viewport bottom -->
    <div v-if="!detected.isDesktop && latest && !error" class="fixed inset-x-0 bottom-0 z-50 flex justify-center px-4 pb-[calc(0.75rem+env(safe-area-inset-bottom))] pt-3 bg-gradient-to-t from-background to-transparent">
      <Button size="lg" class="w-4/5" :disabled="isExpired()" @click="download(latest)">
        <Download class="size-4" />
        {{ t('detail.download') }}
      </Button>
    </div>

    <Dialog v-model:open="dialogOpen" :title="t('detail.passwordTitle')" max-width="md">
      <div class="grid gap-4">
        <p class="text-muted-foreground text-sm">{{ t('detail.passwordBody') }}</p>
        <div class="grid gap-2">
          <Label for="download-password">{{ t('common.password') }}</Label>
          <Input
            v-if="passwordPrompt"
            id="download-password"
            v-model="passwordPrompt.password"
            type="password"
            autocomplete="off"
            autofocus
            @keyup.enter="submitPassword"
          />
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="closePasswordPrompt">{{ t('common.cancel') }}</Button>
          <Button @click="submitPassword">{{ t('detail.passwordContinue') }}</Button>
        </div>
      </div>
    </Dialog>
  </div>
</template>
