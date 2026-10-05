# 0005: Serve the API same-origin through the frontend proxy, fail closed on storage and origin

- Status: Accepted
- Date: 2026-10-05

## Context

The shipped deployments were inconsistent: Docker Compose built the SPA for an API on port 8081 while publishing the backend on 3001 and setting `FRONTEND_URL` to the Vite dev port; the Helm chart set `FRONTEND_URL` to the internal service name, routed `/api` but not `/ws`, and ran the root `nginx` image under a non-root security context. The browser Origin never matched `FRONTEND_URL`, so every mutation got 403 and the WebSocket upgrade failed, which pushed operators to disable the checks. The backend also trusted `X-Forwarded-For` from any client, derived the cookie `Secure` flag from a client-controlled header, and fell back to in-memory storage when the database was unreachable while `/health` kept answering 200.

## Decision

The frontend image is `nginxinc/nginx-unprivileged` on port 8080 and proxies `/api` and `/ws` to the backend (`BACKEND_UPSTREAM`), so the browser talks to one origin, cookies are first-party and the CSP keeps `connect-src 'self'`. `FRONTEND_URL` must be an absolute origin; outside `DEV_MODE` plain `http` is accepted only for loopback hosts, and the value is normalized to `scheme://host`. The cookie `Secure` flag follows the `FRONTEND_URL` scheme (override with `COOKIE_SECURE`), and the session cookie uses the `__Host-` prefix when secure. Proxy headers are honoured only from `TRUSTED_PROXY_CIDRS`. `STORAGE_MODE=database` fails at startup when the database is missing or unreachable, and `/health/ready` returns 503 when the store stops answering. The Helm chart derives `FRONTEND_URL` as `https://<first ingress host>` when `ingress.tls` is set, requires `backend.frontendUrl` otherwise (an ingress without TLS would yield a plain-http origin the backend rejects), routes `/ws`, and keeps generated secrets stable across upgrades with `lookup`. Compose publishes only the frontend on loopback and sets the trusted proxy CIDR to the compose network.

## Alternatives considered

- Keep cross-origin API with CORS and `VITE_API_URL`. Rejected because it needs a second public hostname, third-party cookie handling and a wider CSP, and it was what made the shipped configurations drift.
- Derive the cookie `Secure` flag from `X-Forwarded-Proto`. Rejected because any client can set that header when no trusted proxy is configured.
- Keep the memory fallback and log an error. Rejected because a dashboard that looks healthy but lost its data and sessions is worse than a pod that fails readiness.
- Trust all RFC 1918 sources by default in the binary. Rejected: the default is no trusted proxy; the Helm chart sets the private ranges explicitly because it knows it runs behind an ingress.

## Consequences

Operators must provide the public origin (ingress host or `backend.frontendUrl`); the chart refuses to render without it. The OAuth callback URL registered on GitHub is `<public origin>/api/auth/callback`. The frontend container needs writable `/tmp` and `/etc/nginx/conf.d` under a read-only root. A database outage now takes the backend out of rotation instead of serving an empty dashboard.
