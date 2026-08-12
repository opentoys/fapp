<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api/client'

const route = useRoute()
const router = useRouter()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    const res = await api.login(username.value, password.value)
    localStorage.setItem('token', res.data.data.token)
    const redirect = (route.query.redirect as string) || '/admin'
    router.push(redirect)
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="grid-bg" />
    <v-card class="login-card pa-6">
      <div class="eyebrow">▌ SIGN IN</div>
      <h1 class="title">Distribution Console</h1>

      <v-form @submit.prevent="submit">
        <v-text-field
          v-model="username"
          label="Username"
          autocomplete="username"
          autofocus
        />
        <v-text-field
          v-model="password"
          label="Password"
          type="password"
          autocomplete="current-password"
          @keyup.enter="submit"
        />
        <v-alert v-if="error" type="error" variant="outlined" class="mb-3">
          {{ error }}
        </v-alert>
        <v-btn
          color="primary"
          block
          :loading="loading"
          type="submit"
        >
          Authenticate
        </v-btn>
      </v-form>
    </v-card>
  </div>
</template>

<style scoped>
.login-page {
  min-height: calc(100vh - var(--topbar-h));
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  padding: var(--sp-6);
}
.grid-bg {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(to right, var(--border) 1px, transparent 1px),
    linear-gradient(to bottom, var(--border) 1px, transparent 1px);
  background-size: 32px 32px;
  opacity: 0.15;
  pointer-events: none;
}
.login-card {
  position: relative;
  width: 100%;
  max-width: 360px;
  background: var(--surface) !important;
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
  font-size: 1.5rem;
  font-weight: 500;
  letter-spacing: -0.02em;
  margin: 0 0 var(--sp-5) 0;
}
</style>
