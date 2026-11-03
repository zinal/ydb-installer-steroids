import { describe, it, expect, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'

const i18n = createI18n({
  legacy: false, locale: 'en',
  messages: {
    en: {
      install: {
        cancelTitle: 'Cancel installation?',
        cancelMessage: 'This cannot be undone.',
        continueBtn: 'Continue',
        stopConfirmBtn: 'Stop',
      },
    },
  },
})

function mountDialog(visible = true) {
  return mount(ConfirmDialog, {
    props: { modelValue: visible },
    global: { plugins: [i18n] },
    attachTo: document.body,
  })
}

function docText(sel: string) { return document.body.querySelector(sel)?.textContent ?? '' }

describe('ConfirmDialog', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })
  it('renders default title, message and buttons when visible', () => {
    mountDialog(true)
    expect(docText('.dialog-title')).toBe('Cancel installation?')
    expect(docText('.dialog-text')).toBe('This cannot be undone.')
    expect(document.body.querySelectorAll('.dialog-btn')).toHaveLength(2)
  })

  it('emits confirm + closes on Stop click', async () => {
    const wrapper = mountDialog(true)
    // Find Vuetify button components (teleported, but still in the component tree).
    const btns = wrapper.findAllComponents({ name: 'VBtn' })
    // Stop button is the last one (error color).
    await btns[btns.length - 1].trigger('click')
    expect(wrapper.emitted('confirm')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([false])
  })

  it('emits cancel + closes on Continue click', async () => {
    const wrapper = mountDialog(true)
    const btns = wrapper.findAllComponents({ name: 'VBtn' })
    // Continue button is the first one (tonal color).
    await btns[0].trigger('click')
    expect(wrapper.emitted('cancel')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([false])
  })

  it('is hidden when modelValue is false', () => {
    mountDialog(false)
    expect(docText('.dialog-title')).toBe('')
  })
})
