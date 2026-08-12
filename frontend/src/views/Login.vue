<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { api } from '../api/client'

const username = ref('')
const password = ref('')
const error = ref('')
const router = useRouter()
const route = useRoute()

async function submit() {
  try {
    const res = await api.login(username.value, password.value)
    localStorage.setItem('token', res.data.data.token)
    router.push((route.query.redirect as string) || '/admin')
  } catch (e) {
    error.value = (e as Error).message
  }
}
</script>

<template>
  <div class="login">
    <h1>登录</h1>
    <input v-model="username" placeholder="用户名" />
    <input v-model="password" type="password" placeholder="密码" @keyup.enter="submit" />
    <button @click="submit">登录</button>
    <p v-if="error" class="err">{{ error }}</p>
  </div>
</template>

<style scoped>
.login { max-width: 320px; margin: 80px auto; display: flex; flex-direction: column; gap: 12px; }
.err { color: #d33; }
</style>