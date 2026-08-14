# shadcn UI Redesign — Design Spec

Date: 2026-08-14
Status: Approved by user (visual decisions: Neutral theme, single top bar)

## Background

The frontend (`frontend/`, Vue 3 + Vite + TypeScript) currently uses **Vuetify 3** as its
component/styling layer. Every one of the 11 `.vue` files relies on `v-*` components
(46 buttons, 33 cards, 3 data tables, dialogs, menus, chips, etc.), on `@mdi/js` icons,
and on a Vuetify-specific theme composable. `style.css` still contains unused Vite-template
cruft. The goal is to replace the entire UI layer with **Tailwind CSS + shadcn-ui**
(shadcn-vue for Vue) and re-plan the styling per shadcn design conventions.

## Goals

- Replace Vuetify with Tailwind CSS v4 + shadcn-vue components, keeping **all current
  functionality and page structure** (public download flow, admin apps/users/upload,
  stats chart, i18n, light/dark/system theme).
- Adopt the canonical shadcn look: neutral zinc palette, near-black primary buttons,
  CSS-variable theming, rounded corners, subtle borders.
- Remove Vuetify, its Vite plugin, `@mdi/js`, `@iconify/*`, and `sass` from the project.

## Non-Goals

- No backend changes (API, auth, storage) — this is a frontend-only restyle.
- No new features. No re-architecture of the router/API/composables.
- No migration of the dead `style.css` content — it is deleted.

## Design Decisions

### 1. Stack

| Replace | With |
|---|---|
| Vuetify 3 + `vite-plugin-vuetify` + `sass` | Tailwind CSS v4 (`@tailwindcss/vite`, CSS-first config) |
| `vuetify` components | shadcn-vue component set hand-written into `src/components/ui/` |
| `@mdi/js`, `@iconify/vue`, `@iconify-json/material-symbols` | `lucide-vue-next` |
| `v-snackbar` | `vue-sonner` (shadcn Toaster) |

New dependencies: `tailwindcss`, `@tailwindcss/vite`, `reka-ui`,
`class-variance-authority`, `clsx`, `tailwind-merge`, `tailwind-variants`,
`@vueuse/core`, `lucide-vue-next`, `vue-sonner`.

**Approach: hand-write the `ui/` component subset** (rather than the `shadcn-vue` CLI)
because the CLI requires interactive init + registry network access, which is unreliable
in this environment. Component source follows the canonical shadcn-vue implementations.
Only the ~18 components actually used are included.

**No `@/` path alias.** The codebase uses relative imports throughout; keep that
convention to avoid tsconfig/vite config churn.

### 2. Theming

- **Neutral theme** (user-selected): zinc grays, `--primary` near-black (zinc-900) with
  white `--primary-foreground`; `oklch` color tokens defined in `src/index.css`.
- CSS layout (Tailwind v4):
  - `@import "tailwindcss";`
  - `@theme inline { --color-background: var(--background); … }` mapping every
    shadcn token (`--color-primary`, `--color-muted`, `--color-border`, `--color-ring`,
    `--color-success`, `--color-warning`, `--color-info`, `--color-destructive`, …).
  - `:root { … }` light values and `.dark { … }` dark values.
- **Dark mode** = `.dark` class on `<html>`. Rewrite `useTheme.ts` to drop the Vuetify
  dependency:
  - Keep module-level `choice: 'system' | 'light' | 'dark'`, same `localStorage` key
    (`disapp-theme`), same system `prefers-color-scheme` listener.
  - `applyTheme(c)` toggles `document.documentElement.classList` for `dark` and sets
    `color-scheme` via CSS (`:root`/`.dark` blocks).
- Body: `bg-background text-foreground`, system font stack (Tailwind default), no webfont.

### 3. Component inventory (`src/components/ui/`)

button, card, input, label, textarea, select, dialog, alert-dialog, alert, badge,
avatar, table, tabs, dropdown-menu, radio-group, checkbox, skeleton, separator, sonner.
Plus `src/lib/utils.ts` exporting `cn()` (clsx + tailwind-merge).

- **Badge** gets a `variant` set extended for status semantics: `default | secondary |
  outline | destructive | success | warning | info` mapping to `--color-*` tokens, used for
  platform/release/access/status chips.
- **Select** is wrapped once in `src/components/AppSelect.vue` so every call site can pass
  `items: {title, value}[]`, `modelValue`, `placeholder`, and a `multiple` flag, instead of
  repeating trigger/content/item markup.

### 4. Vuetify → shadcn component mapping

| Vuetify | shadcn | Notes |
|---|---|---|
| `v-app-bar` / `v-main` / `v-app` | custom `AppShell.vue` layout (flex column, sticky header, `border-b`, `bg-background`) | nav = ghost Button links with active state from route |
| `v-btn` | `Button` | variants: default/outline/ghost/destructive; sizes: default/sm/icon |
| `v-card`, `v-card-title/text/actions` | `Card` (+ `CardHeader/Title/Description/Content/Footer`) | |
| `v-alert` | `Alert` (+ `AlertTitle/Description`) | error/info/warning via variant class |
| `v-text-field` | `Input` + `Label` | |
| `v-textarea` | `Textarea` | |
| `v-select` | `AppSelect` (shadcn `Select`) | multi-select in Upload → inline `Checkbox` group |
| `v-dialog` (forms) | `Dialog` | create/edit app+user, publish, password |
| `v-dialog` (delete confirm) | `AlertDialog` | delete app/version/user |
| `v-chip` | `Badge` (extended variants) | |
| `v-avatar` | `Avatar` (`AvatarImage`/`AvatarFallback`) | |
| `v-menu` | `DropdownMenu` | user menu, language, theme |
| `v-data-table` | `Table` primitives + `v-for` | users, versions, recent-logs |
| `v-tabs` / `v-window` | `Tabs` (reka) | AdminApp overview/versions/stats |
| `v-radio-group` | `RadioGroup` | access-mode (public/password/expiry) |
| `v-progress-linear` | `Skeleton` / `Loader2` spinner | chart loading |
| `v-file-input` | custom `FileUpload.vue` | hidden `<input type=file>` + styled trigger + optional drag-drop zone |
| `v-img` | plain `<img>` | |
| `v-snackbar` | `vue-sonner` `<Toaster/>` | `showSnack(msg)` → `toast(msg)` |

### 5. Page-by-page

- **AppShell.vue**: header with brand ("Distribution") or, on `/app/*`, the shared app
  icon+name (from `useDownloadApp`); center route nav (Apps/Users, authed only); right
  language menu, theme dropdown, sign-in button (hidden on `/app/*` and `/login`), user
  dropdown (change password / logout) + dialogs. Mount `<Toaster/>`.
- **Home.vue**: stat cards row (3) + responsive app card grid; access badge; app card links
  to `/app/{encodeURIComponent(name)}`.
- **AppDetail.vue / VersionPanel.vue**: keep the decided structure — mobile hero (no-card,
  floating download bar) + desktop per-platform cards; architecture chips stay clickable
  (Badge-based); password dialog stays.
- **Login.vue**: centered `Card` with `Input` fields + `Button`; error via `Alert`.
- **Admin.vue** (apps grid): create/edit dialogs with icon `FileUpload`; delete
  `AlertDialog`; empty/error states; app card click → `/admin/app/{id}`.
- **AdminApp.vue**: Tabs (Overview / Versions / Stats). Overview: app-info card, merged
  download-link + access-permission card (RadioGroup), screenshots card (`FileUpload`
  multi, `<img>` grid). Versions: filter selects (`AppSelect`) + `Table`. Stats: chart
  card (selects + `LineChart` + legend) + version-detail cards + recent-logs `Table`.
- **Upload.vue**: file `FileUpload` (drag-drop) + parsed-meta card + form fields (selects,
  inputs, textarea, inline arch checkboxes).
- **Users.vue**: super-admin info card, `Table`, create/edit `Dialog`, delete `AlertDialog`.
- **LineChart.vue**: keep the dependency-free SVG chart; change series colors from
  `rgb(var(--v-theme-primary))` / `--v-theme-warning` to `var(--color-primary)` /
  `var(--color-warning)` (CSS vars resolve at runtime; chart code otherwise unchanged).

### 6. Files

**Delete**: `frontend/src/plugins/vuetify.ts`, `frontend/src/style.css`,
`frontend/vite.config.ts` (rewritten, Vuetify plugin removed).

**Create**:
- `frontend/src/index.css` — Tailwind import + shadcn theme tokens + base layer.
- `frontend/src/lib/utils.ts` — `cn()`.
- `frontend/src/components/ui/*` — the ~18 components listed above.
- `frontend/src/components/AppSelect.vue`, `frontend/src/components/FileUpload.vue`.

**Rewrite**: `main.ts`, `vite.config.ts`, `AppShell.vue`, `Home.vue`, `AppDetail.vue`,
`VersionPanel.vue`, `Login.vue`, `Admin.vue`, `AdminApp.vue`, `Upload.vue`, `Users.vue`,
`LineChart.vue` (color tokens only), `useTheme.ts`.

**Unchanged**: `useAuth.ts`, `useI18n.ts`, `useDownloadApp.ts`, `api/*`, `router/index.ts`,
`constants/*`, `utils/ua.ts`, `utils/format.ts`, `i18n/messages.ts`.

### 7. package.json

- Remove: `vuetify`, `vite-plugin-vuetify`, `sass`, `@mdi/js`, `@iconify/vue`,
  `@iconify-json/material-symbols`.
- Add: `tailwindcss`, `@tailwindcss/vite`, `reka-ui`, `class-variance-authority`,
  `clsx`, `tailwind-merge`, `tailwind-variants`, `@vueuse/core`, `lucide-vue-next`,
  `vue-sonner`.

## Verification

1. `cd frontend && npm run build` (vue-tsc + vite) — must pass with zero errors.
2. Copy `frontend/dist` → `backend/static/dist`, `go build -o ../bin/disapp ./cmd/server`
   in `backend/`, restart `./bin/disapp` (embed.FS needs restart).
3. Manual pass over all 8 pages (Home, Login, AppDetail desktop+mobile-UA, Admin apps,
   AdminApp 3 tabs, Upload, Users) checking: light/dark/system theme toggle, language
   switch, download flow incl. password dialog, filters, create/edit/delete dialogs,
   chart + selects, arch chips, screenshots upload.

## Risks / Notes

- **reka-ui version drift**: hand-written components pin to a known reka-ui API; verify
  against the installed version (peer-dep) if behavior diverges.
- **Table + Tabs + Select interplay**: all Radix-based; confirm focus rings and controlled
  `v-model` work as expected in vue-tsc.
- **vue-sonner dark styling**: ensure Toaster respects the `.dark` class (uses CSS vars).
- Scope is deliberately frontend-only; the Go side is untouched apart from re-embedding dist.
