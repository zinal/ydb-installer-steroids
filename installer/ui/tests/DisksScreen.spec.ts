import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/api/client', () => ({
  api: {
    getInventory: vi.fn(),
    updateInventory: vi.fn(),
    checkDisks: vi.fn().mockResolvedValue({
      results: [
        {
          host: 'node1.example.com',
          disks: [
            {
              name: 'vdb',
              path: '/dev/vdb',
              type: 'SSD',
              size: '500G',
              sizeBytes: 500 * 1024 * 1024 * 1024,
              mounted: false,
              mountPoint: '',
              filesystem: '',
              isSystemDisk: false,
            },
          ],
        },
      ],
    }),
  },
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn(),
  }),
}))

import { useDisksStore } from '@/stores/disks'
import { useHostsStore } from '@/stores/hosts'
import { useTopologyStore } from '@/stores/topology'
import { useSessionStore } from '@/stores/session'
import DisksScreen from '@/views/DisksScreen.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      common: { step: 'Step' },
      disks: {
        heading: 'Disks',
        topologyLabel: 'Topology: {topology}',
        entryMode: 'Method',
        entryModeAuto: 'Auto-discover',
        entryModeManual: 'Enter manually',
        manualEntryDesc: 'Enter device paths for YDB data storage.',
        manualEntryPlaceholder: '/dev/vdb\n/dev/vdc\n/dev/vdd',
        manualEntryHint: 'One path per line, starting with /dev/',
        manualPreview: 'Disks ({count}):',
        discovery: 'Disk Discovery',
        discoverDisks: 'Discover Disks',
        discovering: 'Discovering disks...',
        autoDiscovering: 'Auto-discovering disks...',
        consistencyWarning: 'Disk Consistency Warning',
        noHosts: 'No hosts',
        addHostsFirst: 'Add hosts first on the "Hosts & SSH" screen',
        selectAtLeastOne: 'Select at least one disk',
        selectAtLeastThree: 'Select at least 3 disks',
        selectAtLeastThreeManual: 'Enter at least 3 device paths',
        referenceHostNote: '(Selection applies to all hosts)',
        noDisks: 'No disks discovered',
        clickDiscover: 'Click "Discover Disks" to find available disks',
        device: 'Device',
        type: 'Type',
        size: 'Size',
        status: 'Status',
        free: 'Free',
        mounted: 'Mounted',
        hasFS: 'Has filesystem',
        total: 'Total',
        statusAvailable: 'Available',
        statusSystemDisk: 'System disk',
        statusMounted: 'Mounted',
        statusMountedAt: 'Mounted at {mountPoint}',
        smallDiskWarning: 'Disks smaller than 80 GB are not recommended for YDB',
        recheckDisks: 'Re-check Disks',
        validateDisks: 'Validate Disks',
        validating: 'Validating...',
        validationOk: 'All disks valid',
        validationErrors: 'Some disks invalid',
        systemRequirementsLink: 'System requirements',
      },
      screens: {
        '4': 'Disks',
      },
    },
    ru: {
      common: { step: 'Шаг' },
      disks: {
        heading: 'Диски',
        topologyLabel: 'Топология: {topology}',
        entryMode: 'Способ',
        entryModeAuto: 'Автоопределение',
        entryModeManual: 'Ввести вручную',
        manualEntryDesc: 'Укажите пути устройств для хранения данных YDB.',
        manualEntryPlaceholder: '/dev/vdb\n/dev/vdc\n/dev/vdd',
        manualEntryHint: 'Один путь на строку, начиная с /dev/',
        manualPreview: 'Диски ({count}):',
        discovery: 'Обнаружение дисков',
        discoverDisks: 'Обнаружить диски',
        discovering: 'Обнаружение дисков...',
        autoDiscovering: 'Автоматическое обнаружение дисков...',
        consistencyWarning: 'Предупреждение о согласованности дисков',
        noHosts: 'Нет хостов',
        addHostsFirst: 'Сначала добавьте хосты на экране «Хосты и SSH»',
        selectAtLeastOne: 'Выберите хотя бы один диск',
        selectAtLeastThree: 'Выберите минимум 3 диска',
        selectAtLeastThreeManual: 'Укажите не менее 3 путей к устройствам',
        referenceHostNote: '(Выбор применяется ко всем хостам)',
        noDisks: 'Диски не обнаружены',
        clickDiscover: 'Нажмите «Обнаружить диски» для поиска доступных дисков',
        device: 'Устройство',
        type: 'Тип',
        size: 'Размер',
        status: 'Статус',
        free: 'Свободен',
        mounted: 'Смонтирован',
        hasFS: 'Содержит ФС',
        total: 'Всего',
        statusAvailable: 'Доступен',
        statusSystemDisk: 'Системный диск',
        statusMounted: 'Смонтирован',
        statusMountedAt: 'Смонтирован в {mountPoint}',
        smallDiskWarning: 'Диски размером менее 80 ГБ не рекомендуются для YDB',
        recheckDisks: 'Перепроверить диски',
        validateDisks: 'Проверить диски',
        validating: 'Проверка...',
        validationOk: 'Все диски валидны',
        validationErrors: 'Некоторые диски невалидны',
        systemRequirementsLink: 'Системные требования',
      },
      screens: {
        '4': 'Диски',
      },
    },
  },
})

describe('DisksScreen — re-check disks button', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('does not show re-check button initially (no disks discovered)', () => {
    const wrapper = mount(DisksScreen, {
      global: { plugins: [i18n] },
    })
    const recheckBtn = wrapper.find('.recheck-btn')
    expect(recheckBtn.exists()).toBe(false)
  })

  it('shows re-check button after disks are discovered', async () => {
    const disksStore = useDisksStore()
    const hostsStore = useHostsStore()

    hostsStore.addDCHost('DC-1', 'node1.example.com')
    hostsStore.setSsh({
      login: 'root',
      auth_type: 'key',
      use_jump_host: false,
    })

    disksStore.setDiskResults('node1.example.com', [
      {
        name: 'vdb',
        path: '/dev/vdb',
        type: 'SSD',
        size: '500G',
        sizeBytes: 500 * 1024 * 1024 * 1024,
        mounted: false,
        mountPoint: '',
        filesystem: '',
        isSystemDisk: false,
      },
    ])

    const wrapper = mount(DisksScreen, {
      global: { plugins: [i18n] },
    })

    await wrapper.vm.$nextTick()

    const recheckBtn = wrapper.find('.recheck-btn')
    expect(recheckBtn.exists()).toBe(true)
  })

  it('shows re-check button after a check error', async () => {
    const disksStore = useDisksStore()
    const hostsStore = useHostsStore()

    hostsStore.addDCHost('DC-1', 'node1.example.com')
    hostsStore.setSsh({
      login: 'root',
      auth_type: 'key',
      use_jump_host: false,
    })

    disksStore.setDiscoveryError('Failed to discover disks')

    const wrapper = mount(DisksScreen, {
      global: { plugins: [i18n] },
    })

    await wrapper.vm.$nextTick()

    const recheckBtn = wrapper.find('.recheck-btn')
    expect(recheckBtn.exists()).toBe(true)
  })

  it('stale inventory disks are removed after discovery', async () => {
    const disksStore = useDisksStore()
    const hostsStore = useHostsStore()

    disksStore.toggleDisk('/dev/vdb', 'ydb_disk_1')
    disksStore.toggleDisk('/dev/vdc', 'ydb_disk_2')
    disksStore.toggleDisk('/dev/vdd', 'ydb_disk_3')
    expect(disksStore.autoSelectedDisks).toHaveLength(3)

    hostsStore.addDCHost('DC-1', 'node1.example.com')
    hostsStore.setSsh({
      login: 'root',
      auth_type: 'key',
      use_jump_host: false,
    })

    disksStore.setDiskResults('node1.example.com', [
      {
        name: 'vdb',
        path: '/dev/vdb',
        type: 'SSD',
        size: '500G',
        sizeBytes: 500 * 1024 * 1024 * 1024,
        mounted: false,
        mountPoint: '',
        filesystem: '',
        isSystemDisk: false,
      },
      {
        name: 'vdc',
        path: '/dev/vdc',
        type: 'SSD',
        size: '500G',
        sizeBytes: 500 * 1024 * 1024 * 1024,
        mounted: false,
        mountPoint: '',
        filesystem: '',
        isSystemDisk: false,
      },
    ])

    disksStore.reconcileSelection()

    expect(disksStore.autoSelectedDisks).toHaveLength(2)
    const paths = disksStore.autoSelectedDisks.map(d => d.path)
    expect(paths).toContain('/dev/vdb')
    expect(paths).toContain('/dev/vdc')
    expect(paths).not.toContain('/dev/vdd')
  })
})

describe('DisksScreen — mirror-3-dc disk minimum', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  function addHostsPerDC(count: number) {
    const hosts = useHostsStore()
    for (let i = 0; i < count; i++) {
      hosts.addDCHost('DC-1', `a${i + 1}.test`)
      hosts.addDCHost('DC-2', `b${i + 1}.test`)
      hosts.addDCHost('DC-3', `d${i + 1}.test`)
    }
  }

  it('9 hosts: a single auto-selected disk enables Next', async () => {
    const topology = useTopologyStore()
    const disks = useDisksStore()
    const session = useSessionStore()

    topology.select('mirror-3-dc')
    addHostsPerDC(3)

    disks.setEntryMode('auto')
    disks.toggleDisk('/dev/vdb', 'ydb_disk_1')

    const wrapper = mount(DisksScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()

    expect(session.canGoNext).toBe(true)
  })

  it('6 hosts: a single auto-selected disk does not enable Next', async () => {
    const topology = useTopologyStore()
    const disks = useDisksStore()
    const session = useSessionStore()

    topology.select('mirror-3-dc')
    addHostsPerDC(2)

    disks.setEntryMode('auto')
    disks.toggleDisk('/dev/vdb', 'ydb_disk_1')

    const wrapper = mount(DisksScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()

    expect(session.canGoNext).toBe(false)
  })

  it('9 hosts: a single manual path enables Next', async () => {
    const topology = useTopologyStore()
    const disks = useDisksStore()
    const session = useSessionStore()

    topology.select('mirror-3-dc')
    addHostsPerDC(3)

    disks.setEntryMode('manual')
    disks.setManualDiskInput('/dev/vdb')

    const wrapper = mount(DisksScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()

    expect(session.canGoNext).toBe(true)
  })

  it('6 hosts: a single manual path does not enable Next', async () => {
    const topology = useTopologyStore()
    const disks = useDisksStore()
    const session = useSessionStore()

    topology.select('mirror-3-dc')
    addHostsPerDC(2)

    disks.setEntryMode('manual')
    disks.setManualDiskInput('/dev/vdb')

    const wrapper = mount(DisksScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()

    expect(session.canGoNext).toBe(false)
  })

  it('block-4-2: a single manual path enables Next', async () => {
    const topology = useTopologyStore()
    const disks = useDisksStore()
    const session = useSessionStore()

    topology.select('block-4-2')
    addHostsPerDC(1)

    disks.setEntryMode('manual')
    disks.setManualDiskInput('/dev/vdb')

    const wrapper = mount(DisksScreen, { global: { plugins: [i18n] } })
    await wrapper.vm.$nextTick()

    expect(session.canGoNext).toBe(true)
  })
})

describe('DisksScreen — system requirements docs link', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    i18n.global.locale.value = 'en'
  })

  it('renders the docs link with the English URL and visible text', () => {
    const wrapper = mount(DisksScreen, { global: { plugins: [i18n] } })
    const link = wrapper.find('.disks-doc-link')

    expect(link.exists()).toBe(true)
    expect(link.attributes('href')).toBe(
      'https://ydb.tech/docs/en/devops/concepts/system-requirements',
    )
    expect(link.attributes('target')).toBe('_blank')
    expect(link.text()).toContain('System requirements')
  })

  it('switches the docs URL to Russian when the locale changes', async () => {
    const wrapper = mount(DisksScreen, { global: { plugins: [i18n] } })
    const link = wrapper.find('.disks-doc-link')

    expect(link.attributes('href')).toBe(
      'https://ydb.tech/docs/en/devops/concepts/system-requirements',
    )

    i18n.global.locale.value = 'ru'
    await wrapper.vm.$nextTick()

    expect(link.attributes('href')).toBe(
      'https://ydb.tech/docs/ru/devops/concepts/system-requirements',
    )
  })
})
