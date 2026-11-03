import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

vi.mock('@/api/client', () => ({
  api: {
    getInventory: vi.fn().mockResolvedValue({ inventory_found: false, data: null }),
    updateInventory: vi.fn().mockResolvedValue({ status: 'ok' }),
    getEMInventory: vi.fn().mockResolvedValue({ em_inventory_found: false, data: null }),
    updateEMInventory: vi.fn().mockResolvedValue({ status: 'ok' }),
  },
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

import { useEmStore } from '@/stores/em'
import { useYdbConfigStore } from '@/stores/ydbConfig'
import { useSessionStore } from '@/stores/session'
import EmParamsScreen from '@/views/EmParamsScreen.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      common: { step: 'Step' },
      emParams: {
        heading: 'YDB EM Parameters',
        subtitle: 'Configure settings.',
        credentialsLabel: 'YDB credentials',
        emDbLabel: 'YDB EM database',
        user: 'YDB user',
        userPlaceholder: 'root',
        password: 'YDB password',
        passwordPlaceholder: 'Enter or generate',
        generatePassword: 'Generate',
        dbRoot: 'Database root',
        dbRootPlaceholder: '/Root',
        dbName: 'Database name',
        dbNamePlaceholder: 'db',
        emHostRequired: 'EM host required',
        credentialsRequired: 'Enter credentials',
        fieldsRequired: 'Fill database root and database name',
      },
    },
  },
})

function fillFields() {
  const em = useEmStore()
  em.dbConnectionDBRoot = '/Root'
  em.dbConnectionDB = 'db'
  em.hosts = [{ fqdn: 'em.example.com' }]
  useYdbConfigStore().user = 'root'
}

describe('EmParamsScreen', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  function mountScreen() {
    return mount(EmParamsScreen, {
      global: {
        plugins: [i18n],
        stubs: { HelpTooltip: true },
      },
    })
  }

  it('renders credentials and EM database sections (no AZ)', async () => {
    const wrapper = mountScreen()
    await nextTick()
    expect(wrapper.text()).toContain('YDB EM Parameters')
    expect(wrapper.text()).toContain('YDB credentials')
    expect(wrapper.text()).toContain('YDB EM database')
    expect(wrapper.text()).toContain('YDB user')
    expect(wrapper.text()).toContain('YDB password')
    expect(wrapper.text()).toContain('Database root')
  })

  it('does not render AZ input section', async () => {
    const wrapper = mountScreen()
    await nextTick()
    expect(wrapper.text()).not.toContain('Availability zones')
  })

  it('typing updates ydbConfig and em stores', async () => {
    const wrapper = mountScreen()
    await nextTick()
    const inputs = wrapper.findAll('input.ssh-input')
    expect(inputs.length).toBe(4)
    await inputs[0].setValue('admin')
    await inputs[1].setValue('secret')
    await inputs[2].setValue('/Root')
    await inputs[3].setValue('mydb')
    await nextTick()
    expect(useYdbConfigStore().user).toBe('admin')
    expect(useYdbConfigStore().password).toBe('secret')
    expect(useEmStore().dbConnectionDB).toBe('mydb')
  })

  it('disables Next when fields are incomplete', async () => {
    const wrapper = mountScreen()
    await nextTick()
    const session = useSessionStore()
    expect(session.canGoNext).toBe(false)
  })

  it('enables Next when all fields are filled', async () => {
    const wrapper = mountScreen()
    await nextTick()
    fillFields()
    await nextTick()
    const session = useSessionStore()
    expect(session.canGoNext).toBe(true)
  })

  it('disables Next when EM host is missing', async () => {
    const wrapper = mountScreen()
    await nextTick()
    const em = useEmStore()
    em.dbConnectionDBRoot = '/Root'
    em.dbConnectionDB = 'db'
    useYdbConfigStore().user = 'root'
    await nextTick()
    expect(useSessionStore().canGoNext).toBe(false)
    expect(wrapper.text()).toContain('EM host required')
  })

  it('disables Next when credentials are missing', async () => {
    const wrapper = mountScreen()
    await nextTick()
    const em = useEmStore()
    em.dbConnectionDBRoot = '/Root'
    em.dbConnectionDB = 'db'
    em.hosts = [{ fqdn: 'em.example.com' }]
    await nextTick()
    expect(useSessionStore().canGoNext).toBe(false)
    expect(wrapper.text()).toContain('Enter credentials')
  })

  it('disables Next when DB fields are missing', async () => {
    const wrapper = mountScreen()
    await nextTick()
    const em = useEmStore()
    em.hosts = [{ fqdn: 'em.example.com' }]
    useYdbConfigStore().user = 'root'
    await nextTick()
    expect(useSessionStore().canGoNext).toBe(false)
    expect(wrapper.text()).toContain('Fill database root and database name')
  })
})
