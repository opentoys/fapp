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
  <v-container class="d-flex align-center justify-center" style="min-height: calc(100vh - 64px);">
    <v-card max-width="400" width="100%">
      <v-card-title>Sign in</v-card-title>
      <v-card-text>
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
          <v-alert v-if="error" type="error" variant="tonal" class="mb-3">
            {{ error }}
          </v-alert>
          <v-btn
            color="primary"
            variant="flat"
            block
            :loading="loading"
            type="submit"
          >
            Sign in
          </v-btn>
        </v-form>
      </v-card-text>
    </v-card>
  </v-container>
</template>
