<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../../api/client'
import type { AppItem } from '../../api/types'

const apps = ref<AppItem[]>([])
const name = ref('')
const err = ref('')

onMounted(load)
async function load() {
  apps.value = await api.adminApps()
}
async function create() {
  if (!name.value) return
  try {
    await api.createApp({ name: name.value })
    name.value = ''
    load()
  } catch (e) {
    err.value = (e as Error).message
  }
}
async function remove(id: number) {
  if (!confirm('删除该应用？关联版本与渠道将一并删除。')) return
  await api.deleteApp(id)
  load()
}
</script>

<template>
  <div class="admin">
    <h1>应用管理</h1>
    <div class="create">
      <input v-model="name" placeholder="应用名称" @keyup.enter="create" />
      <button @click="create">新建</button>
    </div>
    <p v-if="err" class="err">{{ err }}</p>
    <table>
      <tr v-for="a in apps" :key="a.id">
        <td><router-link :to="`/admin/app/${a.id}`">{{ a.name }}</router-link></td>
        <td><button @click="remove(a.id)">删除</button></td>
      </tr>
    </table>
    <router-link to="/admin/upload">上传新版本</router-link>
  </div>
</template>

<style scoped>
.admin { max-width: 720px; margin: 0 auto; padding: 24px; }
.create { display: flex; gap: 8px; margin-bottom: 16px; }
table { width: 100%; border-collapse: collapse; }
td { padding: 8px; border-bottom: 1px solid #eee; }
.err { color: #d33; }
</style>