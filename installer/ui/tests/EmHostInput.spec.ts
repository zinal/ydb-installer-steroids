import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import EmHostInput from '@/components/hosts/EmHostInput.vue'
import type { EMHost } from '@/types/inventory'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      emHosts: {
        hostnameLabel: 'Hostname / FQDN',
        hostnamePlaceholder: 'hostname.example.com',
        addHost: 'Add host',
        removeHost: 'Remove host',
      },
    },
  },
})

function mountInput(hosts: EMHost[], maxHosts = 1) {
  return mount(EmHostInput, {
    props: { hosts, maxHosts },
    global: { plugins: [i18n] },
  })
}

describe('EmHostInput', () => {
  it('renders host rows without an empty-state message', () => {
    const wrapper = mountInput([{ fqdn: '' }])
    expect(wrapper.find('.em-host-empty').exists()).toBe(false)
    expect(wrapper.find('.em-host-row').exists()).toBe(true)
    expect(wrapper.find('.em-host-input').exists()).toBe(true)
  })

  it('shows Hostname / FQDN label once even with multiple hosts', () => {
    const wrapper = mountInput(
      [{ fqdn: 'a.example.com' }, { fqdn: 'b.example.com' }],
      5,
    )
    const labels = wrapper.findAll('.em-host-label')
    expect(labels).toHaveLength(1)
    expect(labels[0].text()).toBe('Hostname / FQDN')
    expect(wrapper.findAll('.em-host-row')).toHaveLength(2)
  })

  it('hides Add button when hosts.length >= maxHosts', () => {
    const wrapper = mountInput([{ fqdn: 'em.example.com' }], 1)
    expect(wrapper.find('.em-host-add').exists()).toBe(false)
  })

  it('shows Add button when under maxHosts', () => {
    const wrapper = mountInput([{ fqdn: 'a.example.com' }], 3)
    expect(wrapper.find('.em-host-add').exists()).toBe(true)
  })

  it('emits add when Add host is clicked', async () => {
    const wrapper = mountInput([{ fqdn: 'a.example.com' }], 3)
    await wrapper.find('.em-host-add').trigger('click')
    expect(wrapper.emitted('add')).toHaveLength(1)
  })

  it('renders FQDN input and emits update on input', async () => {
    const wrapper = mountInput([{ fqdn: 'old.example.com' }], 1)
    const input = wrapper.find('.em-host-input')
    expect((input.element as HTMLInputElement).value).toBe('old.example.com')

    await input.setValue('new.example.com')
    const updates = wrapper.emitted('update')
    expect(updates).toBeTruthy()
    expect(updates![updates!.length - 1]).toEqual([0, 'new.example.com'])
  })

  it('hides remove when only one host remains', () => {
    const wrapper = mountInput([{ fqdn: 'em.example.com' }], 1)
    expect(wrapper.find('.em-host-remove').exists()).toBe(false)
  })

  it('shows remove and emits remove when multiple hosts exist', async () => {
    const wrapper = mountInput(
      [{ fqdn: 'a.example.com' }, { fqdn: 'b.example.com' }],
      5,
    )
    expect(wrapper.findAll('.em-host-remove')).toHaveLength(2)
    await wrapper.findAll('.em-host-remove')[0].trigger('click')
    expect(wrapper.emitted('remove')).toEqual([[0]])
  })

  it('treats maxHosts <= 0 as unlimited', () => {
    const wrapper = mountInput(
      [{ fqdn: 'a' }, { fqdn: 'b' }, { fqdn: 'c' }],
      0,
    )
    expect(wrapper.find('.em-host-add').exists()).toBe(true)
    expect(wrapper.findAll('.em-host-row')).toHaveLength(3)
  })

  it('shows 1-based index badges', () => {
    const wrapper = mountInput(
      [{ fqdn: 'a.example.com' }, { fqdn: 'b.example.com' }],
      5,
    )
    const indexes = wrapper.findAll('.em-host-index').map(n => n.text())
    expect(indexes).toEqual(['1', '2'])
  })
})
