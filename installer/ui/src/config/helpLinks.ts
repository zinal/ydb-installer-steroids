export const HELP_LINKS = {
    /** YDB cluster system requirements (CPU, RAM, disks) */
    systemRequirements: '/devops/concepts/system-requirements',

    /** Jump-host / bastion proxy server */
    jumpHost: 'https://ydb.tech/docs/en/devops/deployment-options/manual/initial-deployment',

    /** YDB root domain path */
    ydbDomain: 'https://ydb.tech/docs/en/devops/deployment-options/ansible/initial-deployment',

    /** CPU cores allocated per storage node (static) */
    coresStatic: 'https://ydb.tech/docs/en/devops/deployment-options/ansible/initial-deployment',

    /** Enforce user token authentication */
    enforceUserToken: 'https://ydb.tech/docs/en/devops/deployment-options/ansible/initial-deployment',

    /** Require client certificate (mTLS) for dynamic nodes */
    requestClientCert: 'https://ydb.tech/docs/en/devops/deployment-options/ansible/initial-deployment',

    /** CPU affinity for static nodes */
    affinityStatic: 'https://ydb.tech/docs/en/devops/deployment-options/ansible/initial-deployment',

    /** CPU cores allocated per dynamic node */
    coresDynamic: 'https://ydb.tech/docs/en/devops/deployment-options/ansible/initial-deployment',

    /** Database name */
    dbName: 'https://ydb.tech/docs/en/devops/deployment-options/ansible/initial-deployment',

    /** Number of storage groups */
    dbGroups: 'https://ydb.tech/docs/en/devops/deployment-options/ansible/initial-deployment',

    /** CPU affinity for dynamic nodes */
    affinityDynamic: 'https://ydb.tech/docs/en/devops/deployment-options/ansible/initial-deployment',

    /** YDB CLI binary */
    ydbCli: 'https://ydb.tech/docs/en/reference/ydb-cli',

    /** YDB server daemon (ydbd) binary */
    ydbd: 'https://ydb.tech/docs/en/concepts/glossary#node',

    /** ydbops administrative tool binary */
    ydbops: 'https://ydb.tech/docs/en/reference/ydbops',

    /** dstool distributed storage management binary */
    dstool: 'https://ydb.tech/docs/en/reference/ydb-dstool',
} as const

/** Union type of all HELP_LINKS keys — useful for typed access. */
export type HelpLinkKey = keyof typeof HELP_LINKS

const DOCS_BASE = 'https://ydb.tech/docs'

/**
 * Build the system-requirements docs URL for a UI locale.
 *
 * HELP_LINKS.systemRequirements stores only the path after the locale segment;
 * the /ru/ ↔ /en/ swap keeps the link in the user's current language, matching
 * the bilingual docs experience everywhere else in the app.
 */
export function systemRequirementsUrl(locale: 'ru' | 'en'): string {
    return `${DOCS_BASE}/${locale}${HELP_LINKS.systemRequirements}`
}