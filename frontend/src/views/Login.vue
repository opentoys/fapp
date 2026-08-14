<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api/client'
import { useAuth } from '../composables/useAuth'
import { useI18n } from '../composables/useI18n'
import { Button } from '../components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'
import { Alert } from '../components/ui/alert'

const route = useRoute()
const router = useRouter()
const { setToken } = useAuth()
const { t } = useI18n()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    const res = await api.login(username.value, password.value)
    setToken(res.data.data.token)
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
  <div class="flex min-h-[calc(100vh-3.5rem)] items-center justify-center px-4 py-12">
    <Card class="w-full max-w-sm">
      <CardHeader>
        <CardTitle class="text-lg">{{ t('login.title') }}</CardTitle>
      </CardHeader>
      <CardContent>
        <form class="grid gap-4" @submit.prevent="submit">
          <div class="grid gap-2">
            <Label for="username">{{ t('common.username') }}</Label>
            <Input id="username" v-model="username" autocomplete="username" autofocus />
          </div>
          <div class="grid gap-2">
            <Label for="password">{{ t('common.password') }}</Label>
            <Input id="password" v-model="password" type="password" autocomplete="current-password" @keyup.enter="submit" />
          </div>
          <Alert v-if="error" variant="destructive">{{ error }}</Alert>
          <Button type="submit" :disabled="loading" class="w-full">
            {{ t('login.submit') }}
          </Button>
        </form>
      </CardContent>
    </Card>
  </div>
</template>
