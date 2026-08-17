<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Globe, LogIn, Moon, Sun, SunMoon, ChevronDown, UserRound,
} from 'lucide-vue-next'
import { useTheme } from '../composables/useTheme'
import { useAuth } from '../composables/useAuth'
import { useI18n } from '../composables/useI18n'
import { useDownloadApp } from '../composables/useDownloadApp'
import { api } from '../api/client'
import { Button } from './ui/button'
import { Avatar } from './ui/avatar'
import { DropdownMenu } from './ui/dropdown-menu'
import { Dialog } from './ui/dialog'
import { AlertDialog } from './ui/alert-dialog'
import { Alert } from './ui/alert'
import { Input } from './ui/input'
import { Label } from './ui/label'
import { Sonner } from './ui/sonner'
import type { ThemeChoice } from '../composables/useTheme'
import type { Locale } from '../i18n/messages'

const route = useRoute()
const router = useRouter()
const { choice, setChoice } = useTheme()
const { isAuthed, isSuperAdmin, username, clearToken } = useAuth()
const { t, locale, setLocale } = useI18n()
const { app } = useDownloadApp()

const isDownloadPage = computed(() => route.path.startsWith('/app/'))
const currentDownloadApp = computed(() => (isDownloadPage.value ? app.value : null))

const tabs = computed(() => {
  if (!isAuthed.value) return []
  const result = [
    { label: t('home.title'), to: '/admin', match: '/admin' },
  ]
  if (isSuperAdmin.value) {
    result.push({ label: t('adminUsers.title'), to: '/admin/users', match: '/admin/users' })
  }
  result.push({ label: t('adminKeys.title'), to: '/admin/keys', match: '/admin/keys' })
  result.push({ label: t('apiDoc.title'), to: '/admin/keys/doc', match: '/admin/keys/doc' })
  result.push({ label: t('notify.title'), to: '/admin/subscriptions', match: '/admin/subscriptions' })
  return result
})

const activeTab = computed(() => {
  const p = route.path
  // Longest matching tab wins so that e.g. /admin/users highlights the
  // "Users" tab instead of the prefix "Apps" (/admin) one.
  const hit = tabs.value
    .filter((tab) => p === tab.match || p.startsWith(tab.match + '/'))
    .sort((a, b) => b.match.length - a.match.length)[0]
  return hit?.to ?? ''
})

const sonnerTheme = computed<ThemeChoice>(() => choice.value)

const themeLabel = computed(() => {
  if (choice.value === 'system') return t('app.themeSystem')
  if (choice.value === 'light') return t('app.themeLight')
  return t('app.themeDark')
})

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

function onThemeSelect(i: number) {
  const order: ThemeChoice[] = ['light', 'dark', 'system']
  setChoice(order[i])
}

function onLangSelect(i: number) {
  const order: Locale[] = ['en', 'zh']
  setLocale(order[i])
}
</script>

<template>
  <div class="flex min-h-screen flex-col">
    <header class="border-b bg-background/95 sticky top-0 z-40 backdrop-blur">
      <div class="flex h-14 items-center gap-3 px-4 sm:px-6">
        <template v-if="currentDownloadApp && currentDownloadApp.name">
          <Avatar :src="currentDownloadApp.icon" :fallback="currentDownloadApp.name.charAt(0).toUpperCase()" class="size-7" />
          <span class="truncate text-sm font-semibold">{{ currentDownloadApp.name }}</span>
        </template>
        <template v-else>
          <span class="text-sm font-semibold">{{ t('app.title') }}</span>
        </template>

        <nav v-if="isAuthed" class="mx-auto hidden items-center gap-1 sm:flex">
          <Button
            v-for="tab in tabs"
            :key="tab.to"
            as-child
            variant="ghost"
            size="sm"
            :class="activeTab === tab.to ? 'bg-accent text-accent-foreground' : ''"
          >
            <RouterLink :to="tab.to">{{ tab.label }}</RouterLink>
          </Button>
        </nav>

        <div class="ml-auto flex items-center gap-1">
          <!-- Language -->
          <DropdownMenu :items="[{ label: 'English', value: 'en' }, { label: '中文', value: 'zh' }]" :selected="locale" @select="onLangSelect">
            <template #trigger>
              <Button variant="ghost" size="icon" :title="t('lang.en')">
                <Globe class="size-4" />
              </Button>
            </template>
          </DropdownMenu>

          <!-- Theme -->
          <DropdownMenu :items="[{ label: t('app.themeLight'), value: 'light' }, { label: t('app.themeDark'), value: 'dark' }, { label: t('app.themeSystem'), value: 'system' }]" :selected="choice" @select="onThemeSelect">
            <template #trigger>
              <Button variant="ghost" size="icon" :title="themeLabel">
                <Sun v-if="choice === 'light'" class="size-4" />
                <Moon v-else-if="choice === 'dark'" class="size-4" />
                <SunMoon v-else class="size-4" />
              </Button>
            </template>
          </DropdownMenu>

          <!-- Sign in -->
          <Button v-if="!isAuthed && route.path !== '/login' && !isDownloadPage" as-child variant="ghost" size="sm">
            <RouterLink to="/login">
              <LogIn class="size-4" />
              {{ t('app.signin') }}
            </RouterLink>
          </Button>

          <!-- User menu -->
          <DropdownMenu
            v-if="isAuthed"
            :items="[
              { label: username ?? '', value: username ?? '' },
              { label: '', divider: true },
              ...(!isSuperAdmin ? [{ label: t('app.changePassword'), value: '' }] : []),
              { label: t('app.logout'), value: '', danger: true },
            ]"
            @select="(i: number) => { if (!isSuperAdmin && i === 2) openPwDialog(); if (i === (isSuperAdmin ? 2 : 3)) logoutDialog = true }"
          >
            <template #trigger>
              <Button variant="ghost" size="sm" class="gap-1.5">
                <UserRound class="size-4" />
                {{ username }}
                <ChevronDown class="size-3 opacity-50" />
              </Button>
            </template>
          </DropdownMenu>
        </div>
      </div>
    </header>

    <main class="flex-1">
      <router-view />
    </main>

    <Sonner :theme="sonnerTheme" />

    <!-- Password change dialog -->
    <Dialog v-model:open="pwDialog" :title="t('app.changePassword')" max-width="md">
      <div class="grid gap-4">
        <div class="grid gap-2">
          <Label>{{ t('app.oldPassword') }}</Label>
          <Input v-model="oldPassword" type="password" />
        </div>
        <div class="grid gap-2">
          <Label>{{ t('app.newPassword') }}</Label>
          <Input v-model="newPassword" type="password" @keyup.enter="submitPassword" />
        </div>
        <Alert v-if="pwError" variant="destructive">{{ pwError }}</Alert>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="pwDialog = false">{{ t('common.cancel') }}</Button>
          <Button :disabled="pwLoading" @click="submitPassword">{{ t('common.save') }}</Button>
        </div>
      </div>
    </Dialog>

    <!-- Logout confirmation -->
    <AlertDialog v-model:open="logoutDialog" :title="t('app.confirmLogout')">
      <template #footer>
        <Button variant="outline" @click="logoutDialog = false">{{ t('common.cancel') }}</Button>
        <Button variant="destructive" @click="logout">{{ t('app.logout') }}</Button>
      </template>
    </AlertDialog>
  </div>
</template>
