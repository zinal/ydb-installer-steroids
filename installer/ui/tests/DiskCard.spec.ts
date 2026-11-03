import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import DiskCard from '@/components/disks/DiskCard.vue'
import type { DiskInfo } from '@/types/disks'
import { MIN_RECOMMENDED_DISK_SIZE_BYTES } from '@/types/disks'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      disks: {
        statusAvailable: 'Available',
        statusSystemDisk: 'System disk',
        statusMounted: 'Mounted',
        smallDiskWarning: 'Disks smaller than 80 GB are not recommended for YDB',
      },
    },
  },
})

function createDisk(overrides: Partial<DiskInfo> = {}): DiskInfo {
  return {
    name: 'vda',
    path: '/dev/vda',
    type: 'SSD',
    size: '100G',
    sizeBytes: 100 * 1024 * 1024 * 1024,
    mounted: false,
    mountPoint: '',
    filesystem: '',
    isSystemDisk: false,
    ...overrides,
  }
}

describe('DiskCard', () => {
  it('does not show warning icon for disks >= 80 GB', () => {
    const disk = createDisk({ sizeBytes: 100 * 1024 * 1024 * 1024 })
    const wrapper = mount(DiskCard, {
      props: { disk, selected: false },
      global: { plugins: [i18n] },
    })
    const warnIcon = wrapper.find('.disk-card__small-warn')
    expect(warnIcon.exists()).toBe(false)
  })

  it('does not show warning icon for disks at exactly 80 GB', () => {
    const disk = createDisk({ sizeBytes: MIN_RECOMMENDED_DISK_SIZE_BYTES })
    const wrapper = mount(DiskCard, {
      props: { disk, selected: false },
      global: { plugins: [i18n] },
    })
    const warnIcon = wrapper.find('.disk-card__small-warn')
    expect(warnIcon.exists()).toBe(false)
  })

  it('shows warning icon for disks smaller than 80 GB', () => {
    const disk = createDisk({ sizeBytes: 50 * 1024 * 1024 * 1024, size: '50G' })
    const wrapper = mount(DiskCard, {
      props: { disk, selected: false },
      global: { plugins: [i18n] },
    })
    const warnIcon = wrapper.find('.disk-card__small-warn')
    expect(warnIcon.exists()).toBe(true)
  })

  it('does not show warning icon when sizeBytes is 0 (unknown size)', () => {
    const disk = createDisk({ sizeBytes: 0, size: '0' })
    const wrapper = mount(DiskCard, {
      props: { disk, selected: false },
      global: { plugins: [i18n] },
    })
    const warnIcon = wrapper.find('.disk-card__small-warn')
    expect(warnIcon.exists()).toBe(false)
  })

  it('shows warning icon for very small disks (e.g. 10 GB)', () => {
    const disk = createDisk({ sizeBytes: 10 * 1024 * 1024 * 1024, size: '10G' })
    const wrapper = mount(DiskCard, {
      props: { disk, selected: false },
      global: { plugins: [i18n] },
    })
    const warnIcon = wrapper.find('.disk-card__small-warn')
    expect(warnIcon.exists()).toBe(true)
  })

  it('does not show warning icon for large disks (e.g. 1 TB)', () => {
    const disk = createDisk({ sizeBytes: 1024 * 1024 * 1024 * 1024, size: '1T' })
    const wrapper = mount(DiskCard, {
      props: { disk, selected: false },
      global: { plugins: [i18n] },
    })
    const warnIcon = wrapper.find('.disk-card__small-warn')
    expect(warnIcon.exists()).toBe(false)
  })
})
