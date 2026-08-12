<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../api/client'
import type { AppItem } from '../api/types'

const apps = ref<AppItem[]>([])
const error = ref('')

onMounted(async () => {
  try {
    apps.value = await api.apps()
  } catch (e) {
    error.value = (e as Error).message
  }
})

function fmtSize(n: number): string {
  if (n > 1024 * 1024 * 1024) return (n / 1024 / 1024 / 1024).toFixed(2) + ' GB'
  if (n > 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MB'
  if (n > 1024) return (n / 1024).toFixed(1) + ' KB'
  return n + ' B'
}
</script>

<template>
  <div class="home">
    <header><h1>App 分发平台</h1></header>
    <p v-if="error" class="err">{{ error }}</p>
    <div class="grid">
      <router-link v-for="a in apps" :key="a.id" class="card" :to="`/app/${a.id}`">
        <img v-if="a.icon" :src="a.icon" alt="" class="icon" />
        <div class="name">{{ a.name }}</div>
        <div class="ver" v-if="a.latest_version">
          {{ a.latest_version.version_name }} · {{ fmtSize(a.latest_version.file_size) }}
        </div>
        <div class="desc">{{ a.description }}</div>
      </router-link>
    </div>
    <p v-if="!apps.length && !error">暂无应用</p>
  </div>
</template>

<style scoped>
.home { max-width: 960px; margin: 0 auto; padding: 24px; }
.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 16px; }
.card { border: 1px solid #ddd; border-radius: 8px; padding: 16px; text-decoration: none; color: inherit; }
.icon { width: 48px; height: 48px; object-fit: contain; }
.ver { color: #888; font-size: 13px; }
.err { color: #d33; }
</style>