import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import DepsPanel from '@/components/common/DepsPanel.vue'
import type { DepState } from '@/composables/useDeps'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      deps: {
        loading: 'Checking...',
        instruction: 'Instructions',
        skip: 'Skip',
        skippedWarning: 'Skipped. May fail.',
        notFound: 'not found',
        notChecked: 'not checked',
        invalidVersion: 'invalid version, need {required}',
        bundledBanner: 'All dependencies are bundled with the installer.',
        systemDepsHint: 'To use the host tools, restart with --use-system-deps.',
        source: {
          bundled: 'Bundled',
          system: 'System',
          missing: 'Missing',
        },
      },
    },
  },
})

function makeOkDep(name: string): DepState {
  return {
    name,
    ok: true,
    version: '1.2.3',
    required: '>=1.0',
    installed: true,
    skipped: false,
    instructionOpen: false,
    instruction: null,
    instructionLoading: false,
  }
}

function makeFailedDep(name: string): DepState {
  return {
    name,
    ok: false,
    version: '',
    required: '>=1.0',
    installed: false,
    skipped: false,
    instructionOpen: false,
    instruction: null,
    instructionLoading: false,
  }
}

function makeSkippedDep(name: string): DepState {
  return { ...makeFailedDep(name), skipped: true }
}

describe('DepsPanel', () => {
  it('shows loading spinner when loading=true', () => {
    const wrapper = mount(DepsPanel, {
      global: { plugins: [i18n] },
      props: { deps: [], loading: true },
    })
    expect(wrapper.text()).toContain('Checking...')
    expect(wrapper.find('.deps-list').exists()).toBe(false)
  })

  it('renders ok deps without action buttons', () => {
    const deps = [makeOkDep('python')]
    const wrapper = mount(DepsPanel, {
      global: { plugins: [i18n] },
      props: { deps, loading: false },
    })
    expect(wrapper.find('.deps-list').exists()).toBe(true)
    expect(wrapper.text()).toContain('python')
    expect(wrapper.text()).toContain('1.2.3')
    expect(wrapper.find('.deps-actions').exists()).toBe(false)
  })

  it('renders instruction and skip buttons for failed dep', () => {
    const deps = [makeFailedDep('ansible')]
    const wrapper = mount(DepsPanel, {
      global: { plugins: [i18n] },
      props: { deps, loading: false },
    })
    expect(wrapper.find('.deps-actions').exists()).toBe(true)
    const buttons = wrapper.findAll('.deps-btn')
    const labels = buttons.map((b) => b.text())
    expect(labels.some((l) => l.includes('Instructions'))).toBe(true)
    expect(labels.some((l) => l.includes('Skip'))).toBe(true)
  })

  it('does NOT render an install button', () => {
    const deps = [makeFailedDep('ansible')]
    const wrapper = mount(DepsPanel, {
      global: { plugins: [i18n] },
      props: { deps, loading: false },
    })
    const allText = wrapper.text().toLowerCase()
    expect(allText).not.toContain('install')
    expect(allText).not.toContain('установить')
  })

  it('shows version with invalid version warning when installed but fails check', () => {
    const dep: DepState = {
      name: 'ansible',
      ok: false,
      version: '2.10.8',
      required: '>=2.12',
      installed: true,
      skipped: false,
      instructionOpen: false,
      instruction: null,
      instructionLoading: false,
    }
    const wrapper = mount(DepsPanel, {
      global: { plugins: [i18n] },
      props: { deps: [dep], loading: false },
    })
    const verText = wrapper.find('.deps-ver').text()
    expect(verText).toContain('2.10.8')
    expect(verText).toContain('invalid version')
    expect(wrapper.find('.deps-actions').exists()).toBe(true)
  })

  it('emits toggle-instruction when instruction button clicked', async () => {
    const deps = [makeFailedDep('ansible')]
    const wrapper = mount(DepsPanel, {
      global: { plugins: [i18n] },
      props: { deps, loading: false },
    })
    const instrBtn = wrapper.findAll('.deps-btn').find((b) => b.text().includes('Instructions'))
    await instrBtn?.trigger('click')
    expect(wrapper.emitted('toggle-instruction')).toBeTruthy()
    expect(wrapper.emitted('toggle-instruction')![0]).toEqual(['ansible'])
  })

  it('emits skip when skip button clicked', async () => {
    const deps = [makeFailedDep('ansible')]
    const wrapper = mount(DepsPanel, {
      global: { plugins: [i18n] },
      props: { deps, loading: false },
    })
    const skipBtn = wrapper.findAll('.deps-btn').find((b) => b.text().includes('Skip'))
    await skipBtn?.trigger('click')
    expect(wrapper.emitted('skip')).toBeTruthy()
    expect(wrapper.emitted('skip')![0]).toEqual(['ansible'])
  })

  it('shows skip warning for skipped dep', () => {
    const deps = [makeSkippedDep('ansible')]
    const wrapper = mount(DepsPanel, {
      global: { plugins: [i18n] },
      props: { deps, loading: false },
    })
    expect(wrapper.find('.deps-skip-warning').exists()).toBe(true)
    expect(wrapper.text()).toContain('Skipped')
  })

  it('shows instruction block when instructionOpen=true', () => {
    const dep: DepState = {
      ...makeFailedDep('ansible'),
      instructionOpen: true,
      instruction: 'sudo apt install ansible',
    }
    const wrapper = mount(DepsPanel, {
      global: { plugins: [i18n] },
      props: { deps: [dep], loading: false },
    })
    expect(wrapper.find('.deps-instruction').exists()).toBe(true)
    expect(wrapper.find('.deps-code').text()).toContain('sudo apt install ansible')
  })

  it('renders multiple deps correctly', () => {
    const deps = [makeOkDep('python'), makeFailedDep('ansible'), makeOkDep('ydb_platform.ydb')]
    const wrapper = mount(DepsPanel, {
      global: { plugins: [i18n] },
      props: { deps, loading: false },
    })
    const items = wrapper.findAll('.deps-item')
    expect(items).toHaveLength(3)
  })

  it('shows bundled banner when bundled=true', () => {
    const deps = [makeOkDep('python')]
    const wrapper = mount(DepsPanel, {
      global: { plugins: [i18n] },
      props: { deps, loading: false, bundled: true },
    })
    expect(wrapper.find('.deps-bundled-banner').exists()).toBe(true)
    expect(wrapper.text()).toContain('bundled with the installer')
  })

  it('shows system-deps CLI hint when bundled=true', () => {
    const deps = [makeOkDep('python')]
    const wrapper = mount(DepsPanel, {
      global: { plugins: [i18n] },
      props: { deps, loading: false, bundled: true },
    })
    expect(wrapper.find('.deps-system-hint').exists()).toBe(true)
    expect(wrapper.text()).toContain('--use-system-deps')
  })

  it('does not show bundled banner when bundled=false', () => {
    const deps = [makeOkDep('python')]
    const wrapper = mount(DepsPanel, {
      global: { plugins: [i18n] },
      props: { deps, loading: false, bundled: false },
    })
    expect(wrapper.find('.deps-bundled-banner').exists()).toBe(false)
    expect(wrapper.find('.deps-system-hint').exists()).toBe(false)
  })

  it('renders a source badge reflecting the dep source', () => {
    const dep: DepState = { ...makeOkDep('ansible'), source: 'bundled' }
    const wrapper = mount(DepsPanel, {
      global: { plugins: [i18n] },
      props: { deps: [dep], loading: false, bundled: true },
    })
    const badge = wrapper.find('.deps-source')
    expect(badge.exists()).toBe(true)
    expect(badge.classes()).toContain('is-bundled')
    expect(badge.text()).toBe('Bundled')
  })

  it('renders a system source badge', () => {
    const dep: DepState = { ...makeOkDep('python'), source: 'system' }
    const wrapper = mount(DepsPanel, {
      global: { plugins: [i18n] },
      props: { deps: [dep], loading: false },
    })
    const badge = wrapper.find('.deps-source')
    expect(badge.classes()).toContain('is-system')
    expect(badge.text()).toBe('System')
  })
})
