<script setup lang="ts">
import { computed, ref } from 'vue'
import { mdiDownload } from '@mdi/js'
import { useI18n } from '../composables/useI18n'
import { detectUA } from '../utils/ua'
import { fmtDate } from '../utils/format'
import type { Architecture, Version } from '../api/types'

const props = defineProps<{
  version: Version
  fallbackName: string
  fallbackIcon: string
  // App-level access scope (set once on the app's Overview page). Download
  // gating uses this instead of per-version fields.
  accessMode?: string
  expiresAt?: string | null
  // When true, the inline download button is omitted (e.g. the mobile hero,
  // which renders the download action as a floating bottom bar instead).
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
const releaseColor = computed(() => {
  if (props.version.release_type === 'beta') return 'info'
  if (props.version.release_type === 'canary') return 'warning'
  return 'primary'
})

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
  <div class="d-flex flex-column" style="gap: 12px;">
    <!-- Icon + release tag -->
    <div class="d-flex align-center" style="gap: 12px;">
      <v-avatar v-if="icon" :image="icon" size="56" />
      <v-avatar v-else color="primary" size="56">
        <span class="text-h5">{{ appName.charAt(0).toUpperCase() }}</span>
      </v-avatar>
      <v-chip size="small" :color="releaseColor" variant="tonal">
        {{ t('release.' + version.release_type) }}
      </v-chip>
    </div>

    <!-- App name + package -->
    <div>
      <div class="text-h6 font-weight-medium">{{ appName }}</div>
      <div v-if="version.package_name" class="text-caption text-medium-emphasis">
        <code>{{ version.package_name }}</code>
      </div>
    </div>

    <!-- Platform + version name (build code) -->
    <div class="d-flex align-center" style="gap: 8px;">
      <v-chip v-if="version.platform" size="x-small" variant="outlined">
        {{ t('platform.' + version.platform) }}
      </v-chip>
      <span class="text-body-1 font-weight-medium">
        {{ version.version_name }}
        <span v-if="version.version_code" class="text-body-2 text-medium-emphasis">
          ({{ version.version_code }})
        </span>
      </span>
    </div>

    <!-- Clickable architecture chips -->
    <div class="d-flex align-center" style="gap: 6px;">
      <span class="text-caption text-medium-emphasis">{{ t('detail.arch') }}:</span>
      <template v-if="archList.length">
        <v-chip
          v-for="a in archList"
          :key="a"
          size="small"
          :color="currentArch === a ? 'primary' : undefined"
          variant="tonal"
          @click="archChoice = a"
        >
          {{ t('arch.' + a) }}
        </v-chip>
      </template>
      <span v-else class="text-caption text-medium-emphasis">—</span>
    </div>

    <!-- Size · time -->
    <div class="text-body-2 text-medium-emphasis">
      {{ fmtSize(version.file_size) }} · {{ fmtDate(version.created_at) }}
    </div>

    <!-- Changelog -->
    <div v-if="version.changelog">
      <div class="text-body-2 font-weight-medium">{{ t('detail.changelog') }}:</div>
      <div class="text-body-2" style="white-space: pre-wrap;">{{ version.changelog }}</div>
    </div>

    <!-- Download -->
    <div v-if="!noDownload">
      <v-btn
        color="primary"
        size="large"
        variant="flat"
        :disabled="isExpired()"
        @click="emit('download', version)"
      >
        <v-icon start :icon="mdiDownload" />
        {{ t('detail.download') }}
      </v-btn>
    </div>
  </div>
</template>
