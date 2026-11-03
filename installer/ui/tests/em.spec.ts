import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useEmStore } from '@/stores/em'
import type { EMConfig } from '@/types/inventory'

const tlsGenerate = vi.fn()
const tlsValidate = vi.fn()
const tlsExists = vi.fn()

vi.mock('@/api/client', () => ({
  api: {
    validateFile: vi.fn().mockResolvedValue({ valid: true, size: '1MB' }),
    tlsGenerate: (...args: unknown[]) => tlsGenerate(...args),
    tlsValidate: (...args: unknown[]) => tlsValidate(...args),
    tlsExists: (...args: unknown[]) => tlsExists(...args),
  },
}))

describe('useEmStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('has initial empty state', () => {
    const store = useEmStore()
    expect(store.hosts).toEqual([])
    expect(store.total).toBe(0)
    expect(store.hasDuplicates).toEqual([])
    expect(store.emInventoryFound).toBe(false)
  })

  it('addEmptyHost appends an empty fillable row', () => {
    const store = useEmStore()
    store.addEmptyHost()
    expect(store.hosts).toEqual([{ fqdn: '' }])
    expect(store.total).toBe(1)
  })

  it('allows multiple empty rows via addEmptyHost', () => {
    const store = useEmStore()
    store.addEmptyHost()
    store.addEmptyHost()
    expect(store.hosts).toHaveLength(2)
    expect(store.hosts.every(h => h.fqdn === '')).toBe(true)
  })

  it('addHost trims and rejects empty or duplicate FQDNs', () => {
    const store = useEmStore()
    store.addHost('  em.example.com  ')
    expect(store.hosts).toEqual([{ fqdn: 'em.example.com' }])

    store.addHost('em.example.com')
    expect(store.hosts).toHaveLength(1)

    store.addHost('')
    store.addHost('   ')
    expect(store.hosts).toHaveLength(1)
  })

  it('updateHost updates FQDN and allows clearing to empty', () => {
    const store = useEmStore()
    store.addEmptyHost()
    store.updateHost(0, '  node1.example.com ')
    expect(store.hosts[0].fqdn).toBe('node1.example.com')

    store.updateHost(0, '')
    expect(store.hosts[0].fqdn).toBe('')
  })

  it('updateHost rejects non-empty duplicates', () => {
    const store = useEmStore()
    store.addHost('a.example.com')
    store.addHost('b.example.com')
    store.updateHost(1, 'a.example.com')
    expect(store.hosts[1].fqdn).toBe('b.example.com')
  })

  it('updateHost preserves location when location arg is omitted', () => {
    const store = useEmStore()
    store.addHost('em.example.com', 'az1')
    store.updateHost(0, 'em2.example.com')
    expect(store.hosts[0]).toEqual({ fqdn: 'em2.example.com', location: 'az1' })
  })

  it('removeHost removes by index', () => {
    const store = useEmStore()
    store.addHost('a.example.com')
    store.addHost('b.example.com')
    store.removeHost(0)
    expect(store.hosts).toEqual([{ fqdn: 'b.example.com' }])
    store.removeHost(0)
    expect(store.hosts).toEqual([])
  })

  it('hasDuplicates returns duplicate non-empty FQDNs', () => {
    const store = useEmStore()
    store.addEmptyHost()
    store.addEmptyHost()
    store.hosts = [
      { fqdn: 'dup.example.com' },
      { fqdn: 'dup.example.com' },
      { fqdn: 'other.example.com' },
      { fqdn: '' },
    ]
    expect(store.hasDuplicates).toEqual(['dup.example.com'])
  })

  it('toEMConfig filters empty hosts', () => {
    const store = useEmStore()
    store.addEmptyHost()
    store.addHost('em.example.com')
    store.addEmptyHost()
    const cfg = store.toEMConfig()
    expect(cfg.hosts).toEqual([{ fqdn: 'em.example.com' }])
  })

  it('applyFromEMConfig / toEMConfig round-trip', () => {
    const store = useEmStore()
    const input: EMConfig = {
      hosts: [{ fqdn: 'em1.example.com', location: 'dc1' }],
      cluster_name: 'prod',
      db_connection_db_root: '/Root',
      db_connection_db: '/Root/db',
      db_connection_endpoint: 'grpcs://em1.example.com:2135',
      meta_ui_binary: '/opt/meta',
      cp_binary: '/opt/cp',
      cp_agent_binary: '/opt/agent',
      agent_cpu: 2,
      agent_memory: 4,
      use_dynamic_config: true,
    }
    store.applyFromEMConfig(input)
    expect(store.hosts).toEqual(input.hosts)
    expect(store.clusterName).toBe('prod')
    expect(store.useDynamicConfig).toBe(true)
    expect(store.dbConnectionEndpoint).toBe('grpcs://em1.example.com:2135')
    expect(store.distMode).toBe('binaries')

    const out = store.toEMConfig()
    expect(out.hosts).toEqual(input.hosts)
    expect(out.cluster_name).toBe('prod')
    expect(out.db_connection_db_root).toBe('/Root')
    expect(out.db_connection_endpoint).toBe('grpcs://em1.example.com:2135')
    expect(out.agent_cpu).toBe(2)
    expect(out.use_dynamic_config).toBe(true)
    expect(out.meta_ui_binary).toBe('/opt/meta')
    expect(out.cp_binary).toBe('/opt/cp')
    expect(out.cp_agent_binary).toBe('/opt/agent')
    expect(out.archive_path).toBeUndefined()
    expect(out.az_locations).toBeUndefined()
  })

  it('defaults to archive distMode and distCanProceed false', () => {
    const store = useEmStore()
    expect(store.distMode).toBe('archive')
    expect(store.archivePath).toBe('')
    expect(store.distCanProceed).toBe(false)
  })

  it('applyFromEMConfig with archives sets archive mode', () => {
    const store = useEmStore()
    store.applyFromEMConfig({
      archive_path: '/opt/dist/ydb-em.tar.gz',
      meta_ui_archive: '/opt/dist/ui.tar.gz',
    })
    expect(store.distMode).toBe('archive')
    expect(store.archivePath).toBe('/opt/dist/ydb-em.tar.gz')
    expect(store.metaUIArchivePath).toBe('/opt/dist/ui.tar.gz')
  })

  it('strips Jinja paths and does not force binaries mode or wipe store', () => {
    const store = useEmStore()
    store.distMode = 'archive'
    store.metaUIBinary = '/keep/me'
    store.applyFromEMConfig({
      meta_ui_binary: '{{ ansible_config_file | dirname }}/files/ydb-em-gateway',
      cp_binary: '{{ ansible_config_file | dirname }}/files/ydb-em-cp',
      cp_agent_binary: '{{ ansible_config_file | dirname }}/files/ydb-em-agent',
    })
    expect(store.metaUIBinary).toBe('/keep/me')
    expect(store.distMode).toBe('archive')
  })

  it('applyFromEMConfig archive payload does not clear store binaries', () => {
    const store = useEmStore()
    store.distMode = 'binaries'
    store.metaUIBinary = '/b/gateway'
    store.cpBinary = '/b/cp'
    store.cpAgentBinary = '/b/agent'
    store.applyFromEMConfig({
      archive_path: '/a/em.tar.gz',
      meta_ui_archive: '/a/ui.tar.gz',
    })
    expect(store.distMode).toBe('archive')
    expect(store.archivePath).toBe('/a/em.tar.gz')
    expect(store.metaUIBinary).toBe('/b/gateway')
    expect(store.cpBinary).toBe('/b/cp')
    expect(store.cpAgentBinary).toBe('/b/agent')
  })

  it('applyFromEMConfig binaries payload does not clear store archives', () => {
    const store = useEmStore()
    store.distMode = 'archive'
    store.archivePath = '/a/em.tar.gz'
    store.metaUIArchivePath = '/a/ui.tar.gz'
    store.applyFromEMConfig({
      meta_ui_binary: '/b/gateway',
      cp_binary: '/b/cp',
      cp_agent_binary: '/b/agent',
    })
    expect(store.distMode).toBe('binaries')
    expect(store.metaUIBinary).toBe('/b/gateway')
    expect(store.archivePath).toBe('/a/em.tar.gz')
    expect(store.metaUIArchivePath).toBe('/a/ui.tar.gz')
  })

  it('toEMConfig archive mode emits archives and omits binaries', () => {
    const store = useEmStore()
    store.distMode = 'archive'
    store.archivePath = '/a/em.tar.gz'
    store.metaUIArchivePath = '/a/ui.tar.gz'
    store.metaUIBinary = '/b/gateway'
    store.cpBinary = '/b/cp'
    const out = store.toEMConfig()
    expect(out.archive_path).toBe('/a/em.tar.gz')
    expect(out.meta_ui_archive).toBe('/a/ui.tar.gz')
    expect(out.meta_ui_binary).toBeUndefined()
    expect(out.cp_binary).toBeUndefined()
  })

  it('toEMConfig binaries mode emits binaries and omits archives', () => {
    const store = useEmStore()
    store.distMode = 'binaries'
    store.archivePath = '/a/em.tar.gz'
    store.metaUIArchivePath = '/a/ui.tar.gz'
    store.metaUIBinary = '/b/gateway'
    store.cpBinary = '/b/cp'
    store.cpAgentBinary = '/b/agent'
    const out = store.toEMConfig()
    expect(out.archive_path).toBeUndefined()
    expect(out.meta_ui_archive).toBeUndefined()
    expect(out.meta_ui_binary).toBe('/b/gateway')
    expect(out.cp_binary).toBe('/b/cp')
    expect(out.cp_agent_binary).toBe('/b/agent')
  })

  it('distCanProceed requires validated paths for active mode', () => {
    const store = useEmStore()
    store.distMode = 'archive'
    expect(store.distCanProceed).toBe(false)
    store.archiveValid = true
    expect(store.distCanProceed).toBe(false)
    store.metaUIArchiveValid = true
    expect(store.distCanProceed).toBe(true)

    store.distMode = 'binaries'
    expect(store.distCanProceed).toBe(false)
    store.metaUIBinaryValid = true
    store.cpBinaryValid = true
    store.cpAgentBinaryValid = true
    expect(store.distCanProceed).toBe(true)
  })

  it('setDistMode preserves opposite-mode path values', () => {
    const store = useEmStore()
    store.archivePath = '/a/em.tar.gz'
    store.metaUIArchivePath = '/a/ui.tar.gz'
    store.metaUIBinary = '/b/gateway'
    store.cpBinary = '/b/cp'
    store.cpAgentBinary = '/b/agent'
    store.setDistMode('binaries')
    expect(store.archivePath).toBe('/a/em.tar.gz')
    store.setDistMode('archive')
    expect(store.metaUIBinary).toBe('/b/gateway')
  })

  it('setEMInventoryFound updates flag', () => {
    const store = useEmStore()
    store.setEMInventoryFound(true)
    expect(store.emInventoryFound).toBe(true)
  })

  it('firstEmHost and derivedEndpoint use first non-empty ydb_em host', () => {
    const store = useEmStore()
    store.hosts = [{ fqdn: '' }, { fqdn: 'em.example.com' }]
    expect(store.firstEmHost).toBe('em.example.com')
    expect(store.derivedEndpoint).toBe('grpcs://em.example.com:2135')
  })

  it('databasePath joins root and db name', () => {
    const store = useEmStore()
    store.dbConnectionDBRoot = '/Root'
    store.dbConnectionDB = 'db'
    expect(store.databasePath).toBe('/Root/db')
    store.dbConnectionDB = '/Root/custom'
    expect(store.databasePath).toBe('/Root/custom')
  })

  it('ensureParamsDefaults fills empty DB fields', () => {
    const store = useEmStore()
    store.ensureParamsDefaults()
    expect(store.dbConnectionDBRoot).toBe('/Root')
    expect(store.dbConnectionDB).toBe('db')
  })

  it('paramsFieldsValid requires DB root and name', () => {
    const store = useEmStore()
    expect(store.paramsFieldsValid).toBe(false)

    store.dbConnectionDBRoot = '/Root'
    expect(store.paramsFieldsValid).toBe(false)

    store.dbConnectionDB = 'db'
    expect(store.paramsFieldsValid).toBe(true)
  })

  it('defaults tlsMode generate and tlsCanProceed false', () => {
    const store = useEmStore()
    expect(store.tlsMode).toBe('generate')
    expect(store.tlsCertPath).toBe('')
    expect(store.tlsDays).toBe(365)
    expect(store.tlsCanProceed).toBe(false)
  })

  it('setTlsMode preserves generation and validation results', () => {
    const store = useEmStore()
    store.tlsGenerationResult = {
      success: true,
      outputDir: '/certs',
      caFile: '/certs/ca.crt',
      nodeFiles: [],
    }
    store.tlsValidationResult = {
      valid: true,
      caValid: true,
      nodeFiles: [],
      missing: [],
      errors: [],
      totalNodes: 1,
      foundNodes: 1,
    }
    store.setTlsMode('existing')
    expect(store.tlsMode).toBe('existing')
    expect(store.tlsGenerationResult?.success).toBe(true)
    expect(store.tlsValidationResult?.valid).toBe(true)
    store.setTlsMode('generate')
    expect(store.tlsGenerationResult?.success).toBe(true)
  })

  it('setTlsCertPath invalidates results when path changes', () => {
    const store = useEmStore()
    store.tlsGenerationResult = {
      success: true,
      outputDir: '/a',
      caFile: '/a/ca.crt',
      nodeFiles: [],
    }
    store.tlsValidationResult = {
      valid: true,
      caValid: true,
      nodeFiles: [],
      missing: [],
      errors: [],
      totalNodes: 0,
      foundNodes: 0,
    }
    store.tlsCertPath = '/a'
    store.setTlsCertPath('/b')
    expect(store.tlsCertPath).toBe('/b')
    expect(store.tlsGenerationResult).toBeNull()
    expect(store.tlsValidationResult).toBeNull()
  })

  it('toEMConfig emits em_tls_cert_path when set', () => {
    const store = useEmStore()
    store.tlsCertPath = 'TLS/CA/certs/certs-per-host'
    const out = store.toEMConfig()
    expect(out.em_tls_cert_path).toBe('TLS/CA/certs/certs-per-host')
  })

  it('applyFromEMConfig reads em_tls_cert_path and preserves generate mode', () => {
    const store = useEmStore()
    store.applyFromEMConfig({
      em_tls_cert_path: 'TLS/CA/certs/certs-per-host',
    })
    expect(store.tlsCertPath).toBe('TLS/CA/certs/certs-per-host')
    expect(store.tlsMode).toBe('generate')
  })

  it('tlsCanProceed true after successful generate result', () => {
    const store = useEmStore()
    store.tlsMode = 'generate'
    store.tlsGenerationResult = {
      success: true,
      outputDir: '/out',
      caFile: '/out/ca.crt',
      nodeFiles: [],
    }
    expect(store.tlsCanProceed).toBe(true)
  })

  it('tlsCanProceed true after valid validation result', () => {
    const store = useEmStore()
    store.tlsMode = 'existing'
    store.tlsValidationResult = {
      valid: true,
      caValid: true,
      nodeFiles: [],
      missing: [],
      errors: [],
      totalNodes: 1,
      foundNodes: 1,
    }
    expect(store.tlsCanProceed).toBe(true)
  })

  it('generateEmTls calls API with EM hosts and sets path on success', async () => {
    tlsGenerate.mockResolvedValue({
      success: true,
      output_dir: '/cwd/TLS/CA/certs/certs-per-host',
      ca_file: '/cwd/TLS/CA/certs/certs-per-host/ca.crt',
      node_files: [{
        hostname: 'em.example.com',
        short_name: 'em',
        cert_file: '/cwd/TLS/CA/certs/certs-per-host/em/node.crt',
        key_file: '/cwd/TLS/CA/certs/certs-per-host/em/node.key',
        web_pem: '/cwd/TLS/CA/certs/certs-per-host/em/web.pem',
      }],
    })
    const store = useEmStore()
    store.hosts = [{ fqdn: 'em.example.com' }]
    store.tlsDays = 180
    const result = await store.generateEmTls()
    expect(tlsGenerate).toHaveBeenCalledWith({ hosts: ['em.example.com'], days: 180 })
    expect(result.success).toBe(true)
    expect(store.tlsCertPath).toBe('/cwd/TLS/CA/certs/certs-per-host')
    expect(store.tlsCanProceed).toBe(true)
  })

  it('validateEmTls maps API response', async () => {
    tlsValidate.mockResolvedValue({
      valid: true,
      ca_valid: true,
      node_files: [{
        index: 1,
        hostname: 'em.example.com',
        short_name: 'em',
        key_found: true,
        cert_found: true,
        cert_valid: true,
        key_valid: true,
        pair_match: true,
        match: true,
      }],
      missing: [],
      errors: [],
      total_nodes: 1,
      found_nodes: 1,
    })
    const store = useEmStore()
    store.hosts = [{ fqdn: 'em.example.com' }]
    store.tlsMode = 'existing'
    store.tlsCertPath = 'TLS/CA/certs/certs-per-host'
    const result = await store.validateEmTls()
    expect(tlsValidate).toHaveBeenCalledWith({
      cert_path: 'TLS/CA/certs/certs-per-host',
      hosts: ['em.example.com'],
    })
    expect(result.valid).toBe(true)
    expect(store.tlsCanProceed).toBe(true)
  })

  it('emHostsList filters empty FQDNs', () => {
    const store = useEmStore()
    store.hosts = [{ fqdn: '' }, { fqdn: 'em.example.com' }, { fqdn: '  ' }]
    expect(store.emHostsList).toEqual(['em.example.com'])
  })

  it('checkEmTlsExists promotes path when tlsCertPath is empty', async () => {
    tlsExists.mockResolvedValue({
      exists: false,
      block_generate: false,
      path: '/TLS/CA/certs/certs-per-host',
      reasons: [],
      ca_present: true,
    })
    const store = useEmStore()
    store.hosts = [{ fqdn: 'em.example.com' }]
    await store.checkEmTlsExists()
    expect(store.tlsCertPath).toBe('/TLS/CA/certs/certs-per-host')
    expect(store.tlsCaPresent).toBe(true)
    expect(store.tlsPathAutoPromoted).toBe(true)
  })

  it('checkEmTlsExists does not overwrite non-empty tlsCertPath', async () => {
    tlsExists.mockResolvedValue({
      exists: false,
      block_generate: false,
      path: '/discovered/path',
      reasons: [],
      ca_present: true,
    })
    const store = useEmStore()
    store.hosts = [{ fqdn: 'em.example.com' }]
    store.tlsCertPath = '/custom/path'
    await store.checkEmTlsExists()
    expect(store.tlsCertPath).toBe('/custom/path')
    expect(store.tlsPathAutoPromoted).toBe(false)
  })

  it('checkEmTlsExists sets tlsCertsExist from block_generate', async () => {
    tlsExists.mockResolvedValue({
      exists: false,
      block_generate: true,
      path: '/TLS',
      reasons: ['em/node.crt'],
      ca_present: true,
      host_conflicts: ['em/node.crt', 'em/node.key'],
    })
    const store = useEmStore()
    store.hosts = [{ fqdn: 'em.example.com' }]
    await store.checkEmTlsExists()
    expect(store.tlsCertsExist).toBe(true)
    expect(store.tlsExistReasons).toEqual(['em/node.crt'])
  })

  it('tlsCaReuse true when CA present and not blocked', () => {
    const store = useEmStore()
    store.tlsMode = 'generate'
    store.hosts = [{ fqdn: 'em.example.com' }]
    store.tlsCaPresent = true
    store.tlsCertsExist = false
    expect(store.tlsCaReuse).toBe(true)
  })

  it('tlsCaReuse false when blocked', () => {
    const store = useEmStore()
    store.tlsMode = 'generate'
    store.hosts = [{ fqdn: 'em.example.com' }]
    store.tlsCaPresent = true
    store.tlsCertsExist = true
    expect(store.tlsCaReuse).toBe(false)
  })

  it('switchToExistingWithDiscoveredPath fills path from discovery', () => {
    const store = useEmStore()
    store.tlsExistPath = '/discovered/path'
    store.tlsCertPath = ''
    store.switchToExistingWithDiscoveredPath()
    expect(store.tlsMode).toBe('existing')
    expect(store.tlsCertPath).toBe('/discovered/path')
  })

  it('switchToExistingWithDiscoveredPath does not overwrite existing path', () => {
    const store = useEmStore()
    store.tlsExistPath = '/discovered/path'
    store.tlsCertPath = '/my/path'
    store.switchToExistingWithDiscoveredPath()
    expect(store.tlsCertPath).toBe('/my/path')
  })

  it('setTlsMode existing fills path from discovery when empty', () => {
    const store = useEmStore()
    store.tlsExistPath = '/discovered/path'
    store.tlsCertPath = ''
    store.setTlsMode('existing')
    expect(store.tlsCertPath).toBe('/discovered/path')
  })

  it('setTlsCertPath resets tlsPathAutoPromoted', () => {
    const store = useEmStore()
    store.tlsPathAutoPromoted = true
    store.setTlsCertPath('/other')
    expect(store.tlsPathAutoPromoted).toBe(false)
  })

  it('resetEmTls clears tlsCaPresent and tlsPathAutoPromoted', () => {
    const store = useEmStore()
    store.tlsCaPresent = true
    store.tlsPathAutoPromoted = true
    store.resetEmTls()
    expect(store.tlsCaPresent).toBe(false)
    expect(store.tlsPathAutoPromoted).toBe(false)
  })

  it('tlsCanGenerate false during existChecking', async () => {
    const store = useEmStore()
    store.hosts = [{ fqdn: 'em.example.com' }]
    tlsExists.mockImplementationOnce(() => new Promise(() => { }))
    const _promise = store.checkEmTlsExists()
    expect(store.tlsCanGenerate).toBe(false)
  })
})

describe('useEmStore — Screen 24 pre-flight', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  /** Fully configured archive-mode store — the happy path baseline. */
  function makeReadyStore() {
    const store = useEmStore()
    store.hosts = [{ fqdn: 'em.example.com' }]
    store.dbConnectionDBRoot = '/Root'
    store.dbConnectionDB = 'db'
    store.distMode = 'archive'
    store.archivePath = '/opt/dist/ydb-em.tar.gz'
    store.metaUIArchivePath = '/opt/dist/ui.tar.gz'
    store.archiveValid = true
    store.metaUIArchiveValid = true
    store.tlsCertPath = 'TLS/CA/certs/certs-per-host'
    store.useDynamicConfig = true
    return store
  }

  function rowById(store: ReturnType<typeof useEmStore>, id: string) {
    return store.emPreflightRows.find(r => r.id === id)!
  }

  it('emCanInstall is false on a fresh store', () => {
    const store = useEmStore()
    expect(store.emCanInstall).toBe(false)
  })

  it('emCanInstall is true when everything is configured', () => {
    const store = makeReadyStore()
    expect(store.emCanInstall).toBe(true)
  })

  it('exposes one row per requirement with stable ids', () => {
    const store = makeReadyStore()
    expect(store.emPreflightRows.map(r => r.id)).toEqual([
      'hosts', 'endpoint', 'database', 'distribution', 'tls', 'dynamicConfig',
    ])
  })

  it('hosts row reports the configured EM hosts', () => {
    const store = makeReadyStore()
    store.hosts = [{ fqdn: 'em1.example.com' }, { fqdn: 'em2.example.com' }]
    const row = rowById(store, 'hosts')
    expect(row.ok).toBe(true)
    expect(row.value).toBe('em1.example.com, em2.example.com')
  })

  it('blocks when no EM host is set', () => {
    const store = makeReadyStore()
    store.hosts = []
    expect(rowById(store, 'hosts').ok).toBe(false)
    expect(store.emCanInstall).toBe(false)
  })

  it('ignores blank host rows when evaluating the hosts requirement', () => {
    const store = makeReadyStore()
    store.hosts = [{ fqdn: '   ' }]
    expect(rowById(store, 'hosts').ok).toBe(false)
    expect(store.emCanInstall).toBe(false)
  })

  it('endpoint row falls back to the endpoint derived from the first EM host', () => {
    const store = makeReadyStore()
    store.dbConnectionEndpoint = ''
    const row = rowById(store, 'endpoint')
    expect(row.ok).toBe(true)
    expect(row.value).toBe('grpcs://em.example.com:2135')
  })

  it('endpoint row prefers an explicitly configured endpoint', () => {
    const store = makeReadyStore()
    store.dbConnectionEndpoint = 'grpcs://balancer.example.com:2135'
    expect(rowById(store, 'endpoint').value).toBe('grpcs://balancer.example.com:2135')
  })

  it('blocks when the database params are incomplete', () => {
    const store = makeReadyStore()
    store.dbConnectionDB = ''
    expect(rowById(store, 'database').ok).toBe(false)
    expect(store.emCanInstall).toBe(false)
  })

  it('archive mode reports archive paths and blocks until both validate', () => {
    const store = makeReadyStore()
    store.metaUIArchiveValid = null
    const row = rowById(store, 'distribution')
    expect(row.labelKey).toBe('emInstall.preflight.distArchive')
    expect(row.ok).toBe(false)
    expect(store.emCanInstall).toBe(false)
  })

  it('binaries mode reports the three binary paths only', () => {
    const store = makeReadyStore()
    store.distMode = 'binaries'
    store.metaUIBinary = '/opt/dist/gateway'
    store.cpBinary = '/opt/dist/cp'
    store.cpAgentBinary = '/opt/dist/agent'
    store.metaUIBinaryValid = true
    store.cpBinaryValid = true
    store.cpAgentBinaryValid = true

    const row = rowById(store, 'distribution')
    expect(row.labelKey).toBe('emInstall.preflight.distBinaries')
    expect(row.ok).toBe(true)
    expect(row.value).not.toContain('ydb-em.tar.gz')
    expect(store.emCanInstall).toBe(true)
  })

  it('blocks when the TLS cert path is empty', () => {
    const store = makeReadyStore()
    store.tlsCertPath = ''
    expect(rowById(store, 'tls').ok).toBe(false)
    expect(store.emCanInstall).toBe(false)
  })

  it('dynamicConfig is advisory — not ok but never blocking', () => {
    const store = makeReadyStore()
    store.useDynamicConfig = false
    const row = rowById(store, 'dynamicConfig')
    expect(row.ok).toBe(false)
    expect(row.severity).toBe('warning')
    expect(store.emCanInstall).toBe(true)
  })

  it('every blocking row carries a hint key for the UI', () => {
    const store = useEmStore()
    const blocking = store.emPreflightRows.filter(r => r.severity === 'error')
    expect(blocking.length).toBeGreaterThan(0)
    expect(blocking.every(r => !!r.hintKey)).toBe(true)
  })
})
