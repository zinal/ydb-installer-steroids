import { createI18n } from 'vue-i18n'
import ru from '@/locales/ru.json'
import en from '@/locales/en.json'

const STORAGE_KEY = 'ydb-installer.locale'

/**
 * Resolve the initial locale.
 *
 *   1. Explicit user choice from localStorage wins.
 *   2. Otherwise we trust whatever the backend wrote into the HTML
 *      (`<html lang="...">`) — this is set after the session is loaded.
 *   3. Fallback to "ru".
 */
function initialLocale(): 'ru' | 'en' {
    const stored = (typeof localStorage !== 'undefined' ? localStorage.getItem(STORAGE_KEY) : null) as
        | 'ru'
        | 'en'
        | null
    if (stored === 'ru' || stored === 'en') return stored
    if (typeof document !== 'undefined' && document.documentElement.lang === 'en') return 'en'
    return 'ru'
}

export const i18n = createI18n({
  legacy: false,
  globalInjection: true,
    locale: initialLocale(),
  fallbackLocale: 'en',
    messages: { ru, en },
})

/** Persist a user-chosen locale across reloads. */
export function setLocale(locale: 'ru' | 'en') {
    i18n.global.locale.value = locale
    if (typeof localStorage !== 'undefined') localStorage.setItem(STORAGE_KEY, locale)
    if (typeof document !== 'undefined') document.documentElement.lang = locale
}
