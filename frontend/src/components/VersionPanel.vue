<script setup lang="ts">
import { computed, ref } from 'vue'
import { Download } from 'lucide-vue-next'
import { useI18n } from '../composables/useI18n'
import { detectUA } from '../utils/ua'
import { fmtDate } from '../utils/format'
import { Badge } from './ui/badge'
import { Avatar } from './ui/avatar'
import { Button } from './ui/button'
import type { Architecture, Version } from '../api/types'

const props = defineProps<{
  version: Version
  fallbackName: string
  fallbackIcon: string
  accessMode?: string
  expiresAt?: string | null
  noDownload?: boolean
}>()

const emit = defineEmits<{ download: [v: Version] }>()

const { t } = useI18n()
const detected = detectUA()

// Clickable architecture selection. Architecture is metadata — one file per
// version — so switching only changes the highlighted chip.
const archChoice = ref<Architecture | null>(null)
const archList = computed(() =>
  (props.version.arch || '').split(',').filter(Boolean) as Architecture[]
)
const currentArch = computed(() => archChoice.value ?? defaultArch())
function defaultArch(): Architecture {
  const list = archList.value
  if (!list.length) return 'universal'
  if (detected.arch && list.includes(detected.arch)) return detected.arch
  return list[0]
}

const icon = computed(() => props.version.icon_url || props.fallbackIcon)
const appName = computed(() => props.version.app_name || props.fallbackName)

function releaseVariant(rt: string): 'default' | 'info' | 'warning' {
  if (rt === 'beta') return 'info'
  if (rt === 'canary') return 'warning'
  return 'default'
}

function isExpired(): boolean {
  return props.accessMode === 'expiry' && !!props.expiresAt && new Date(props.expiresAt) < new Date()
}

function fmtSize(n: number): string {
  if (n > 1024 * 1024 * 1024) return (n / 1024 / 1024 / 1024).toFixed(2) + ' GB'
  if (n > 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MB'
  if (n > 1024) return (n / 1024).toFixed(1) + ' KB'
  return n + ' B'
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <div class="flex items-center gap-3">
      <Avatar :src="icon" :fallback="appName.charAt(0).toUpperCase()" class="size-14" />
      <Badge :variant="releaseVariant(version.release_type)">
        {{ t('release.' + version.release_type) }}
      </Badge>
    </div>

    <div>
      <div class="font-semibold">{{ appName }}</div>
      <div v-if="version.package_name" class="text-muted-foreground text-xs">
        <code>{{ version.package_name }}</code>
      </div>
    </div>

    <div class="flex items-center gap-2">
      <Badge v-if="version.platform" variant="outline" class="text-xs">
        {{ t('platform.' + version.platform) }}
      </Badge>
      <span class="font-medium">
        {{ version.version_name }}
        <span v-if="version.version_code" class="text-muted-foreground text-sm">
          ({{ version.version_code }})
        </span>
      </span>
    </div>

    <div class="flex items-center gap-1.5">
      <span class="text-muted-foreground text-xs">{{ t('detail.arch') }}:</span>
      <template v-if="archList.length">
        <Badge
          v-for="a in archList"
          :key="a"
          variant="outline"
          :class="currentArch === a ? 'border-primary text-primary' : 'cursor-pointer hover:bg-accent'"
          @click="archChoice = a"
        >
          {{ t('arch.' + a) }}
        </Badge>
      </template>
      <span v-else class="text-muted-foreground text-xs">—</span>
    </div>

    <div class="text-muted-foreground text-sm">
      {{ fmtSize(version.file_size) }} · {{ fmtDate(version.created_at) }}
    </div>

    <div v-if="version.changelog">
      <div class="text-sm font-medium">{{ t('detail.changelog') }}:</div>
      <div class="text-sm whitespace-pre-wrap">{{ version.changelog }}</div>
    </div>

    <div v-if="!noDownload">
      <Button size="lg" :disabled="isExpired()" @click="emit('download', version)">
        <Download class="size-4" />
        {{ t('detail.download') }}
      </Button>
    </div>
  </div>
</template>
