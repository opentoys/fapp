# Frontend — Vue 3 + Tailwind CSS v4 + shadcn-vue + TypeScript + Vite

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
- Icons: `lucide-vue-next` components; toasts via `vue-sonner`.
- UI: use the shadcn primitives in `src/components/ui/*` (Button, Card, Dialog,
  Table, …); Tailwind v4 utilities with oklch tokens in `src/index.css`.
- All `computed()` values used for reactive data (tables, filters, chart series).
- Module-level `ref` in composables for shared state across components.
- Locale auto-detected from `navigator.language` on first visit (zh → Chinese, else English).
- `noUnusedLocals`/`noUnusedParameters` are on — unused imports fail `npm run build`.