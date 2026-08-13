<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api/client'
import type { AppDetail } from '../api/types'

const route = useRoute()
const data = ref<AppDetail | null>(null)
const error = ref('')
const passwordPrompt = ref<{ versionId: number; password: string } | null>(null)
const dialogOpen = ref(false)
const passwordError = ref('')

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

function fmtSize(n: number): string {
  if (n > 1024 * 1024 * 1024) return (n / 1024 / 1024 / 1024).toFixed(2) + ' GB'
  if (n > 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MB'
  if (n > 1024) return (n / 1024).toFixed(1) + ' KB'
  return n + ' B'
}

function fmtDate(s: string): string {
  return new Date(s).toISOString().replace('T', ' ').slice(0, 19)
}

function accessColor(mode: string, enabled: boolean): string {
  if (!enabled) return 'error'
  if (mode === 'public') return 'success'
  if (mode === 'password' || mode === 'expiry') return 'warning'
  return 'grey'
}

function accessLabel(mode: string, enabled: boolean): string {
  if (!enabled) return 'taken down'
  return mode
}

async function download(v: { id: number; access_mode: string }) {
  passwordError.value = ''
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
      <v-row>
        <v-col cols="12" md="4">
          <h1 class="text-h4 mb-2">{{ data.app.name }}</h1>
          <p v-if="data.app.description" class="text-body-1 mb-4">{{ data.app.description }}</p>

          <div v-if="data.channels.length" class="mb-4">
            <div class="text-overline mb-2">Channels</div>
            <v-chip
              v-for="c in data.channels"
              :key="c.id"
              class="mr-2 mb-2"
              variant="outlined"
            >
              {{ c.name }}
            </v-chip>
          </div>
        </v-col>

        <v-col cols="12" md="8">
          <div class="text-overline mb-3">Versions</div>

          <v-card v-if="data.versions.length" variant="outlined">
            <v-list lines="three">
              <v-list-item
                v-for="v in data.versions"
                :key="v.id"
                :class="{ 'text-disabled': !v.enabled }"
              >
                <template #prepend>
                  <v-chip
                    :color="accessColor(v.access_mode, v.enabled)"
                    size="small"
                    variant="tonal"
                    class="mr-3"
                  >
                    {{ accessLabel(v.access_mode, v.enabled) }}
                  </v-chip>
                </template>

                <v-list-item-title>
                  <span class="text-h6">{{ v.version_name }}</span>
                  <span class="text-body-2 text-medium-emphasis ml-2">
                    code {{ v.version_code }} · {{ fmtSize(v.file_size) }}
                  </span>
                </v-list-item-title>

                <v-list-item-subtitle>
                  <code class="text-caption">{{ v.sha256.slice(0, 16) }}…</code>
                  <span class="text-caption text-medium-emphasis ml-2">{{ fmtDate(v.created_at) }}</span>
                </v-list-item-subtitle>

                <template v-if="v.changelog" #append>
                  <v-list-item-subtitle class="text-body-2 mt-1" style="white-space: pre-wrap;">
                    {{ v.changelog }}
                  </v-list-item-subtitle>
                </template>

                <template #append>
                  <v-btn
                    v-if="v.enabled"
                    color="primary"
                    variant="flat"
                    size="small"
                    :disabled="v.access_mode === 'expiry' && !!v.expires_at && new Date(v.expires_at) < new Date()"
                    @click="download(v)"
                  >
                    Download
                  </v-btn>
                </template>
              </v-list-item>
            </v-list>
          </v-card>

          <v-card v-else variant="tonal" class="text-center pa-8">
            <v-card-text>No versions yet.</v-card-text>
          </v-card>
        </v-col>
      </v-row>
    </template>

    <v-dialog v-model="dialogOpen" max-width="400">
      <v-card>
        <v-card-title>Password required</v-card-title>
        <v-card-text>
          <p class="mb-3">This version is password protected.</p>
          <v-text-field
            v-if="passwordPrompt"
            v-model="passwordPrompt.password"
            label="Password"
            type="password"
            autofocus
            @keyup.enter="submitPassword"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="closePasswordPrompt">Cancel</v-btn>
          <v-btn color="primary" variant="flat" @click="submitPassword">Continue</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>
