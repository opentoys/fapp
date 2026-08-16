<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Download, Loader2 } from 'lucide-vue-next'
import { api } from '../api/client'
import { useI18n } from '../composables/useI18n'
import { loadDownloadApp } from '../composables/useDownloadApp'
import { detectUA } from '../utils/ua'
import { Alert } from '../components/ui/alert'
import { Avatar } from '../components/ui/avatar'
import { Badge } from '../components/ui/badge'
import { Button } from '../components/ui/button'
import { Card, CardContent, CardFooter } from '../components/ui/card'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'
import VersionPanel from '../components/VersionPanel.vue'
import type { AppDetail, Version } from '../api/types'

const route = useRoute()
const { t } = useI18n()
const detected = detectUA()

const data = ref<AppDetail | null>(null)
const error = ref('')
const unlocked = ref(false)
const unlocking = ref(false)
const password = ref('')
const passwordError = ref('')

// The backend exposes only the app's single current version.
const latest = computed(() => data.value?.versions[0] ?? null)

onMounted(load)
watch(() => route.params.name, (name) => { if (name) load() })

async function load() {
  data.value = null
  unlocked.value = false
  error.value = ''
  try {
    data.value = await loadDownloadApp(String(route.params.name))
  } catch (e) {
    error.value = (e as Error).message
  }
}

const appAccess = computed(() => data.value?.app.access_mode ?? 'public')
const appLocked = computed(() => data.value?.app.access_mode === 'password' && !unlocked.value)

async function unlock() {
  const v = latest.value
  if (!v) return
  unlocking.value = true
  passwordError.value = ''
  try {
    await api.verify(v.id, password.value)
    unlocked.value = true
  } catch (e) {
    passwordError.value = (e as Error).message
  } finally {
    unlocking.value = false
  }
}

async function download(v: Version | null) {
  if (!v || appLocked.value) return
  await doDownload(v.id, appAccess.value === 'password' ? password.value : undefined)
}

async function doDownload(versionId: number, pw: string | undefined) {
  try {
    const url = await api.downloadUrl(versionId, pw)
    window.location.href = url
  } catch (e) {
    error.value = (e as Error).message
  }
}

</script>

<template>
  <div class="mx-auto max-w-6xl px-4 py-8 sm:px-6">
    <Alert v-if="error" variant="destructive" class="mb-4">
      {{ error }}
    </Alert>

    <!-- Password gate: entry-level. Replace the info until verified. -->
    <Card v-if="data && appLocked" class="mx-auto max-w-md">
      <CardContent class="p-6">
        <div class="mb-4 flex items-center gap-3">
          <Avatar :src="data.app.icon" :fallback="data.app.name.charAt(0).toUpperCase()" class="size-10" />
          <h1 class="text-lg font-semibold tracking-tight">{{ data.app.name }}</h1>
        </div>
        <p class="text-muted-foreground mb-4 text-sm">{{ t('detail.passwordBody') }}</p>
        <div class="grid gap-2">
          <Label for="app-password">{{ t('common.password') }}</Label>
          <Input
            id="app-password"
            v-model="password"
            type="password"
            autocomplete="off"
            autofocus
            :disabled="unlocking"
            @keyup.enter="unlock"
          />
        </div>
        <div v-if="passwordError" class="mt-2 text-sm text-destructive">{{ passwordError }}</div>
        <Button class="mt-4 w-full" :disabled="unlocking" @click="unlock">
          <Loader2 v-if="unlocking" class="animate-spin size-4" />
          {{ t('detail.passwordContinue') }}
        </Button>
        <CardFooter class="p-0 pt-4">
          <p class="text-muted-foreground text-xs">{{ t('detail.passwordGateHint') }}</p>
        </CardFooter>
      </CardContent>
    </Card>

    <template v-else-if="data">
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
          :no-download="true"
          @download="download"
        />
      </div>

      <!-- Desktop: current version card (the only publicly visible version) -->
      <template v-else-if="latest">
        <div class="max-w-[560px]">
          <Card class="p-5">
            <CardContent class="p-0!">
              <VersionPanel
                :version="latest"
                :fallback-name="data.app.name"
                :fallback-icon="data.app.icon"
                @download="download"
              />
            </CardContent>
          </Card>
        </div>
      </template>

      <Card v-else-if="!error" class="text-center">
        <CardContent class="py-12 text-muted-foreground">{{ t('detail.empty') }}</CardContent>
      </Card>
    </template>

    <!-- Floating download button: 80% width, pinned to the viewport bottom -->
    <div v-if="!detected.isDesktop && latest && !error && !appLocked" class="fixed inset-x-0 bottom-0 z-50 flex justify-center px-4 pb-[calc(0.75rem+env(safe-area-inset-bottom))] pt-3 bg-gradient-to-t from-background to-transparent">
      <Button size="lg" class="w-4/5" @click="download(latest)">
        <Download class="size-4" />
        {{ t('detail.download') }}
      </Button>
    </div>
  </div>
</template>
