# Frontend — Vue 3 + Vuetify 3 + TypeScript + Vite

## Commands

```bash
# Type-check + build
npm run build

# Dev server (hot-reload, proxies /api → :8080)
npm run dev

# Type-check only
npx vue-tsc -b
```

## Key conventions

- `useI18n()` composable for all UI strings. API messages are NOT translated.
- `useAuth()` composable for auth state (reactive). JWT decoded for username/uid.
- `useTheme()` composable for theme (light/dark/system), module-level ref.
- Icons: `@mdi/js` SVG paths, imported individually. `vuetify/iconsets/mdi-svg` icon set.
- `<v-icon :icon="mdiIconName" />` — never use `icon="string"` for SVG paths.
- All Vuetify data-table headers are `computed()` for reactivity.
- Module-level `ref` in composables for shared state across components.
- Locale auto-detected from `navigator.language` on first visit (zh → Chinese, else English).