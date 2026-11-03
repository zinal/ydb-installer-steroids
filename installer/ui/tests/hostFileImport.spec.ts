import { describe, it, expect } from 'vitest'
import { useHostFileImport } from '@/composables/useHostFileImport'

describe('useHostFileImport', () => {
  it('parses hosts from text content', async () => {
    const { readHostFile } = useHostFileImport()
    const file = new File(['host1\nhost2\nhost3'], 'hosts.txt', { type: 'text/plain' })
    const hosts = await readHostFile(file)
    expect(hosts).toEqual(['host1', 'host2', 'host3'])
  })

  it('skips empty lines and comments', async () => {
    const { readHostFile } = useHostFileImport()
    const content = '\n  \nhost1\n# comment\nhost2\n'
    const file = new File([content], 'hosts.txt', { type: 'text/plain' })
    const hosts = await readHostFile(file)
    expect(hosts).toEqual(['host1', 'host2'])
  })

  it('trims whitespace from each line', async () => {
    const { readHostFile } = useHostFileImport()
    const content = '  host1  \n\thost2\t\n  host3  '
    const file = new File([content], 'hosts.txt', { type: 'text/plain' })
    const hosts = await readHostFile(file)
    expect(hosts).toEqual(['host1', 'host2', 'host3'])
  })

  it('returns error for empty file', async () => {
    const { readHostFile, importError } = useHostFileImport()
    const file = new File([''], 'empty.txt', { type: 'text/plain' })
    const hosts = await readHostFile(file)
    expect(hosts).toEqual([])
    expect(importError.value).toBe('hosts.fileEmpty')
  })

  it('returns error for whitespace-only file', async () => {
    const { readHostFile, importError } = useHostFileImport()
    const file = new File(['   \n\n   \n'], 'whitespace.txt', { type: 'text/plain' })
    const hosts = await readHostFile(file)
    expect(hosts).toEqual([])
    expect(importError.value).toBe('hosts.fileEmpty')
  })

  it('returns error for comment-only file', async () => {
    const { readHostFile, importError } = useHostFileImport()
    const file = new File(['# comment1\n# comment2\n'], 'comments.txt', { type: 'text/plain' })
    const hosts = await readHostFile(file)
    expect(hosts).toEqual([])
    expect(importError.value).toBe('hosts.fileEmpty')
  })

  it('handles mixed content with hosts and comments', async () => {
    const { readHostFile } = useHostFileImport()
    const content = '# List of hosts\nhost1.example.com\n\n# More hosts\nhost2.example.com\nhost3.example.com\n'
    const file = new File([content], 'mixed.txt', { type: 'text/plain' })
    const hosts = await readHostFile(file)
    expect(hosts).toEqual(['host1.example.com', 'host2.example.com', 'host3.example.com'])
  })

  it('sets importing state during read', async () => {
    const { readHostFile, importing } = useHostFileImport()
    const file = new File(['host1'], 'hosts.txt', { type: 'text/plain' })
    expect(importing.value).toBe(false)
    const promise = readHostFile(file)
    expect(importing.value).toBe(true)
    await promise
    expect(importing.value).toBe(false)
  })
})
