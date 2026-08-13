<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../../api/client'
import { useI18n } from '../../composables/useI18n'
import type { AppItem, Channel } from '../../api/types'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const file = ref<File | null>(null)
const initialAppId = Number(route.query.app_id)
const appId = ref<number | null>(Number.isFinite(initialAppId) && initialAppId > 0 ? initialAppId : null)
const channelId = ref<number | null>(null)
const versionName = ref('')
const versionCode = ref<number | null>(null)
const changelog = ref('')
const accessMode = ref<'public' | 'password' | 'expiry'>('public')
const password = ref('')
const expiresAt = ref('')
const error = ref('')
const loading = ref(false)

const apps = ref<AppItem[]>([])
const channels = ref<Channel[]>([])

onMounted(async () => {
  try {
    apps.value = await api.adminApps()
  } catch (e) {
    error.value = (e as Error).message
  }
})

const appItems = computed(() =>
  apps.value.map((a) => ({ title: a.name, value: a.id }))
)

const channelItems = computed(() =>
  channels.value.map((c) => ({ title: c.name, value: c.id }))
)

watch(appId, async (id) => {
  if (!id) {
    channels.value = []
    return
  }
  try {
    channels.value = await api.channels(id)
  } catch (e) {
    error.value = (e as Error).message
  }
})

function onFileChange(f: File | File[] | null) {
  if (Array.isArray(f)) file.value = f[0] ?? null
  else file.value = f
}

async function submit() {
  if (!file.value || !appId.value || !versionName.value) {
    error.value = t('upload.required')
    return
  }
  error.value = ''
  loading.value = true

  const form = new FormData()
  form.append('file', file.value)
  form.append('app_id', String(appId.value))
  if (channelId.value) form.append('channel_id', String(channelId.value))
  form.append('version_name', versionName.value)
  if (versionCode.value) form.append('version_code', String(versionCode.value))
  form.append('changelog', changelog.value)
  form.append('access_mode', accessMode.value)
  if (accessMode.value === 'password') form.append('password', password.value)
  if (accessMode.value === 'expiry' && expiresAt.value) {
    form.append('expires_at', new Date(expiresAt.value).toISOString())
  }

  try {
    await api.uploadVersion(form)
    router.push(`/admin/app/${appId.value}`)
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <v-container class="pa-6" max-width="720">
    <h1 class="text-h4 mb-6">{{ t('upload.title') }}</h1>

    <v-alert v-if="error" type="error" variant="tonal" class="mb-4" closable>
      {{ error }}
    </v-alert>

    <v-form @submit.prevent="submit">
      <v-card variant="outlined" class="mb-4">
        <v-card-text>
          <v-file-input
            :model-value="file"
            :label="t('upload.file')"
            accept=".apk,.aab,.ipa,.exe,.dmg"
            prepend-icon=""
            show-size
            @update:model-value="onFileChange"
          />
        </v-card-text>
      </v-card>

      <v-card variant="outlined" class="mb-4">
        <v-card-text>
          <v-select
            v-model="appId"
            :items="appItems"
            :label="t('upload.app')"
          />
          <v-select
            v-model="channelId"
            :items="channelItems"
            :label="t('upload.channel')"
            :disabled="!appId"
            clearable
          />
          <v-row>
            <v-col cols="12" sm="6">
              <v-text-field v-model="versionName" :label="t('upload.versionName')" placeholder="1.0.0" />
            </v-col>
            <v-col cols="12" sm="6">
              <v-text-field
                v-model.number="versionCode"
                :label="t('upload.versionCode')"
                type="number"
                placeholder="1"
              />
            </v-col>
          </v-row>
          <v-textarea
            v-model="changelog"
            :label="t('upload.changelog')"
            rows="3"
            auto-grow
          />
        </v-card-text>
      </v-card>

      <v-card variant="outlined" class="mb-4">
        <v-card-text>
          <div class="text-overline mb-2">{{ t('upload.access') }}</div>
          <v-radio-group v-model="accessMode">
            <v-radio :label="t('upload.accessPublic')" value="public" />
            <v-radio :label="t('upload.accessPassword')" value="password" />
            <v-radio :label="t('upload.accessExpiry')" value="expiry" />
          </v-radio-group>
          <v-text-field
            v-if="accessMode === 'password'"
            v-model="password"
            :label="t('upload.downloadPassword')"
            type="password"
          />
          <v-text-field
            v-if="accessMode === 'expiry'"
            v-model="expiresAt"
            :label="t('upload.expiresAt')"
            type="datetime-local"
          />
        </v-card-text>
      </v-card>

      <div class="d-flex justify-end" style="gap: 8px;">
        <v-btn @click="router.back()">{{ t('common.cancel') }}</v-btn>
        <v-btn
          color="primary"
          variant="flat"
          :loading="loading"
          :disabled="!file || !appId || !versionName"
          @click="submit"
        >
          {{ t('upload.submit') }}
        </v-btn>
      </div>
    </v-form>
  </v-container>
</template>
