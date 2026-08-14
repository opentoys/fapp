<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '../api/client'
import { useI18n } from '../composables/useI18n'
import { Badge } from '../components/ui/badge'
import { Card, CardContent } from '../components/ui/card'
import { Alert } from '../components/ui/alert'
import { Avatar } from '../components/ui/avatar'
import type { AppItem } from '../api/types'

const { t } = useI18n()
const apps = ref<AppItem[]>([])
const error = ref('')

onMounted(async () => {
  try {
    apps.value = await api.apps()
  } catch (e) {
    error.value = (e as Error).message
  }
})

const totalDownloads = computed(() =>
  apps.value.reduce((s, a) => s + (a.latest_version?.download_count ?? 0), 0)
)
const totalVersions = computed(() =>
  apps.value.filter((a) => a.latest_version).length
)

function fmtSize(n: number): string {
  if (n > 1024 * 1024 * 1024) return (n / 1024 / 1024 / 1024).toFixed(2) + ' GB'
  if (n > 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MB'
  if (n > 1024) return (n / 1024).toFixed(1) + ' KB'
  return n + ' B'
}

function accessVariant(mode: string): 'success' | 'warning' | 'secondary' {
  if (mode === 'public') return 'success'
  if (mode === 'password' || mode === 'expiry') return 'warning'
  return 'secondary'
}
</script>

<template>
  <div class="mx-auto max-w-6xl px-4 py-8 sm:px-6">
    <h1 class="text-2xl font-semibold tracking-tight mb-6">{{ t('home.title') }}</h1>

    <Alert v-if="error" variant="destructive" class="mb-4">{{ error }}</Alert>

    <div class="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
      <Card>
        <CardContent class="!py-4">
          <div class="text-muted-foreground text-xs uppercase tracking-wider">{{ t('home.stat.apps') }}</div>
          <div class="text-3xl font-semibold">{{ apps.length }}</div>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="!py-4">
          <div class="text-muted-foreground text-xs uppercase tracking-wider">{{ t('home.stat.versions') }}</div>
          <div class="text-3xl font-semibold">{{ totalVersions }}</div>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="!py-4">
          <div class="text-muted-foreground text-xs uppercase tracking-wider">{{ t('home.stat.downloads') }}</div>
          <div class="text-3xl font-semibold">{{ totalDownloads }}</div>
        </CardContent>
      </Card>
    </div>

    <div v-if="apps.length" class="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3">
      <RouterLink
        v-for="a in apps"
        :key="a.id"
        :to="`/app/${encodeURIComponent(a.name)}`"
        class="group rounded-xl border bg-card text-card-foreground p-5 shadow-sm transition-colors hover:bg-accent/50"
      >
        <div class="mb-3 flex items-center gap-3">
          <Avatar :src="a.icon" :fallback="a.name.charAt(0).toUpperCase()" class="size-10" />
          <span class="truncate font-semibold">{{ a.name }}</span>
        </div>
        <div v-if="a.latest_version" class="text-muted-foreground mb-1 text-sm">
          <code>{{ a.latest_version.version_name }}</code>
          · {{ fmtSize(a.latest_version.file_size) }}
        </div>
        <Badge :variant="accessVariant(a.access_mode)">
          {{ t('access.' + a.access_mode) }}
        </Badge>
        <p v-if="a.description" class="text-muted-foreground mt-2 text-sm">{{ a.description }}</p>
      </RouterLink>
    </div>

    <Card v-else-if="!error" class="text-center">
      <CardContent class="py-12 text-muted-foreground">{{ t('home.empty') }}</CardContent>
    </Card>
  </div>
</template>
