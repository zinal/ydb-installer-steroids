import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

vi.mock('@/api/client', () => ({
  api: {
    getRuntimeInfo: vi.fn().mockResolvedValue({ embedded: true, extracted: true }),
  },
}))

import DatabasesScreen from '@/views/DatabasesScreen.vue'
import { useDatabasesStore } from '@/stores/databases'
import { useSessionStore } from '@/stores/session'
import { useHostsStore } from '@/stores/hosts'
import { useYdbConfigStore } from '@/stores/ydbConfig'
import { useEmStore } from '@/stores/em'
import { useRuntimeStore } from '@/stores/runtime'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  missingWarn: false,
  fallbackWarn: false,
  messages: {
    en: {
      common: { step: 'Step', soon: 'soon' },
      screens: { '7': 'Databases' },
      databases: {
        heading: 'Databases',
        topologyLabel: 'Topology: {topology}',
        variantLabel: 'Configuration variant',
        variantSingle: 'Single DB',
        variantMultiple: 'Multiple DBs',
        variantSinglePlusEm: 'Single DB + YDB EM',
        commonParams: 'Common parameters',
        coresDynamic: 'Dynamic cores',
        coresRequired: 'Cores must be greater than 0',
        databaseLabel: 'Database',
        dbName: 'Database name',
        dbNamePlaceholder: 'user_db',
        dbNameRequired: 'Name is required',
        groups: 'Storage groups',
        hosts: 'Hosts',
        hostsAll: 'All hosts',
        hostsSelect: 'Select hosts',
        allHostsHint: '{count} hosts will be used',
        advancedSettings: 'Advanced settings',
        affinity: 'Affinity',
        affinityPlaceholder: 'Disabled in this release',
        affinityDesc: 'Dynamic node affinity',
        singlePlusEmHint: 'EM database name is fixed',
        userDbLabel: 'User database',
        emDbLabel: 'YDB EM database',
        addDatabase: 'Add database',
        duplicateName: 'Duplicate name',
      },
      help: {
        coresDynamic: 'Cores for dynamic nodes',
        dbName: 'Database name',
        dbGroups: 'Storage groups',
        affinityDynamic: 'Affinity',
        docsLink: 'Docs',
      },
      coresInfo: {
        title: 'Resources',
        available: 'Available',
        requested: 'Requested',
        perHost: 'per host',
        warning: 'Requested more than available',
        recommendation: 'Recommendation',
      },
    },
  },
})

function mountScreen() {
  return mount(DatabasesScreen, {
    global: { plugins: [i18n] },
  })
}

const variantButtons = (wrapper: ReturnType<typeof mountScreen>) =>
  wrapper.findAll('.variant-option')

describe('DatabasesScreen', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders the two non-EM variants without runtime flags', async () => {
    const wrapper = mountScreen()
    const db = useDatabasesStore()
    expect(variantButtons(wrapper)).toHaveLength(2)
    expect(db.configVariant).toBe('single')

    await variantButtons(wrapper)[1].trigger('click')
    expect(db.configVariant).toBe('multiple')
    expect(variantButtons(wrapper)[1].classes()).toContain('is-active')

    await variantButtons(wrapper)[0].trigger('click')
    expect(db.configVariant).toBe('single')
  })

  it('single-plus-em toggles session.showEmStep (step 7.1 visibility)', async () => {
    const { api } = await import('@/api/client')
    vi.mocked(api.getRuntimeInfo).mockResolvedValue({ embedded: true, extracted: true, show_em_scenarios: true })
    await useRuntimeStore().load()

    const wrapper = mountScreen()
    const db = useDatabasesStore()
    const session = useSessionStore()

    expect(session.showEmStep).toBe(false)

    db.setConfigVariant('single-plus-em')
    await nextTick()
    expect(session.showEmStep).toBe(true)

    db.setConfigVariant('single')
    await nextTick()
    expect(session.showEmStep).toBe(false)
    wrapper.unmount()
  })

  it('single-plus-em seeds the EM store from YDB domain and first host', async () => {
    const { api } = await import('@/api/client')
    vi.mocked(api.getRuntimeInfo).mockResolvedValue({ embedded: true, extracted: true, show_em_scenarios: true })
    await useRuntimeStore().load()

    useHostsStore().dcHosts['DC-1'] = ['ydb-1.example.com']
    useYdbConfigStore().domain = 'Root'

    const wrapper = mountScreen()
    const db = useDatabasesStore()
    const em = useEmStore()

    db.setConfigVariant('single-plus-em')
    await nextTick()

    expect(em.useDynamicConfig).toBe(true)
    expect(em.dbConnectionDBRoot).toBe('/Root')
    expect(em.dbConnectionEndpoint).toBe('grpcs://ydb-1.example.com:2135')
    expect(db.entries[1]?.name).toBe('ydb_em')
    expect(em.dbConnectionDB).toBe('ydb_em')
    expect(em.hosts[0]?.fqdn).toBe('ydb-1.example.com')
    wrapper.unmount()
  })

  it('updates the single DB entry name and flags an empty name', async () => {
    const wrapper = mountScreen()
    const db = useDatabasesStore()

    const nameInput = wrapper.find('.text-input input')
    await nameInput.setValue('my_db')
    expect(db.entries[0].name).toBe('my_db')

    await nameInput.setValue('   ')
    expect(wrapper.find('.validation-hint.is-error').exists()).toBe(true)
  })

  it('multiple variant adds and removes entries via the buttons', async () => {
    const db = useDatabasesStore()
    db.setConfigVariant('multiple')
    await nextTick()
    const initial = db.entries.length

    const wrapper = mountScreen()
    await nextTick()

    const addBtn = wrapper.find('.add-entry-row button')
    expect(addBtn).toBeTruthy()
    await addBtn!.trigger('click')
    expect(db.entries.length).toBe(initial + 1)

    const removeBtns = wrapper.findAll('.remove-btn')
    expect(removeBtns.length).toBeGreaterThan(0)
    await removeBtns[removeBtns.length - 1].trigger('click')
    expect(db.entries.length).toBe(initial)
  })

  it('detects duplicate database names in multiple variant', async () => {
    const wrapper = mountScreen()
    const db = useDatabasesStore()

    db.setConfigVariant('multiple')
    db.addEntry()
    db.updateEntry(0, 'name', 'same_db')
    db.updateEntry(1, 'name', 'same_db')
    await nextTick()

    expect(db.duplicateNames).toContain('same_db')
    wrapper.unmount()
  })

  it('mirrors db.canProceed into session.canGoNext', async () => {
    const wrapper = mountScreen()
    const db = useDatabasesStore()
    const session = useSessionStore()

    expect(session.canGoNext).toBe(db.canProceed)

    db.coresDynamic = 4
    db.updateEntry(0, 'name', 'user_db')
    await nextTick()
    expect(session.canGoNext).toBe(db.canProceed)
    wrapper.unmount()
  })

  it('hides the single-plus-em variant when the flag is off', async () => {
    const wrapper = mountScreen()
    const db = useDatabasesStore()

    expect(variantButtons(wrapper)).toHaveLength(2)
    expect(wrapper.text()).not.toContain('Single DB + YDB EM')
    expect(db.configVariant).toBe('single')
    wrapper.unmount()
  })

  it('enables the single-plus-em variant when the flag is on', async () => {
    const { api } = await import('@/api/client')
    vi.mocked(api.getRuntimeInfo).mockResolvedValue({ embedded: true, extracted: true, show_em_scenarios: true })
    await useRuntimeStore().load()

    const wrapper = mountScreen()
    const db = useDatabasesStore()

    expect(variantButtons(wrapper)).toHaveLength(3)
    const emBtn = variantButtons(wrapper)[2]
    expect(emBtn.attributes('disabled')).toBeUndefined()

    await emBtn.trigger('click')
    expect(db.configVariant).toBe('single-plus-em')
    wrapper.unmount()
  })

  it('enables single-plus-em without a bundled runtime', async () => {
    const { api } = await import('@/api/client')
    vi.mocked(api.getRuntimeInfo).mockResolvedValue({ embedded: false, extracted: false, show_em_scenarios: true })
    await useRuntimeStore().load()

    const wrapper = mountScreen()
    const db = useDatabasesStore()

    const emBtn = variantButtons(wrapper)[2]
    expect(emBtn.attributes('disabled')).toBeUndefined()

    await emBtn.trigger('click')
    expect(db.configVariant).toBe('single-plus-em')
    wrapper.unmount()
  })

  it('resets a persisted single-plus-em variant to single when the flag is off', async () => {
    const db = useDatabasesStore()
    const session = useSessionStore()

    // Simulate an existing inventory with single-plus-em before the screen
    // mounts; the hydration watch resets it on the immediate evaluation.
    db.setConfigVariant('single-plus-em')
    expect(db.configVariant).toBe('single-plus-em')

    const wrapper = mountScreen()
    await nextTick()

    expect(db.configVariant).toBe('single')
    expect(session.showEmStep).toBe(false)
    wrapper.unmount()
  })

  it('enables the multiple-DB variant without runtime flags', async () => {
    const wrapper = mountScreen()
    const db = useDatabasesStore()

    const multipleBtn = variantButtons(wrapper)[1]
    expect(multipleBtn.attributes('disabled')).toBeUndefined()

    await multipleBtn.trigger('click')
    expect(db.configVariant).toBe('multiple')
    wrapper.unmount()
  })

  it('preserves a multiple variant loaded before the screen mounts', async () => {
    const db = useDatabasesStore()
    db.setConfigVariant('multiple')
    const wrapper = mountScreen()
    await nextTick()

    const multipleBtn = variantButtons(wrapper)[1]
    expect(db.configVariant).toBe('multiple')
    expect(multipleBtn.classes()).toContain('is-active')
    wrapper.unmount()
  })
})
