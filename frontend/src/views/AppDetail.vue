<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { mdiDownload } from '@mdi/js'
import { api } from '../api/client'
import { useI18n } from '../composables/useI18n'
import { detectUA } from '../utils/ua'
import { PLATFORMS } from '../constants/platform'
import VersionPanel from '../components/VersionPanel.vue'
import type { AppDetail, Platform, Version } from '../api/types'

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

// Latest version per platform (first encounter wins in a desc-sorted list).
const byPlatform = computed(() => {
  const map = new Map<Platform, Version>()
  for (const v of versions.value) {
    if (v.platform && !map.has(v.platform)) map.set(v.platform, v)
  }
  return map
})

// Platform cards shown on desktop / unknown UA, in canonical platform order.
const platformList = computed(() => {
  const result: { platform: Platform; version: Version }[] = []
  for (const p of PLATFORMS) {
    const v = byPlatform.value.get(p)
    if (v) result.push({ platform: p, version: v })
  }
  return result
})

// Mobile (android/ios): show that platform's latest version as a no-card hero
// instead of the per-platform grid. If the app has no version for the detected
// platform, fall back to the newest version so mobile styles always apply.
const mobileVersion = computed(() => {
  if (detected.isDesktop || !detected.platform) return null
  return byPlatform.value.get(detected.platform) ?? versions.value[0] ?? null
})

function openPasswordPrompt(versionId: number) {
  passwordPrompt.value = { versionId, password: '' }
  dialogOpen.value = true
}
function closePasswordPrompt() {
  dialogOpen.value = false
  passwordPrompt.value = null
}

onMounted(load)
async function load() {
  try {
    data.value = await api.appDetail(Number(route.params.id))
  } catch (e) {
    error.value = (e as Error).message
  }
}

function isExpired(v: { access_mode: string; expires_at: string | null } | null): boolean {
  return !!v && v.access_mode === 'expiry' && !!v.expires_at && new Date(v.expires_at) < new Date()
}

async function download(v: Version | null) {
  if (!v) return
  if (v.access_mode === 'password') {
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
</script>

<template>
  <v-container class="pa-6" max-width="1200">
    <v-alert v-if="error" type="error" variant="tonal" class="mb-4" closable>
      {{ error }}
    </v-alert>

    <template v-if="data">
      <div class="d-flex align-center mb-2" style="gap: 12px;">
        <v-avatar v-if="data.app.icon" :image="data.app.icon" size="40" />
        <v-avatar v-else color="primary" size="40">
          <span class="text-h6">{{ data.app.name.charAt(0).toUpperCase() }}</span>
        </v-avatar>
        <h1 class="text-h4 mb-0">{{ data.app.name }}</h1>
      </div>
      <p v-if="data.app.description" class="text-body-1 mb-6">{{ data.app.description }}</p>

      <!-- Mobile: detected platform's latest version, no card wrapper -->
      <div v-if="mobileVersion" style="max-width: 560px;" class="pb-18">
        <VersionPanel
          :version="mobileVersion"
          :fallback-name="data.app.name"
          :fallback-icon="data.app.icon"
          :no-download="true"
          @download="download"
        />
      </div>

      <!-- Desktop / unknown UA: one card per platform -->
      <v-row v-else-if="platformList.length">
        <v-col
          v-for="{ platform, version } in platformList"
          :key="platform"
          cols="12"
          md="6"
        >
          <v-card :variant="detected.isDesktop ? 'outlined' : 'flat'" class="pa-4 h-100">
            <VersionPanel
              :version="version"
              :fallback-name="data.app.name"
              :fallback-icon="data.app.icon"
              @download="download"
            />
          </v-card>
        </v-col>
      </v-row>

      <v-card v-else-if="!error" variant="tonal" class="text-center pa-8">
        <v-card-text>{{ t('detail.empty') }}</v-card-text>
      </v-card>
    </template>

    <!-- Floating download button: 80% width, pinned to the viewport bottom on
         the mobile hero view. -->
    <div v-if="mobileVersion && !error" class="download-fab">
      <v-btn
        color="primary"
        size="large"
        variant="flat"
        :disabled="isExpired(mobileVersion)"
        @click="download(mobileVersion)"
      >
        <v-icon start :icon="mdiDownload" />
        {{ t('detail.download') }}
      </v-btn>
    </div>

    <v-dialog v-model="dialogOpen" max-width="400">
      <v-card>
        <v-card-title>{{ t('detail.passwordTitle') }}</v-card-title>
        <v-card-text>
          <p class="mb-3">{{ t('detail.passwordBody') }}</p>
          <v-text-field
            v-if="passwordPrompt"
            v-model="passwordPrompt.password"
            :label="t('common.password')"
            type="password"
            autofocus
            @keyup.enter="submitPassword"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="closePasswordPrompt">{{ t('common.cancel') }}</v-btn>
          <v-btn color="primary" variant="flat" @click="submitPassword">{{ t('detail.passwordContinue') }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<style scoped>
.download-fab {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  justify-content: center;
  padding: 12px 0;
  padding-bottom: calc(12px + env(safe-area-inset-bottom));
  /* Fade so content scrolling underneath stays legible behind the button. */
  background: linear-gradient(to top, rgb(var(--v-theme-surface)) 55%, transparent);
  z-index: 100;
}
.download-fab .v-btn {
  width: 80%;
}
</style>
