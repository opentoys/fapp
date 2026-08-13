<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTheme } from '../composables/useTheme'
import { useAuth } from '../composables/useAuth'
import { mdiWeatherSunny, mdiWeatherNight, mdiThemeLightDark, mdiLogout, mdiLogin } from '@mdi/js'

const route = useRoute()
const router = useRouter()
const { choice, cycle } = useTheme()
const { isAuthed, clearToken } = useAuth()

const tabs = computed(() => {
  if (!isAuthed.value) return []
  return [
    { label: 'Apps', to: '/' },
    { label: 'Admin', to: '/admin' },
    { label: 'Users', to: '/admin/users' },
    { label: 'Upload', to: '/admin/upload' },
  ]
})

const themeIcon = computed(() => {
  if (choice.value === 'system') return mdiThemeLightDark
  if (choice.value === 'light') return mdiWeatherSunny
  return mdiWeatherNight
})

const themeLabel = computed(() => `Theme: ${choice.value}`)

function logout() {
  clearToken()
  router.push('/login')
}
</script>

<template>
  <v-app>
    <v-app-bar color="primary" density="compact">
      <v-app-bar-title>Distribution</v-app-bar-title>

      <v-tabs v-if="isAuthed" :model-value="route.path" align-tabs="center">
        <v-tab
          v-for="t in tabs"
          :key="t.to"
          :value="t.to"
          :to="t.to"
        >
          {{ t.label }}
        </v-tab>
      </v-tabs>

      <v-spacer />

      <v-btn
        :icon="themeIcon"
        :title="themeLabel"
        variant="text"
        @click="cycle"
      />

      <v-btn
        v-if="!isAuthed && route.path !== '/login'"
        :prepend-icon="mdiLogin"
        to="/login"
        variant="text"
      >
        Sign in
      </v-btn>

      <v-btn
        v-if="isAuthed"
        :prepend-icon="mdiLogout"
        variant="text"
        @click="logout"
      >
        Logout
      </v-btn>
    </v-app-bar>

    <v-main>
      <router-view />
    </v-main>
  </v-app>
</template>
