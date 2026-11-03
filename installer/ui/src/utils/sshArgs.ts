export interface ParsedJumpHost {
  host: string
  user: string
  format: 'proxycommand'
}

/**
 * Parse jump host configuration from SSH common args string.
 * Only the ProxyCommand format is recognized; the legacy -J flag and any
 * other SSH arguments do NOT trigger jump host auto-fill.
 *
 * @example
 * parseJumpHost("-o ProxyCommand='ssh -o StrictHostKeyChecking=no -W %h:%p user@bastion.example.com'")
 * // Returns: { host: "bastion.example.com", user: "user", format: "proxycommand" }
 *
 * @example
 * parseJumpHost("-J {{ ansible_user }}@bastion.example.com")
 * // Returns: null  (—J format is not auto-filled)
 */
export function parseJumpHost(args: string): ParsedJumpHost | null {
  if (!args) return null

  // Jinja2 template user with inner spaces (`{{ ansible_user }}`) is matched
  // first: the plain \S+ capture below would only grab the closing `}}`
  // (space-separated from the variable name) and lose the username.
  const templateMatch = args.match(
    /-o\s+ProxyCommand='ssh[^']*?\s+\{\{\s*([^{}'\s]+)\s*\}\}\s*@\s*(\S+)'/
  )
  if (templateMatch) {
    return {
      user: templateMatch[1],
      host: templateMatch[2],
      format: 'proxycommand',
    }
  }

  const proxyMatch = args.match(/-o\s+ProxyCommand='ssh[^']*?\s+(\S+)\s*@\s*(\S+)'/)
  if (proxyMatch) {
    // Strip Jinja2 template markers from the captured username.
    // The regex \S+ greedily captures `ansible_user}}` when the braces
    // are contiguous with the username (e.g. `{{ansible_user}}` or a
    // stray `}}`). Real OS usernames never start with `{` or end with
    // `}`, so stripping is always safe.
    let user = proxyMatch[1].trim()
    user = user.replace(/^\{+/, '').replace(/\}+$/, '').trim()
    return {
      user,
      host: proxyMatch[2],
      format: 'proxycommand',
    }
  }

  return null
}

export function buildProxyCommand(user: string, host: string): string {
  return `-o ProxyCommand='ssh -o StrictHostKeyChecking=no -W %h:%p ${user}@${host}'`
}

export function buildJFlag(user: string, host: string): string {
  return `-J ${user}@${host}`
}

/**
 * Remove jump host configuration from SSH args string.
 * Only ProxyCommand format is removed; -J and other args are preserved.
 */
export function removeJumpHostFromArgs(args: string, host?: string): string {
  if (!args) return ''

  let result = args

  if (host) {
    const escapedHost = host.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    result = result.replace(
      new RegExp(`-o\\s+ProxyCommand='ssh[^']*?@${escapedHost}'`, 'g'),
      ''
    )
  } else {
    result = result.replace(/-o\s+ProxyCommand='ssh[^']*'/g, '')
  }

  return result.trim()
}

export function updateJumpHost(
  currentArgs: string | undefined,
  jumpHost: string | null | undefined,
  jumpUser: string = '{{ ansible_user }}'
): string {
  let cleaned = removeJumpHostFromArgs(currentArgs ?? '')

  if (jumpHost) {
    const proxyCmd = buildProxyCommand(jumpUser, jumpHost)
    
    if (cleaned) {
      return `${proxyCmd} ${cleaned}`
    }
    return proxyCmd
  }

  return cleaned
}

export function hasJumpHost(args: string): boolean {
  return parseJumpHost(args) !== null
}
