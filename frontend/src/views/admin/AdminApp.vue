<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../../api/client'
import type { AppDetail } from '../../api/types'

const route = useRoute()
const appId = Number(route.params.id)
const detail = ref<AppDetail | null>(null)
const chName = ref('')
const confirmPassword = ref<Record<number, string>>({})

onMounted(load)
async function load() {
  detail.value = await api.appDetail(appId)
}
async function addChannel() {
  if (!chName.value) return
  await api.createChannel(appId, chName.value)
  chName.value = ''
  load()
}
async function toggle(v: { id: number; enabled: boolean }) {
  await api.updateVersion(v.id, { enabled: !v.enabled })
  load()
}
async function removeVersion(id: number) {
  if (!confirm('删除该版本？')) return
  await api.deleteVersion(id, true)
  load()
}
async function setPassword(v: { id: number }, pwd: string) {
  if (!pwd) return
  await api.updateVersion(v.id, { password: pwd, access_mode: 'password' })
  delete confirmPassword.value[v.id]
  load()
}
function fmtSize(n: number): string {
  if (n > 1024 * 1024 * 1024) return (n / 1024 / 1024 / 1024).toFixed(2) + ' GB'
  if (n > 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MB'
  if (n > 1024) return (n / 1024).toFixed(1) + ' KB'
  return n + ' B'
}
</script>

<template>
  <div v-if="detail" class="page">
    <h1>{{ detail.app.name }}</h1>

    <h3>渠道</h3>
    <div class="row">
      <span v-for="c in detail.channels" :key="c.id" class="chip">{{ c.name }}</span>
      <input v-model="chName" placeholder="新渠道名" @keyup.enter="addChannel" />
      <button @click="addChannel">添加</button>
    </div>

    <h3>版本</h3>
    <router-link to="/admin/upload" class="link">上传新版本 →</router-link>
    <div class="version" v-for="v in detail.versions" :key="v.id">
      <div class="vinfo">
        <b>{{ v.version_name }}</b>
        <span>{{ v.file_type.toUpperCase() }} · {{ fmtSize(v.file_size) }}</span>
        <span :class="v.enabled ? 'on' : 'off'">{{ v.enabled ? '上架中' : '已下架' }}</span>
        <span>{{ v.access_mode }}</span>
        <span>下载 {{ v.download_count }} · 安装 {{ v.install_count }}</span>
      </div>
      <div class="actions">
        <button @click="toggle(v)">{{ v.enabled ? '下架' : '上架' }}</button>
        <button @click="removeVersion(v.id)">删除</button>
      </div>
      <div class="actions" v-if="v.access_mode === 'password' || v.access_mode === 'public'">
        <input v-model="confirmPassword[v.id]" type="password" placeholder="设访问密码" />
        <button @click="setPassword(v, confirmPassword[v.id])">设置密码</button>
      </div>
      <router-link :to="`/app/${appId}`">查看下载页</router-link>
    </div>
  </div>
</template>

<style scoped>
.page { max-width: 720px; margin: 0 auto; padding: 24px; }
.row { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.chip { background: #eef; padding: 4px 10px; border-radius: 12px; }
.version { border: 1px solid #ddd; border-radius: 8px; padding: 12px; margin: 12px 0; }
.vinfo { display: flex; gap: 12px; align-items: center; flex-wrap: wrap; }
.actions { display: flex; gap: 8px; margin-top: 8px; }
.on { color: #2a2; } .off { color: #a22; }
.link { display: inline-block; margin-bottom: 8px; }
</style>