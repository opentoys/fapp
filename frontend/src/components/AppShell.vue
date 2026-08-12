<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTheme } from '../composables/useTheme'
import { mdiWeatherSunny, mdiWeatherNight, mdiThemeLightDark, mdiLogout } from '@mdi/js'

const route = useRoute()
const router = useRouter()
const { choice, cycle } = useTheme()

const isAuthed = computed(() => !!localStorage.getItem('token'))

const tabs = computed(() => {
  if (isAuthed.value) {
    return [
      { label: 'Apps', to: '/' },
      { label: 'Admin', to: '/admin' },
      { label: 'Upload', to: '/admin/upload' },
    ]
  }
  return [
    { label: 'Apps', to: '/' },
    { label: 'Login', to: '/login' },
  ]
})

const themeIcon = computed(() => {
  if (choice.value === 'system') return mdiThemeLightDark
  if (choice.value === 'light') return mdiWeatherSunny
  return mdiWeatherNight
})

const themeLabel = computed(() => {
  return `Theme: ${choice.value}`
})

function logout() {
  localStorage.removeItem('token')
  router.push('/login')
}
</script>

<template>
  <v-app>
    <v-app-bar>
      <v-app-bar-title class="wordmark">▌ DISTRIBUTION</v-app-bar-title>

      <v-tabs v-if="!isAuthed" :model-value="route.path" align-tabs="center">
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
        density="comfortable"
        @click="cycle"
      />

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

<style scoped>
.wordmark {
  font-family: var(--font-mono);
  font-size: 0.85rem;
  letter-spacing: 0.2em;
  color: var(--accent) !important;
  text-transform: uppercase;
}
</style>
