# Security

## Supported versions

| Version | Security fixes |
|---------|---------------|
| latest (main) | Yes |
| older tags | No — upgrade to latest |

## Credential storage model

Credentials are split into two layers:

| File | Contents | Scope |
|------|----------|-------|
| `~/.corezoid/credentials` | `ACCESS_TOKEN`, `ACCESS_TOKEN_EXPIRES_AT` | User — shared across all projects |
| `<project>/.env` | `WORKSPACE_ID`, `COREZOID_STAGE_ID`, API URLs | Project — one per workspace |

**Token file:** written with permissions `0600`; the `~/.corezoid/` directory is created with `0700`.  
**Plugin package:** `plugins/corezoid/.mcp.json` ships without any credentials — tokens are never bundled in the marketplace package.  
**Load order:** the MCP server loads `~/.corezoid/credentials` first, then the project `.env`. A token in `.env` overrides the user-level one (for environments that manage credentials externally).

## What the MCP server sends over the network

- OAuth2 flows go to `account.corezoid.com` only.
- All Corezoid API calls go to the `COREZOID_API_URL` configured in `.env` (default: `https://api.corezoid.com`).
- TLS verification is enabled by default. It can be disabled with `COREZOID_INSECURE_TLS=1` — only for on-premises installations with self-signed certificates.
- Anonymous tool-call telemetry (tool name, duration ms, error type, workspace ID, API hostname) is sent to `www.corezoid.com`. No tokens, process content, or personally identifiable data are included. Set `COREZOID_ANALYTICS_DISABLED=1` to opt out.

## What not to commit

- `ACCESS_TOKEN` or OAuth2 refresh tokens of any kind
- `.env` files (add to `.gitignore`)
- `~/.corezoid/credentials`
- Workspace IDs or stage IDs tied to private environments
- Exported process files (`.conv.json`) that contain private business logic or customer data

## Reporting a vulnerability

If you discover a security issue — including a secret accidentally committed to this repository — **do not open a public GitHub issue**. Instead:

1. Open a private security advisory at `https://github.com/corezoid/corezoid-ai-plugin/security/advisories/new`.
2. Or email `support@corezoid.com` with subject `[SECURITY] corezoid-ai-plugin`.

We will acknowledge within 2 business days and coordinate a fix before any public disclosure.
