# Vuetify Frontend Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the hand-rolled CSS frontend with a Vuetify-based Industrial Blueprint design (navy + teal, hairline borders, mono data + sans body, dark + light themes with a system-preference default).

**Architecture:** Single-page Vue 3 app served by the existing Go binary. Add Vuetify 3 with `vite-plugin-vuetify` for tree-shaking, register two custom themes in `plugins/vuetify.ts`, and override Material defaults globally via a single SCSS file (border-radius: 0, no elevation, hairline borders). Build three reusable components (StatusDot, MonoText, StatBlock) and an AppShell wrapper. Rewrite each of the six view files to use the new design.

**Tech Stack:** Vue 3.5, TypeScript, Vite 8, Vuetify 3, vite-plugin-vuetify, @mdi/js, sass, vue-router 5, axios.

---

## File Structure

Files created or modified by this plan:

```
frontend/
├── package.json                     # MODIFIED: + vuetify, sass, vite-plugin-vuetify, @mdi/js
├── vite.config.ts                   # MODIFIED: + vuetify plugin
├── src/
│   ├── main.ts                      # MODIFIED: install Vuetify, import styles
│   ├── App.vue                      # MODIFIED: render <AppShell>
│   ├── plugins/
│   │   └── vuetify.ts               # NEW: createVuetify with both themes
│   ├── composables/
│   │   └── useTheme.ts              # NEW: localStorage + media query + cycle
│   ├── components/
│   │   ├── AppShell.vue             # NEW: v-app-bar + tabs + theme toggle
│   │   ├── StatusDot.vue            # NEW: signature element
│   │   ├── MonoText.vue             # NEW: inline mono span
│   │   └── StatBlock.vue            # NEW: stat tile for home page
│   ├── styles/
│   │   ├── tokens.css               # NEW: CSS custom props for both themes
│   │   ├── fonts.css                # NEW: @import Inter + JetBrains Mono
│   │   └── overrides.scss           # NEW: global Vuetify overrides
│   ├── views/
│   │   ├── Home.vue                 # MODIFIED: rewrite with Vuetify
│   │   ├── AppDetail.vue            # MODIFIED: rewrite with Vuetify
│   │   ├── Login.vue                # MODIFIED: rewrite with Vuetify
│   │   └── admin/
│   │       ├── Admin.vue            # MODIFIED: rewrite with Vuetify
│   │       ├── AdminApp.vue         # MODIFIED: rewrite with Vuetify
│   │       └── Upload.vue           # MODIFIED: rewrite with Vuetify
│   ├── api/                         # UNCHANGED: client.ts, types.ts
│   └── router/                      # UNCHANGED
```

API contract is unchanged. All backend endpoints serve the same JSON shapes.

---

## Task 1: Add Vuetify dependencies and configure Vite

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/vite.config.ts`

- [ ] **Step 1: Add dependencies to package.json**

Open `frontend/package.json`. Replace the `dependencies` and `devDependencies` blocks with:

```json
{
  "name": "frontend",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vue-tsc -b && vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "@mdi/js": "^7.4.47",
    "axios": "^1.19.0",
    "vue": "^3.5.40",
    "vue-router": "^5.2.0",
    "vuetify": "^3.7.0"
  },
  "devDependencies": {
    "@types/node": "^24.13.3",
    "@vitejs/plugin-vue": "^6.0.8",
    "@vue/tsconfig": "^0.9.1",
    "sass": "^1.80.0",
    "typescript": "~6.0.2",
    "vite": "^8.2.0",
    "vite-plugin-vuetify": "^2.0.4",
    "vue-tsc": "^3.3.8"
  }
}
```

- [ ] **Step 2: Update vite.config.ts to load the Vuetify plugin**

Replace `frontend/vite.config.ts` with:

```ts
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vuetify from 'vite-plugin-vuetify'

export default defineConfig({
  plugins: [
    vue(),
    vuetify({ autoImport: true }),
  ],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
```

- [ ] **Step 3: Install dependencies**

Run:
```bash
cd frontend && npm install
```

Expected: no errors. If a peer-dependency warning appears for `vue-router` (vuetify lists `@4`), it is safe to ignore — vue-router 5 works with vuetify 3.

- [ ] **Step 4: Verify build still works**

Run:
```bash
cd frontend && npm run build
```

Expected: build succeeds, no Vuetify errors yet (Vuetify is not installed in `main.ts`).

- [ ] **Step 5: Commit**

```bash
cd frontend && git add package.json package-lock.json vite.config.ts
git commit -m "build: add vuetify, sass, vite-plugin-vuetify dependencies"
```

---

## Task 2: Create design tokens (CSS custom properties)

**Files:**
- Create: `frontend/src/styles/tokens.css`

- [ ] **Step 1: Create the tokens file**

Create `frontend/src/styles/tokens.css`:

```css
/* Design tokens — Industrial Blueprint. All colors flow from here. */

:root {
  /* Type scale */
  --font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  --font-mono: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;

  --text-xs: 0.75rem;
  --text-sm: 0.875rem;
  --text-base: 1rem;
  --text-lg: 1.25rem;
  --text-xl: 1.75rem;
  --text-2xl: 2.25rem;

  /* Spacing (4px base) */
  --sp-1: 4px;
  --sp-2: 8px;
  --sp-3: 12px;
  --sp-4: 16px;
  --sp-6: 24px;
  --sp-8: 32px;

  /* Layout */
  --max-w: 1200px;
  --topbar-h: 56px;
}

/* Dark: Control Room (default) */
:root,
[data-theme='dark'] {
  --bg: #0d1b2a;
  --surface: #1b263b;
  --surface-2: #243349;
  --border: #415a77;
  --text: #e0e1dd;
  --text-mute: #778da9;
  --accent: #a8dadc;
  --success: #a8dadc;
  --warning: #f4a261;
  --danger: #e07a5f;
}

/* Light: Blueprint Paper */
[data-theme='light'] {
  --bg: #f4f1ea;
  --surface: #e8e3d6;
  --surface-2: #dcd6c5;
  --border: #415a77;
  --text: #0d1b2a;
  --text-mute: #415a77;
  --accent: #0d6e6e;
  --success: #0d6e6e;
  --warning: #b8651b;
  --danger: #a83232;
}

html, body, #app {
  height: 100%;
  margin: 0;
  background: var(--bg);
  color: var(--text);
  font-family: var(--font-sans);
  font-size: var(--text-sm);
  line-height: 1.5;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

a {
  color: var(--accent);
  text-decoration: none;
}
a:hover { text-decoration: underline; }

:focus-visible {
  outline: 1px solid var(--accent);
  outline-offset: 2px;
}

@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

- [ ] **Step 2: Import tokens in main.ts**

Open `frontend/src/main.ts`. Replace the contents with:

```ts
import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import './styles/tokens.css'

createApp(App).use(router).mount('#app')
```

(We will add the Vuetify install in a later task. For now, just add the CSS import.)

- [ ] **Step 3: Verify build**

Run:
```bash
cd frontend && npm run build
```

Expected: build succeeds. The CSS bundle now includes the tokens.

- [ ] **Step 4: Commit**

```bash
cd frontend && git add src/styles/tokens.css src/main.ts
git commit -m "feat: design tokens for industrial blueprint theme"
```

---

## Task 3: Add fonts and global overrides

**Files:**
- Create: `frontend/src/styles/fonts.css`
- Create: `frontend/src/styles/overrides.scss`
- Modify: `frontend/src/main.ts`

- [ ] **Step 1: Create fonts.css**

Create `frontend/src/styles/fonts.css`:

```css
/* Inter for body, JetBrains Mono for data. */
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@400;500&display=swap');
```

- [ ] **Step 2: Create overrides.scss**

Create `frontend/src/styles/overrides.scss`:

```scss
/* Strip Material defaults to match the blueprint look. */

/* No rounded corners anywhere. */
.v-card,
.v-btn,
.v-text-field .v-field,
.v-select .v-field,
.v-textarea .v-field,
.v-data-table,
.v-chip,
.v-dialog .v-card,
.v-snackbar__wrapper,
.v-list,
.v-toolbar,
.v-app-bar,
.v-tab,
.v-file-input .v-field {
  border-radius: 0 !important;
}

/* No shadows. */
.v-card,
.v-dialog .v-card,
.v-menu__content,
.v-overlay__scrim,
.v-list,
.v-sheet {
  box-shadow: none !important;
}

/* Hairline borders. */
.v-card,
.v-data-table,
.v-dialog .v-card {
  border: 1px solid var(--border);
}

/* Card hover. */
.v-card.hoverable {
  transition: background 100ms ease;
  cursor: pointer;
}
.v-card.hoverable:hover {
  background: var(--surface-2);
}

/* App bar: flat with hairline bottom. */
.v-app-bar {
  border-bottom: 1px solid var(--border) !important;
  background: var(--surface) !important;
  color: var(--text) !important;
}

/* Tabs: hide the slider, use underline on active. */
.v-tab--selected {
  color: var(--accent) !important;
}
.v-tabs-slider {
  display: none !important;
}
.v-tab {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  letter-spacing: 0.15em;
  text-transform: uppercase;
  min-width: 80px;
}

/* Buttons: outlined default, mono caps. */
.v-btn {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  font-weight: 500;
  border-radius: 0 !important;
}

/* Form fields: outlined, mono labels. */
.v-field {
  background: var(--surface) !important;
}
.v-field__outline {
  --v-field-border-opacity: 1;
  color: var(--border) !important;
}
.v-label {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  opacity: 0.8;
}

/* Data tables: hairline rows, eyebrow headers. */
.v-data-table {
  background: var(--surface) !important;
  color: var(--text) !important;
}
.v-data-table th {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  letter-spacing: 0.15em;
  text-transform: uppercase;
  color: var(--text-mute) !important;
  background: var(--surface) !important;
  border-bottom: 1px solid var(--border) !important;
}
.v-data-table td {
  border-bottom: 1px solid var(--border) !important;
}
.v-data-table tr:hover td {
  background: var(--surface-2) !important;
}

/* Chips (status). */
.v-chip {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  border: 1px solid var(--border);
}

/* Dialog: no shadow, hairline border. */
.v-overlay__content > .v-card {
  background: var(--bg) !important;
  border: 1px solid var(--border) !important;
}

/* Snackbar: hairline border. */
.v-snackbar__wrapper {
  border: 1px solid var(--border) !important;
  background: var(--surface) !important;
  color: var(--text) !important;
}

/* File input dropzone. */
.v-file-input .v-field {
  border-style: dashed !important;
}
```

- [ ] **Step 3: Import fonts and overrides in main.ts**

Open `frontend/src/main.ts`. Replace its contents with:

```ts
import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import './styles/fonts.css'
import './styles/tokens.css'
import './styles/overrides.scss'

createApp(App).use(router).mount('#app')
```

- [ ] **Step 4: Verify build**

Run:
```bash
cd frontend && npm run build
```

Expected: build succeeds. CSS bundle now includes fonts and overrides. (No visual change yet — overrides only matter once Vuetify components are in use.)

- [ ] **Step 5: Commit**

```bash
cd frontend && git add src/styles/ src/main.ts
git commit -m "feat: add font import and global vuetify overrides"
```

---

## Task 4: Create the Vuetify plugin with two themes

**Files:**
- Create: `frontend/src/plugins/vuetify.ts`
- Modify: `frontend/src/main.ts`

- [ ] **Step 1: Create plugins/vuetify.ts**

Create `frontend/src/plugins/vuetify.ts`:

```ts
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import { aliases, mdi } from 'vuetify/iconsets/mdi'

// Dark: Control Room
const dark = {
  dark: true,
  colors: {
    background: '#0d1b2a',
    surface: '#1b263b',
    'surface-variant': '#243349',
    'on-surface-variant': '#e0e1dd',
    primary: '#a8dadc',
    secondary: '#778da9',
    error: '#e07a5f',
    warning: '#f4a261',
    info: '#a8dadc',
    success: '#a8dadc',
  },
}

// Light: Blueprint Paper
const light = {
  dark: false,
  colors: {
    background: '#f4f1ea',
    surface: '#e8e3d6',
    'surface-variant': '#dcd6c5',
    'on-surface-variant': '#0d1b2a',
    primary: '#0d6e6e',
    secondary: '#415a77',
    error: '#a83232',
    warning: '#b8651b',
    info: '#0d6e6e',
    success: '#0d6e6e',
  },
}

export const vuetify = createVuetify({
  components,
  directives,
  theme: {
    defaultTheme: 'dark',
    themes: { dark, light },
  },
  icons: {
    defaultSet: 'mdi',
    aliases,
    sets: { mdi },
  },
  defaults: {
    VBtn: { variant: 'outlined', density: 'comfortable' },
    VCard: { flat: true },
    VTextField: { variant: 'outlined', density: 'comfortable' },
    VSelect: { variant: 'outlined', density: 'comfortable' },
    VTextarea: { variant: 'outlined', density: 'comfortable' },
    VDataTable: { density: 'comfortable', hover: true },
    VChip: { variant: 'outlined', size: 'small' },
    VAppBar: { flat: true, density: 'compact' },
  },
})
```

- [ ] **Step 2: Install Vuetify in main.ts**

Open `frontend/src/main.ts`. Replace its contents with:

```ts
import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import { vuetify } from './plugins/vuetify'
import 'vuetify/styles'
import './styles/fonts.css'
import './styles/tokens.css'
import './styles/overrides.scss'

createApp(App).use(router).use(vuetify).mount('#app')
```

- [ ] **Step 3: Verify build**

Run:
```bash
cd frontend && npm run build
```

Expected: build succeeds. Vuetify is now wired up. The empty `<router-view />` in `App.vue` still has no visual change, but the CSS bundle will grow noticeably with Vuetify styles.

- [ ] **Step 4: Commit**

```bash
cd frontend && git add src/plugins/vuetify.ts src/main.ts
git commit -m "feat: register vuetify plugin with dark and light themes"
```

---

## Task 5: Create the useTheme composable

**Files:**
- Create: `frontend/src/composables/useTheme.ts`

- [ ] **Step 1: Create the composable**

Create `frontend/src/composables/useTheme.ts`:

```ts
import { ref, watch, onMounted } from 'vue'

export type ThemeChoice = 'system' | 'light' | 'dark'

const STORAGE_KEY = 'disapp-theme'

// Module-level state so all callers share the same value.
const choice = ref<ThemeChoice>('system')
const resolved = ref<'light' | 'dark'>('dark')

function readStored(): ThemeChoice {
  const v = localStorage.getItem(STORAGE_KEY)
  if (v === 'light' || v === 'dark' || v === 'system') return v
  return 'system'
}

function systemPrefersDark(): boolean {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

function applyTheme(c: ThemeChoice) {
  const r = c === 'system' ? (systemPrefersDark() ? 'dark' : 'light') : c
  resolved.value = r
  document.documentElement.setAttribute('data-theme', r)
  localStorage.setItem(STORAGE_KEY, c)
}

function nextChoice(c: ThemeChoice): ThemeChoice {
  return c === 'system' ? 'light' : c === 'light' ? 'dark' : 'system'
}

export function useTheme() {
  onMounted(() => {
    choice.value = readStored()
    applyTheme(choice.value)

    // React to system changes when in 'system' mode.
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    mq.addEventListener('change', () => {
      if (choice.value === 'system') applyTheme('system')
    })
  })

  watch(choice, (c) => applyTheme(c))

  function cycle() {
    choice.value = nextChoice(choice.value)
  }

  return { choice, resolved, cycle }
}
```

- [ ] **Step 2: Verify build**

Run:
```bash
cd frontend && npm run build
```

Expected: build succeeds. No visual change yet — the composable is unused.

- [ ] **Step 3: Commit**

```bash
cd frontend && git add src/composables/useTheme.ts
git commit -m "feat: useTheme composable with system/light/dark cycle"
```

---

## Task 6: Create the StatusDot signature component

**Files:**
- Create: `frontend/src/components/StatusDot.vue`

- [ ] **Step 1: Create StatusDot.vue**

Create `frontend/src/components/StatusDot.vue`:

```vue
<script setup lang="ts">
import { computed } from 'vue'

type Mode = 'public' | 'password' | 'expiry' | 'enabled' | 'taken_down' | 'live'

const props = defineProps<{
  mode: Mode
  label?: string
}>()

const colorVar = computed(() => {
  switch (props.mode) {
    case 'public':
    case 'enabled':
    case 'live':
      return 'var(--success)'
    case 'password':
      return 'var(--warning)'
    case 'expiry':
      return 'var(--warning)'
    case 'taken_down':
      return 'var(--danger)'
    default:
      return 'var(--text-mute)'
  }
})

const displayLabel = computed(() => {
  if (props.label) return props.label
  return props.mode.toUpperCase().replace('_', ' ')
})
</script>

<template>
  <span class="status-dot">
    <span class="dot" :style="{ color: colorVar }">●</span>
    <span class="label">{{ displayLabel }}</span>
  </span>
</template>

<style scoped>
.status-dot {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 0.7rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--text-mute);
}
.dot {
  font-size: 0.6rem;
  line-height: 1;
}
.label {
  color: var(--text-mute);
}
</style>
```

- [ ] **Step 2: Verify build**

Run:
```bash
cd frontend && npm run build
```

Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
cd frontend && git add src/components/StatusDot.vue
git commit -m "feat: StatusDot signature component"
```

---

## Task 7: Create MonoText and StatBlock components

**Files:**
- Create: `frontend/src/components/MonoText.vue`
- Create: `frontend/src/components/StatBlock.vue`

- [ ] **Step 1: Create MonoText.vue**

Create `frontend/src/components/MonoText.vue`:

```vue
<script setup lang="ts">
defineProps<{
  muted?: boolean
}>()
</script>

<template>
  <span class="mono" :class="{ muted }"><slot /></span>
</template>

<style scoped>
.mono {
  font-family: var(--font-mono);
  font-feature-settings: 'tnum' 1;
}
.mono.muted {
  color: var(--text-mute);
}
</style>
```

- [ ] **Step 2: Create StatBlock.vue**

Create `frontend/src/components/StatBlock.vue`:

```vue
<script setup lang="ts">
defineProps<{
  label: string
  value: number | string
  emphasis?: boolean
}>()
</script>

<template>
  <div class="stat" :class="{ emphasis }">
    <div class="label">{{ label }}</div>
    <div class="value">{{ value }}</div>
  </div>
</template>

<style scoped>
.stat {
  border: 1px solid var(--border);
  background: var(--surface);
  padding: var(--sp-3) var(--sp-4);
}
.label {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  letter-spacing: 0.15em;
  text-transform: uppercase;
  color: var(--text-mute);
  margin-bottom: var(--sp-1);
}
.value {
  font-family: var(--font-sans);
  font-size: 1.75rem;
  font-weight: 500;
  letter-spacing: -0.02em;
  color: var(--text);
}
.emphasis .value {
  color: var(--accent);
}
</style>
```

- [ ] **Step 3: Verify build**

Run:
```bash
cd frontend && npm run build
```

Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
cd frontend && git add src/components/MonoText.vue src/components/StatBlock.vue
git commit -m "feat: MonoText and StatBlock presentational components"
```

---

## Task 8: Create the AppShell (top bar + tabs + theme toggle)

**Files:**
- Create: `frontend/src/components/AppShell.vue`

- [ ] **Step 1: Create AppShell.vue**

Create `frontend/src/components/AppShell.vue`:

```vue
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
```

- [ ] **Step 2: Update App.vue to render the shell**

Replace `frontend/src/App.vue` with:

```vue
<script setup lang="ts">
import AppShell from './components/AppShell.vue'
</script>

<template>
  <AppShell />
</template>
```

- [ ] **Step 3: Verify build**

Run:
```bash
cd frontend && npm run build
```

Expected: build succeeds. Run dev server (`npm run dev`) and visit http://localhost:5173 — you should see a dark navy top bar with the wordmark, two tabs ("Apps", "Login"), and a theme toggle on the right. Pages will still use the old hand-rolled styles, but the shell and theme toggle work.

- [ ] **Step 4: Commit**

```bash
cd frontend && git add src/components/AppShell.vue src/App.vue
git commit -m "feat: AppShell with top bar, tabs, and theme toggle"
```

---

## Task 9: Rewrite Home.vue

**Files:**
- Modify: `frontend/src/views/Home.vue`

- [ ] **Step 1: Rewrite Home.vue**

Replace `frontend/src/views/Home.vue` with:

```vue
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '../api/client'
import type { AppItem } from '../api/types'
import StatusDot from '../components/StatusDot.vue'
import StatBlock from '../components/StatBlock.vue'
import MonoText from '../components/MonoText.vue'

const apps = ref<AppItem[]>([])
const error = ref('')

onMounted(async () => {
  try {
    apps.value = await api.apps()
  } catch (e) {
    error.value = (e as Error).message
  }
})

const totalDownloads = computed(() =>
  apps.value.reduce((s, a) => s + (a.latest_version?.download_count ?? 0), 0)
)
const totalVersions = computed(() =>
  apps.value.filter((a) => a.latest_version).length
)

function fmtSize(n: number): string {
  if (n > 1024 * 1024 * 1024) return (n / 1024 / 1024 / 1024).toFixed(2) + ' GB'
  if (n > 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MB'
  if (n > 1024) return (n / 1024).toFixed(1) + ' KB'
  return n + ' B'
}
</script>

<template>
  <div class="home">
    <div class="page-header">
      <div class="eyebrow">▌ DISTRIBUTION</div>
      <h1 class="title">Apps</h1>
      <div class="stat-strip">
        <StatBlock label="Apps" :value="apps.length" />
        <StatBlock label="Versions" :value="totalVersions" />
        <StatBlock label="Downloads" :value="totalDownloads" emphasis />
      </div>
    </div>

    <v-alert v-if="error" type="error" variant="outlined" class="mb-4">
      {{ error }}
    </v-alert>

    <div v-if="apps.length" class="grid">
      <router-link
        v-for="a in apps"
        :key="a.id"
        :to="`/app/${a.id}`"
        class="card-link"
      >
        <v-card class="hoverable pa-4">
          <div class="card-head">
            <img v-if="a.icon" :src="a.icon" alt="" class="icon" />
            <div v-else class="icon-fallback">{{ a.name.charAt(0).toUpperCase() }}</div>
            <div class="card-name">{{ a.name }}</div>
          </div>
          <div v-if="a.latest_version" class="card-meta">
            <MonoText>{{ a.latest_version.version_name }}</MonoText>
            <MonoText muted> · {{ fmtSize(a.latest_version.file_size) }}</MonoText>
          </div>
          <div v-if="a.latest_version" class="card-status">
            <StatusDot :mode="a.latest_version.access_mode" />
          </div>
          <div v-if="a.description" class="card-desc">{{ a.description }}</div>
        </v-card>
      </router-link>
    </div>

    <div v-else-if="!error" class="empty">
      <div class="eyebrow">▌ NO APPS</div>
      <p>no applications yet</p>
    </div>
  </div>
</template>

<style scoped>
.home {
  max-width: var(--max-w);
  margin: 0 auto;
  padding: var(--sp-8) var(--sp-6);
}
.page-header {
  margin-bottom: var(--sp-8);
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
  font-size: 2.25rem;
  font-weight: 500;
  letter-spacing: -0.02em;
  margin: 0 0 var(--sp-6) 0;
}
.stat-strip {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--sp-2);
  max-width: 600px;
}
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: var(--sp-4);
}
.card-link {
  text-decoration: none;
  color: inherit;
}
.card-head {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  margin-bottom: var(--sp-3);
}
.icon {
  width: 40px;
  height: 40px;
  object-fit: contain;
}
.icon-fallback {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border);
  background: var(--surface-2);
  font-family: var(--font-mono);
  font-size: 1.1rem;
  color: var(--accent);
}
.card-name {
  font-size: 1.1rem;
  font-weight: 500;
}
.card-meta {
  margin-bottom: var(--sp-2);
}
.card-status {
  margin-bottom: var(--sp-3);
}
.card-desc {
  font-size: 0.85rem;
  color: var(--text-mute);
  line-height: 1.4;
}
.empty {
  text-align: center;
  padding: var(--sp-8) 0;
}
.empty p {
  color: var(--text-mute);
  margin: 0;
}
@media (max-width: 600px) {
  .stat-strip { grid-template-columns: 1fr; }
}
</style>
```

- [ ] **Step 2: Verify build and dev**

Run:
```bash
cd frontend && npm run build
```

Run dev server:
```bash
cd frontend && npm run dev
```

Visit http://localhost:5173. Expected: home page renders the eyebrow "▌ DISTRIBUTION", title "Apps", a 3-up stat strip, and a card grid. The dark theme should be active by default. Click the theme toggle (top-right) — it should cycle through `system` → `light` → `dark`.

- [ ] **Step 3: Commit**

```bash
cd frontend && git add src/views/Home.vue
git commit -m "feat: redesign Home page with vuetify and blueprint styling"
```

---

## Task 10: Rewrite AppDetail.vue

**Files:**
- Modify: `frontend/src/views/AppDetail.vue`

- [ ] **Step 1: Rewrite AppDetail.vue**

Replace `frontend/src/views/AppDetail.vue` with:

```vue
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api/client'
import type { AppDetail } from '../api/types'
import StatusDot from '../components/StatusDot.vue'
import MonoText from '../components/MonoText.vue'

const route = useRoute()
const data = ref<AppDetail | null>(null)
const error = ref('')
const passwordPrompt = ref<{ versionId: number; password: string } | null>(null)
const passwordError = ref('')

onMounted(load)
async function load() {
  try {
    data.value = await api.appDetail(Number(route.params.id))
  } catch (e) {
    error.value = (e as Error).message
  }
}

function fmtSize(n: number): string {
  if (n > 1024 * 1024 * 1024) return (n / 1024 / 1024 / 1024).toFixed(2) + ' GB'
  if (n > 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MB'
  if (n > 1024) return (n / 1024).toFixed(1) + ' KB'
  return n + ' B'
}

function fmtDate(s: string): string {
  return new Date(s).toISOString().replace('T', ' ').slice(0, 19)
}

async function download(v: { id: number; access_mode: string }) {
  passwordError.value = ''
  if (v.access_mode === 'password') {
    passwordPrompt.value = { versionId: v.id, password: '' }
    return
  }
  await doDownload(v.id, undefined)
}

async function submitPassword() {
  if (!passwordPrompt.value) return
  const pwd = passwordPrompt.value.password
  const vid = passwordPrompt.value.versionId
  passwordPrompt.value = null
  await doDownload(vid, pwd)
}

async function doDownload(versionId: number, password: string | undefined) {
  try {
    const url = await api.downloadUrl(versionId, password)
    window.location.href = url
  } catch (e) {
    error.value = (e as Error).message
  }
}
</script>

<template>
  <div class="app-detail">
    <v-alert v-if="error" type="error" variant="outlined" class="mb-4">
      {{ error }}
    </v-alert>

    <div v-if="data" class="layout">
      <aside class="left">
        <div class="eyebrow">▌ APP</div>
        <h1 class="title">{{ data.app.name }}</h1>
        <p v-if="data.app.description" class="desc">{{ data.app.description }}</p>

        <div v-if="data.channels.length" class="channels">
          <div class="eyebrow">▌ CHANNELS</div>
          <div class="channel-list">
            <span v-for="c in data.channels" :key="c.id" class="channel">
              <MonoText>{{ c.name }}</MonoText>
            </span>
          </div>
        </div>
      </aside>

      <section class="right">
        <div class="eyebrow">▌ VERSIONS</div>
        <div v-if="data.versions.length" class="version-list">
          <div
            v-for="v in data.versions"
            :key="v.id"
            class="version-row"
            :class="{ disabled: !v.enabled }"
          >
            <div class="ver-head">
              <MonoText class="ver-name">{{ v.version_name }}</MonoText>
              <MonoText muted> · code {{ v.version_code }} · {{ fmtSize(v.file_size) }}</MonoText>
              <span v-if="!v.enabled" class="taken-down">TAKEN DOWN</span>
            </div>
            <div class="ver-meta">
              <MonoText muted class="sha">{{ v.sha256.slice(0, 16) }}…</MonoText>
              <MonoText muted> · {{ fmtDate(v.created_at) }}</MonoText>
            </div>
            <div class="ver-status">
              <StatusDot :mode="v.enabled ? v.access_mode : 'taken_down'" />
            </div>
            <p v-if="v.changelog" class="changelog">{{ v.changelog }}</p>
            <div v-if="v.enabled" class="actions">
              <v-btn
                variant="outlined"
                size="small"
                :disabled="v.access_mode === 'expiry' && !!v.expires_at && new Date(v.expires_at) < new Date()"
                @click="download(v)"
              >
                Download
              </v-btn>
            </div>
          </div>
        </div>
        <div v-else class="empty">
          <p>no versions yet</p>
        </div>
      </section>
    </div>

    <v-dialog v-model="passwordPrompt" max-width="400">
      <v-card class="pa-5">
        <div class="eyebrow">▌ PASSWORD REQUIRED</div>
        <p class="dialog-body">This version is password protected.</p>
        <v-text-field
          v-model="passwordPrompt.password"
          label="Password"
          type="password"
          autofocus
          @keyup.enter="submitPassword"
        />
        <div class="dialog-actions">
          <v-btn variant="text" @click="passwordPrompt = null">Cancel</v-btn>
          <v-btn color="primary" @click="submitPassword">Continue</v-btn>
        </div>
      </v-card>
    </v-dialog>
  </div>
</template>

<style scoped>
.app-detail {
  max-width: var(--max-w);
  margin: 0 auto;
  padding: var(--sp-8) var(--sp-6);
}
.layout {
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: var(--sp-8);
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
  font-size: 2.25rem;
  font-weight: 500;
  letter-spacing: -0.02em;
  margin: 0 0 var(--sp-3) 0;
}
.desc {
  color: var(--text-mute);
  margin: 0 0 var(--sp-6) 0;
}
.channels {
  margin-top: var(--sp-4);
}
.channel-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--sp-2);
}
.channel {
  display: inline-block;
  padding: 4px 8px;
  border: 1px solid var(--border);
  background: var(--surface);
  font-size: 0.8rem;
}
.right .eyebrow {
  margin-bottom: var(--sp-3);
}
.version-list {
  border-top: 1px solid var(--border);
}
.version-row {
  padding: var(--sp-4) 0;
  border-bottom: 1px solid var(--border);
}
.version-row.disabled { opacity: 0.5; }
.ver-head {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  margin-bottom: var(--sp-1);
  flex-wrap: wrap;
}
.ver-name {
  font-size: 1.1rem;
  color: var(--text);
}
.taken-down {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  letter-spacing: 0.1em;
  color: var(--danger);
  border: 1px solid var(--danger);
  padding: 2px 6px;
}
.ver-meta {
  margin-bottom: var(--sp-2);
}
.sha {
  font-size: 0.75rem;
}
.ver-status {
  margin-bottom: var(--sp-2);
}
.changelog {
  font-size: 0.85rem;
  color: var(--text-mute);
  margin: var(--sp-2) 0;
  white-space: pre-wrap;
}
.actions {
  margin-top: var(--sp-2);
}
.empty {
  padding: var(--sp-6) 0;
  color: var(--text-mute);
  text-align: center;
}
.dialog-body {
  margin: var(--sp-3) 0;
  color: var(--text-mute);
}
.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--sp-2);
  margin-top: var(--sp-3);
}
@media (max-width: 900px) {
  .layout { grid-template-columns: 1fr; }
}
</style>
```

- [ ] **Step 2: Verify build and dev**

Run:
```bash
cd frontend && npm run build
```

In dev server, navigate to an app detail page (e.g. http://localhost:5173/app/1). Expected: two-column layout, version rows with mono name + code + size, status dots per row, hairline-divided rows. Click Download on a password-protected version — the dialog should appear.

- [ ] **Step 3: Commit**

```bash
cd frontend && git add src/views/AppDetail.vue
git commit -m "feat: redesign AppDetail with two-column layout and password dialog"
```

---

## Task 11: Rewrite Login.vue

**Files:**
- Modify: `frontend/src/views/Login.vue`

- [ ] **Step 1: Rewrite Login.vue**

Replace `frontend/src/views/Login.vue` with:

```vue
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
    localStorage.setItem('token', res.data.token)
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
```

- [ ] **Step 2: Verify build and dev**

Run:
```bash
cd frontend && npm run build
```

In dev server, visit http://localhost:5173/login. Expected: centered card on a faint blueprint-grid background, "▌ SIGN IN" eyebrow, "Distribution Console" title, two fields, and an Authenticate button. Sign in with the admin credentials — should redirect to /admin.

- [ ] **Step 3: Commit**

```bash
cd frontend && git add src/views/Login.vue
git commit -m "feat: redesign Login page with blueprint grid background"
```

---

## Task 12: Rewrite admin/Admin.vue

**Files:**
- Modify: `frontend/src/views/admin/Admin.vue`

- [ ] **Step 1: Rewrite Admin.vue**

Replace `frontend/src/views/admin/Admin.vue` with:

```vue
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../../api/client'
import type { AppItem } from '../../api/types'
import MonoText from '../../components/MonoText.vue'

const apps = ref<AppItem[]>([])
const name = ref('')
const error = ref('')
const deleteTarget = ref<AppItem | null>(null)
const snackbar = ref('')

onMounted(load)
async function load() {
  try {
    apps.value = await api.adminApps()
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function create() {
  if (!name.value) return
  try {
    await api.createApp({ name: name.value })
    name.value = ''
    await load()
    snackbar.value = 'App created'
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  const id = deleteTarget.value.id
  deleteTarget.value = null
  try {
    await api.deleteApp(id)
    await load()
    snackbar.value = 'App deleted'
  } catch (e) {
    error.value = (e as Error).message
  }
}

function fmtDate(s: string): string {
  return new Date(s).toISOString().replace('T', ' ').slice(0, 19)
}
</script>

<template>
  <div class="admin">
    <div class="page-header">
      <div class="eyebrow">▌ ADMIN</div>
      <h1 class="title">Applications</h1>
    </div>

    <v-alert v-if="error" type="error" variant="outlined" class="mb-4">
      {{ error }}
    </v-alert>

    <div class="create-row">
      <v-text-field
        v-model="name"
        label="New application name"
        density="comfortable"
        hide-details
        @keyup.enter="create"
      />
      <v-btn color="primary" :disabled="!name" @click="create">Create</v-btn>
    </div>

    <v-data-table
      :items="apps"
      :headers="[
        { title: 'Name', key: 'name' },
        { title: 'Created', key: 'created_at' },
        { title: '', key: 'actions', sortable: false, align: 'end' },
      ]"
      class="mt-6"
      hide-default-footer
      :items-per-page="-1"
    >
      <template #item.name="{ item }">
        <router-link :to="`/admin/app/${item.id}`" class="name-link">
          {{ item.name }}
        </router-link>
      </template>
      <template #item.created_at="{ item }">
        <MonoText muted>{{ fmtDate(item.created_at) }}</MonoText>
      </template>
      <template #item.actions="{ item }">
        <v-btn
          variant="text"
          size="small"
          color="error"
          @click="deleteTarget = item"
        >
          Delete
        </v-btn>
      </template>
    </v-data-table>

    <v-dialog v-model="deleteTarget" max-width="400">
      <v-card class="pa-5">
        <div class="eyebrow">▌ CONFIRM DELETE</div>
        <p class="dialog-body">
          Delete <b>{{ deleteTarget?.name }}</b>? Associated versions and channels will be removed.
        </p>
        <div class="dialog-actions">
          <v-btn variant="text" @click="deleteTarget = null">Cancel</v-btn>
          <v-btn color="error" @click="confirmDelete">Delete</v-btn>
        </div>
      </v-card>
    </v-dialog>

    <v-snackbar v-model="snackbar" :timeout="2000">
      {{ snackbar }}
    </v-snackbar>
  </div>
</template>

<style scoped>
.admin {
  max-width: var(--max-w);
  margin: 0 auto;
  padding: var(--sp-8) var(--sp-6);
}
.page-header {
  margin-bottom: var(--sp-6);
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
  font-size: 2.25rem;
  font-weight: 500;
  letter-spacing: -0.02em;
  margin: 0;
}
.create-row {
  display: flex;
  gap: var(--sp-2);
  align-items: start;
  max-width: 600px;
}
.name-link {
  color: var(--accent);
  font-weight: 500;
}
.dialog-body {
  margin: var(--sp-3) 0;
  color: var(--text-mute);
}
.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--sp-2);
}
</style>
```

- [ ] **Step 2: Verify build and dev**

Run:
```bash
cd frontend && npm run build
```

In dev server, log in and visit /admin. Expected: page header, create row with input + button, data table with hairline rows, delete confirm dialog.

- [ ] **Step 3: Commit**

```bash
cd frontend && git add src/views/admin/Admin.vue
git commit -m "feat: redesign Admin apps list with data table and confirm dialog"
```

---

## Task 13: Rewrite admin/AdminApp.vue

**Files:**
- Modify: `frontend/src/views/admin/AdminApp.vue`

- [ ] **Step 1: Rewrite AdminApp.vue**

Replace `frontend/src/views/admin/AdminApp.vue` with:

```vue
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../../api/client'
import type { AppDetail, Channel, Version } from '../../api/types'
import StatusDot from '../../components/StatusDot.vue'
import MonoText from '../../components/MonoText.vue'

const route = useRoute()
const data = ref<AppDetail | null>(null)
const channels = ref<Channel[]>([])
const newChannelName = ref('')
const stats = ref<{ download_count: number; install_count: number; recent: Array<{ ip: string; user_agent: string; created_at: string }> } | null>(null)
const error = ref('')
const tab = ref<'versions' | 'channels' | 'stats'>('versions')
const deleteTarget = ref<Version | null>(null)
const snackbar = ref('')

onMounted(load)
async function load() {
  const id = Number(route.params.id)
  try {
    data.value = await api.appDetail(id)
    channels.value = await api.channels(id)
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function createChannel() {
  if (!newChannelName.value || !data.value) return
  try {
    await api.createChannel(data.value.app.id, newChannelName.value)
    newChannelName.value = ''
    channels.value = await api.channels(data.value.app.id)
    snackbar.value = 'Channel created'
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function loadStats(v: Version) {
  try {
    stats.value = await api.versionStats(v.id)
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function deleteVersion() {
  if (!deleteTarget.value) return
  const id = deleteTarget.value.id
  deleteTarget.value = null
  try {
    await api.deleteVersion(id, true)
    await load()
    snackbar.value = 'Version deleted'
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function toggleEnabled(v: Version) {
  try {
    await api.updateVersion(v.id, { enabled: !v.enabled })
    await load()
  } catch (e) {
    error.value = (e as Error).message
  }
}

const versions = computed(() => data.value?.versions ?? [])

function fmtSize(n: number): string {
  if (n > 1024 * 1024 * 1024) return (n / 1024 / 1024 / 1024).toFixed(2) + ' GB'
  if (n > 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MB'
  if (n > 1024) return (n / 1024).toFixed(1) + ' KB'
  return n + ' B'
}

function fmtDate(s: string): string {
  return new Date(s).toISOString().replace('T', ' ').slice(0, 19)
}
</script>

<template>
  <div class="admin-app">
    <div v-if="data" class="page-header">
      <div class="eyebrow">▌ ADMIN</div>
      <h1 class="title">{{ data.app.name }}</h1>
    </div>

    <v-alert v-if="error" type="error" variant="outlined" class="mb-4">
      {{ error }}
    </v-alert>

    <v-tabs v-model="tab" class="mt-4">
      <v-tab value="versions">Versions</v-tab>
      <v-tab value="channels">Channels</v-tab>
      <v-tab value="stats">Stats</v-tab>
    </v-tabs>

    <v-divider />

    <!-- Versions tab -->
    <div v-if="tab === 'versions'" class="tab-body">
      <v-data-table
        :items="versions"
        :headers="[
          { title: 'Version', key: 'version_name' },
          { title: 'Size', key: 'file_size' },
          { title: 'Access', key: 'access_mode' },
          { title: 'Downloads', key: 'download_count' },
          { title: 'Status', key: 'enabled' },
          { title: '', key: 'actions', sortable: false, align: 'end' },
        ]"
        hide-default-footer
        :items-per-page="-1"
      >
        <template #item.version_name="{ item }">
          <MonoText>{{ item.version_name }}</MonoText>
          <MonoText muted> · code {{ item.version_code }}</MonoText>
        </template>
        <template #item.file_size="{ item }">
          <MonoText muted>{{ fmtSize(item.file_size) }}</MonoText>
        </template>
        <template #item.access_mode="{ item }">
          <StatusDot :mode="item.enabled ? item.access_mode : 'taken_down'" />
        </template>
        <template #item.download_count="{ item }">
          <MonoText>{{ item.download_count }}</MonoText>
        </template>
        <template #item.enabled="{ item }">
          <v-btn variant="text" size="small" @click="toggleEnabled(item)">
            {{ item.enabled ? 'Take down' : 'Re-enable' }}
          </v-btn>
        </template>
        <template #item.actions="{ item }">
          <v-btn
            variant="text"
            size="small"
            color="error"
            @click="deleteTarget = item"
          >
            Delete
          </v-btn>
        </template>
      </v-data-table>
    </div>

    <!-- Channels tab -->
    <div v-else-if="tab === 'channels'" class="tab-body">
      <div class="create-row">
        <v-text-field
          v-model="newChannelName"
          label="New channel name"
          density="comfortable"
          hide-details
          @keyup.enter="createChannel"
        />
        <v-btn color="primary" :disabled="!newChannelName" @click="createChannel">
          Create
        </v-btn>
      </div>
      <v-data-table
        :items="channels"
        :headers="[
          { title: 'Name', key: 'name' },
          { title: 'ID', key: 'id' },
        ]"
        class="mt-6"
        hide-default-footer
        :items-per-page="-1"
      >
        <template #item.name="{ item }">
          <MonoText>{{ item.name }}</MonoText>
        </template>
        <template #item.id="{ item }">
          <MonoText muted>{{ item.id }}</MonoText>
        </template>
      </v-data-table>
    </div>

    <!-- Stats tab -->
    <div v-else-if="tab === 'stats'" class="tab-body">
      <div v-if="!stats" class="empty">
        <p>Select a version from the Versions tab to view stats.</p>
      </div>
      <div v-else>
        <div class="stat-strip">
          <div class="stat-cell">
            <div class="stat-label">Downloads</div>
            <div class="stat-value">{{ stats.download_count }}</div>
          </div>
          <div class="stat-cell">
            <div class="stat-label">Installs</div>
            <div class="stat-value">{{ stats.install_count }}</div>
          </div>
        </div>
        <div class="eyebrow mt-6">▌ RECENT</div>
        <v-data-table
          :items="stats.recent"
          :headers="[
            { title: 'Time', key: 'created_at' },
            { title: 'IP', key: 'ip' },
            { title: 'User Agent', key: 'user_agent' },
          ]"
          class="mt-2"
          hide-default-footer
          :items-per-page="-1"
        >
          <template #item.created_at="{ item }">
            <MonoText muted>{{ fmtDate(item.created_at) }}</MonoText>
          </template>
          <template #item.ip="{ item }">
            <MonoText>{{ item.ip }}</MonoText>
          </template>
          <template #item.user_agent="{ item }">
            <MonoText muted class="ua-cell">{{ item.user_agent }}</MonoText>
          </template>
        </v-data-table>
        <v-btn class="mt-4" variant="text" @click="stats = null">Clear</v-btn>
      </div>
    </div>

    <v-dialog v-model="deleteTarget" max-width="400">
      <v-card class="pa-5">
        <div class="eyebrow">▌ CONFIRM DELETE</div>
        <p class="dialog-body">
          Delete version <MonoText>{{ deleteTarget?.version_name }}</MonoText>
          and its storage file?
        </p>
        <div class="dialog-actions">
          <v-btn variant="text" @click="deleteTarget = null">Cancel</v-btn>
          <v-btn color="error" @click="deleteVersion">Delete</v-btn>
        </div>
      </v-card>
    </v-dialog>

    <v-snackbar v-model="snackbar" :timeout="2000">
      {{ snackbar }}
    </v-snackbar>

    <!-- Hidden helper to expose version selection from Versions tab -->
    <v-btn class="d-none" @click="loadStats">_</v-btn>
  </div>
</template>

<style scoped>
.admin-app {
  max-width: var(--max-w);
  margin: 0 auto;
  padding: var(--sp-8) var(--sp-6);
}
.page-header {
  margin-bottom: var(--sp-4);
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
  font-size: 2.25rem;
  font-weight: 500;
  letter-spacing: -0.02em;
  margin: 0;
}
.tab-body {
  padding: var(--sp-6) 0;
}
.create-row {
  display: flex;
  gap: var(--sp-2);
  align-items: start;
  max-width: 500px;
}
.stat-strip {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--sp-2);
  max-width: 400px;
}
.stat-cell {
  border: 1px solid var(--border);
  background: var(--surface);
  padding: var(--sp-3) var(--sp-4);
}
.stat-label {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  letter-spacing: 0.15em;
  text-transform: uppercase;
  color: var(--text-mute);
  margin-bottom: var(--sp-1);
}
.stat-value {
  font-size: 1.75rem;
  font-weight: 500;
  color: var(--accent);
}
.ua-cell {
  max-width: 400px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: inline-block;
}
.empty {
  padding: var(--sp-8) 0;
  color: var(--text-mute);
  text-align: center;
}
.dialog-body {
  margin: var(--sp-3) 0;
  color: var(--text-mute);
}
.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--sp-2);
}
</style>
```

- [ ] **Step 2: Verify build and dev**

Run:
```bash
cd frontend && npm run build
```

In dev server, navigate to an admin app page. Expected: header, three tabs (Versions/Channels/Stats), Versions tab shows a data table with action buttons; Channels tab has create form + table; Stats tab has empty state.

- [ ] **Step 3: Commit**

```bash
cd frontend && git add src/views/admin/AdminApp.vue
git commit -m "feat: redesign AdminApp with tabs for versions/channels/stats"
```

---

## Task 14: Rewrite admin/Upload.vue

**Files:**
- Modify: `frontend/src/views/admin/Upload.vue`

- [ ] **Step 1: Rewrite Upload.vue**

Replace `frontend/src/views/admin/Upload.vue` with:

```vue
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../../api/client'
import type { AppItem, Channel } from '../../api/types'

const router = useRouter()

const file = ref<File | null>(null)
const appId = ref<number | null>(null)
const channelId = ref<number | null>(null)
const versionName = ref('')
const versionCode = ref<number | null>(null)
const changelog = ref('')
const accessMode = ref<'public' | 'password' | 'expiry'>('public')
const password = ref('')
const expiresAt = ref('')
const error = ref('')
const loading = ref(false)

const apps = ref<AppItem[]>([])
const channels = ref<Channel[]>([])

onMounted(async () => {
  try {
    apps.value = await api.adminApps()
  } catch (e) {
    error.value = (e as Error).message
  }
})

const appItems = computed(() =>
  apps.value.map((a) => ({ title: a.name, value: a.id }))
)

const channelItems = computed(() =>
  channels.value.map((c) => ({ title: c.name, value: c.id }))
)

watch(appId, async (id) => {
  if (!id) {
    channels.value = []
    return
  }
  try {
    channels.value = await api.channels(id)
  } catch (e) {
    error.value = (e as Error).message
  }
})

function onFileChange(f: File | File[] | null) {
  if (Array.isArray(f)) file.value = f[0] ?? null
  else file.value = f
}

async function submit() {
  if (!file.value || !appId.value || !versionName.value) {
    error.value = 'File, app, and version name are required.'
    return
  }
  error.value = ''
  loading.value = true

  const form = new FormData()
  form.append('file', file.value)
  form.append('app_id', String(appId.value))
  if (channelId.value) form.append('channel_id', String(channelId.value))
  form.append('version_name', versionName.value)
  if (versionCode.value) form.append('version_code', String(versionCode.value))
  form.append('changelog', changelog.value)
  form.append('access_mode', accessMode.value)
  if (accessMode.value === 'password') form.append('password', password.value)
  if (accessMode.value === 'expiry' && expiresAt.value) {
    form.append('expires_at', new Date(expiresAt.value).toISOString())
  }

  try {
    await api.uploadVersion(form)
    router.push(`/admin/app/${appId.value}`)
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="upload">
    <div class="page-header">
      <div class="eyebrow">▌ UPLOAD</div>
      <h1 class="title">New Version</h1>
    </div>

    <v-alert v-if="error" type="error" variant="outlined" class="mb-4">
      {{ error }}
    </v-alert>

    <div class="form">
      <section class="form-section">
        <div class="eyebrow">▌ FILE</div>
        <v-file-input
          :model-value="file"
          label="Choose installation package"
          accept=".apk,.aab,.ipa,.exe,.dmg"
          prepend-icon=""
          show-size
          density="comfortable"
          @update:model-value="onFileChange"
        />
      </section>

      <section class="form-section">
        <div class="eyebrow">▌ METADATA</div>
        <v-select
          v-model="appId"
          :items="appItems"
          label="Application"
        />
        <v-select
          v-model="channelId"
          :items="channelItems"
          label="Channel"
          :disabled="!appId"
          clearable
        />
        <div class="row-2">
          <v-text-field v-model="versionName" label="Version name" placeholder="1.0.0" />
          <v-text-field
            v-model.number="versionCode"
            label="Version code"
            type="number"
            placeholder="1"
          />
        </div>
        <v-textarea
          v-model="changelog"
          label="Changelog"
          rows="3"
          auto-grow
        />
      </section>

      <section class="form-section">
        <div class="eyebrow">▌ ACCESS</div>
        <v-radio-group v-model="accessMode" inline>
          <v-radio label="Public" value="public" />
          <v-radio label="Password" value="password" />
          <v-radio label="Expires" value="expiry" />
        </v-radio-group>
        <v-text-field
          v-if="accessMode === 'password'"
          v-model="password"
          label="Download password"
          type="password"
        />
        <v-text-field
          v-if="accessMode === 'expiry'"
          v-model="expiresAt"
          label="Expires at"
          type="datetime-local"
        />
      </section>

      <div class="actions">
        <v-btn variant="text" @click="router.back()">Cancel</v-btn>
        <v-btn
          color="primary"
          :loading="loading"
          :disabled="!file || !appId || !versionName"
          @click="submit"
        >
          Upload
        </v-btn>
      </div>
    </div>
  </div>
</template>

<style scoped>
.upload {
  max-width: 720px;
  margin: 0 auto;
  padding: var(--sp-8) var(--sp-6);
}
.page-header {
  margin-bottom: var(--sp-6);
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
  font-size: 2.25rem;
  font-weight: 500;
  letter-spacing: -0.02em;
  margin: 0 0 var(--sp-6) 0;
}
.form {
  display: flex;
  flex-direction: column;
  gap: var(--sp-6);
}
.form-section {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
  padding-bottom: var(--sp-6);
  border-bottom: 1px solid var(--border);
}
.form-section:last-of-type {
  border-bottom: none;
}
.row-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--sp-3);
}
.actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--sp-2);
}
@media (max-width: 600px) {
  .row-2 { grid-template-columns: 1fr; }
}
</style>
```

- [ ] **Step 2: Verify build and dev**

Run:
```bash
cd frontend && npm run build
```

In dev server, visit /admin/upload. Expected: three sections (File, Metadata, Access) with hairline dividers, conditional fields for password/expiry, Upload button at the bottom.

- [ ] **Step 3: Commit**

```bash
cd frontend && git add src/views/admin/Upload.vue
git commit -m "feat: redesign Upload with sectioned form and conditional access"
```

---

## Task 15: End-to-end smoke test

**Files:** (none — verification only)

- [ ] **Step 1: Build the single binary**

Run from the project root:
```bash
make build
```

Expected: `bin/disapp` is built. The Go server is unchanged — it only sees the new frontend dist via `go:embed`.

- [ ] **Step 2: Run the server and exercise each route**

Start the server:
```bash
APP_CONFIG=config.json ./bin/disapp &
```

Then verify in another terminal:
```bash
# Home returns the new HTML
curl -s http://localhost:8080/ | head -c 200
# Login route works
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/login
# Admin route works
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/admin
# API still works
curl -s http://localhost:8080/api/v1/apps | head -c 100
# Static assets load
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/assets/index-CsUDhMuy.css
```

Expected: home returns 200 with the new HTML (Inter font, eyebrow text), login/admin return 200, API returns 200 with JSON, CSS asset returns 200.

- [ ] **Step 3: Visual verification**

Open http://localhost:8080 in a browser. Walk through:
1. Home page — eyebrow, title, stat strip, card grid. Dark theme by default.
2. Click the theme toggle top-right — should cycle through three states.
3. Click an app — two-column detail with version rows.
4. Log in — grid background, centered card.
5. After login — top bar gains "Admin" and "Upload" tabs.
6. Admin page — data table, create form, delete confirm dialog.
7. AdminApp — three tabs work.
8. Upload — three sections, conditional access fields.
9. Toggle theme to light — paper background, navy ink, same structure.

Stop the server:
```bash
pkill -f bin/disapp
```

- [ ] **Step 4: Commit any final tweaks**

If you made any visual tweaks during smoke testing, commit them with a descriptive message. Otherwise, no commit is needed — the plan is complete.

---

## Self-Review

**Spec coverage check** (mapping spec sections to tasks):

| Spec section | Task(s) |
|---|---|
| Visual tokens (color, type, spacing) | Task 2 (tokens.css), Task 3 (fonts, overrides) |
| Layout & navigation (app shell, top bar) | Task 4 (vuetify.ts), Task 8 (AppShell) |
| Components & Vuetify map | Task 3 (overrides.scss), Task 4 (defaults), Tasks 6-7 (custom components) |
| Per-page breakdown | Tasks 9-14 |
| Theming strategy (3-state toggle) | Task 5 (useTheme), Task 8 (AppShell toggle button) |
| Signature element (StatusDot) | Task 6 |
| File structure | Tasks 1-14 each create/modify their listed files |
| Testing (build + smoke) | Task 15 |

**Placeholder scan:** No "TBD", "TODO", or "implement later" found. Every code step shows the actual code.

**Type consistency:** `useTheme` returns `{ choice, resolved, cycle }` in both Task 5 and used in Task 8 (`choice`, `cycle`). `StatusDot` prop is `mode: 'public' | 'password' | 'expiry' | 'enabled' | 'taken_down' | 'live'` consistently in Tasks 6, 9, 10, 12, 13. `StatBlock` props are `{ label, value, emphasis? }` in Task 7 and `{ label, value, emphasis }` in Task 9. `MonoText` props are `{ muted? }` in Task 7 and used as `<MonoText>` / `<MonoText muted>` consistently throughout. All `fmtSize` / `fmtDate` helpers are duplicated in views — this is intentional for view isolation, not a typo.

**Ambiguity check:** "Three-state toggle cycles system → light → dark → system" is explicit in Task 5. "Eyebrow always above section title" is consistent. "Data table hides default footer" appears in both Task 12 and 13 — explicit and consistent.
