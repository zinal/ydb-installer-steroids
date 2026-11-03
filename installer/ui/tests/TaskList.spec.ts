import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import TaskList from '@/components/common/TaskList.vue'
import type { Task } from '@/types/install'

const tasks: Task[] = [
  { id: 1, name: 'Stop Dynamic nodes', status: 'done' },
  { id: 2, name: 'Stop Storage nodes', status: 'running' },
  { id: 3, name: 'Delete disk data', status: 'pending' },
  { id: 4, name: 'Delete config files', status: 'error' },
]

describe('TaskList', () => {
  it('renders all tasks', () => {
    const wrapper = mount(TaskList, { props: { tasks } })
    const items = wrapper.findAll('.task-item')
    expect(items).toHaveLength(4)
  })

  it('renders task names', () => {
    const wrapper = mount(TaskList, { props: { tasks } })
    expect(wrapper.text()).toContain('Stop Dynamic nodes')
    expect(wrapper.text()).toContain('Stop Storage nodes')
    expect(wrapper.text()).toContain('Delete disk data')
    expect(wrapper.text()).toContain('Delete config files')
  })

  it('applies correct status class to each item', () => {
    const wrapper = mount(TaskList, { props: { tasks } })
    const items = wrapper.findAll('.task-item')
    expect(items[0].classes()).toContain('task-item--done')
    expect(items[1].classes()).toContain('task-item--running')
    expect(items[2].classes()).toContain('task-item--pending')
    expect(items[3].classes()).toContain('task-item--error')
  })

  it('renders task numbers', () => {
    const wrapper = mount(TaskList, { props: { tasks } })
    expect(wrapper.text()).toContain('1.')
    expect(wrapper.text()).toContain('2.')
    expect(wrapper.text()).toContain('3.')
    expect(wrapper.text()).toContain('4.')
  })

  it('renders empty list without errors', () => {
    const wrapper = mount(TaskList, { props: { tasks: [] } })
    expect(wrapper.find('.task-list').exists()).toBe(true)
    expect(wrapper.findAll('.task-item')).toHaveLength(0)
  })

  it('shows spinner for running task', () => {
    const runningTask: Task[] = [{ id: 1, name: 'Running task', status: 'running' }]
    const wrapper = mount(TaskList, { props: { tasks: runningTask } })
    expect(wrapper.findComponent({ name: 'VProgressCircular' }).exists()).toBe(true)
  })

  it('shows pending dot for pending task', () => {
    const pendingTask: Task[] = [{ id: 1, name: 'Pending task', status: 'pending' }]
    const wrapper = mount(TaskList, { props: { tasks: pendingTask } })
    expect(wrapper.find('.task-dot--pending').exists()).toBe(true)
  })
})
