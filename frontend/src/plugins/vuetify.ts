import { watch } from 'vue'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import { aliases, mdi } from 'vuetify/iconsets/mdi-svg'
import { choice } from '../composables/useTheme'

export const vuetify = createVuetify({
  components,
  directives,
  icons: {
    defaultSet: 'mdi',
    aliases,
    sets: { mdi },
  },
})

// Interim bridge: `useTheme` now drives the shadcn `.dark` class on <html>;
// this keeps the still-Vuetify UI in sync until the views are rewritten and
// this file is deleted. Remove together with this file.
const vuetifyTheme = vuetify.theme.global
watch(
  choice,
  (c) => {
    const resolved =
      c === 'system'
        ? window.matchMedia('(prefers-color-scheme: dark)').matches
          ? 'dark'
          : 'light'
        : c
    vuetifyTheme.name.value = resolved
  },
  { immediate: true },
)