<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '../api/client'
import { useI18n } from '../composables/useI18n'
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

function accessColor(mode: string): string {
  if (mode === 'public') return 'success'
  if (mode === 'password' || mode === 'expiry') return 'warning'
  return 'grey'
}

function accessLabel(mode: string): string {
  return t(`access.${mode}`)
}
</script>

<template>
  <v-container class="pa-6" max-width="1200">
    <h1 class="text-h4 mb-6">{{ t('home.title') }}</h1>

    <v-alert v-if="error" type="error" variant="tonal" class="mb-4">
      {{ error }}
    </v-alert>

    <v-row class="mb-6">
      <v-col cols="12" sm="4">
        <v-card variant="tonal">
          <v-card-text>
            <div class="text-overline">{{ t('home.stat.apps') }}</div>
            <div class="text-h4">{{ apps.length }}</div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" sm="4">
        <v-card variant="tonal">
          <v-card-text>
            <div class="text-overline">{{ t('home.stat.versions') }}</div>
            <div class="text-h4">{{ totalVersions }}</div>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" sm="4">
        <v-card variant="tonal" color="primary">
          <v-card-text>
            <div class="text-overline">{{ t('home.stat.downloads') }}</div>
            <div class="text-h4">{{ totalDownloads }}</div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-row v-if="apps.length">
      <v-col
        v-for="a in apps"
        :key="a.id"
        cols="12"
        sm="6"
        md="4"
      >
        <v-card :to="`/app/${a.id}`" hover>
          <v-card-text>
            <div class="d-flex align-center mb-2">
              <v-avatar v-if="a.icon" :image="a.icon" size="40" class="mr-3" />
              <v-avatar v-else color="primary" size="40" class="mr-3">
                <span class="text-h6">{{ a.name.charAt(0).toUpperCase() }}</span>
              </v-avatar>
              <span class="text-h6">{{ a.name }}</span>
            </div>
            <div v-if="a.latest_version" class="text-body-2 text-medium-emphasis mb-1">
              <code>{{ a.latest_version.version_name }}</code>
              · {{ fmtSize(a.latest_version.file_size) }}
            </div>
            <v-chip
              v-if="a.latest_version"
              :color="accessColor(a.latest_version.access_mode)"
              size="small"
              variant="tonal"
            >
              {{ accessLabel(a.latest_version.access_mode) }}
            </v-chip>
            <p v-if="a.description" class="text-body-2 mt-2 mb-0">{{ a.description }}</p>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-card v-else-if="!error" variant="tonal" class="text-center pa-8">
      <v-card-text>{{ t('home.empty') }}</v-card-text>
    </v-card>
  </v-container>
</template>
