import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useTlsStore } from '@/stores/tls'

vi.mock('@/api/client', () => ({
  api: {
    tlsGenerate: vi.fn(),
    tlsValidate: vi.fn(),
  },
}))

import { api } from '@/api/client'

const FQDN_HOSTS = [
  'static-node-1.ydb-cluster.com',
  'static-node-2.ydb-cluster.com',
  'static-node-3.ydb-cluster.com',
]

const MOCK_GENERATE_SUCCESS = {
  success: true,
  output_dir: '/opt/ydb/TLS/CA/certs/certs-per-host',
  ca_file: '/opt/ydb/TLS/CA/certs/certs-per-host/ca.crt',
  node_files: [
    {
      hostname: 'static-node-1.ydb-cluster.com',
      short_name: 'static-node-1',
      cert_file: '/opt/ydb/TLS/CA/certs/certs-per-host/static-node-1/node.crt',
      key_file: '/opt/ydb/TLS/CA/certs/certs-per-host/static-node-1/node.key',
      web_pem: '/opt/ydb/TLS/CA/certs/certs-per-host/static-node-1/web.pem',
    },
    {
      hostname: 'static-node-2.ydb-cluster.com',
      short_name: 'static-node-2',
      cert_file: '/opt/ydb/TLS/CA/certs/certs-per-host/static-node-2/node.crt',
      key_file: '/opt/ydb/TLS/CA/certs/certs-per-host/static-node-2/node.key',
      web_pem: '/opt/ydb/TLS/CA/certs/certs-per-host/static-node-2/web.pem',
    },
    {
      hostname: 'static-node-3.ydb-cluster.com',
      short_name: 'static-node-3',
      cert_file: '/opt/ydb/TLS/CA/certs/certs-per-host/static-node-3/node.crt',
      key_file: '/opt/ydb/TLS/CA/certs/certs-per-host/static-node-3/node.key',
      web_pem: '/opt/ydb/TLS/CA/certs/certs-per-host/static-node-3/web.pem',
    },
  ],
}

const MOCK_VALIDATE_SUCCESS = {
  valid: true,
  ca_valid: true,
  node_files: [
    {
      index: 1,
      hostname: 'static-node-1.ydb-cluster.com',
      short_name: 'static-node-1',
      key_found: true,
      cert_found: true,
      cert_valid: true,
      key_valid: true,
      pair_match: true,
      match: true,
    },
    {
      index: 2,
      hostname: 'static-node-2.ydb-cluster.com',
      short_name: 'static-node-2',
      key_found: true,
      cert_found: true,
      cert_valid: true,
      key_valid: true,
      pair_match: true,
      match: true,
    },
  ],
  missing: [],
  errors: [],
  total_nodes: 2,
  found_nodes: 2,
}

describe('TLS Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('initial state', () => {
    it('should have correct default values', () => {
      const store = useTlsStore()

      expect(store.mode).toBe('generate')
      expect(store.certPath).toBe('')
      expect(store.days).toBe(365)
      expect(store.generating).toBe(false)
      expect(store.validating).toBe(false)
      expect(store.generationResult).toBe(null)
      expect(store.validationResult).toBe(null)
      expect(store.error).toBe('')
    })
  })

  describe('setMode', () => {
    it('should set mode to generate', () => {
      const store = useTlsStore()
      store.setMode('generate')
      expect(store.mode).toBe('generate')
    })

    it('should set mode to existing', () => {
      const store = useTlsStore()
      store.setMode('existing')
      expect(store.mode).toBe('existing')
    })

    it('should clear only the mode-agnostic error when changing mode (results are preserved)', () => {
      const store = useTlsStore()
      store.error = 'some error'
      store.generationResult = {
        success: true,
        outputDir: '/test',
        caFile: '/test/ca.crt',
        nodeFiles: [],
      }

      store.setMode('existing')

      expect(store.error).toBe('')
      expect(store.generationResult).not.toBeNull()
      expect(store.generationResult?.outputDir).toBe('/test')
    })

    it('preserves both generation and validation results across a full mode round-trip', () => {
      const store = useTlsStore()

      store.setMode('generate')
      store.generationResult = {
        success: true,
        outputDir: '/gen',
        caFile: '/gen/ca.crt',
        nodeFiles: [],
      }
      store.setMode('existing')
      store.validationResult = {
        valid: true,
        caValid: true,
        nodeFiles: [],
        missing: [],
        errors: [],
        totalNodes: 0,
        foundNodes: 0,
      }

      store.setMode('existing')
      expect(store.generationResult?.outputDir).toBe('/gen')

      store.setMode('generate')
      expect(store.generationResult?.outputDir).toBe('/gen')
      store.setMode('existing')
      expect(store.validationResult?.valid).toBe(true)
    })
  })

  describe('setCertPath', () => {
    it('should set cert path', () => {
      const store = useTlsStore()
      store.setCertPath('/etc/ydb/certs')
      expect(store.certPath).toBe('/etc/ydb/certs')
    })

    it('should clear error when setting cert path', () => {
      const store = useTlsStore()
      store.error = 'some error'
      store.setCertPath('/etc/ydb/certs')
      expect(store.error).toBe('')
    })

    it('should invalidate generationResult when cert path changes', () => {
      const store = useTlsStore()
      store.generationResult = {
        success: true,
        outputDir: '/old/path',
        caFile: '/old/path/ca.crt',
        nodeFiles: [],
      }
      store.setCertPath('/new/path')
      expect(store.generationResult).toBeNull()
    })

    it('should invalidate validationResult when cert path changes', () => {
      const store = useTlsStore()
      store.validationResult = {
        valid: true,
        caValid: true,
        nodeFiles: [],
        missing: [],
        errors: [],
        totalNodes: 1,
        foundNodes: 1,
      }
      store.setCertPath('/new/path')
      expect(store.validationResult).toBeNull()
    })

    it('should NOT invalidate results when cert path stays the same', () => {
      const store = useTlsStore()
      store.certPath = '/same/path'
      store.generationResult = {
        success: true,
        outputDir: '/same/path',
        caFile: '/same/path/ca.crt',
        nodeFiles: [],
      }
      store.validationResult = {
        valid: true,
        caValid: true,
        nodeFiles: [],
        missing: [],
        errors: [],
        totalNodes: 1,
        foundNodes: 1,
      }
      store.setCertPath('/same/path')
      expect(store.generationResult).not.toBeNull()
      expect(store.validationResult).not.toBeNull()
    })
  })

  describe('setDays', () => {
    it('should set days', () => {
      const store = useTlsStore()
      store.setDays(730)
      expect(store.days).toBe(730)
    })
  })

  describe('applyFromInventory', () => {
    it('should apply TLS config from inventory', () => {
      const store = useTlsStore()
      store.mode = 'existing'
      store.applyFromInventory({
        tls: { cert_path: '/custom/path' },
      })

      expect(store.certPath).toBe('/custom/path')
      expect(store.mode).toBe('existing')
    })

    it('should preserve generate mode when cert_path is provided', () => {
      const store = useTlsStore()
      store.mode = 'generate'
      store.applyFromInventory({
        tls: { cert_path: '/custom/path' },
      })

      expect(store.certPath).toBe('/custom/path')
      expect(store.mode).toBe('generate')
    })

    it('should use defaults when TLS config is missing', () => {
      const store = useTlsStore()
      store.applyFromInventory({})

      expect(store.mode).toBe('generate')
      expect(store.certPath).toBe('')
    })

    it('should invalidate generationResult when cert_path from inventory differs', () => {
      const store = useTlsStore()
      store.certPath = '/old/path'
      store.generationResult = {
        success: true,
        outputDir: '/old/path',
        caFile: '/old/path/ca.crt',
        nodeFiles: [],
      }
      store.applyFromInventory({
        tls: { cert_path: '/new/path' },
      })
      expect(store.generationResult).toBeNull()
    })

    it('should invalidate validationResult when cert_path from inventory differs', () => {
      const store = useTlsStore()
      store.certPath = '/old/path'
      store.validationResult = {
        valid: true,
        caValid: true,
        nodeFiles: [],
        missing: [],
        errors: [],
        totalNodes: 1,
        foundNodes: 1,
      }
      store.applyFromInventory({
        tls: { cert_path: '/new/path' },
      })
      expect(store.validationResult).toBeNull()
    })

    it('should NOT invalidate results when cert_path from inventory is the same', () => {
      const store = useTlsStore()
      store.certPath = '/same/path'
      store.generationResult = {
        success: true,
        outputDir: '/same/path',
        caFile: '/same/path/ca.crt',
        nodeFiles: [],
      }
      store.validationResult = {
        valid: true,
        caValid: true,
        nodeFiles: [],
        missing: [],
        errors: [],
        totalNodes: 1,
        foundNodes: 1,
      }
      store.applyFromInventory({
        tls: { cert_path: '/same/path' },
      })
      expect(store.generationResult).not.toBeNull()
      expect(store.validationResult).not.toBeNull()
    })
  })

  describe('toTLSConfig', () => {
    it('should convert store state to TLSConfig', () => {
      const store = useTlsStore()
      store.mode = 'existing'
      store.certPath = '/test/path'

      const config = store.toTLSConfig()

      expect(config.cert_path).toBe('/test/path')
    })
  })

  describe('generateCertificates', () => {
    it('should generate certificates successfully and map short_name + web_pem', async () => {
      const store = useTlsStore()
      vi.mocked(api.tlsGenerate).mockResolvedValue(MOCK_GENERATE_SUCCESS)

      const result = await store.generateCertificates(FQDN_HOSTS)

      expect(store.generating).toBe(false)
      expect(result.success).toBe(true)
      expect(store.generationResult).not.toBeNull()
      expect(store.generationResult?.success).toBe(true)
      expect(store.generationResult?.outputDir).toBe('/opt/ydb/TLS/CA/certs/certs-per-host')

      expect(store.certPath).toBe('/opt/ydb/TLS/CA/certs/certs-per-host')

      const nodeFiles = store.generationResult?.nodeFiles ?? []
      expect(nodeFiles).toHaveLength(3)

      expect(nodeFiles[0].hostname).toBe('static-node-1.ydb-cluster.com')
      expect(nodeFiles[0].shortName).toBe('static-node-1')
      expect(nodeFiles[0].certFile).toBe('/opt/ydb/TLS/CA/certs/certs-per-host/static-node-1/node.crt')
      expect(nodeFiles[0].keyFile).toBe('/opt/ydb/TLS/CA/certs/certs-per-host/static-node-1/node.key')
      expect(nodeFiles[0].webPem).toBe('/opt/ydb/TLS/CA/certs/certs-per-host/static-node-1/web.pem')

      expect(nodeFiles[1].shortName).toBe('static-node-2')
      expect(nodeFiles[2].shortName).toBe('static-node-3')
    })

    it('should handle generation failure', async () => {
      const store = useTlsStore()
      vi.mocked(api.tlsGenerate).mockResolvedValue({
        success: false,
        error: 'Generation failed',
      })

      const result = await store.generateCertificates(FQDN_HOSTS)

      expect(store.generating).toBe(false)
      expect(result.success).toBe(false)
      expect(store.generationResult?.success).toBe(false)
      expect(store.generationResult?.error).toBe('Generation failed')
      expect(store.error).toBe('Generation failed')
    })

    it('should handle API error', async () => {
      const store = useTlsStore()
      vi.mocked(api.tlsGenerate).mockRejectedValue(new Error('Network error'))

      const result = await store.generateCertificates(FQDN_HOSTS)

      expect(store.generating).toBe(false)
      expect(store.error).toBe('Network error')
      expect(result.success).toBe(false)
    })

    it('should send days parameter to API', async () => {
      const store = useTlsStore()
      store.setDays(730)
      vi.mocked(api.tlsGenerate).mockResolvedValue(MOCK_GENERATE_SUCCESS)

      await store.generateCertificates(['node1.example.com'])

      expect(vi.mocked(api.tlsGenerate)).toHaveBeenCalledWith(
        expect.objectContaining({ days: 730 })
      )
    })

    it('should pass force=false by default', async () => {
      const store = useTlsStore()
      vi.mocked(api.tlsGenerate).mockResolvedValue(MOCK_GENERATE_SUCCESS)

      await store.generateCertificates(['node1.example.com'])

      expect(vi.mocked(api.tlsGenerate)).toHaveBeenCalledWith(
        expect.objectContaining({ force: false })
      )
    })

    it('should pass force=true when requested', async () => {
      const store = useTlsStore()
      vi.mocked(api.tlsGenerate).mockResolvedValue(MOCK_GENERATE_SUCCESS)

      await store.generateCertificates(['node1.example.com'], true)

      expect(vi.mocked(api.tlsGenerate)).toHaveBeenCalledWith(
        expect.objectContaining({ force: true })
      )
    })
  })

  describe('validateCertificates', () => {
    it('should validate certificates successfully and map short_name', async () => {
      const store = useTlsStore()
      store.certPath = '/opt/ydb/TLS/CA/certs/certs-per-host'
      vi.mocked(api.tlsValidate).mockResolvedValue(MOCK_VALIDATE_SUCCESS)

      const result = await store.validateCertificates(FQDN_HOSTS.slice(0, 2))

      expect(store.validating).toBe(false)
      expect(result.valid).toBe(true)
      expect(store.validationResult).not.toBeNull()
      expect(store.validationResult?.valid).toBe(true)

      const nodeFiles = store.validationResult?.nodeFiles ?? []
      expect(nodeFiles).toHaveLength(2)

      expect(nodeFiles[0].hostname).toBe('static-node-1.ydb-cluster.com')
      expect(nodeFiles[0].shortName).toBe('static-node-1')
      expect(nodeFiles[0].certFound).toBe(true)
      expect(nodeFiles[0].keyFound).toBe(true)
      expect(nodeFiles[0].match).toBe(true)

      expect(nodeFiles[1].shortName).toBe('static-node-2')
    })

    it('should return error when cert path is empty', async () => {
      const store = useTlsStore()
      store.certPath = ''

      const result = await store.validateCertificates(FQDN_HOSTS)

      expect(store.error).toBe('Certificate path is required')
      expect(result.valid).toBe(false)
    })

    it('should handle validation failure with missing files', async () => {
      const store = useTlsStore()
      store.certPath = '/opt/ydb/TLS/CA/certs/certs-per-host'
      vi.mocked(api.tlsValidate).mockResolvedValue({
        valid: false,
        ca_valid: true,
        node_files: [
          {
            index: 1,
            hostname: 'static-node-1.ydb-cluster.com',
            short_name: 'static-node-1',
            key_found: false,
            cert_found: false,
            cert_valid: false,
            key_valid: false,
            pair_match: false,
            match: false,
          },
        ],
        missing: ['static-node-1/node.crt', 'static-node-1/node.key'],
        errors: [],
        total_nodes: 1,
        found_nodes: 0,
      })

      const result = await store.validateCertificates(['static-node-1.ydb-cluster.com'])

      expect(result.valid).toBe(false)
      expect(store.validationResult?.valid).toBe(false)
      expect(store.error).toContain('Missing')
      expect(store.error).toContain('static-node-1')
    })

    it('should handle API error', async () => {
      const store = useTlsStore()
      store.certPath = '/some/path'
      vi.mocked(api.tlsValidate).mockRejectedValue(new Error('Network error'))

      const result = await store.validateCertificates(FQDN_HOSTS)

      expect(store.validating).toBe(false)
      expect(store.error).toBe('Network error')
      expect(result.valid).toBe(false)
    })
  })

  describe('canProceed', () => {
    it('should return true when in generate mode and generation succeeded', () => {
      const store = useTlsStore()
      store.mode = 'generate'
      store.generationResult = {
        success: true,
        outputDir: '/test',
        caFile: '/test/ca.crt',
        nodeFiles: [],
      }

      expect(store.canProceed).toBe(true)
    })

    it('should return false when in generate mode and no result', () => {
      const store = useTlsStore()
      store.mode = 'generate'
      store.generationResult = null

      expect(store.canProceed).toBe(false)
    })

    it('should return false when in generate mode and generation failed', () => {
      const store = useTlsStore()
      store.mode = 'generate'
      store.generationResult = {
        success: false,
        outputDir: '',
        caFile: '',
        nodeFiles: [],
        error: 'failed',
      }

      expect(store.canProceed).toBe(false)
    })

    it('should return true when in existing mode and validation succeeded', () => {
      const store = useTlsStore()
      store.mode = 'existing'
      store.validationResult = {
        valid: true,
        caValid: true,
        nodeFiles: [],
        missing: [],
        errors: [],
        totalNodes: 1,
        foundNodes: 1,
      }

      expect(store.canProceed).toBe(true)
    })

    it('should return false when in existing mode and validation failed', () => {
      const store = useTlsStore()
      store.mode = 'existing'
      store.validationResult = {
        valid: false,
        caValid: true,
        nodeFiles: [],
        missing: [],
        errors: [],
        totalNodes: 1,
        foundNodes: 0,
      }

      expect(store.canProceed).toBe(false)
    })
  })

  describe('clearError', () => {
    it('should clear error', () => {
      const store = useTlsStore()
      store.error = 'some error'
      store.clearError()
      expect(store.error).toBe('')
    })
  })

  describe('reset', () => {
    it('should reset store to initial state', () => {
      const store = useTlsStore()
      store.mode = 'existing'
      store.certPath = '/test/path'
      store.days = 730
      store.error = 'some error'
      store.generationResult = {
        success: true,
        outputDir: '/test',
        caFile: '/test/ca.crt',
        nodeFiles: [],
      }

      store.reset()

      expect(store.mode).toBe('generate')
      expect(store.certPath).toBe('')
      expect(store.days).toBe(365)
      expect(store.error).toBe('')
      expect(store.generationResult).toBe(null)
      expect(store.validationResult).toBe(null)
    })
  })
})
