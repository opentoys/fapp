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
