<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTheme } from '../composables/useTheme'
import { useAuth } from '../composables/useAuth'
import { useI18n } from '../composables/useI18n'
import { api } from '../api/client'
import {
  mdiWeatherSunny, mdiWeatherNight, mdiThemeLightDark,
  mdiLogin, mdiLogout, mdiTranslate, mdiAccountCircle,
  mdiMenuDown, mdiLock,
} from '@mdi/js'
import type { ThemeChoice } from '../composables/useTheme'
import type { Locale } from '../i18n/messages'

const route = useRoute()
const router = useRouter()
const { choice, setChoice } = useTheme()
const { isAuthed, isSuperAdmin, username, clearToken } = useAuth()
const { t, locale, setLocale } = useI18n()

// The public download page hides the sign-in entry point.
const isDownloadPage = computed(() => route.path.startsWith('/app/'))

const tabs = computed(() => {
  if (!isAuthed.value) return []
  const result = [
    { label: t('home.title'), to: '/admin', match: '/admin' },
  ]
  if (isSuperAdmin.value) {
    result.push({ label: t('adminUsers.title'), to: '/admin/users', match: '/admin/users' })
  }
  return result
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

const themeLabel = computed(() => {
  if (choice.value === 'system') return t('app.themeSystem')
  if (choice.value === 'light') return t('app.themeLight')
  return t('app.themeDark')
})

const themeItems = computed(() => [
  { title: t('app.themeLight'), value: 'light' as ThemeChoice, icon: mdiWeatherSunny },
  { title: t('app.themeDark'), value: 'dark' as ThemeChoice, icon: mdiWeatherNight },
  { title: t('app.themeSystem'), value: 'system' as ThemeChoice, icon: mdiThemeLightDark },
])

const langItems = computed(() => [
  { title: t('lang.en'), value: 'en' as Locale },
  { title: t('lang.zh'), value: 'zh' as Locale },
])

// --- Password change dialog ---
const pwDialog = ref(false)
const oldPassword = ref('')
const newPassword = ref('')
const pwError = ref('')
const pwLoading = ref(false)

function openPwDialog() {
  oldPassword.value = ''
  newPassword.value = ''
  pwError.value = ''
  pwDialog.value = true
}

async function submitPassword() {
  if (!oldPassword.value || !newPassword.value) {
    pwError.value = t('adminUsers.required')
    return
  }
  pwError.value = ''
  pwLoading.value = true
  try {
    await api.changePassword(oldPassword.value, newPassword.value)
    pwDialog.value = false
  } catch (e) {
    pwError.value = (e as Error).message
  } finally {
    pwLoading.value = false
  }
}

const logoutDialog = ref(false)

function logout() {
  logoutDialog.value = false
  clearToken()
  router.push('/login')
}
</script>

<template>
  <v-app>
    <v-app-bar color="primary" density="compact">
      <v-app-bar-title>{{ t('app.title') }}</v-app-bar-title>

      <v-tabs v-if="isAuthed" :model-value="activeTab" align-tabs="center">
        <v-tab v-for="tab in tabs" :key="tab.to" :value="tab.to" :to="tab.to">
          {{ tab.label }}
        </v-tab>
      </v-tabs>

      <v-spacer />

      <!-- Language switcher -->
      <v-menu>
        <template #activator="{ props }">
          <v-btn variant="text" v-bind="props">
            <v-icon :icon="mdiTranslate" />
          </v-btn>
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

      <!-- Theme dropdown -->
      <v-menu>
        <template #activator="{ props }">
          <v-btn :title="themeLabel" variant="text" v-bind="props">
            <v-icon :icon="themeIcon" />
          </v-btn>
        </template>
        <v-list density="compact" :selected="[choice]">
          <v-list-item
            v-for="item in themeItems"
            :key="item.value"
            :value="item.value"
            @click="setChoice(item.value)"
          >
            <template #prepend>
              <v-icon :icon="item.icon" size="small" />
            </template>
            <v-list-item-title>{{ item.title }}</v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>

      <!-- Sign in button -->
      <v-btn
        v-if="!isAuthed && route.path !== '/login' && !isDownloadPage"
        to="/login"
        variant="text"
      >
        <v-icon :icon="mdiLogin" />
        {{ t('app.signin') }}
      </v-btn>

      <!-- User menu -->
      <v-menu v-if="isAuthed">
        <template #activator="{ props }">
          <v-btn variant="text" v-bind="props">
            <v-icon :icon="mdiAccountCircle" />
            {{ username }}
            <v-icon :icon="mdiMenuDown" />
          </v-btn>
        </template>
        <v-list density="compact">
          <v-list-item v-if="!isSuperAdmin" @click="openPwDialog">
            <template #prepend>
              <v-icon :icon="mdiLock" size="small" />
            </template>
            <v-list-item-title>{{ t('app.changePassword') }}</v-list-item-title>
          </v-list-item>
          <v-list-item @click="logoutDialog = true">
            <template #prepend>
              <v-icon :icon="mdiLogout" size="small" />
            </template>
            <v-list-item-title>{{ t('app.logout') }}</v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>
    </v-app-bar>

    <v-main>
      <router-view />
    </v-main>

    <!-- Password change dialog -->
    <v-dialog v-model="pwDialog" max-width="400">
      <v-card>
        <v-card-title>{{ t('app.changePassword') }}</v-card-title>
        <v-card-text>
          <v-text-field
            v-model="oldPassword"
            :label="t('app.oldPassword')"
            type="password"
            autofocus
          />
          <v-text-field
            v-model="newPassword"
            :label="t('app.newPassword')"
            type="password"
            @keyup.enter="submitPassword"
          />
          <v-alert v-if="pwError" type="error" variant="tonal" density="compact" class="mt-2">
            {{ pwError }}
          </v-alert>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="pwDialog = false">{{ t('common.cancel') }}</v-btn>
          <v-btn
            color="primary"
            variant="flat"
            :loading="pwLoading"
            :disabled="!oldPassword || !newPassword"
            @click="submitPassword"
          >
            {{ t('common.save') }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Logout confirmation dialog -->
    <v-dialog v-model="logoutDialog" max-width="400">
      <v-card>
        <v-card-title>{{ t('app.confirmLogout') }}</v-card-title>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="logoutDialog = false">{{ t('common.cancel') }}</v-btn>
          <v-btn color="error" variant="flat" @click="logout">{{ t('app.logout') }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-app>
</template>