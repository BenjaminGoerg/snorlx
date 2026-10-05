# Snorlx MCP

Connect Cursor (or any MCP client) to a running Snorlx instance using a personal API token.

## Prerequisites

1. Snorlx backend running (default `http://localhost:8080`)
2. Sign in to the dashboard and open **Settings → API tokens**
3. Create a token. Prefer **read-only** unless you need the write tools (sync, rerun, cancel). Copy it once; it is not shown again.

The MCP process acts as the user who minted the token: it sees exactly the repositories that user synced, nothing else.

## Install

From the repo root:

```bash
pnpm install
pnpm mcp:build
```

## Local Cursor (stdio)

Add to your Cursor MCP config (e.g. `~/.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "snorlx": {
      "command": "node",
      "args": ["/absolute/path/to/snorlx/mcp/dist/index.js"],
      "env": {
        "SNORLX_API_URL": "http://localhost:8080",
        "SNORLX_API_TOKEN": "snorlx_..."
      }
    }
  }
}
```

## Product / remote (Streamable HTTP)

HTTP mode requires a **separate** client credential (`MCP_HTTP_TOKEN`) and binds to localhost by default.

```bash
SNORLX_API_URL=https://snorlx.example.com \
SNORLX_API_TOKEN=snorlx_... \
MCP_HTTP_TOKEN="$(openssl rand -hex 32)" \
MCP_HOST=127.0.0.1 \
MCP_PORT=3100 \
pnpm mcp:http
```

| Variable | Purpose |
|----------|---------|
| `SNORLX_API_TOKEN` | Personal API token used by the MCP process to call Snorlx |
| `MCP_HTTP_TOKEN` | Required. Clients must send `Authorization: Bearer <MCP_HTTP_TOKEN>` on `/mcp` |
| `MCP_HOST` | Bind address (default `127.0.0.1`). Set to `0.0.0.0` only behind an authenticated reverse proxy / mTLS |
| `MCP_PORT` | Listen port (default `3100`) |
| `MCP_ALLOWED_HOSTS` | Comma-separated `Host` header values accepted on `/mcp` (DNS rebinding protection). Defaults to the loopback names when `MCP_HOST` is loopback; set it when binding elsewhere |
| `MCP_SESSION_TTL_SECONDS` | Idle MCP sessions are closed after this many seconds (default 1800) |

Without `MCP_HTTP_TOKEN`, anyone who can reach `/mcp` could drive all tools using the process Snorlx token. Treat `/mcp` as a private control plane: keep it on loopback, or terminate TLS and auth at a reverse proxy before opening it on a network. Request failures return a generic `500`; details are written to the MCP process log only.

## Tools

Read tools cover dashboard summary/trends, orgs, repos, scores, workflows, runs, jobs, annotations, truncated logs, and active pipelines.

Write tools (require token `write` scope; confirm with the user before use):

- `sync_repositories` / `sync_repository`
- `backfill_deployment_runs`
- `update_workflow`
- `rerun_workflow` / `cancel_run`

## Security notes

- Tokens are hashed at rest; plaintext is shown only once at creation.
- Validated Bearer API tokens skip CSRF; cookie sessions still use Origin checks. A bogus `Authorization` header does not bypass CSRF.
- Never put OAuth client secrets or GitHub tokens into MCP env vars; use a user-minted `snorlx_` API token only.
- Stdio mode keeps secrets in the local Cursor process env; do not commit them.
- Every HTTP client shares the identity of the token owner. Mint one token per agent or per person so revocation stays targeted.
