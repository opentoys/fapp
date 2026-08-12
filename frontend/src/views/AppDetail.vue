<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api/client'
import type { AppDetail } from '../api/types'
import StatusDot from '../components/StatusDot.vue'
import MonoText from '../components/MonoText.vue'

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
  <div class="app-detail">
    <v-alert v-if="error" type="error" variant="outlined" class="mb-4">
      {{ error }}
    </v-alert>

    <div v-if="data" class="layout">
      <aside class="left">
        <div class="eyebrow">▌ APP</div>
        <h1 class="title">{{ data.app.name }}</h1>
        <p v-if="data.app.description" class="desc">{{ data.app.description }}</p>

        <div v-if="data.channels.length" class="channels">
          <div class="eyebrow">▌ CHANNELS</div>
          <div class="channel-list">
            <span v-for="c in data.channels" :key="c.id" class="channel">
              <MonoText>{{ c.name }}</MonoText>
            </span>
          </div>
        </div>
      </aside>

      <section class="right">
        <div class="eyebrow">▌ VERSIONS</div>
        <div v-if="data.versions.length" class="version-list">
          <div
            v-for="v in data.versions"
            :key="v.id"
            class="version-row"
            :class="{ disabled: !v.enabled }"
          >
            <div class="ver-head">
              <MonoText class="ver-name">{{ v.version_name }}</MonoText>
              <MonoText muted> · code {{ v.version_code }} · {{ fmtSize(v.file_size) }}</MonoText>
              <span v-if="!v.enabled" class="taken-down">TAKEN DOWN</span>
            </div>
            <div class="ver-meta">
              <MonoText muted class="sha">{{ v.sha256.slice(0, 16) }}…</MonoText>
              <MonoText muted> · {{ fmtDate(v.created_at) }}</MonoText>
            </div>
            <div class="ver-status">
              <StatusDot :mode="v.enabled ? v.access_mode : 'taken_down'" />
            </div>
            <p v-if="v.changelog" class="changelog">{{ v.changelog }}</p>
            <div v-if="v.enabled" class="actions">
              <v-btn
                variant="outlined"
                size="small"
                :disabled="v.access_mode === 'expiry' && !!v.expires_at && new Date(v.expires_at) < new Date()"
                @click="download(v)"
              >
                Download
              </v-btn>
            </div>
          </div>
        </div>
        <div v-else class="empty">
          <p>no versions yet</p>
        </div>
      </section>
    </div>

    <v-dialog v-model="dialogOpen" max-width="400">
      <v-card class="pa-5">
        <div class="eyebrow">▌ PASSWORD REQUIRED</div>
        <p class="dialog-body">This version is password protected.</p>
        <v-text-field
          v-if="passwordPrompt"
          v-model="passwordPrompt.password"
          label="Password"
          type="password"
          autofocus
          @keyup.enter="submitPassword"
        />
        <div class="dialog-actions">
          <v-btn variant="text" @click="closePasswordPrompt">Cancel</v-btn>
          <v-btn color="primary" @click="submitPassword">Continue</v-btn>
        </div>
      </v-card>
    </v-dialog>
  </div>
</template>

<style scoped>
.app-detail {
  max-width: var(--max-w);
  margin: 0 auto;
  padding: var(--sp-8) var(--sp-6);
}
.layout {
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: var(--sp-8);
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
  margin: 0 0 var(--sp-3) 0;
}
.desc {
  color: var(--text-mute);
  margin: 0 0 var(--sp-6) 0;
}
.channels {
  margin-top: var(--sp-4);
}
.channel-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--sp-2);
}
.channel {
  display: inline-block;
  padding: 4px 8px;
  border: 1px solid var(--border);
  background: var(--surface);
  font-size: 0.8rem;
}
.right .eyebrow {
  margin-bottom: var(--sp-3);
}
.version-list {
  border-top: 1px solid var(--border);
}
.version-row {
  padding: var(--sp-4) 0;
  border-bottom: 1px solid var(--border);
}
.version-row.disabled { opacity: 0.5; }
.ver-head {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  margin-bottom: var(--sp-1);
  flex-wrap: wrap;
}
.ver-name {
  font-size: 1.1rem;
  color: var(--text);
}
.taken-down {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  letter-spacing: 0.1em;
  color: var(--danger);
  border: 1px solid var(--danger);
  padding: 2px 6px;
}
.ver-meta {
  margin-bottom: var(--sp-2);
}
.sha {
  font-size: 0.75rem;
}
.ver-status {
  margin-bottom: var(--sp-2);
}
.changelog {
  font-size: 0.85rem;
  color: var(--text-mute);
  margin: var(--sp-2) 0;
  white-space: pre-wrap;
}
.actions {
  margin-top: var(--sp-2);
}
.empty {
  padding: var(--sp-6) 0;
  color: var(--text-mute);
  text-align: center;
}
.dialog-body {
  margin: var(--sp-3) 0;
  color: var(--text-mute);
}
.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--sp-2);
  margin-top: var(--sp-3);
}
@media (max-width: 900px) {
  .layout { grid-template-columns: 1fr; }
}
</style>
