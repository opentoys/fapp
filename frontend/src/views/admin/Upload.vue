<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../../api/client'
import type { AppItem, Channel } from '../../api/types'

const router = useRouter()

const file = ref<File | null>(null)
const appId = ref<number | null>(null)
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
    error.value = 'File, app, and version name are required.'
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
  <div class="upload">
    <div class="page-header">
      <div class="eyebrow">▌ UPLOAD</div>
      <h1 class="title">New Version</h1>
    </div>

    <v-alert v-if="error" type="error" variant="outlined" class="mb-4">
      {{ error }}
    </v-alert>

    <div class="form">
      <section class="form-section">
        <div class="eyebrow">▌ FILE</div>
        <v-file-input
          :model-value="file"
          label="Choose installation package"
          accept=".apk,.aab,.ipa,.exe,.dmg"
          prepend-icon=""
          show-size
          density="comfortable"
          @update:model-value="onFileChange"
        />
      </section>

      <section class="form-section">
        <div class="eyebrow">▌ METADATA</div>
        <v-select
          v-model="appId"
          :items="appItems"
          label="Application"
        />
        <v-select
          v-model="channelId"
          :items="channelItems"
          label="Channel"
          :disabled="!appId"
          clearable
        />
        <div class="row-2">
          <v-text-field v-model="versionName" label="Version name" placeholder="1.0.0" />
          <v-text-field
            v-model.number="versionCode"
            label="Version code"
            type="number"
            placeholder="1"
          />
        </div>
        <v-textarea
          v-model="changelog"
          label="Changelog"
          rows="3"
          auto-grow
        />
      </section>

      <section class="form-section">
        <div class="eyebrow">▌ ACCESS</div>
        <v-radio-group v-model="accessMode" inline>
          <v-radio label="Public" value="public" />
          <v-radio label="Password" value="password" />
          <v-radio label="Expires" value="expiry" />
        </v-radio-group>
        <v-text-field
          v-if="accessMode === 'password'"
          v-model="password"
          label="Download password"
          type="password"
        />
        <v-text-field
          v-if="accessMode === 'expiry'"
          v-model="expiresAt"
          label="Expires at"
          type="datetime-local"
        />
      </section>

      <div class="actions">
        <v-btn variant="text" @click="router.back()">Cancel</v-btn>
        <v-btn
          color="primary"
          :loading="loading"
          :disabled="!file || !appId || !versionName"
          @click="submit"
        >
          Upload
        </v-btn>
      </div>
    </div>
  </div>
</template>

<style scoped>
.upload {
  max-width: 720px;
  margin: 0 auto;
  padding: var(--sp-8) var(--sp-6);
}
.page-header {
  margin-bottom: var(--sp-6);
}
.eyebrow {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: var(--text-mute);
  margin-bottom: var(--sp-2);
}
.title {
  font-size: 2.25rem;
  font-weight: 500;
  letter-spacing: -0.02em;
  margin: 0 0 var(--sp-6) 0;
}
.form {
  display: flex;
  flex-direction: column;
  gap: var(--sp-6);
}
.form-section {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
  padding-bottom: var(--sp-6);
  border-bottom: 1px solid var(--border);
}
.form-section:last-of-type {
  border-bottom: none;
}
.row-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--sp-3);
}
.actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--sp-2);
}
@media (max-width: 600px) {
  .row-2 { grid-template-columns: 1fr; }
}
</style>
