import { describe, it, expect } from 'vitest'
import {
  parseJumpHost,
  buildProxyCommand,
  buildJFlag,
  removeJumpHostFromArgs,
  updateJumpHost,
  hasJumpHost,
} from '@/utils/sshArgs'

describe('parseJumpHost', () => {
  describe('ProxyCommand format', () => {
    it('should parse ProxyCommand with explicit user', () => {
      const args = "-o ProxyCommand='ssh -o StrictHostKeyChecking=no -W %h:%p user@bastion.example.com'"
      const result = parseJumpHost(args)
      
      expect(result).toEqual({
        host: 'bastion.example.com',
        user: 'user',
        format: 'proxycommand',
      })
    })

    it('should parse ProxyCommand with Jinja2 template user and strip braces', () => {
      const args = "-o ProxyCommand='ssh -o StrictHostKeyChecking=no -W %h:%p {{ ansible_user }}@bastion.example.com'"
      const result = parseJumpHost(args)
      
      expect(result).toEqual({
        host: 'bastion.example.com',
        user: 'ansible_user',
        format: 'proxycommand',
      })
    })

    it('should strip only trailing braces from captured username', () => {
      const args = "-o ProxyCommand='ssh -o StrictHostKeyChecking=no -W %h:%p ansible_user}}@bastion.example.com'"
      const result = parseJumpHost(args)

      expect(result).toEqual({
        host: 'bastion.example.com',
        user: 'ansible_user',
        format: 'proxycommand',
      })
    })

    it('should parse ProxyCommand with additional args before', () => {
      const args = "-o StrictHostKeyChecking=no -o ProxyCommand='ssh -W %h:%p user@host'"
      const result = parseJumpHost(args)
      
      expect(result).toEqual({
        host: 'host',
        user: 'user',
        format: 'proxycommand',
      })
    })
  })

  describe('-J flag format (not auto-filled)', () => {
    it('should return null for -J with explicit user', () => {
      const args = '-J user@bastion.example.com'
      const result = parseJumpHost(args)

      expect(result).toBeNull()
    })

    it('should return null for -J with Jinja2 template user', () => {
      const args = '-J {{ ansible_user }}@bastion.example.com'
      const result = parseJumpHost(args)

      expect(result).toBeNull()
    })

    it('should return null for -J with additional args after', () => {
      const args = '-J user@host -o StrictHostKeyChecking=no'
      const result = parseJumpHost(args)

      expect(result).toBeNull()
    })
  })

  describe('edge cases', () => {
    it('should return null for empty string', () => {
      expect(parseJumpHost('')).toBeNull()
    })

    it('should return null for null/undefined', () => {
      expect(parseJumpHost(null as any)).toBeNull()
      expect(parseJumpHost(undefined as any)).toBeNull()
    })

    it('should return null for args without jump host', () => {
      expect(parseJumpHost('-o StrictHostKeyChecking=no')).toBeNull()
      expect(parseJumpHost('-o ConnectTimeout=30')).toBeNull()
    })

    it('should prefer ProxyCommand over -J when both present', () => {
      const args = "-o ProxyCommand='ssh -W %h:%p user@host1' -J user@host2"
      const result = parseJumpHost(args)

      // Should match ProxyCommand (—J is ignored)
      expect(result?.host).toBe('host1')
      expect(result?.format).toBe('proxycommand')
    })
  })
})

describe('buildProxyCommand', () => {
  it('should build ProxyCommand with explicit user', () => {
    const result = buildProxyCommand('admin', 'bastion.example.com')
    
    expect(result).toBe("-o ProxyCommand='ssh -o StrictHostKeyChecking=no -W %h:%p admin@bastion.example.com'")
  })

  it('should build ProxyCommand with Jinja2 template user', () => {
    const result = buildProxyCommand('{{ ansible_user }}', 'bastion.example.com')
    
    expect(result).toBe("-o ProxyCommand='ssh -o StrictHostKeyChecking=no -W %h:%p {{ ansible_user }}@bastion.example.com'")
  })

  it('should handle IP addresses', () => {
    const result = buildProxyCommand('user', '192.168.1.1')
    
    expect(result).toBe("-o ProxyCommand='ssh -o StrictHostKeyChecking=no -W %h:%p user@192.168.1.1'")
  })
})

describe('buildJFlag', () => {
  it('should build -J flag with explicit user', () => {
    const result = buildJFlag('admin', 'bastion.example.com')
    
    expect(result).toBe('-J admin@bastion.example.com')
  })

  it('should build -J flag with Jinja2 template user', () => {
    const result = buildJFlag('{{ ansible_user }}', 'bastion.example.com')
    
    expect(result).toBe('-J {{ ansible_user }}@bastion.example.com')
  })
})

describe('removeJumpHostFromArgs', () => {
  describe('without specific host', () => {
    it('should remove ProxyCommand from args', () => {
      const args = "-o ProxyCommand='ssh -W %h:%p user@host' -o ConnectTimeout=30"
      const result = removeJumpHostFromArgs(args)

      expect(result).toBe('-o ConnectTimeout=30')
    })

    it('should NOT remove -J flag from args (preserved)', () => {
      const args = '-J user@host -o ConnectTimeout=30'
      const result = removeJumpHostFromArgs(args)

      expect(result).toBe('-J user@host -o ConnectTimeout=30')
    })

    it('should remove only ProxyCommand, preserve -J', () => {
      const args = "-o ProxyCommand='ssh -W %h:%p user@host1' -J user@host2 -o ConnectTimeout=30"
      const result = removeJumpHostFromArgs(args)

      expect(result).toBe('-J user@host2 -o ConnectTimeout=30')
    })

    it('should return empty string for args with only ProxyCommand', () => {
      const args = "-o ProxyCommand='ssh -W %h:%p user@host'"
      const result = removeJumpHostFromArgs(args)

      expect(result).toBe('')
    })

    it('should handle empty input', () => {
      expect(removeJumpHostFromArgs('')).toBe('')
      expect(removeJumpHostFromArgs(null as any)).toBe('')
    })
  })

  describe('with specific host', () => {
    it('should remove only ProxyCommand for specific host', () => {
      const args = "-o ProxyCommand='ssh -W %h:%p user@host1' -o ProxyCommand='ssh -W %h:%p user@host2'"
      const result = removeJumpHostFromArgs(args, 'host1')

      expect(result).toBe("-o ProxyCommand='ssh -W %h:%p user@host2'")
    })
  })
})

describe('updateJumpHost', () => {
  it('should add jump host to empty args', () => {
    const result = updateJumpHost('', 'bastion.example.com')
    
    expect(result).toBe("-o ProxyCommand='ssh -o StrictHostKeyChecking=no -W %h:%p {{ ansible_user }}@bastion.example.com'")
  })

  it('should replace existing jump host', () => {
    const args = "-o ProxyCommand='ssh -W %h:%p olduser@oldhost.example.com' -o ConnectTimeout=30"
    const result = updateJumpHost(args, 'newbastion.example.com')
    
    expect(result).toBe("-o ProxyCommand='ssh -o StrictHostKeyChecking=no -W %h:%p {{ ansible_user }}@newbastion.example.com' -o ConnectTimeout=30")
  })

  it('should remove jump host when null provided', () => {
    const args = "-o ProxyCommand='ssh -W %h:%p user@host' -o ConnectTimeout=30"
    const result = updateJumpHost(args, null)
    
    expect(result).toBe('-o ConnectTimeout=30')
  })

  it('should use custom user when provided', () => {
    const result = updateJumpHost('', 'bastion.example.com', 'admin')
    
    expect(result).toBe("-o ProxyCommand='ssh -o StrictHostKeyChecking=no -W %h:%p admin@bastion.example.com'")
  })

  it('should preserve non-jump-host args when removing', () => {
    const args = '-o ConnectTimeout=30 -o StrictHostKeyChecking=no'
    const result = updateJumpHost(args, null)
    
    expect(result).toBe('-o ConnectTimeout=30 -o StrictHostKeyChecking=no')
  })
})

describe('hasJumpHost', () => {
  it('should return true for ProxyCommand format', () => {
    const args = "-o ProxyCommand='ssh -W %h:%p user@host'"
    expect(hasJumpHost(args)).toBe(true)
  })

  it('should return false for -J format (not recognized)', () => {
    const args = '-J user@host'
    expect(hasJumpHost(args)).toBe(false)
  })

  it('should return false for args without jump host', () => {
    expect(hasJumpHost('-o ConnectTimeout=30')).toBe(false)
    expect(hasJumpHost('')).toBe(false)
  })
})
