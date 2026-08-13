<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTheme } from '../composables/useTheme'
import { useAuth } from '../composables/useAuth'
import { useI18n } from '../composables/useI18n'
import { mdiWeatherSunny, mdiWeatherNight, mdiThemeLightDark, mdiLogout, mdiLogin, mdiTranslate } from '@mdi/js'
import type { Locale } from '../i18n/messages'

const route = useRoute()
const router = useRouter()
const { choice, cycle } = useTheme()
const { isAuthed, clearToken } = useAuth()
const { t, locale, setLocale } = useI18n()

const tabs = computed(() => {
  if (!isAuthed.value) return []
  return [
    { label: t('home.title'), to: '/admin', match: '/admin' },
    { label: t('adminUsers.title'), to: '/admin/users', match: '/admin/users' },
  ]
})

const activeTab = computed(() => {
  const p = route.path
  const hit = tabs.value.find((t) => p === t.match || p.startsWith(t.match + '/'))
  return hit?.to ?? ''
})

const themeIcon = computed(() => {
  if (choice.value === 'system') return mdiThemeLightDark
  if (choice.value === 'light') return mdiWeatherSunny
  return mdiWeatherNight
})

const themeLabel = computed(() => t('app.theme', { name: choice.value }))

const langItems = computed(() => [
  { title: t('lang.en'), value: 'en' as Locale },
  { title: t('lang.zh'), value: 'zh' as Locale },
])

function logout() {
  clearToken()
  router.push('/login')
}
</script>

<template>
  <v-app>
    <v-app-bar color="primary" density="compact">
      <v-app-bar-title>{{ t('app.title') }}</v-app-bar-title>

      <v-tabs v-if="isAuthed" :model-value="activeTab" align-tabs="center">
        <v-tab
          v-for="tab in tabs"
          :key="tab.to"
          :value="tab.to"
          :to="tab.to"
        >
          {{ tab.label }}
        </v-tab>
      </v-tabs>

      <v-spacer />

      <v-menu>
        <template #activator="{ props }">
          <v-btn :icon="mdiTranslate" variant="text" v-bind="props" />
        </template>
        <v-list density="compact" :selected="[locale]">
          <v-list-item
            v-for="item in langItems"
            :key="item.value"
            :value="item.value"
            @click="setLocale(item.value)"
          >
            <v-list-item-title>{{ item.title }}</v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>

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
        {{ t('app.signin') }}
      </v-btn>

      <v-btn
        v-if="isAuthed"
        :prepend-icon="mdiLogout"
        variant="text"
        @click="logout"
      >
        {{ t('app.logout') }}
      </v-btn>
    </v-app-bar>

    <v-main>
      <router-view />
    </v-main>
  </v-app>
</template>
