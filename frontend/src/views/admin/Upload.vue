<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../../api/client'
import type { AppItem, Channel } from '../../api/types'

const router = useRouter()
const apps = ref<AppItem[]>([])
const channels = ref<Channel[]>([])

const appId = ref(0)
const channelId = ref(0)
const versionName = ref('')
const versionCode = ref(0)
const changelog = ref('')
const accessMode = ref<'public' | 'password' | 'expiry'>('public')
const password = ref('')
const expiresAt = ref('')
const file = ref<File | null>(null)
const err = ref('')

onMounted(async () => {
  apps.value = await api.adminApps()
})
async function pickApp() {
  channels.value = await api.channels(appId.value)
  channelId.value = channels.value[0]?.id ?? 0
}
function onFile(e: Event) {
  file.value = (e.target as HTMLInputElement).files?.[0] ?? null
}
async function submit() {
  if (!file.value || !appId.value || !versionName.value) {
    err.value = '请填写应用、版本号与文件'
    return
  }
  const form = new FormData()
  form.append('file', file.value)
  form.append('app_id', String(appId.value))
  form.append('channel_id', String(channelId.value))
  form.append('version_name', versionName.value)
  form.append('version_code', String(versionCode.value))
  form.append('changelog', changelog.value)
  form.append('access_mode', accessMode.value)
  if (accessMode.value === 'password') form.append('password', password.value)
  if (accessMode.value === 'expiry' && expiresAt.value) form.append('expires_at', new Date(expiresAt.value).toISOString())
  try {
    await api.uploadVersion(form)
    router.push(`/admin/app/${appId.value}`)
  } catch (e) {
    err.value = (e as Error).message
  }
}
</script>

<template>
  <div class="upload">
    <h1>上传新版本</h1>
    <p v-if="err" class="err">{{ err }}</p>

    <label>应用</label>
    <select v-model.number="appId" @change="pickApp">
      <option :value="0" disabled>选择应用</option>
      <option v-for="a in apps" :key="a.id" :value="a.id">{{ a.name }}</option>
    </select>

    <label>渠道</label>
    <select v-model.number="channelId">
      <option v-for="c in channels" :key="c.id" :value="c.id">{{ c.name }}</option>
    </select>

    <label>版本号（如 1.2.3）</label>
    <input v-model="versionName" />

    <label>版本 Code</label>
    <input v-model.number="versionCode" type="number" />

    <label>更新日志</label>
    <textarea v-model="changelog" rows="4"></textarea>

    <label>访问模式</label>
    <select v-model="accessMode">
      <option value="public">公开</option>
      <option value="password">密码</option>
      <option value="expiry">有效期</option>
    </select>

    <label v-if="accessMode === 'password'">访问密码</label>
    <input v-if="accessMode === 'password'" v-model="password" />

    <label v-if="accessMode === 'expiry'">过期时间</label>
    <input v-if="accessMode === 'expiry'" v-model="expiresAt" type="datetime-local" />

    <label>安装包文件</label>
    <input type="file" @change="onFile" />

    <button @click="submit">上传</button>
  </div>
</template>

<style scoped>
.upload { max-width: 480px; margin: 0 auto; padding: 24px; display: flex; flex-direction: column; gap: 8px; }
.err { color: #d33; }
</style>