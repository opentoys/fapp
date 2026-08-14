# shadcn UI Redesign — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Vuetify 3 with Tailwind CSS v4 + shadcn-vue (reka-ui primitives) across the Vue 3 frontend, restyled per the approved shadcn spec, preserving all functionality.

**Architecture:** Tailwind v4 provides the design system via CSS-variable tokens in `src/index.css`. A hand-written `src/components/ui/*` set (reka-ui + cva) supplies shadcn primitives. Views are rewritten one at a time against those primitives; Vuetify stays installed until the last task so every intermediate commit still builds. `useTheme` is decoupled from Vuetify and drives a `.dark` class.

**Tech Stack:** Vue 3.5, Vite 8, Tailwind CSS v4 (`@tailwindcss/vite`), reka-ui, class-variance-authority, clsx, tailwind-merge, tailwind-variants, @vueuse/core, lucide-vue-next, vue-sonner, tw-animate-css.

> **Testing note:** This frontend has no unit-test runner (no vitest/cypress). The gate for "the step works" is `npm run build` (runs `vue-tsc -b && vite build`), which type-checks every `.vue`/`.ts` file, plus a manual smoke check at the end. Each code step therefore verifies via `npm run build` rather than a failing test. Vuetify remains a dependency until Task 16 so untouched `v-*` views still type-check during the transition.

---

### Task 1: Install new dependencies and add Tailwind v4 pipeline

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/vite.config.ts`
- Create: `frontend/src/index.css` (minimal, real theme lands in Task 2)
- Create: `frontend/src/lib/utils.ts`

- [ ] **Step 1: Install dependencies**

```bash
cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend
npm install tailwindcss @tailwindcss/vite reka-ui class-variance-authority clsx tailwind-merge tailwind-variants @vueuse/core lucide-vue-next vue-sonner tw-animate-css
```

Expected: packages added to `dependencies` in `package.json`.

- [ ] **Step 2: Create `src/lib/utils.ts`**

```ts
import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
```

- [ ] **Step 3: Create `src/index.css` (scaffold; full theme in Task 2)**

```css
@import "tailwindcss";
@import "tw-animate-css";
```

- [ ] **Step 4: Update `vite.config.ts`**

```ts
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

// Dev workflow: run `go run ./cmd/server` in backend/ for the API, and
// `npm run dev` here. Vite serves the SPA on :5173 and forwards /api/*
// to the Go server, so the browser hits one origin and CORS doesn't get
// in the way. Override VITE_API_TARGET if your backend isn't on :8080.
const API_TARGET = process.env.VITE_API_TARGET || 'http://localhost:8080'

export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
    // vite-plugin-vuetify is removed in Task 16; kept for now so existing v-* views build.
  ],
  server: {
    host: true,
    proxy: {
      '/api': {
        target: API_TARGET,
        changeOrigin: true,
      },
    },
  },
})
```

- [ ] **Step 5: Verify build**

Run: `cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend && npm run build`
Expected: build succeeds (Vuetify + Tailwind coexist).

- [ ] **Step 6: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/vite.config.ts frontend/src/lib/utils.ts frontend/src/index.css
git commit -m "feat: add tailwind v4 + shadcn foundation deps"
```

---

### Task 2: Full shadcn neutral theme + decouple `useTheme`

**Files:**
- Rewrite: `frontend/src/index.css`
- Rewrite: `frontend/src/composables/useTheme.ts`
- Modify: `frontend/src/main.ts`

- [ ] **Step 1: Rewrite `src/index.css`**

```css
@import "tailwindcss";
@import "tw-animate-css";

@custom-variant dark (&:is(.dark *));

:root {
  --radius: 0.625rem;
  --background: oklch(1 0 0);
  --foreground: oklch(0.141 0.005 285.823);
  --card: oklch(1 0 0);
  --card-foreground: oklch(0.141 0.005 285.823);
  --popover: oklch(1 0 0);
  --popover-foreground: oklch(0.141 0.005 285.823);
  --primary: oklch(0.21 0.006 285.885);
  --primary-foreground: oklch(0.985 0 0);
  --secondary: oklch(0.967 0.001 286.375);
  --secondary-foreground: oklch(0.21 0.006 285.885);
  --muted: oklch(0.967 0.001 286.375);
  --muted-foreground: oklch(0.552 0.016 285.938);
  --accent: oklch(0.967 0.001 286.375);
  --accent-foreground: oklch(0.21 0.006 285.885);
  --destructive: oklch(0.577 0.245 27.325);
  --border: oklch(0.92 0.004 286.32);
  --input: oklch(0.92 0.004 286.32);
  --ring: oklch(0.705 0.015 286.067);
  --success: oklch(0.596 0.145 163.225);
  --success-foreground: oklch(0.985 0 0);
  --warning: oklch(0.769 0.188 70.08);
  --warning-foreground: oklch(0.985 0 0);
  --info: oklch(0.623 0.214 259.815);
  --info-foreground: oklch(0.985 0 0);
  --chart-1: oklch(0.646 0.222 41.116);
  --chart-2: oklch(0.696 0.17 162.48);
}

.dark {
  --background: oklch(0.141 0.005 285.823);
  --foreground: oklch(0.985 0 0);
  --card: oklch(0.21 0.006 285.885);
  --card-foreground: oklch(0.985 0 0);
  --popover: oklch(0.21 0.006 285.885);
  --popover-foreground: oklch(0.985 0 0);
  --primary: oklch(0.92 0.004 286.32);
  --primary-foreground: oklch(0.21 0.006 285.885);
  --secondary: oklch(0.274 0.006 286.033);
  --secondary-foreground: oklch(0.985 0 0);
  --muted: oklch(0.274 0.006 286.033);
  --muted-foreground: oklch(0.705 0.015 286.067);
  --accent: oklch(0.274 0.006 286.033);
  --accent-foreground: oklch(0.985 0 0);
  --destructive: oklch(0.704 0.191 22.216);
  --border: oklch(1 0 0 / 10%);
  --input: oklch(1 0 0 / 15%);
  --ring: oklch(0.552 0.016 285.938);
  --success: oklch(0.696 0.17 162.48);
  --success-foreground: oklch(0.985 0 0);
  --warning: oklch(0.828 0.189 84.429);
  --warning-foreground: oklch(0.985 0 0);
  --info: oklch(0.685 0.169 237.323);
  --info-foreground: oklch(0.985 0 0);
  --chart-1: oklch(0.488 0.243 264.376);
  --chart-2: oklch(0.696 0.17 162.48);
}

@theme inline {
  --color-background: var(--background);
  --color-foreground: var(--foreground);
  --color-card: var(--card);
  --color-card-foreground: var(--card-foreground);
  --color-popover: var(--popover);
  --color-popover-foreground: var(--popover-foreground);
  --color-primary: var(--primary);
  --color-primary-foreground: var(--primary-foreground);
  --color-secondary: var(--secondary);
  --color-secondary-foreground: var(--secondary-foreground);
  --color-muted: var(--muted);
  --color-muted-foreground: var(--muted-foreground);
  --color-accent: var(--accent);
  --color-accent-foreground: var(--accent-foreground);
  --color-destructive: var(--destructive);
  --color-border: var(--border);
  --color-input: var(--input);
  --color-ring: var(--ring);
  --color-success: var(--success);
  --color-success-foreground: var(--success-foreground);
  --color-warning: var(--warning);
  --color-warning-foreground: var(--warning-foreground);
  --color-info: var(--info);
  --color-info-foreground: var(--info-foreground);
  --color-chart-1: var(--chart-1);
  --color-chart-2: var(--chart-2);
  --radius-sm: calc(var(--radius) - 4px);
  --radius-md: calc(var(--radius) - 2px);
  --radius-lg: var(--radius);
  --radius-xl: calc(var(--radius) + 4px);
}

@layer base {
  * {
    @apply border-border outline-ring/50;
  }
  body {
    @apply bg-background text-foreground;
    font-family: system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif;
  }
}
```

- [ ] **Step 2: Rewrite `src/composables/useTheme.ts`**

```ts
import { ref, watch, onMounted } from 'vue'

export type ThemeChoice = 'system' | 'light' | 'dark'

const STORAGE_KEY = 'disapp-theme'

function systemPrefersDark(): boolean {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

function applyTheme(c: ThemeChoice) {
  const resolved = c === 'system' ? (systemPrefersDark() ? 'dark' : 'light') : c
  document.documentElement.classList.toggle('dark', resolved === 'dark')
  document.documentElement.style.colorScheme = resolved
  localStorage.setItem(STORAGE_KEY, c)
}

function readStored(): ThemeChoice {
  const v = localStorage.getItem(STORAGE_KEY)
  if (v === 'light' || v === 'dark' || v === 'system') return v
  return 'system'
}

// Module-level state so all callers share the same value.
const choice = ref<ThemeChoice>(readStored())

// Apply immediately (avoid a flash before mount), then keep listening to the OS.
applyTheme(choice.value)
let listenerBound = false

export function useTheme() {
  onMounted(() => {
    if (listenerBound) return
    listenerBound = true
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    mq.addEventListener('change', () => {
      if (choice.value === 'system') applyTheme('system')
    })
  })

  watch(choice, (c) => applyTheme(c))

  function setChoice(c: ThemeChoice) {
    choice.value = c
  }

  return { choice, setChoice }
}
```

- [ ] **Step 3: Update `src/main.ts`** — import the new CSS (keep `vuetify/styles` until Task 16)

```ts
import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import { vuetify } from './plugins/vuetify'
import 'vuetify/styles'
import './index.css'

createApp(App).use(router).use(vuetify).mount('#app')
```

- [ ] **Step 4: Verify build**

Run: `cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend && npm run build`
Expected: build succeeds.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/index.css frontend/src/composables/useTheme.ts frontend/src/main.ts
git commit -m "feat: shadcn neutral theme tokens + class-based dark mode"
```

---

### Task 3: UI primitives — button, badge, avatar, skeleton, separator, card

**Files (all created under `frontend/src/components/ui/`):**
- `button/index.ts`, `button/Button.vue`
- `badge/index.ts`, `badge/Badge.vue`
- `avatar/index.ts`, `avatar/Avatar.vue`
- `skeleton/index.ts`, `skeleton/Skeleton.vue`
- `separator/index.ts`, `separator/Separator.vue`
- `card/index.ts`, `card/Card.vue`, `card/CardHeader.vue`, `card/CardTitle.vue`, `card/CardDescription.vue`, `card/CardContent.vue`, `card/CardFooter.vue`

- [ ] **Step 1: `button/index.ts`**

```ts
import { cva, type VariantProps } from 'class-variance-authority'

export { default as Button } from './Button.vue'

export const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-all disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 shrink-0 [&_svg]:shrink-0 outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px]",
  {
    variants: {
      variant: {
        default: 'bg-primary text-primary-foreground shadow-xs hover:bg-primary/90',
        destructive: 'bg-destructive text-white shadow-xs hover:bg-destructive/90 focus-visible:ring-destructive/20 dark:bg-destructive/60',
        outline: 'border bg-background shadow-xs hover:bg-accent hover:text-accent-foreground dark:bg-input/30 dark:border-input dark:hover:bg-input/50',
        secondary: 'bg-secondary text-secondary-foreground shadow-xs hover:bg-secondary/80',
        ghost: 'hover:bg-accent hover:text-accent-foreground dark:hover:bg-accent/50',
        link: 'text-primary underline-offset-4 hover:underline',
      },
      size: {
        default: 'h-9 px-4 py-2 has-[>svg]:px-3',
        sm: 'h-8 rounded-md gap-1.5 px-3 has-[>svg]:px-2.5',
        lg: 'h-10 rounded-md px-6 has-[>svg]:px-4',
        icon: 'size-9',
      },
    },
    defaultVariants: { variant: 'default', size: 'default' },
  },
)

export type ButtonVariants = VariantProps<typeof buttonVariants>
```

- [ ] **Step 2: `button/Button.vue`**

```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { Primitive, type PrimitiveProps } from 'reka-ui'
import { cn } from '../../../lib/utils'
import { buttonVariants, type ButtonVariants } from '.'

interface Props extends PrimitiveProps {
  variant?: ButtonVariants['variant']
  size?: ButtonVariants['size']
  class?: HTMLAttributes['class']
}
const props = withDefaults(defineProps<Props>(), {
  as: 'button',
  variant: 'default',
  size: 'default',
})
</script>

<template>
  <Primitive
    :as="as"
    :as-child="asChild"
    :class="cn(buttonVariants({ variant, size }), props.class)"
  >
    <slot />
  </Primitive>
</template>
```

- [ ] **Step 3: `badge/index.ts`**

```ts
import { cva, type VariantProps } from 'class-variance-authority'

export { default as Badge } from './Badge.vue'

export const badgeVariants = cva(
  'inline-flex items-center justify-center gap-1 rounded-md border px-2 py-0.5 text-xs font-medium w-fit whitespace-nowrap shrink-0 [&>svg]:size-3 [&>svg]:pointer-events-none',
  {
    variants: {
      variant: {
        default: 'border-transparent bg-primary text-primary-foreground',
        secondary: 'border-transparent bg-secondary text-secondary-foreground',
        destructive: 'border-transparent bg-destructive text-white dark:bg-destructive/60',
        outline: 'text-foreground',
        success: 'border-transparent bg-success text-success-foreground',
        warning: 'border-transparent bg-warning text-warning-foreground',
        info: 'border-transparent bg-info text-info-foreground',
      },
    },
    defaultVariants: { variant: 'default' },
  },
)

export type BadgeVariants = VariantProps<typeof badgeVariants>
```

- [ ] **Step 4: `badge/Badge.vue`**

```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { Primitive, type PrimitiveProps } from 'reka-ui'
import { cn } from '../../../lib/utils'
import { badgeVariants, type BadgeVariants } from '.'

interface Props extends PrimitiveProps {
  variant?: BadgeVariants['variant']
  class?: HTMLAttributes['class']
}
const props = withDefaults(defineProps<Props>(), { as: 'span', variant: 'default' })
</script>

<template>
  <Primitive :as="as" :as-child="asChild" :class="cn(badgeVariants({ variant }), props.class)">
    <slot />
  </Primitive>
</template>
```

- [ ] **Step 5: `avatar/index.ts` + `avatar/Avatar.vue`** (simplified app wrapper: `src` + `fallback` + size class)

`avatar/index.ts`:
```ts
export { default as Avatar } from './Avatar.vue'
```

`avatar/Avatar.vue`:
```vue
<script setup lang="ts">
import { AvatarFallback, AvatarImage, AvatarRoot } from 'reka-ui'
import { cn } from '../../../lib/utils'

const props = withDefaults(defineProps<{ src?: string; fallback?: string; class?: string }>(), {
  src: '',
  fallback: '',
})
</script>

<template>
  <AvatarRoot :class="cn('relative flex shrink-0 overflow-hidden rounded-full', props.class)">
    <AvatarImage v-if="props.src" :src="props.src" alt="" class="aspect-square size-full object-cover" />
    <AvatarFallback class="bg-muted text-muted-foreground flex size-full items-center justify-center rounded-full text-xs font-medium uppercase">
      {{ props.fallback }}
    </AvatarFallback>
  </AvatarRoot>
</template>
```

- [ ] **Step 6: `skeleton/index.ts` + `skeleton/Skeleton.vue`**

`index.ts`:
```ts
export { default as Skeleton } from './Skeleton.vue'
```

`Skeleton.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '../../../lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <div :class="cn('bg-primary/10 animate-pulse rounded-md', props.class)" />
</template>
```

- [ ] **Step 7: `separator/index.ts` + `separator/Separator.vue`**

`index.ts`:
```ts
export { default as Separator } from './Separator.vue'
```

`Separator.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { Separator, type SeparatorProps } from 'reka-ui'
import { cn } from '../../../lib/utils'

const props = withDefaults(defineProps<SeparatorProps & { class?: HTMLAttributes['class'] }>(), {
  orientation: 'horizontal',
  decorative: true,
})
</script>

<template>
  <Separator
    :class="cn('bg-border shrink-0 data-[orientation=horizontal]:h-px data-[orientation=horizontal]:w-full data-[orientation=vertical]:h-full data-[orientation=vertical]:w-px', props.class)"
    v-bind="props"
  />
</template>
```

- [ ] **Step 8: `card/index.ts` + Card components**

`card/index.ts`:
```ts
export { default as Card } from './Card.vue'
export { default as CardHeader } from './CardHeader.vue'
export { default as CardTitle } from './CardTitle.vue'
export { default as CardDescription } from './CardDescription.vue'
export { default as CardContent } from './CardContent.vue'
export { default as CardFooter } from './CardFooter.vue'
```

`card/Card.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '../../../lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <div :class="cn('bg-card text-card-foreground flex flex-col gap-6 rounded-xl border py-6 shadow-sm', props.class)">
    <slot />
  </div>
</template>
```

`card/CardHeader.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '../../../lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <div :class="cn('grid flex-1 items-start gap-1.5 px-6', props.class)">
    <slot />
  </div>
</template>
```

`card/CardTitle.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '../../../lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <div :class="cn('font-semibold leading-none', props.class)">
    <slot />
  </div>
</template>
```

`card/CardDescription.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '../../../lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <div :class="cn('text-muted-foreground text-sm', props.class)">
    <slot />
  </div>
</template>
```

`card/CardContent.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '../../../lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <div :class="cn('px-6', props.class)">
    <slot />
  </div>
</template>
```

`card/CardFooter.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '../../../lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <div :class="cn('flex items-center px-6', props.class)">
    <slot />
  </div>
</template>
```

- [ ] **Step 9: Verify build**

Run: `cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend && npm run build`
Expected: build succeeds (new components are imported by nothing yet, but vue-tsc still compiles them).

- [ ] **Step 10: Commit**

```bash
git add frontend/src/components/ui/button frontend/src/components/ui/badge frontend/src/components/ui/avatar frontend/src/components/ui/skeleton frontend/src/components/ui/separator frontend/src/components/ui/card
git commit -m "feat: add shadcn ui primitives (button/badge/avatar/skeleton/separator/card)"
```

---

### Task 4: UI primitives — input, label, textarea, checkbox, radio-group, table

**Files (created under `frontend/src/components/ui/`):**
- `input/index.ts`, `input/Input.vue`
- `label/index.ts`, `label/Label.vue`
- `textarea/index.ts`, `textarea/Textarea.vue`
- `checkbox/index.ts`, `checkbox/Checkbox.vue`
- `radio-group/index.ts`, `radio-group/RadioGroup.vue`, `radio-group/RadioGroupItem.vue`
- `table/index.ts`, `table/Table.vue`, `table/TableHeader.vue`, `table/TableBody.vue`, `table/TableRow.vue`, `table/TableHead.vue`, `table/TableCell.vue`, `table/TableCaption.vue`, `table/TableEmpty.vue`

- [ ] **Step 1: `input/index.ts` + `input/Input.vue`**

`input/index.ts`:
```ts
export { default as Input } from './Input.vue'
```

`input/Input.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { useVModel } from '@vueuse/core'
import { cn } from '../../../lib/utils'

const props = defineProps<{
  defaultValue?: string | number
  modelValue?: string | number | null
  class?: HTMLAttributes['class']
}>()
const emits = defineEmits<{ 'update:modelValue': [value: string | number | null] }>()

const modelValue = useVModel(props, 'modelValue', emits, { passive: true, defaultValue: props.defaultValue })
</script>

<template>
  <input
    v-model="modelValue"
    :class="cn('border-input bg-transparent shadow-xs transition-[color,box-shadow] outline-none file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 md:text-sm text-base h-10 w-full min-w-0 rounded-md border px-3 py-1 focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px]', props.class)"
  />
</template>
```

- [ ] **Step 2: `label/index.ts` + `label/Label.vue`**

`label/index.ts`:
```ts
export { default as Label } from './Label.vue'
```

`label/Label.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '../../../lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <label :class="cn('text-sm font-medium leading-4 text-foreground peer-disabled:cursor-not-allowed peer-disabled:opacity-70', props.class)">
    <slot />
  </label>
</template>
```

- [ ] **Step 3: `textarea/index.ts` + `textarea/Textarea.vue`**

`textarea/index.ts`:
```ts
export { default as Textarea } from './Textarea.vue'
```

`textarea/Textarea.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { useVModel } from '@vueuse/core'
import { cn } from '../../../lib/utils'

const props = defineProps<{
  defaultValue?: string | number
  modelValue?: string | number | null
  class?: HTMLAttributes['class']
}>()
const emits = defineEmits<{ 'update:modelValue': [value: string | number | null] }>()

const modelValue = useVModel(props, 'modelValue', emits, { passive: true, defaultValue: props.defaultValue })
</script>

<template>
  <textarea
    v-model="modelValue"
    :class="cn('border-input bg-transparent shadow-xs transition-[color,box-shadow] outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50 flex min-h-16 w-full rounded-md border px-3 py-2 text-base md:text-sm', props.class)"
  />
</template>
```

- [ ] **Step 4: `checkbox/index.ts` + `checkbox/Checkbox.vue`**

`checkbox/index.ts`:
```ts
export { default as Checkbox } from './Checkbox.vue'
```

`checkbox/Checkbox.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { Check } from 'lucide-vue-next'
import { CheckboxIndicator, CheckboxRoot, type CheckboxRootEmits, type CheckboxRootProps } from 'reka-ui'
import { cn } from '../../../lib/utils'

const props = withDefaults(defineProps<CheckboxRootProps & { class?: HTMLAttributes['class'] }>(), { modelValue: false })
const emits = defineEmits<CheckboxRootEmits>()
</script>

<template>
  <CheckboxRoot
    v-bind="props"
    :class="cn('peer size-4 shrink-0 rounded-sm border border-input shadow-xs transition-shadow outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:border-primary data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground', props.class)"
    @update:checked="emits('update:checked', $event)"
  >
    <CheckboxIndicator class="flex items-center justify-center text-current">
      <Check class="size-3.5" />
    </CheckboxIndicator>
  </CheckboxRoot>
</template>
```

- [ ] **Step 5: `radio-group/index.ts` + `radio-group/RadioGroup.vue` + `radio-group/RadioGroupItem.vue`**

`radio-group/index.ts`:
```ts
export { default as RadioGroup } from './RadioGroup.vue'
export { default as RadioGroupItem } from './RadioGroupItem.vue'
```

`radio-group/RadioGroup.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { RadioGroupRoot, type RadioGroupRootEmits, type RadioGroupRootProps } from 'reka-ui'
import { cn } from '../../../lib/utils'

const props = withDefaults(defineProps<RadioGroupRootProps & { class?: HTMLAttributes['class'] }>(), {})
const emits = defineEmits<RadioGroupRootEmits>()
</script>

<template>
  <RadioGroupRoot
    v-bind="props"
    :class="cn('grid gap-2', props.class)"
    @update:model-value="emits('update:modelValue', $event)"
  >
    <slot />
  </RadioGroupRoot>
</template>
```

`radio-group/RadioGroupItem.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { Circle } from 'lucide-vue-next'
import { RadioGroupIndicator, RadioGroupItem, type RadioGroupItemProps } from 'reka-ui'
import { cn } from '../../../lib/utils'

const props = defineProps<RadioGroupItemProps & { class?: HTMLAttributes['class'] }>()
</script>

<template>
  <RadioGroupItem
    v-bind="props"
    :class="cn('border-input text-primary shadow-xs focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] outline-none disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:border-primary size-4 shrink-0 rounded-full border', props.class)"
  >
    <RadioGroupIndicator class="relative flex items-center justify-center">
      <Circle class="fill-primary absolute top-1/2 left-1/2 size-2 -translate-x-1/2 -translate-y-1/2" />
    </RadioGroupIndicator>
  </RadioGroupItem>
</template>
```

- [ ] **Step 6: `table/index.ts` + Table components**

`table/index.ts`:
```ts
export { default as Table } from './Table.vue'
export { default as TableBody } from './TableBody.vue'
export { default as TableCaption } from './TableCaption.vue'
export { default as TableCell } from './TableCell.vue'
export { default as TableEmpty } from './TableEmpty.vue'
export { default as TableHead } from './TableHead.vue'
export { default as TableHeader } from './TableHeader.vue'
export { default as TableRow } from './TableRow.vue'
```

`table/Table.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '../../../lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <div class="relative w-full overflow-auto">
    <table :class="cn('w-full caption-bottom text-sm', props.class)">
      <slot />
    </table>
  </div>
</template>
```

`table/TableHeader.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '../../../lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <thead :class="cn('[&_tr]:border-b', props.class)">
    <slot />
  </thead>
</template>
```

`table/TableBody.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '../../../lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <tbody :class="cn('[&_tr:last-child]:border-0', props.class)">
    <slot />
  </tbody>
</template>
```

`table/TableRow.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '../../../lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <tr :class="cn('border-b transition-colors hover:bg-muted/50 data-[state=selected]:bg-muted', props.class)">
    <slot />
  </tr>
</template>
```

`table/TableHead.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '../../../lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <th :class="cn('text-muted-foreground h-10 px-2 text-left align-middle font-medium whitespace-nowrap [&:has([role=checkbox])]:pr-0', props.class)">
    <slot />
  </th>
</template>
```

`table/TableCell.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '../../../lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <td :class="cn('p-2 align-middle whitespace-nowrap [&:has([role=checkbox])]:pr-0', props.class)">
    <slot />
  </td>
</template>
```

`table/TableCaption.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '../../../lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <caption :class="cn('text-muted-foreground mt-4 text-sm', props.class)">
    <slot />
  </caption>
</template>
```

`table/TableEmpty.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '../../../lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <tr :class="props.class">
    <td :colspan="props.colspan ?? 1" class="text-muted-foreground p-6 text-center">
      <slot />
    </td>
  </tr>
</template>
```

- [ ] **Step 7: Verify build**

Run: `cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend && npm run build`
Expected: build succeeds.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/components/ui/input frontend/src/components/ui/label frontend/src/components/ui/textarea frontend/src/components/ui/checkbox frontend/src/components/ui/radio-group frontend/src/components/ui/table
git commit -m "feat: add shadcn ui primitives (input/label/textarea/checkbox/radio-group/table)"
```

---

### Task 5: UI primitives — alert, dialog, alert-dialog

**Files (created under `frontend/src/components/ui/`):**
- `alert/index.ts`, `alert/Alert.vue`
- `dialog/index.ts`, `dialog/Dialog.vue` (simplified wrapper)
- `alert-dialog/index.ts`, `alert-dialog/AlertDialog.vue` (simplified wrapper)

- [ ] **Step 1: `alert/index.ts` + `alert/Alert.vue`**

`alert/index.ts`:
```ts
export { default as Alert } from './Alert.vue'
```

`alert/Alert.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '../../../lib/utils'

const props = withDefaults(defineProps<{ class?: HTMLAttributes['class']; variant?: 'default' | 'destructive' | 'info' | 'warning' | 'success' }>(), { variant: 'default' })

const variantClass = {
  default: 'bg-card text-card-foreground',
  destructive: 'border-destructive/50 text-destructive bg-destructive/5',
  info: 'border-info/50 text-foreground bg-info/5',
  warning: 'border-warning/50 text-foreground bg-warning/5',
  success: 'border-success/50 text-foreground bg-success/5',
}
</script>

<template>
  <div :class="cn('relative w-full rounded-lg border px-4 py-3 text-sm', variantClass[props.variant], props.class)" role="alert">
    <slot />
  </div>
</template>
```

- [ ] **Step 2: `dialog/index.ts` + `dialog/Dialog.vue`**

`dialog/index.ts`:
```ts
export { default as Dialog } from './Dialog.vue'
```

`dialog/Dialog.vue`:
```vue
<script setup lang="ts">
import { X } from 'lucide-vue-next'
import { DialogContent, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { cn } from '../../../lib/utils'

const props = withDefaults(defineProps<{ title?: string; class?: string; maxWidth?: string }>(), {
  title: '',
  class: '',
  maxWidth: 'lg',
})

const open = defineModel<boolean>('open', { default: false })
</script>

<template>
  <DialogRoot v-model:open="open">
    <slot name="trigger" />
    <DialogPortal>
      <DialogOverlay class="bg-black/80 fixed inset-0 z-50" />
      <DialogContent
        :class="cn('bg-background data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 fixed top-1/2 left-1/2 z-50 grid w-full max-w-[calc(100%-2rem)] -translate-x-1/2 -translate-y-1/2 gap-4 rounded-lg border p-6 shadow-lg duration-200', maxWidth === 'md' ? 'sm:max-w-md' : 'sm:max-w-lg', props.class)"
      >
        <div class="flex items-start justify-between gap-4">
          <DialogTitle v-if="title" class="text-lg font-semibold leading-none">{{ title }}</DialogTitle>
          <span v-else />
          <button
            type="button"
            class="text-muted-foreground hover:text-foreground rounded-sm opacity-70 transition-opacity hover:opacity-100 focus-visible:outline-none"
            @click="open = false"
          >
            <X class="size-4" />
          </button>
        </div>
        <slot />
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
```

- [ ] **Step 3: `alert-dialog/index.ts` + `alert-dialog/AlertDialog.vue`**

`alert-dialog/index.ts`:
```ts
export { default as AlertDialog } from './AlertDialog.vue'
```

`alert-dialog/AlertDialog.vue`:
```vue
<script setup lang="ts">
import {
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogOverlay,
  AlertDialogPortal,
  AlertDialogRoot,
  AlertDialogTitle,
} from 'reka-ui'

const props = withDefaults(defineProps<{ title?: string; description?: string; maxWidth?: string }>(), {
  title: '',
  description: '',
  maxWidth: 'md',
})

const open = defineModel<boolean>('open', { default: false })
</script>

<template>
  <AlertDialogRoot v-model:open="open">
    <slot name="trigger" />
    <AlertDialogPortal>
      <AlertDialogOverlay class="bg-black/80 fixed inset-0 z-50" />
      <AlertDialogContent class="bg-background data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 fixed top-1/2 left-1/2 z-50 grid w-full max-w-[calc(100%-2rem)] -translate-x-1/2 -translate-y-1/2 gap-4 rounded-lg border p-6 shadow-lg duration-200 sm:max-w-md">
        <AlertDialogHeader>
          <AlertDialogTitle v-if="title" class="text-lg font-semibold">{{ title }}</AlertDialogTitle>
          <AlertDialogDescription v-if="description" class="text-muted-foreground text-sm" v-html="description" />
        </AlertDialogHeader>
        <AlertDialogFooter class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <slot name="footer" />
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialogPortal>
  </AlertDialogRoot>
</template>
```

- [ ] **Step 4: Verify build**

Run: `cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend && npm run build`
Expected: build succeeds.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/ui/alert frontend/src/components/ui/dialog frontend/src/components/ui/alert-dialog
git commit -m "feat: add shadcn ui primitives (alert/dialog/alert-dialog)"
```

---

### Task 6: UI radix set — dropdown-menu, tabs, select, sonner + app wrappers

**Files (created under `frontend/src/components/ui/`):**
- `dropdown-menu/index.ts` + `dropdown-menu/DropdownMenu.vue` (simplified: items-driven)
- `tabs/index.ts`, `tabs/Tabs.vue`, `tabs/TabsList.vue`, `tabs/TabsTrigger.vue`, `tabs/TabsContent.vue`
- `select/index.ts`, `select/Select.vue` (only used by `AppSelect`)
- `sonner/index.ts`, `sonner/Sonner.vue`

**Files (created under `frontend/src/components/`):**
- `AppSelect.vue`
- `FileUpload.vue`

- [ ] **Step 1: `dropdown-menu/index.ts` + `dropdown-menu/DropdownMenu.vue`**

`dropdown-menu/index.ts`:
```ts
export { default as DropdownMenu } from './DropdownMenu.vue'
```

`dropdown-menu/DropdownMenu.vue` (items-driven; `select` emits the item index):
```vue
<script setup lang="ts">
import { Check } from 'lucide-vue-next'
import { DropdownMenuContent, DropdownMenuItem, DropdownMenuPortal, DropdownMenuRoot } from 'reka-ui'
import { cn } from '../../../lib/utils'

export interface DropdownItem {
  label: string
  value?: string
  danger?: boolean
  divider?: boolean
}

const props = withDefaults(defineProps<{
  items: DropdownItem[]
  selected?: string
  class?: string
}>(), { items: () => [], selected: '' })

const emit = defineEmits<{ select: [index: number] }>()
</script>

<template>
  <DropdownMenuRoot>
    <slot name="trigger" />
    <DropdownMenuPortal>
      <DropdownMenuContent
        :class="cn('bg-popover text-popover-foreground z-50 min-w-32 overflow-hidden rounded-md border p-1 shadow-md', props.class)"
        :side-offset="4"
        align="end"
      >
        <template v-for="(item, i) in props.items" :key="i">
          <div v-if="item.divider" class="bg-border my-1 h-px" />
          <DropdownMenuItem
            v-else
            :class="cn(
              'focus:bg-accent focus:text-accent-foreground data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground cursor-pointer rounded-sm px-2 py-1.5 text-sm outline-none',
              item.danger && 'text-destructive data-[highlighted]:bg-destructive data-[highlighted]:text-white',
            )"
            @select="emit('select', i)"
          >
            <span class="flex w-full items-center justify-between gap-2">
              <span>{{ item.label }}</span>
              <Check v-if="item.value && item.value === props.selected" class="size-4" />
            </span>
          </DropdownMenuItem>
        </template>
      </DropdownMenuContent>
    </DropdownMenuPortal>
  </DropdownMenuRoot>
</template>
```

- [ ] **Step 2: `tabs/index.ts` + tabs components**

`tabs/index.ts`:
```ts
export { default as Tabs } from './Tabs.vue'
export { default as TabsList } from './TabsList.vue'
export { default as TabsTrigger } from './TabsTrigger.vue'
export { default as TabsContent } from './TabsContent.vue'
```

`tabs/Tabs.vue`:
```vue
<script setup lang="ts">
import { TabsRoot, type TabsRootEmits, type TabsRootProps } from 'reka-ui'

const props = withDefaults(defineProps<TabsRootProps>(), {})
const emits = defineEmits<TabsRootEmits>()
</script>

<template>
  <TabsRoot v-bind="props" @update:model-value="emits('update:modelValue', $event)">
    <slot />
  </TabsRoot>
</template>
```

`tabs/TabsList.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { TabsList, type TabsListProps } from 'reka-ui'
import { cn } from '../../../lib/utils'

const props = defineProps<TabsListProps & { class?: HTMLAttributes['class'] }>()
</script>

<template>
  <TabsList
    v-bind="props"
    :class="cn('text-muted-foreground bg-muted inline-flex h-9 items-center justify-center rounded-lg p-1', props.class)"
  >
    <slot />
  </TabsList>
</template>
```

`tabs/TabsTrigger.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { TabsTrigger, type TabsTriggerProps } from 'reka-ui'
import { cn } from '../../../lib/utils'

const props = defineProps<TabsTriggerProps & { class?: HTMLAttributes['class'] }>()
</script>

<template>
  <TabsTrigger
    v-bind="props"
    :class="cn('data-[state=active]:bg-background data-[state=active]:text-foreground focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] text-muted-foreground inline-flex items-center justify-center gap-1.5 rounded-md px-3 py-1 text-sm font-medium whitespace-nowrap transition-[color,box-shadow] focus-visible:outline-none disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 shrink-0 [&_svg]:shrink-0 data-[state=active]:shadow-sm', props.class)"
  >
    <slot />
  </TabsTrigger>
</template>
```

`tabs/TabsContent.vue`:
```vue
<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { TabsContent, type TabsContentProps } from 'reka-ui'
import { cn } from '../../../lib/utils'

const props = defineProps<TabsContentProps & { class?: HTMLAttributes['class'] }>()
</script>

<template>
  <TabsContent
    v-bind="props"
    :class="cn('focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] flex-1 outline-none', props.class)"
  >
    <slot />
  </TabsContent>
</template>
```

- [ ] **Step 3: `select/index.ts` + `select/Select.vue`** (canonical trigger/content; consumed by `AppSelect`)

`select/index.ts`:
```ts
export { default as Select } from './Select.vue'
```

`select/Select.vue`:
```vue
<script setup lang="ts">
import { SelectContent, SelectPortal, SelectRoot, SelectTrigger, SelectValue } from 'reka-ui'
import { cn } from '../../../lib/utils'

export interface SelectOption {
  title: string
  value: string | number
}

const props = withDefaults(defineProps<{
  items: SelectOption[]
  modelValue?: string | number | null
  placeholder?: string
  class?: string
  disabled?: boolean
}>(), { placeholder: 'Select…' })

const emit = defineEmits<{ 'update:modelValue': [value: string | number | null] }>()

function toStr(v: string | number | null | undefined): string | undefined {
  if (v === null || v === undefined) return undefined
  return String(v)
}
function fromStr(s: string): string | number | null {
  return props.items.find((i) => String(i.value) === s)?.value ?? null
}
</script>

<template>
  <SelectRoot
    :model-value="toStr(props.modelValue)"
    :disabled="props.disabled"
    @update:model-value="(s: string) => emit('update:modelValue', fromStr(s))"
  >
    <SelectTrigger
      :class="cn('border-input bg-transparent shadow-xs data-[placeholder]:text-muted-foreground flex h-10 w-full items-center justify-between gap-2 rounded-md border px-3 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50', props.class)"
    >
      <SelectValue :placeholder="props.placeholder" />
    </SelectTrigger>
    <SelectPortal>
      <SelectContent class="bg-popover text-popover-foreground data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 relative z-50 max-h-96 min-w-32 origin-(--reka-select-content-transform-origin) overflow-y-auto rounded-md border shadow-md">
        <div class="p-1">
          <template v-for="item in props.items" :key="String(item.value)">
            <div
              class="data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground cursor-pointer select-none rounded-sm px-2 py-1.5 text-sm outline-none"
              data-reka-select-item=""
              :data-value="String(item.value)"
              @click="emit('update:modelValue', fromStr(String(item.value)))"
            >
              {{ item.title }}
            </div>
          </template>
        </div>
      </SelectContent>
    </SelectPortal>
  </SelectRoot>
</template>
```

> Note: `Select.vue` above renders lightweight custom option rows inside a reka-ui `SelectContent` portal. If focus/keyboard behavior is off, replace the `data-reka-select-item` rows with `SelectItem` components imported from `reka-ui` (canonical shadcn-vue) — see Task 6 step 5 fallback. Either way the external `SelectOption` interface and `modelValue`/`@update:model-value` contract stay the same.

- [ ] **Step 4: `sonner/index.ts` + `sonner/Sonner.vue`**

`sonner/index.ts`:
```ts
export { default as Sonner } from './Sonner.vue'
```

`sonner/Sonner.vue`:
```vue
<script setup lang="ts">
import { Toaster as Sonner, type ToasterProps } from 'vue-sonner'

const props = defineProps<ToasterProps>()
</script>

<template>
  <Sonner
    :position="props.position ?? 'bottom-center'"
    :rich-colors="props.richColors ?? true"
    :theme="props.theme ?? 'system'"
    :toast-options="{ class: '!font-sans' }"
  />
</template>
```

- [ ] **Step 5: `AppSelect.vue`** (the canonical wrapper actually used by views; takes over from `select/Select.vue`)

```vue
<script setup lang="ts">
import { Check, ChevronDown } from 'lucide-vue-next'
import { SelectContent, SelectItem, SelectPortal, SelectRoot, SelectTrigger, SelectValue, SelectViewport } from 'reka-ui'
import { cn } from '../lib/utils'

export interface SelectOption {
  title: string
  value: string | number
}

const props = withDefaults(defineProps<{
  items: SelectOption[]
  modelValue?: string | number | null
  placeholder?: string
  class?: string
  disabled?: boolean
}>(), { placeholder: 'Select…' })

const emit = defineEmits<{ 'update:modelValue': [value: string | number | null] }>()

function toStr(v: string | number | null | undefined): string | undefined {
  if (v === null || v === undefined) return undefined
  return String(v)
}
function fromStr(s: string): string | number | null {
  return props.items.find((i) => String(i.value) === s)?.value ?? null
}
</script>

<template>
  <SelectRoot
    :model-value="toStr(props.modelValue)"
    :disabled="props.disabled"
    @update:model-value="(s: string) => emit('update:modelValue', fromStr(s))"
  >
    <SelectTrigger
      :class="cn('border-input bg-transparent shadow-xs data-[placeholder]:text-muted-foreground flex h-10 w-full items-center justify-between gap-2 rounded-md border px-3 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50', props.class)"
    >
      <SelectValue :placeholder="props.placeholder" />
      <ChevronDown class="size-4 shrink-0 opacity-50" />
    </SelectTrigger>
    <SelectPortal>
      <SelectContent class="bg-popover text-popover-foreground z-50 max-h-96 min-w-32 overflow-y-auto rounded-md border shadow-md">
        <SelectViewport class="p-1">
          <SelectItem
            v-for="item in props.items"
            :key="String(item.value)"
            :value="String(item.value)"
            class="data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground cursor-pointer rounded-sm px-2 py-1.5 text-sm outline-none"
          >
            <span class="flex items-center justify-between gap-2">
              <span>{{ item.title }}</span>
              <Check class="size-4 opacity-0 data-[state=checked]:opacity-100" />
            </span>
          </SelectItem>
        </SelectViewport>
      </SelectContent>
    </SelectPortal>
  </SelectRoot>
</template>
```

- [ ] **Step 6: `FileUpload.vue`**

```vue
<script setup lang="ts">
import { computed, ref } from 'vue'
import { Upload } from 'lucide-vue-next'
import { cn } from '../lib/utils'

const props = withDefaults(defineProps<{
  label?: string
  accept?: string
  multiple?: boolean
  dropZone?: boolean
  disabled?: boolean
}>(), { label: '', accept: '', multiple: false, dropZone: false })

const inputEl = ref<HTMLInputElement | null>(null)

const model = defineModel<File | File[] | null>({ default: null })
const emit = defineEmits<{ change: [File | File[]] }>()

const fileName = computed(() => {
  if (Array.isArray(model.value)) return model.value.map((f) => f.name).join(', ')
  return model.value?.name ?? ''
})

function onPick(e: Event) {
  const files = (e.target as HTMLInputElement).files
  if (!files?.length) return
  const value = props.multiple ? Array.from(files) : files[0]
  model.value = value
  emit('change', value)
  if (inputEl.value) inputEl.value.value = ''
}

function open() {
  if (!props.disabled) inputEl.value?.click()
}
</script>

<template>
  <div>
    <input ref="inputEl" type="file" class="hidden" :accept="props.accept" :multiple="props.multiple" @change="onPick" />
    <button
      type="button"
      :disabled="props.disabled"
      @click="open"
      :class="cn(
        'border-input inline-flex items-center justify-center gap-2 rounded-md border text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50',
        props.dropZone ? 'border-dashed h-28 w-full flex-col px-4 py-6 text-center text-muted-foreground' : 'h-9 px-4 py-2',
      )"
    >
      <Upload class="size-4" />
      <span class="max-w-[220px] truncate">{{ fileName || props.label || 'Choose file' }}</span>
    </button>
  </div>
</template>
```

- [ ] **Step 7: Verify build**

Run: `cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend && npm run build`
Expected: build succeeds.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/components/ui/dropdown-menu frontend/src/components/ui/tabs frontend/src/components/ui/select frontend/src/components/ui/sonner frontend/src/components/AppSelect.vue frontend/src/components/FileUpload.vue
git commit -m "feat: add dropdown-menu/tabs/select/sonner + AppSelect & FileUpload wrappers"
```

---

### Task 7: Rewrite `AppShell.vue`

**Files:**
- Rewrite: `frontend/src/components/AppShell.vue`

- [ ] **Step 1: Rewrite the file**

```vue
<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import {
  Globe, LogIn, LogOut, Moon, LockKeyhole, Sun, SunMoon, ChevronDown, UserRound,
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
  return result
})

const activeTab = computed(() => {
  const p = route.path
  const hit = tabs.value.find((tab) => p === tab.match || p.startsWith(tab.match + '/'))
  return hit?.to ?? ''
})

const sonnerTheme = computed<ThemeChoice>(() => choice.value)

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
        <template v-if="currentDownloadApp">
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
              <Button variant="ghost" size="icon">
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
            @select="(i: number) => { if (!isSuperAdmin && i === 2) openPwDialog(); if (i === (isSuperAdmin ? 1 : 3)) logoutDialog = true }"
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
          <Button :loading="pwLoading" :disabled="!oldPassword || !newPassword" @click="submitPassword">{{ t('common.save') }}</Button>
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
```

> Note: `toast` is imported for potential use; if unused, drop the import to satisfy `noUnusedLocals`. The `Button` component does not render a `loading` prop — add `:disabled="pwLoading"` and swap label to a spinner manually if needed; simplest is `:disabled="pwLoading"` plus text. Remove the `loading` attribute usage below.

- [ ] **Step 2: Verify build**

Run: `cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend && npm run build`
Expected: build succeeds. Fix any unused-import or prop-type errors (e.g. remove `:loading="pwLoading"` — our `Button` has no `loading` prop).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/AppShell.vue
git commit -m "feat: rewrite AppShell with shadcn header, menus, dialogs"
```

---

### Task 8: Rewrite `Home.vue`

**Files:**
- Rewrite: `frontend/src/views/Home.vue`

- [ ] **Step 1: Rewrite the file**

```vue
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '../api/client'
import { useI18n } from '../composables/useI18n'
import { Badge } from '../components/ui/badge'
import { Card, CardContent } from '../components/ui/card'
import { Alert } from '../components/ui/alert'
import { Avatar } from '../components/ui/avatar'
import type { AppItem } from '../api/types'

const { t } = useI18n()
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

function accessVariant(mode: string): 'success' | 'warning' | 'secondary' {
  if (mode === 'public') return 'success'
  if (mode === 'password' || mode === 'expiry') return 'warning'
  return 'secondary'
}
</script>

<template>
  <div class="mx-auto max-w-6xl px-4 py-8 sm:px-6">
    <h1 class="text-2xl font-semibold tracking-tight mb-6">{{ t('home.title') }}</h1>

    <Alert v-if="error" variant="destructive" class="mb-4">{{ error }}</Alert>

    <div class="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
      <Card>
        <CardContent class="!py-4">
          <div class="text-muted-foreground text-xs uppercase tracking-wider">{{ t('home.stat.apps') }}</div>
          <div class="text-3xl font-semibold">{{ apps.length }}</div>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="!py-4">
          <div class="text-muted-foreground text-xs uppercase tracking-wider">{{ t('home.stat.versions') }}</div>
          <div class="text-3xl font-semibold">{{ totalVersions }}</div>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="!py-4">
          <div class="text-muted-foreground text-xs uppercase tracking-wider">{{ t('home.stat.downloads') }}</div>
          <div class="text-3xl font-semibold">{{ totalDownloads }}</div>
        </CardContent>
      </Card>
    </div>

    <div v-if="apps.length" class="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3">
      <RouterLink
        v-for="a in apps"
        :key="a.id"
        :to="`/app/${encodeURIComponent(a.name)}`"
        class="group rounded-xl border bg-card text-card-foreground p-5 shadow-sm transition-colors hover:bg-accent/50"
      >
        <div class="mb-3 flex items-center gap-3">
          <Avatar :src="a.icon" :fallback="a.name.charAt(0).toUpperCase()" class="size-10" />
          <span class="truncate font-semibold">{{ a.name }}</span>
        </div>
        <div v-if="a.latest_version" class="text-muted-foreground mb-1 text-sm">
          <code>{{ a.latest_version.version_name }}</code>
          · {{ fmtSize(a.latest_version.file_size) }}
        </div>
        <Badge :variant="accessVariant(a.access_mode)">
          {{ t('access.' + a.access_mode) }}
        </Badge>
        <p v-if="a.description" class="text-muted-foreground mt-2 text-sm">{{ a.description }}</p>
      </RouterLink>
    </div>

    <Card v-else-if="!error" class="text-center">
      <CardContent class="py-12 text-muted-foreground">{{ t('home.empty') }}</CardContent>
    </Card>
  </div>
</template>
```

- [ ] **Step 2: Verify build**

Run: `cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend && npm run build`
Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/Home.vue
git commit -m "feat: rewrite Home with shadcn cards"
```

---

### Task 9: Rewrite `Login.vue`

**Files:**
- Rewrite: `frontend/src/views/Login.vue`

- [ ] **Step 1: Rewrite the file**

```vue
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
```

- [ ] **Step 2: Verify build**

Run: `cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend && npm run build`
Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/Login.vue
git commit -m "feat: rewrite Login with shadcn card"
```

---

### Task 10: Rewrite `VersionPanel.vue` + `AppDetail.vue`

**Files:**
- Rewrite: `frontend/src/components/VersionPanel.vue`
- Rewrite: `frontend/src/views/AppDetail.vue`

- [ ] **Step 1: Rewrite `VersionPanel.vue`**

```vue
<script setup lang="ts">
import { computed, ref } from 'vue'
import { Download } from 'lucide-vue-next'
import { useI18n } from '../composables/useI18n'
import { detectUA } from '../utils/ua'
import { fmtDate } from '../utils/format'
import { Badge } from './ui/badge'
import { Avatar } from './ui/avatar'
import { Button } from './ui/button'
import type { Architecture, Version } from '../api/types'

const props = defineProps<{
  version: Version
  fallbackName: string
  fallbackIcon: string
  accessMode?: string
  expiresAt?: string | null
  noDownload?: boolean
}>()

const emit = defineEmits<{ download: [v: Version] }>()

const { t } = useI18n()
const detected = detectUA()

// Clickable architecture selection. Architecture is metadata — one file per
// version — so switching only changes the highlighted chip.
const archChoice = ref<Architecture | null>(null)
const archList = computed(() =>
  (props.version.arch || '').split(',').filter(Boolean) as Architecture[]
)
const currentArch = computed(() => archChoice.value ?? defaultArch())
function defaultArch(): Architecture {
  const list = archList.value
  if (!list.length) return 'universal'
  if (detected.arch && list.includes(detected.arch)) return detected.arch
  return list[0]
}

const icon = computed(() => props.version.icon_url || props.fallbackIcon)
const appName = computed(() => props.version.app_name || props.fallbackName)

function releaseVariant(rt: string): 'default' | 'info' | 'warning' {
  if (rt === 'beta') return 'info'
  if (rt === 'canary') return 'warning'
  return 'default'
}

function isExpired(): boolean {
  return props.accessMode === 'expiry' && !!props.expiresAt && new Date(props.expiresAt) < new Date()
}

function fmtSize(n: number): string {
  if (n > 1024 * 1024 * 1024) return (n / 1024 / 1024 / 1024).toFixed(2) + ' GB'
  if (n > 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MB'
  if (n > 1024) return (n / 1024).toFixed(1) + ' KB'
  return n + ' B'
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <div class="flex items-center gap-3">
      <Avatar :src="icon" :fallback="appName.charAt(0).toUpperCase()" class="size-14" />
      <Badge :variant="releaseVariant(version.release_type)">
        {{ t('release.' + version.release_type) }}
      </Badge>
    </div>

    <div>
      <div class="font-semibold">{{ appName }}</div>
      <div v-if="version.package_name" class="text-muted-foreground text-xs">
        <code>{{ version.package_name }}</code>
      </div>
    </div>

    <div class="flex items-center gap-2">
      <Badge v-if="version.platform" variant="outline" class="text-xs">
        {{ t('platform.' + version.platform) }}
      </Badge>
      <span class="font-medium">
        {{ version.version_name }}
        <span v-if="version.version_code" class="text-muted-foreground text-sm">
          ({{ version.version_code }})
        </span>
      </span>
    </div>

    <div class="flex items-center gap-1.5">
      <span class="text-muted-foreground text-xs">{{ t('detail.arch') }}:</span>
      <template v-if="archList.length">
        <Badge
          v-for="a in archList"
          :key="a"
          variant="outline"
          :class="currentArch === a ? 'border-primary text-primary' : 'cursor-pointer hover:bg-accent'"
          @click="archChoice = a"
        >
          {{ t('arch.' + a) }}
        </Badge>
      </template>
      <span v-else class="text-muted-foreground text-xs">—</span>
    </div>

    <div class="text-muted-foreground text-sm">
      {{ fmtSize(version.file_size) }} · {{ fmtDate(version.created_at) }}
    </div>

    <div v-if="version.changelog">
      <div class="text-sm font-medium">{{ t('detail.changelog') }}:</div>
      <div class="text-sm whitespace-pre-wrap">{{ version.changelog }}</div>
    </div>

    <div v-if="!noDownload">
      <Button size="lg" :disabled="isExpired()" @click="emit('download', version)">
        <Download class="size-4" />
        {{ t('detail.download') }}
      </Button>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Rewrite `AppDetail.vue`**

```vue
<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Download } from 'lucide-vue-next'
import { api } from '../api/client'
import { useI18n } from '../composables/useI18n'
import { loadDownloadApp } from '../composables/useDownloadApp'
import { detectUA } from '../utils/ua'
import { PLATFORMS } from '../constants/platform'
import { Alert } from '../components/ui/alert'
import { Avatar } from '../components/ui/avatar'
import { Button } from '../components/ui/button'
import { Card, CardContent } from '../components/ui/card'
import { Dialog } from '../components/ui/dialog'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'
import VersionPanel from '../components/VersionPanel.vue'
import type { AppDetail, Platform, Version } from '../api/types'

const route = useRoute()
const { t } = useI18n()
const detected = detectUA()

const data = ref<AppDetail | null>(null)
const error = ref('')
const passwordPrompt = ref<{ versionId: number; password: string } | null>(null)
const dialogOpen = ref(false)

// Newest first by creation time.
const versions = computed(() =>
  [...(data.value?.versions ?? [])].sort(
    (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  )
)

// Latest version per platform (first encounter wins in a desc-sorted list).
const byPlatform = computed(() => {
  const map = new Map<Platform, Version>()
  for (const v of versions.value) {
    if (v.platform && !map.has(v.platform)) map.set(v.platform, v)
  }
  return map
})

// Platform cards shown on desktop / unknown UA, in canonical platform order.
const platformList = computed(() => {
  const result: { platform: Platform; version: Version }[] = []
  for (const p of PLATFORMS) {
    const v = byPlatform.value.get(p)
    if (v) result.push({ platform: p, version: v })
  }
  return result
})

const mobileVersion = computed(() => {
  if (detected.isDesktop || !detected.platform) return null
  return byPlatform.value.get(detected.platform) ?? versions.value[0] ?? null
})

function openPasswordPrompt(versionId: number) {
  passwordPrompt.value = { versionId, password: '' }
  dialogOpen.value = true
}
function closePasswordPrompt() {
  dialogOpen.value = false
  passwordPrompt.value = null
}

onMounted(load)
watch(() => route.params.name, (name) => { if (name) load() })

async function load() {
  try {
    data.value = await loadDownloadApp(String(route.params.name))
  } catch (e) {
    error.value = (e as Error).message
  }
}

const appAccess = computed(() => data.value?.app.access_mode ?? 'public')
const appExpiresAt = computed(() => data.value?.app.expires_at ?? null)

function isExpired(): boolean {
  return appAccess.value === 'expiry' && !!appExpiresAt.value && new Date(appExpiresAt.value) < new Date()
}

async function download(v: Version | null) {
  if (!v) return
  if (appAccess.value === 'password') {
    openPasswordPrompt(v.id)
    return
  }
  await doDownload(v.id, undefined)
}

async function submitPassword() {
  if (!passwordPrompt.value) return
  const pwd = passwordPrompt.value.password
  const vid = passwordPrompt.value.versionId
  closePasswordPrompt()
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
  <div class="mx-auto max-w-6xl px-4 py-8 sm:px-6">
    <Alert v-if="error" variant="destructive" class="mb-4">
      {{ error }}
    </Alert>

    <template v-if="data">
      <div class="mb-2 flex items-center gap-3">
        <Avatar :src="data.app.icon" :fallback="data.app.name.charAt(0).toUpperCase()" class="size-10" />
        <h1 class="text-2xl font-semibold tracking-tight">{{ data.app.name }}</h1>
      </div>
      <p v-if="data.app.description" class="text-muted-foreground mb-6 text-sm">{{ data.app.description }}</p>

      <!-- Mobile: detected platform's latest version, no card wrapper -->
      <div v-if="mobileVersion" class="max-w-[560px] pb-24">
        <VersionPanel
          :version="mobileVersion"
          :fallback-name="data.app.name"
          :fallback-icon="data.app.icon"
          :access-mode="data.app.access_mode"
          :expires-at="data.app.expires_at"
          :no-download="true"
          @download="download"
        />
      </div>

      <!-- Desktop / unknown UA: one card per platform -->
      <div v-else-if="platformList.length" class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <Card v-for="{ platform, version } in platformList" :key="platform" class="p-5">
          <CardContent class="!p-0">
            <VersionPanel
              :version="version"
              :fallback-name="data.app.name"
              :fallback-icon="data.app.icon"
              :access-mode="data.app.access_mode"
              :expires-at="data.app.expires_at"
              @download="download"
            />
          </CardContent>
        </Card>
      </div>

      <Card v-else-if="!error" class="text-center">
        <CardContent class="py-12 text-muted-foreground">{{ t('detail.empty') }}</CardContent>
      </Card>
    </template>

    <!-- Floating download button: 80% width, pinned to the viewport bottom -->
    <div v-if="mobileVersion && !error" class="fixed inset-x-0 bottom-0 z-50 flex justify-center px-4 pb-[calc(0.75rem+env(safe-area-inset-bottom))] pt-3 bg-gradient-to-t from-background to-transparent">
      <Button size="lg" class="w-4/5" :disabled="isExpired()" @click="download(mobileVersion)">
        <Download class="size-4" />
        {{ t('detail.download') }}
      </Button>
    </div>

    <Dialog v-model:open="dialogOpen" :title="t('detail.passwordTitle')" max-width="md">
      <div class="grid gap-4">
        <p class="text-muted-foreground text-sm">{{ t('detail.passwordBody') }}</p>
        <div class="grid gap-2">
          <Label>{{ t('common.password') }}</Label>
          <Input
            v-if="passwordPrompt"
            v-model="passwordPrompt.password"
            type="password"
            autofocus
            @keyup.enter="submitPassword"
          />
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="closePasswordPrompt">{{ t('common.cancel') }}</Button>
          <Button @click="submitPassword">{{ t('detail.passwordContinue') }}</Button>
        </div>
      </div>
    </Dialog>
  </div>
</template>
```

- [ ] **Step 3: Verify build**

Run: `cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend && npm run build`
Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/VersionPanel.vue frontend/src/views/AppDetail.vue
git commit -m "feat: rewrite download page + version panel with shadcn"
```

---

### Task 11: Rewrite `Admin.vue` (apps grid)

**Files:**
- Rewrite: `frontend/src/views/admin/Admin.vue`

- [ ] **Step 1: Rewrite the file**

```vue
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { ImageIcon, Plus } from 'lucide-vue-next'
import { api } from '../../api/client'
import { useI18n } from '../../composables/useI18n'
import { fmtDate } from '../../utils/format'
import { Button } from '../../components/ui/button'
import { Avatar } from '../../components/ui/avatar'
import { Card, CardContent, CardFooter } from '../../components/ui/card'
import { Alert } from '../../components/ui/alert'
import { Dialog } from '../../components/ui/dialog'
import { AlertDialog } from '../../components/ui/alert-dialog'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Separator } from '../../components/ui/separator'
import FileUpload from '../../components/FileUpload.vue'
import type { AppItem } from '../../api/types'

const router = useRouter()
const { t } = useI18n()
const apps = ref<AppItem[]>([])
const error = ref('')

const newName = ref('')
const createIcon = ref<File | null>(null)
const createIconPreview = ref('')
const createError = ref('')
const creating = ref(false)
const createDialogOpen = ref(false)

const editTarget = ref<AppItem | null>(null)
const editDialogOpen = ref(false)
const editName = ref('')
const editIcon = ref<File | null>(null)
const editIconPreview = ref('')
const editError = ref('')
const editing = ref(false)

const deleteTarget = ref<AppItem | null>(null)
const deleteDialogOpen = ref(false)

onMounted(load)
async function load() {
  try {
    apps.value = await api.adminApps()
    error.value = ''
  } catch (e) {
    error.value = (e as Error).message
  }
}

function openCreate() {
  newName.value = ''
  createIcon.value = null
  createIconPreview.value = ''
  createError.value = ''
  createDialogOpen.value = true
}

function onCreateIcon(file: File | File[]) {
  const f = Array.isArray(file) ? file[0] : file
  createIcon.value = f
  createIconPreview.value = URL.createObjectURL(f)
}

async function confirmCreate() {
  if (!newName.value.trim()) {
    createError.value = t('admin.nameRequired')
    return
  }
  creating.value = true
  createError.value = ''
  try {
    const app = await api.createApp({ name: newName.value.trim() })
    if (createIcon.value) {
      try {
        await api.uploadAppIcon(app.id, createIcon.value)
      } catch (e) {
        toast((e as Error).message)
      }
    }
    createDialogOpen.value = false
    router.push(`/admin/app/${app.id}`)
  } catch (e) {
    createError.value = (e as Error).message
  } finally {
    creating.value = false
  }
}

function openEdit(a: AppItem) {
  editTarget.value = a
  editName.value = a.name
  editIcon.value = null
  editIconPreview.value = a.icon || ''
  editError.value = ''
  editDialogOpen.value = true
}

function onEditIcon(file: File | File[]) {
  const f = Array.isArray(file) ? file[0] : file
  editIcon.value = f
  editIconPreview.value = URL.createObjectURL(f)
}

async function confirmEdit() {
  if (!editTarget.value) return
  if (!editName.value.trim()) {
    editError.value = t('admin.nameRequired')
    return
  }
  editing.value = true
  editError.value = ''
  const id = editTarget.value.id
  try {
    await api.updateApp(id, { name: editName.value.trim() })
    if (editIcon.value) await api.uploadAppIcon(id, editIcon.value)
    await load()
    editDialogOpen.value = false
    toast(t('admin.appUpdated'))
  } catch (e) {
    editError.value = (e as Error).message
  } finally {
    editing.value = false
  }
}

function askDelete(item: AppItem) {
  deleteTarget.value = item
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  const id = deleteTarget.value.id
  deleteDialogOpen.value = false
  try {
    await api.deleteApp(id)
    await load()
    toast(t('admin.appDeleted'))
  } catch (e) {
    toast((e as Error).message)
  }
}
</script>

<template>
  <div class="mx-auto max-w-6xl px-4 py-8 sm:px-6">
    <div class="mb-6 flex items-center justify-between">
      <h1 class="text-2xl font-semibold tracking-tight">{{ t('admin.title') }}</h1>
      <Button @click="openCreate">
        <Plus class="size-4" />
        {{ t('admin.newApp') }}
      </Button>
    </div>

    <Alert v-if="error" variant="destructive" class="mb-4">{{ error }}</Alert>

    <div v-if="apps.length" class="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
      <Card v-for="a in apps" :key="a.id" class="gap-0">
        <div class="flex flex-1 cursor-pointer flex-col gap-3 p-5" @click="router.push(`/admin/app/${a.id}`)">
          <div class="flex items-center gap-3">
            <Avatar :src="a.icon" :fallback="a.name.charAt(0).toUpperCase()" class="size-12" />
            <span class="truncate font-semibold">{{ a.name }}</span>
          </div>
          <p v-if="a.description" class="text-muted-foreground text-sm">{{ a.description }}</p>
        </div>
        <Separator />
        <CardFooter class="justify-between py-3">
          <span class="text-muted-foreground text-xs">{{ fmtDate(a.created_at) }}</span>
          <div class="flex gap-1">
            <Button variant="ghost" size="sm" @click="openEdit(a)">{{ t('common.edit') }}</Button>
            <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="askDelete(a)">{{ t('common.delete') }}</Button>
          </div>
        </CardFooter>
      </Card>
    </div>

    <Card v-else-if="!error" class="text-center">
      <CardContent class="py-12 text-muted-foreground">{{ t('admin.empty') }}</CardContent>
    </Card>

    <!-- Create dialog -->
    <Dialog v-model:open="createDialogOpen" :title="t('admin.newApp')" max-width="md">
      <div class="grid gap-4">
        <div class="grid gap-2">
          <Label>{{ t('admin.appName') }}</Label>
          <Input v-model="newName" autofocus @keyup.enter="confirmCreate" />
        </div>
        <div class="flex items-center gap-3">
          <Avatar :src="createIconPreview" :fallback="(newName || '?').charAt(0).toUpperCase()" class="size-12" />
          <div class="flex-1">
            <FileUpload :label="t('admin.appIcon')" accept="image/*" @change="onCreateIcon" />
          </div>
        </div>
        <Alert v-if="createError" variant="destructive">{{ createError }}</Alert>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="createDialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button :disabled="!newName.trim()" @click="confirmCreate">{{ t('common.create') }}</Button>
        </div>
      </div>
    </Dialog>

    <!-- Edit dialog -->
    <Dialog v-model:open="editDialogOpen" :title="t('admin.editApp')" max-width="md">
      <div class="grid gap-4">
        <div class="grid gap-2">
          <Label>{{ t('admin.appName') }}</Label>
          <Input v-model="editName" autofocus @keyup.enter="confirmEdit" />
        </div>
        <div class="flex items-center gap-3">
          <Avatar :src="editIconPreview" :fallback="(editName || '?').charAt(0).toUpperCase()" class="size-12" />
          <div class="flex-1">
            <FileUpload :label="t('admin.appIcon')" accept="image/*" @change="onEditIcon" />
          </div>
        </div>
        <Alert v-if="editError" variant="destructive">{{ editError }}</Alert>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="editDialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button :disabled="!editName.trim()" @click="confirmEdit">{{ t('common.save') }}</Button>
        </div>
      </div>
    </Dialog>

    <!-- Delete dialog -->
    <AlertDialog v-model:open="deleteDialogOpen" :title="t('common.confirmDelete')" :description="t('admin.confirmDeleteApp', { name: deleteTarget?.name ?? '' })">
      <template #footer>
        <Button variant="outline" @click="deleteDialogOpen = false">{{ t('common.cancel') }}</Button>
        <Button variant="destructive" @click="confirmDelete">{{ t('common.delete') }}</Button>
      </template>
    </AlertDialog>
  </div>
</template>
```

> Note: `ImageIcon` is imported but not used in the final template — remove that import to satisfy `noUnusedLocals`.

- [ ] **Step 2: Verify build**

Run: `cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend && npm run build`
Expected: build succeeds. Remove unused imports as flagged.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/admin/Admin.vue
git commit -m "feat: rewrite admin apps grid with shadcn"
```

---

### Task 12: Rewrite `Users.vue`

**Files:**
- Rewrite: `frontend/src/views/admin/Users.vue`

- [ ] **Step 1: Rewrite the file**

```vue
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { toast } from 'vue-sonner'
import { Plus } from 'lucide-vue-next'
import { api } from '../../api/client'
import { useAuth } from '../../composables/useAuth'
import { useI18n } from '../../composables/useI18n'
import { fmtDate } from '../../utils/format'
import { Button } from '../../components/ui/button'
import { Card, CardContent } from '../../components/ui/card'
import { Alert } from '../../components/ui/alert'
import { Dialog } from '../../components/ui/dialog'
import { AlertDialog } from '../../components/ui/alert-dialog'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../components/ui/table'
import type { User } from '../../api/types'

const { isAuthed } = useAuth()
const { t } = useI18n()
const users = ref<User[]>([])
const error = ref('')

const newUsername = ref('')
const newPassword = ref('')
const createError = ref('')
const createDialogOpen = ref(false)

const editTarget = ref<User | null>(null)
const editDialogOpen = ref(false)
const editUsername = ref('')
const editPassword = ref('')
const editError = ref('')
const editLoading = ref(false)

const deleteTarget = ref<User | null>(null)
const deleteDialogOpen = ref(false)

onMounted(load)

async function load() {
  try {
    users.value = await api.adminUsers()
  } catch (e) {
    error.value = (e as Error).message
  }
}

function openCreate() {
  newUsername.value = ''
  newPassword.value = ''
  createError.value = ''
  createDialogOpen.value = true
}

async function confirmCreate() {
  if (!newUsername.value || !newPassword.value) {
    createError.value = t('adminUsers.required')
    return
  }
  createError.value = ''
  try {
    await api.createUser({ username: newUsername.value, password: newPassword.value })
    createDialogOpen.value = false
    await load()
    toast(t('adminUsers.userCreated'))
  } catch (e) {
    createError.value = (e as Error).message
  }
}

function openEdit(u: User) {
  editTarget.value = u
  editUsername.value = u.username
  editPassword.value = ''
  editError.value = ''
  editDialogOpen.value = true
}

async function confirmEdit() {
  if (!editTarget.value) return
  if (!editUsername.value.trim()) {
    editError.value = t('adminUsers.required')
    return
  }
  editError.value = ''
  editLoading.value = true
  try {
    const data: { username?: string; password?: string } = {}
    if (editUsername.value.trim() !== editTarget.value.username) {
      data.username = editUsername.value.trim()
    }
    if (editPassword.value) {
      data.password = editPassword.value
    }
    await api.updateUser(editTarget.value.id, data)
    editDialogOpen.value = false
    await load()
    toast(t('adminUsers.userUpdated'))
  } catch (e) {
    editError.value = (e as Error).message
  } finally {
    editLoading.value = false
  }
}

function askDelete(u: User) {
  deleteTarget.value = u
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  const id = deleteTarget.value.id
  deleteDialogOpen.value = false
  try {
    await api.deleteUser(id)
    await load()
    toast(t('adminUsers.userDeleted'))
  } catch (e) {
    toast((e as Error).message)
  }
}
</script>

<template>
  <div class="mx-auto max-w-4xl px-4 py-8 sm:px-6">
    <div class="mb-6 flex items-center justify-between">
      <h1 class="text-2xl font-semibold tracking-tight">{{ t('adminUsers.title') }}</h1>
      <Button v-if="isAuthed" @click="openCreate">
        <Plus class="size-4" />
        {{ t('adminUsers.newUser') }}
      </Button>
    </div>

    <Alert v-if="error" variant="destructive" class="mb-4">{{ error }}</Alert>

    <Card class="mb-4">
      <CardContent class="py-4 text-sm">
        <div class="mb-1 text-xs font-semibold uppercase tracking-wider">{{ t('adminUsers.superAdminLabel') }}</div>
        <span class="text-muted-foreground" v-html="t('adminUsers.superAdminNote')" />
      </CardContent>
    </Card>

    <Card>
      <CardContent class="!p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('common.username') }}</TableHead>
              <TableHead>{{ t('common.created') }}</TableHead>
              <TableHead class="text-right"> </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="u in users" :key="u.id">
              <TableCell><code>{{ u.username }}</code></TableCell>
              <TableCell><code class="text-xs">{{ fmtDate(u.created_at) }}</code></TableCell>
              <TableCell class="text-right">
                <Button variant="ghost" size="sm" @click="openEdit(u)">{{ t('common.edit') }}</Button>
                <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="askDelete(u)">{{ t('common.delete') }}</Button>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>

    <Dialog v-model:open="createDialogOpen" :title="t('adminUsers.newUser')" max-width="md">
      <div class="grid gap-4">
        <div class="grid gap-2">
          <Label>{{ t('common.username') }}</Label>
          <Input v-model="newUsername" autofocus />
        </div>
        <div class="grid gap-2">
          <Label>{{ t('common.password') }}</Label>
          <Input v-model="newPassword" type="password" @keyup.enter="confirmCreate" />
        </div>
        <Alert v-if="createError" variant="destructive">{{ createError }}</Alert>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="createDialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button :disabled="!newUsername || !newPassword" @click="confirmCreate">{{ t('common.create') }}</Button>
        </div>
      </div>
    </Dialog>

    <Dialog v-model:open="editDialogOpen" :title="t('adminUsers.editUser')" max-width="md">
      <div class="grid gap-4">
        <div class="grid gap-2">
          <Label>{{ t('common.username') }}</Label>
          <Input v-model="editUsername" autofocus />
        </div>
        <div class="grid gap-2">
          <Label>{{ t('adminUsers.newPassword') }}</Label>
          <Input v-model="editPassword" type="password" @keyup.enter="confirmEdit" />
        </div>
        <Alert v-if="editError" variant="destructive">{{ editError }}</Alert>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="editDialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button :disabled="!editUsername.trim()" @click="confirmEdit">{{ t('common.save') }}</Button>
        </div>
      </div>
    </Dialog>

    <AlertDialog v-model:open="deleteDialogOpen" :title="t('common.confirmDelete')" :description="t('adminUsers.confirmDeleteUser', { name: deleteTarget?.username ?? '' })">
      <template #footer>
        <Button variant="outline" @click="deleteDialogOpen = false">{{ t('common.cancel') }}</Button>
        <Button variant="destructive" @click="confirmDelete">{{ t('common.delete') }}</Button>
      </template>
    </AlertDialog>
  </div>
</template>
```

- [ ] **Step 2: Verify build**

Run: `cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend && npm run build`
Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/admin/Users.vue
git commit -m "feat: rewrite users page with shadcn table + dialogs"
```

---

### Task 13: Rewrite `Upload.vue`

**Files:**
- Rewrite: `frontend/src/views/admin/Upload.vue`

- [ ] **Step 1: Rewrite the file**

```vue
<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../../api/client'
import { useI18n } from '../../composables/useI18n'
import { ARCH_BY_PLATFORM, PLATFORMS, detectPlatformFromName } from '../../constants/platform'
import { Alert } from '../../components/ui/alert'
import { Avatar } from '../../components/ui/avatar'
import { Button } from '../../components/ui/button'
import { Card, CardContent } from '../../components/ui/card'
import { Checkbox } from '../../components/ui/checkbox'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Textarea } from '../../components/ui/textarea'
import AppSelect from '../../components/AppSelect.vue'
import FileUpload from '../../components/FileUpload.vue'
import type { AppItem, Architecture, Platform, ReleaseType } from '../../api/types'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const file = ref<File | null>(null)
const initialAppId = Number(route.query.app_id)
const appId = ref<number | null>(Number.isFinite(initialAppId) && initialAppId > 0 ? initialAppId : null)
const releaseType = ref<ReleaseType>('production')
const platform = ref<Platform | ''>('')
const arch = ref<Architecture[]>([])
const versionName = ref('')
const versionCode = ref<number | null>(null)
const changelog = ref('')
const error = ref('')
const loading = ref(false)

const parsing = ref(false)
const parseError = ref('')
const parsed = ref<{
  platform: Platform
  package: string
  appName: string
  iconDataUri: string
} | null>(null)

const apps = ref<AppItem[]>([])

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

const releaseItems = computed(() => [
  { title: t('release.production'), value: 'production' },
  { title: t('release.beta'), value: 'beta' },
  { title: t('release.canary'), value: 'canary' },
])

const platformItems = computed(() =>
  PLATFORMS.map((p) => ({ title: t('platform.' + p), value: p }))
)

const archOptions = computed(() =>
  platform.value ? ARCH_BY_PLATFORM[platform.value] : []
)

watch(platform, () => { arch.value = [] })

function toggleArch(a: Architecture, checked: boolean) {
  arch.value = checked ? [...arch.value, a] : arch.value.filter((x) => x !== a)
}

function normalizeResult(res: AppInfoParserResult, ext: string) {
  if (ext === 'apk') {
    let appName = res.appName || ''
    if (appName.startsWith('@') || appName.startsWith('resourceId:')) appName = ''
    return {
      platform: 'android' as Platform,
      package: res.package || '',
      versionName: res.versionName || '',
      versionCode: Number(res.versionCode) || 0,
      appName,
      iconDataUri: res.icon || '',
    }
  }
  return {
    platform: 'ios' as Platform,
    package: res.CFBundleIdentifier || '',
    versionName: res.CFBundleShortVersionString || '',
    versionCode: Number(res.CFBundleVersion) || 0,
    appName: (res.CFBundleDisplayName as string) || (res.CFBundleName as string) || '',
    iconDataUri: res.icon || '',
  }
}

async function parseApp(f: File, ext: string) {
  parsing.value = true
  parseError.value = ''
  try {
    const info = normalizeResult(await new window.AppInfoParser(f).parse(), ext)
    parsed.value = info
    versionName.value = info.versionName
    versionCode.value = info.versionCode || null
    platform.value = info.platform
  } catch (e) {
    parseError.value = (e as Error).message || String(e)
  } finally {
    parsing.value = false
  }
}

function onFile(file: File | File[]) {
  const f = Array.isArray(file) ? file[0] : file
  file.value = f
  parsed.value = null
  parseError.value = ''
  if (!f) return
  const ext = (f.name.split('.').pop() ?? '').toLowerCase()
  if (!platform.value) platform.value = detectPlatformFromName(f.name)
  if (ext === 'apk' || ext === 'ipa') {
    parseApp(f, ext)
  }
}

function dataUriToBlob(uri: string): Blob {
  const comma = uri.indexOf(',')
  const mime = /data:([^;]+)/.exec(uri.slice(0, comma))?.[1] || 'image/png'
  const bin = atob(uri.slice(comma + 1))
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
  return new Blob([bytes], { type: mime })
}

async function submit() {
  if (!file.value || !appId.value) {
    error.value = t('upload.required')
    return
  }
  if (!versionName.value) {
    error.value = t('upload.versionNameRequired')
    return
  }
  error.value = ''
  loading.value = true

  const form = new FormData()
  form.append('file', file.value)
  form.append('app_id', String(appId.value))
  form.append('release_type', releaseType.value)
  if (platform.value) form.append('platform', platform.value)
  if (arch.value.length) form.append('arch', arch.value.join(','))
  if (versionName.value) form.append('version_name', versionName.value)
  if (versionCode.value) form.append('version_code', String(versionCode.value))
  form.append('changelog', changelog.value)
  if (parsed.value) {
    if (parsed.value.package) form.append('package_name', parsed.value.package)
    if (parsed.value.appName) form.append('app_name', parsed.value.appName)
    if (parsed.value.iconDataUri) {
      form.append('icon', dataUriToBlob(parsed.value.iconDataUri), 'icon.png')
    }
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
  <div class="mx-auto max-w-2xl px-4 py-8 sm:px-6">
    <h1 class="text-2xl font-semibold tracking-tight mb-6">{{ t('upload.title') }}</h1>

    <Alert v-if="error" variant="destructive" class="mb-4">{{ error }}</Alert>

    <form class="grid gap-4" @submit.prevent="submit">
      <Card>
        <CardContent class="grid gap-4">
          <FileUpload
            :label="t('upload.file')"
            accept=".apk,.aab,.ipa,.exe,.dmg"
            drop-zone
            :disabled="parsing"
            @change="onFile"
          />
          <Alert v-if="parsing" variant="info">{{ t('upload.parsing') }}</Alert>
          <Alert v-else-if="parseError" variant="warning">{{ t('upload.parseFailed') }}</Alert>

          <div v-if="parsed" class="flex items-center gap-3 rounded-lg border bg-muted/30 p-3">
            <Avatar :src="parsed.iconDataUri" :fallback="(parsed.appName || '?').charAt(0).toUpperCase()" class="size-10" />
            <div class="min-w-0 text-sm">
              <div v-if="parsed.appName" class="font-medium">{{ parsed.appName }}</div>
              <code v-if="parsed.package" class="text-muted-foreground text-xs">{{ parsed.package }}</code>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent class="grid gap-4">
          <div class="grid gap-2">
            <Label>{{ t('upload.app') }}</Label>
            <AppSelect v-model="appId" :items="appItems" :placeholder="t('upload.app')" />
          </div>

          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div class="grid gap-2">
              <Label>{{ t('upload.releaseType') }}</Label>
              <AppSelect v-model="releaseType" :items="releaseItems" />
            </div>
            <div class="grid gap-2">
              <Label>{{ t('upload.platform') }}</Label>
              <AppSelect v-model="platform" :items="platformItems" :placeholder="t('upload.platform')" />
            </div>
          </div>

          <div class="grid gap-2">
            <Label>{{ t('upload.arch') }}</Label>
            <div v-if="archOptions.length" class="flex flex-wrap gap-4">
              <label v-for="a in archOptions" :key="a" class="flex cursor-pointer items-center gap-2 text-sm">
                <Checkbox :checked="arch.includes(a)" @update:checked="(c) => toggleArch(a, !!c)" />
                {{ t('arch.' + a) }}
              </label>
            </div>
            <p v-else class="text-muted-foreground text-sm">—</p>
          </div>

          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div class="grid gap-2">
              <Label>{{ t('upload.versionName') }}</Label>
              <Input v-model="versionName" :placeholder="'1.0.0'" />
            </div>
            <div class="grid gap-2">
              <Label>{{ t('upload.versionCode') }}</Label>
              <Input v-model.number="versionCode" type="number" :placeholder="'1'" />
            </div>
          </div>

          <p class="text-muted-foreground text-xs">{{ t('upload.parseHint') }}</p>

          <div class="grid gap-2">
            <Label>{{ t('upload.changelog') }}</Label>
            <Textarea v-model="changelog" rows="3" />
          </div>
        </CardContent>
      </Card>

      <Alert variant="info">{{ t('upload.publishHint') }}</Alert>

      <div class="flex justify-end gap-2">
        <Button variant="outline" @click="router.back()">{{ t('common.cancel') }}</Button>
        <Button type="submit" :disabled="!file || !appId || loading">
          {{ t('upload.submit') }}
        </Button>
      </div>
    </form>
  </div>
</template>
```

> Note: The template references `parsed.iconDataUri` directly as `Avatar src` (a data URI works). The `AppInfoParserResult` global type is provided by `index.html`'s vendor script — reference it as `AppInfoParserResult` (declared as a global in the previous code via `window.AppInfoParser`). Keep the same typing used before.

- [ ] **Step 2: Verify build**

Run: `cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend && npm run build`
Expected: build succeeds. `versionCode` uses `v-model.number` on `Input`; if the `Input` model emits strings, coerce: `v-model="versionCode"` then `String(versionCode.value)` when building the form is already handled. Adjust typing in `submit` if needed.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/admin/Upload.vue
git commit -m "feat: rewrite upload page with shadcn"
```

---

### Task 14: Rewrite `AdminApp.vue`

**Files:**
- Rewrite: `frontend/src/views/admin/AdminApp.vue`

- [ ] **Step 1: Rewrite the script block** — keep ALL existing state/logic (stats chart, filters, dialogs, download link, access save, publish, versions) and change only the Vuetify imports to shadcn imports.

The current script (lines 1–486 of the old file) stays functionally identical except:

1. Replace line 4 `import { mdiContentCopy } from '@mdi/js'` with `import { Copy, Upload as UploadIcon, Plus } from 'lucide-vue-next'`.
2. Add shadcn component imports:
   ```ts
   import { Button } from '../../components/ui/button'
   import { Avatar } from '../../components/ui/avatar'
   import { Badge } from '../../components/ui/badge'
   import { Card, CardContent, CardTitle } from '../../components/ui/card'
   import { Alert } from '../../components/ui/alert'
   import { Dialog } from '../../components/ui/dialog'
   import { AlertDialog } from '../../components/ui/alert-dialog'
   import { Input } from '../../components/ui/input'
   import { Label } from '../../components/ui/label'
   import { Textarea } from '../../components/ui/textarea'
   import { RadioGroup, RadioGroupItem } from '../../components/ui/radio-group'
   import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/tabs'
   import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, TableEmpty } from '../../components/ui/table'
   import { Separator } from '../../components/ui/separator'
   import { Skeleton } from '../../components/ui/skeleton'
   import AppSelect from '../../components/AppSelect.vue'
   import FileUpload from '../../components/FileUpload.vue'
   ```
3. Replace the two chart series color strings in `chartSeries`:
   - `color: 'rgb(var(--v-theme-primary))'` → `color: 'var(--color-primary)'`
   - `color: 'rgb(var(--v-theme-warning))'` → `color: 'var(--color-warning)'`
4. Update `statusColor`/`accessColor`/`releaseColor` helpers to return **Badge variant names** instead of Vuetify color names:
   ```ts
   function statusBadge(v: Version): 'secondary' | 'success' | 'destructive' {
     if (!v.published) return 'secondary'
     return v.enabled ? 'success' : 'destructive'
   }
   function releaseBadge(rt: string): 'default' | 'info' | 'warning' {
     if (rt === 'beta') return 'info'
     if (rt === 'canary') return 'warning'
     return 'default'
   }
   function accessBadge(appAccess: string, v: Version): 'secondary' | 'success' | 'warning' | 'destructive' {
     if (!v.published) return 'secondary'
     if (!v.enabled) return 'destructive'
     if (appAccess === 'public') return 'success'
     if (appAccess === 'password' || appAccess === 'expiry') return 'warning'
     return 'secondary'
   }
   ```
   Remove the old `statusColor`, `statusLabel` (keep statusLabel), `releaseColor`, `accessColor`, `accessLabel` as needed; keep `statusLabel`/`accessLabel`/`actionLabel`.

- [ ] **Step 2: Rewrite the template** — map each `v-*` block per the table in the spec:

**Header:**
```html
<div class="mx-auto max-w-6xl px-4 py-8 sm:px-6">
  <div v-if="data" class="mb-6 flex items-center justify-between gap-3">
    <div class="flex items-center gap-3">
      <Avatar :src="data.app.icon" :fallback="data.app.name.charAt(0).toUpperCase()" class="size-10" />
      <h1 class="text-2xl font-semibold tracking-tight">{{ data.app.name }}</h1>
    </div>
    <Button @click="goUpload">
      <UploadIcon class="size-4" />
      {{ t('adminApp.upload') }}
    </Button>
  </div>

  <Alert v-if="error" variant="destructive" class="mb-4">{{ error }}</Alert>

  <Tabs v-model="tab" class="mt-4">
    <TabsList>
      <TabsTrigger value="overview">{{ t('adminApp.tabOverview') }}</TabsTrigger>
      <TabsTrigger value="versions">{{ t('adminApp.tabVersions') }}</TabsTrigger>
      <TabsTrigger value="stats">{{ t('adminApp.tabStats') }}</TabsTrigger>
    </TabsList>

    <TabsContent value="overview" class="mt-6 grid grid-cols-1 gap-4 md:grid-cols-2">
      <!-- App info -->
      <Card class="p-5">
        <CardTitle class="text-base mb-4">{{ t('adminApp.overviewInfo') }}</CardTitle>
        <div class="mb-4 flex items-center gap-3">
          <Avatar :src="infoIconPreview" :fallback="(infoName || data?.app.name || '?').charAt(0).toUpperCase()" class="size-14" />
          <div class="flex-1">
            <FileUpload :label="t('admin.appIcon')" accept="image/*" @change="onInfoIconChange" />
          </div>
        </div>
        <div class="grid gap-3">
          <div class="grid gap-2">
            <Label>{{ t('admin.appName') }}</Label>
            <Input v-model="infoName" />
          </div>
          <div class="grid gap-2">
            <Label>{{ t('adminApp.appDescription') }}</Label>
            <Textarea v-model="infoDescription" rows="2" />
          </div>
          <Alert v-if="infoError" variant="destructive">{{ infoError }}</Alert>
          <div class="flex justify-end">
            <Button :disabled="!infoName.trim()" @click="saveInfo">{{ t('common.save') }}</Button>
          </div>
        </div>
      </Card>

      <!-- Download link + access permission -->
      <Card class="p-5">
        <CardTitle class="text-base mb-4">{{ t('adminApp.downloadLink') }}</CardTitle>
        <div class="flex gap-2">
          <Input :model-value="downloadLink" readonly class="flex-1" />
          <Button @click="copyLink">
            <Copy class="size-4" />
            {{ t('adminApp.copyLink') }}
          </Button>
        </div>
        <Separator class="my-4" />
        <CardTitle class="text-base mb-4">{{ t('upload.access') }}</CardTitle>
        <RadioGroup v-model="accessMode">
          <div class="flex items-center gap-2 text-sm">
            <RadioGroupItem value="public" id="r-public" />
            <Label for="r-public">{{ t('upload.accessPublic') }}</Label>
          </div>
          <div class="flex items-center gap-2 text-sm">
            <RadioGroupItem value="password" id="r-password" />
            <Label for="r-password">{{ t('upload.accessPassword') }}</Label>
          </div>
          <div class="flex items-center gap-2 text-sm">
            <RadioGroupItem value="expiry" id="r-expiry" />
            <Label for="r-expiry">{{ t('upload.accessExpiry') }}</Label>
          </div>
        </RadioGroup>
        <div v-if="accessMode === 'password'" class="mt-3 grid gap-2">
          <Label>{{ t('upload.downloadPassword') }}</Label>
          <Input v-model="accessPassword" type="password" />
        </div>
        <div v-if="accessMode === 'expiry'" class="mt-3 grid gap-2">
          <Label>{{ t('upload.expiresAt') }}</Label>
          <Input v-model="accessExpiresAt" type="datetime-local" />
        </div>
        <Alert v-if="accessError" variant="destructive" class="mt-2">{{ accessError }}</Alert>
        <div class="mt-3 flex justify-end">
          <Button @click="saveAccess">{{ t('common.save') }}</Button>
        </div>
      </Card>

      <!-- Screenshots -->
      <Card class="p-5 md:col-span-2">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <CardTitle class="text-base">{{ t('adminApp.overviewScreenshots') }}</CardTitle>
          <FileUpload :label="t('adminApp.uploadScreenshots')" accept="image/*" multiple @change="onScreenshotsChange" />
        </div>
        <Alert v-if="shotsError" variant="destructive" class="mb-2">{{ shotsError }}</Alert>
        <div v-if="data && data.app.screenshots.length" class="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4">
          <div v-for="url in data.app.screenshots" :key="url" class="overflow-hidden rounded-lg border">
            <img :src="url" class="aspect-[9/16] w-full object-cover" />
            <div class="flex justify-end p-1">
              <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="deleteScreenshot(url)">{{ t('common.delete') }}</Button>
            </div>
          </div>
        </div>
        <p v-else class="text-muted-foreground py-4 text-center text-sm">{{ t('adminApp.noScreenshots') }}</p>
      </Card>
    </TabsContent>

    <TabsContent value="versions" class="mt-6">
      <div class="mb-4 flex flex-wrap items-center gap-3">
        <AppSelect v-model="statusFilter" :items="statusFilterItems" class="w-40" />
        <AppSelect v-model="releaseFilter" :items="releaseFilterItems" class="w-40" />
        <AppSelect v-model="platformFilter" :items="platformFilterItems" class="w-48" />
        <Button variant="ghost" size="sm" @click="statusFilter = 'all'; releaseFilter = 'all'; platformFilter = 'all'">{{ t('adminApp.filterReset') }}</Button>
      </div>

      <Card>
        <CardContent class="!p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('adminApp.colVersion') }}</TableHead>
                <TableHead>{{ t('adminApp.colAppName') }}</TableHead>
                <TableHead>{{ t('adminApp.colPackage') }}</TableHead>
                <TableHead>{{ t('adminApp.colPlatform') }}</TableHead>
                <TableHead>{{ t('adminApp.colRelease') }}</TableHead>
                <TableHead>{{ t('adminApp.colSize') }}</TableHead>
                <TableHead>{{ t('adminApp.colAccess') }}</TableHead>
                <TableHead>{{ t('adminApp.colDownloads') }}</TableHead>
                <TableHead>{{ t('adminApp.colStatus') }}</TableHead>
                <TableHead class="text-right"> </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="v in filteredVersions" :key="v.id">
                <TableCell>
                  <div class="flex items-center gap-2">
                    <Avatar :src="v.icon_url" :fallback="(v.app_name || v.version_name || '?').charAt(0).toUpperCase()" class="size-7" />
                    <code>{{ v.version_name }}</code>
                    <span class="text-muted-foreground text-xs">{{ t('detail.code') }} {{ v.version_code }}</span>
                  </div>
                </TableCell>
                <TableCell>{{ v.app_name || '—' }}</TableCell>
                <TableCell><code v-if="v.package_name" class="text-xs">{{ v.package_name }}</code><span v-else class="text-muted-foreground">—</span></TableCell>
                <TableCell>
                  <span v-if="v.platform" class="mr-1">
                    <Badge variant="outline">{{ t('platform.' + v.platform) }}</Badge>
                  </span>
                  <span v-if="v.arch" class="text-muted-foreground text-xs">{{ archLabel(v.arch) }}</span>
                  <span v-if="!v.platform && !v.arch" class="text-muted-foreground">—</span>
                </TableCell>
                <TableCell>
                  <Badge v-if="v.release_type" :variant="releaseBadge(v.release_type)">{{ t('release.' + v.release_type) }}</Badge>
                </TableCell>
                <TableCell><code class="text-xs">{{ fmtSize(v.file_size) }}</code></TableCell>
                <TableCell>
                  <Badge :variant="accessBadge(appAccessMode, v)">{{ accessLabel(appAccessMode, v.published, v.enabled) }}</Badge>
                </TableCell>
                <TableCell>{{ v.download_count }}</TableCell>
                <TableCell><Badge :variant="statusBadge(v)">{{ statusLabel(v) }}</Badge></TableCell>
                <TableCell class="text-right">
                  <Button variant="ghost" size="sm" :class="v.published && v.enabled ? '' : 'text-primary'" @click="onMainAction(v)">{{ actionLabel(v) }}</Button>
                  <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="askDelete(v)">{{ t('common.delete') }}</Button>
                </TableCell>
              </TableRow>
              <TableEmpty v-if="!filteredVersions.length">{{ t('adminApp.statsEmpty') }}</TableEmpty>
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </TabsContent>

    <TabsContent value="stats" class="mt-6">
      <Card class="mb-4 p-5">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <CardTitle class="text-base">{{ t('adminApp.chartTitle') }}</CardTitle>
          <div class="flex flex-wrap items-center gap-3">
            <AppSelect v-model="chartFilterPlatform" :items="platformFilterItems" class="w-40" />
            <AppSelect v-model="chartFilterVersion" :items="chartVersionItems" class="w-52" />
            <AppSelect v-model="chartRange" :items="rangeItems" class="w-40" />
          </div>
        </div>
        <Skeleton v-if="chartLoading" class="mb-2 h-8 w-full" />
        <Alert v-else-if="chartError" variant="destructive" class="mb-2">{{ chartError }}</Alert>
        <LineChart :dates="sliced.dates" :series="chartSeries" :empty-text="t('adminApp.chartEmpty')" />
        <div v-if="chartSeries.length" class="mt-2 flex items-center gap-4">
          <div v-for="s in chartSeries" :key="s.name" class="flex items-center gap-1.5 text-xs">
            <span class="inline-block size-2.5 rounded-full" :style="{ background: s.color }" />
            <span>{{ s.name }}</span>
          </div>
        </div>
      </Card>

      <Card v-if="!stats" class="text-center">
        <CardContent class="py-12 text-muted-foreground">{{ t('adminApp.statsEmpty') }}</CardContent>
      </Card>
      <template v-else>
        <div class="mb-4 grid grid-cols-2 gap-4 md:grid-cols-3">
          <Card>
            <CardContent class="py-4">
              <div class="text-muted-foreground text-xs uppercase tracking-wider">{{ t('adminApp.statDownloads') }}</div>
              <div class="text-3xl font-semibold">{{ stats.download_count }}</div>
            </CardContent>
          </Card>
          <Card>
            <CardContent class="py-4">
              <div class="text-muted-foreground text-xs uppercase tracking-wider">{{ t('adminApp.statInstalls') }}</div>
              <div class="text-3xl font-semibold">{{ stats.install_count }}</div>
            </CardContent>
          </Card>
        </div>
        <Card>
          <CardContent class="!p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{{ t('adminApp.colTime') }}</TableHead>
                  <TableHead>{{ t('adminApp.colIp') }}</TableHead>
                  <TableHead>{{ t('adminApp.colUserAgent') }}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="(row, i) in stats.recent" :key="i">
                  <TableCell><code class="text-xs">{{ fmtDate(row.created_at) }}</code></TableCell>
                  <TableCell><code class="text-xs">{{ row.ip }}</code></TableCell>
                  <TableCell><code class="text-xs block max-w-[400px] truncate">{{ row.user_agent }}</code></TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </CardContent>
        </Card>
        <Button variant="ghost" class="mt-4" @click="chartFilterVersion = 'all'">{{ t('adminApp.statsClear') }}</Button>
      </template>
    </TabsContent>
  </Tabs>

  <!-- Publish dialog -->
  <Dialog v-model:open="publishDialogOpen" :title="t('adminApp.publishTitle')" max-width="md">
    <div class="grid gap-4">
      <div class="text-sm">
        <code>{{ publishTarget?.version_name }}</code>
        <span class="text-muted-foreground"> · {{ t('detail.code') }} {{ publishTarget?.version_code }}</span>
      </div>
      <p class="text-muted-foreground text-sm">{{ t('adminApp.publishHint') }}</p>
      <Alert v-if="publishError" variant="destructive">{{ publishError }}</Alert>
      <div class="flex justify-end gap-2">
        <Button variant="outline" @click="closePublish">{{ t('common.cancel') }}</Button>
        <Button @click="submitPublish">{{ t('adminApp.publish') }}</Button>
      </div>
    </div>
  </Dialog>

  <!-- Delete version dialog -->
  <AlertDialog v-model:open="dialogOpen" :title="t('common.confirmDelete')" :description="t('adminApp.confirmDeleteVersion', { name: deleteTarget?.version_name ?? '' })">
    <template #footer>
      <Button variant="outline" @click="cancelDelete">{{ t('common.cancel') }}</Button>
      <Button variant="destructive" @click="deleteVersion">{{ t('common.delete') }}</Button>
    </template>
  </AlertDialog>
</div>
```

> Note: This task's step 1 modifies the script minimally (imports + badge helpers + chart colors) and step 2 replaces the entire template. The **old template** (lines 488–913) is discarded. The `toast` import from `vue-sonner` is needed if any `showSnack` remains — replace `showSnack(msg)` with `toast(msg)` and remove the `snackbar`/`snackbarOpen` refs, OR keep a local `showSnack` wrapper that calls `toast`. Recommend: add `import { toast } from 'vue-sonner'` and replace `showSnack` body with `toast(msg)`.

- [ ] **Step 3: Verify build**

Run: `cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend && npm run build`
Expected: build succeeds. Fix unused vars (e.g. remove `snackbar`/`snackbarOpen`), and ensure `stats.recent` row type matches `{ ip: string; user_agent: string; created_at: string }`.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/views/admin/AdminApp.vue
git commit -m "feat: rewrite AdminApp with shadcn (tabs, tables, dialogs)"
```

---

### Task 15: Update `LineChart.vue` theme colors

**Files:**
- Modify: `frontend/src/components/LineChart.vue` (only the color references)

- [ ] **Step 1: Check current color references**

Run: `cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend && grep -n "v-theme" src/components/LineChart.vue`
Expected: references to `rgb(var(--v-theme-primary))` and/or `--v-theme-warning`.

- [ ] **Step 2: Replace theme color references**

In `src/components/LineChart.vue`, replace every `var(--v-theme-primary)` with `var(--color-primary)` and every `var(--v-theme-warning)` with `var(--color-warning)`. The chart otherwise stays identical (SVG polyline/circle/tooltip logic unchanged).

- [ ] **Step 3: Verify build**

Run: `cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend && npm run build`
Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/LineChart.vue
git commit -m "refactor: point LineChart colors at shadcn theme tokens"
```

---

### Task 16: Remove Vuetify

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/vite.config.ts`
- Modify: `frontend/src/main.ts`
- Delete: `frontend/src/plugins/vuetify.ts`

- [ ] **Step 1: Remove Vuetify deps**

```bash
cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend
npm uninstall vuetify vite-plugin-vuetify sass @mdi/js @iconify/vue @iconify-json/material-symbols
```

Expected: `package.json` no longer lists those packages; `npm ls vuetify` shows empty.

- [ ] **Step 2: Update `vite.config.ts`** — drop the Vuetify plugin comment and finalize

```ts
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

const API_TARGET = process.env.VITE_API_TARGET || 'http://localhost:8080'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  server: {
    host: true,
    proxy: {
      '/api': { target: API_TARGET, changeOrigin: true },
    },
  },
})
```

- [ ] **Step 3: Update `main.ts`**

```ts
import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import './index.css'

createApp(App).use(router).mount('#app')
```

- [ ] **Step 4: Delete `src/plugins/vuetify.ts`**

```bash
rm /Users/xiaqiubo/Desktop/test/go/disapp/frontend/src/plugins/vuetify.ts
```

- [ ] **Step 5: Verify build + no stray Vuetify usage**

Run:
```bash
cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend && npm run build
grep -rn "vuetify\|v-theme\|@mdi\|mdi-" src/ || echo "CLEAN"
```
Expected: build succeeds; grep prints `CLEAN`.

- [ ] **Step 6: Commit**

```bash
git add -A frontend/package.json frontend/package-lock.json frontend/vite.config.ts frontend/src/main.ts
git rm frontend/src/plugins/vuetify.ts
git commit -m "chore: remove Vuetify, @mdi/js, sass from frontend"
```

---

### Task 17: Full verification + deploy

**Files:**
- None (build + deploy only)

- [ ] **Step 1: Frontend build**

Run: `cd /Users/xiaqiubo/Desktop/test/go/disapp/frontend && npm run build`
Expected: `vue-tsc -b && vite build` both succeed, zero errors.

- [ ] **Step 2: Rebuild backend binary and restart**

```bash
cd /Users/xiaqiubo/Desktop/test/go/disapp
rm -rf backend/static/dist && cp -r frontend/dist backend/static/dist
cd backend && go build -o ../bin/disapp ./cmd/server
cd /Users/xiaqiubo/Desktop/test/go/disapp
pkill -f 'bin/disapp' || true
sleep 1
(nohup ./bin/disapp >/tmp/disapp.log 2>&1 &)
sleep 1
pgrep -f 'bin/disapp'
```

Expected: a live PID; server serving the new frontend.

- [ ] **Step 3: API smoke test**

Run:
```bash
curl -s http://localhost:8080/api/v1/apps | head -c 200
```
Expected: `{"code":0,...` JSON.

- [ ] **Step 4: Manual pass**

Open http://localhost:8080 and check:
1. Home — stat cards + app grid; theme toggle (light/dark/system); language switcher.
2. `/app/imchat` desktop UA — per-platform cards, arch chips clickable; mobile UA — hero + floating download bar.
3. Login → admin; password dialog works.
4. Admin apps — create/edit/delete dialogs with icon upload.
5. AdminApp — Overview (info, download-link+access card, screenshots), Versions (filters, table, publish/take-down/delete), Stats (chart filters + LineChart + recent table).
6. Users — table + create/edit/delete.
7. Upload — file pick + parse + arch checkboxes.

- [ ] **Step 5: Commit any remaining cleanup**

```bash
git add -A
git commit -m "feat: complete shadcn UI redesign" || echo "nothing to commit"
```

---

## Self-Review Notes

- **Spec coverage:** Tasks 1–2 cover stack + theming (spec §1, §2); Tasks 3–6 cover the component inventory (spec §3); Task 7–14 rewrite every view including AppShell/Home/AppDetail/Login/Admin/AdminApp/Upload/Users (spec §5); Task 15 covers LineChart colors (spec §5); Task 16 removes Vuetify/deps (spec §6, §7); Task 17 is the verification section.
- **Type consistency:** `SelectOption` (`{ title, value: string | number }`) is defined in `select/Select.vue` and `AppSelect.vue`; views pass `items` computed from `{title, value}` arrays — `appItems`, `statusFilterItems`, etc. already produce that shape. `Input`/`Textarea` use `useVModel` with `modelValue: string | number | null`. `Badge` variants include `default|secondary|destructive|outline|success|warning|info`.
- **Known follow-ups to fix during execution:** unused imports flagged inline (`toast`, `ImageIcon`, `Plus`, `UploadIcon`, `editLoading`), `Button` has no `loading` prop (use `:disabled`), and AdminApp script must drop `snackbar`/`snackbarOpen` refs in favor of `toast`.
