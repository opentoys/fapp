<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api/client'
import type { AppDetail as AD, Version } from '../api/types'

const route = useRoute()
const detail = ref<AD | null>(null)
const pwd = ref<Record<number, string>>({})
const error = ref('')

onMounted(async () => {
  try {
    detail.value = await api.appDetail(Number(route.params.id))
  } catch (e) {
    error.value = (e as Error).message
  }
})

function expired(v: Version): boolean {
  return v.access_mode === 'expiry' && !!v.expires_at && Date.parse(v.expires_at) < Date.now()
}

function fmtSize(n: number): string {
  if (n > 1024 * 1024 * 1024) return (n / 1024 / 1024 / 1024).toFixed(2) + ' GB'
  if (n > 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MB'
  if (n > 1024) return (n / 1024).toFixed(1) + ' KB'
  return n + ' B'
}

async function trigger(v: Version, kind: 'download' | 'install') {
  if (expired(v)) return
  try {
    const url = kind === 'download' ? await api.downloadUrl(v.id, pwd.value[v.id]) : await api.installUrl(v.id, pwd.value[v.id])
    window.open(url, '_blank')
    if (kind === 'download') v.download_count++
  } catch (e) {
    alert((e as Error).message)
  }
}
</script>

<template>
  <div v-if="detail" class="detail">
    <header>
      <h1>{{ detail.app.name }}</h1>
      <p>{{ detail.app.description }}</p>
    </header>
    <p v-if="error" class="err">{{ error }}</p>

    <div class="channels" v-for="c in detail.channels" :key="c.id">
      <h3>渠道：{{ c.name }}</h3>
      <div class="version" v-for="v in detail.versions.filter((x) => x.channel_id === c.id)" :key="v.id">
        <div class="vinfo">
          <span class="ver">{{ v.version_name }}</span>
          <span class="meta">{{ v.file_type.toUpperCase() }} · {{ fmtSize(v.file_size) }}</span>
        </div>
        <pre class="log" v-if="v.changelog">{{ v.changelog }}</pre>
        <div v-if="expired(v)" class="expired">链接已过期</div>
        <div v-else class="actions">
          <input
            v-if="v.access_mode === 'password'"
            v-model="pwd[v.id]"
            type="password"
            placeholder="访问密码"
            @keyup.enter="trigger(v, 'download')"
          />
          <button @click="trigger(v, 'download')">下载</button>
          <button @click="trigger(v, 'install')">安装</button>
          <span class="counts">下载 {{ v.download_count }} · 安装 {{ v.install_count }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.detail { max-width: 720px; margin: 0 auto; padding: 24px; }
.version { border: 1px solid #ddd; border-radius: 8px; padding: 12px; margin: 12px 0; }
.vinfo { display: flex; gap: 12px; align-items: baseline; }
.ver { font-weight: 600; font-size: 18px; }
.meta, .counts { color: #888; font-size: 13px; }
.log { white-space: pre-wrap; background: #f6f6f6; padding: 8px; border-radius: 4px; }
.actions { display: flex; gap: 8px; align-items: center; margin-top: 8px; }
.expired { color: #d33; font-weight: 600; }
.err { color: #d33; }
</style>