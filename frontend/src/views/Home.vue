<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '../api/client'
import type { AppItem } from '../api/types'
import StatusDot from '../components/StatusDot.vue'
import StatBlock from '../components/StatBlock.vue'
import MonoText from '../components/MonoText.vue'

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
</script>

<template>
  <div class="home">
    <div class="page-header">
      <div class="eyebrow">▌ DISTRIBUTION</div>
      <h1 class="title">Apps</h1>
      <div class="stat-strip">
        <StatBlock label="Apps" :value="apps.length" />
        <StatBlock label="Versions" :value="totalVersions" />
        <StatBlock label="Downloads" :value="totalDownloads" emphasis />
      </div>
    </div>

    <v-alert v-if="error" type="error" variant="outlined" class="mb-4">
      {{ error }}
    </v-alert>

    <div v-if="apps.length" class="grid">
      <router-link
        v-for="a in apps"
        :key="a.id"
        :to="`/app/${a.id}`"
        class="card-link"
      >
        <v-card class="hoverable pa-4">
          <div class="card-head">
            <img v-if="a.icon" :src="a.icon" alt="" class="icon" />
            <div v-else class="icon-fallback">{{ a.name.charAt(0).toUpperCase() }}</div>
            <div class="card-name">{{ a.name }}</div>
          </div>
          <div v-if="a.latest_version" class="card-meta">
            <MonoText>{{ a.latest_version.version_name }}</MonoText>
            <MonoText muted> · {{ fmtSize(a.latest_version.file_size) }}</MonoText>
          </div>
          <div v-if="a.latest_version" class="card-status">
            <StatusDot :mode="a.latest_version.access_mode" />
          </div>
          <div v-if="a.description" class="card-desc">{{ a.description }}</div>
        </v-card>
      </router-link>
    </div>

    <div v-else-if="!error" class="empty">
      <div class="eyebrow">▌ NO APPS</div>
      <p>no applications yet</p>
    </div>
  </div>
</template>

<style scoped>
.home {
  max-width: var(--max-w);
  margin: 0 auto;
  padding: var(--sp-8) var(--sp-6);
}
.page-header {
  margin-bottom: var(--sp-8);
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
.stat-strip {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--sp-2);
  max-width: 600px;
}
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: var(--sp-4);
}
.card-link {
  text-decoration: none;
  color: inherit;
}
.card-head {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  margin-bottom: var(--sp-3);
}
.icon {
  width: 40px;
  height: 40px;
  object-fit: contain;
}
.icon-fallback {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border);
  background: var(--surface-2);
  font-family: var(--font-mono);
  font-size: 1.1rem;
  color: var(--accent);
}
.card-name {
  font-size: 1.1rem;
  font-weight: 500;
}
.card-meta {
  margin-bottom: var(--sp-2);
}
.card-status {
  margin-bottom: var(--sp-3);
}
.card-desc {
  font-size: 0.85rem;
  color: var(--text-mute);
  line-height: 1.4;
}
.empty {
  text-align: center;
  padding: var(--sp-8) 0;
}
.empty p {
  color: var(--text-mute);
  margin: 0;
}
@media (max-width: 600px) {
  .stat-strip { grid-template-columns: 1fr; }
}
</style>
