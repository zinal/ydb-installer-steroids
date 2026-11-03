import 'vuetify/styles'
import '@mdi/font/css/materialdesignicons.css'
import { createVuetify } from 'vuetify'
import { aliases, mdi } from 'vuetify/iconsets/mdi'

const ydbLight = {
  dark: false,
  colors: {
      background: '#f7f8fb',
      surface: '#ffffff',
      'surface-variant': '#eef1f6',
      'on-surface': '#0f172a',
      primary: '#2399FF',
      'primary-darken-1': '#1a7acc',
      secondary: '#64748b',
      accent: '#2399FF',
      error: '#dc2626',
      info: '#0284c7',
      success: '#16a34a',
      warning: '#d97706',
      'border-strong': '#e2e8f0',
      'border-soft': '#eef1f6',
  },
}

export const vuetify = createVuetify({
    defaults: {
        VBtn: { rounded: 'lg', class: 'text-none', flat: true, height: 40 },
        VCard: { rounded: 'xl', flat: true, class: 'card-soft' },
        VTextField: { variant: 'outlined', density: 'comfortable', color: 'primary' },
        VSelect: { variant: 'outlined', density: 'comfortable', color: 'primary' },
        VAlert: { rounded: 'lg', density: 'comfortable', variant: 'tonal' },
        VChip: { rounded: 'lg' },
        VList: { density: 'comfortable' },
    },
  theme: {
      defaultTheme: 'ydbLight',
      themes: { ydbLight },
  },
    icons: {
        defaultSet: 'mdi',
        aliases,
        sets: { mdi },
  },
})
