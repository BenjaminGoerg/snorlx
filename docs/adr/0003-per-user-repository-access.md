# 0003: Per-user repository access and login allowlist

- Status: Accepted
- Date: 2026-10-05

## Context

Every repository, workflow, run, job, deployment and score was stored in one global table set with no link to the user who synced it, and any GitHub account could sign in. A user who synced private repositories with their own token exposed their metadata, run history, cached job steps and real-time WebSocket events to every other account that logged in. Actions proxied to GitHub (rerun, cancel, logs) were protected by GitHub itself, but reads from the local store were not.

## Decision

Record visibility per user in a `user_repositories(user_id, repo_id)` table, written by `runSync` when a repository is stored with that user's token. Every list and aggregate query (`ListRepositories`, `ListWorkflows`, `ListRuns`, `ListActivePipelines`, `GetDashboardSummary`, `GetTrends`, `ListLatestRepositoryScores`, `ListOrganizations`, `BackfillDeploymentRuns`) takes the caller's user ID and joins that table. Single-object handlers resolve the object, then call `HasRepositoryAccess` and answer `404 Not found` when the caller may not see the repository, so missing and forbidden objects are indistinguishable. The WebSocket hub addresses every message to explicit user IDs: repository events go to `ListUsersWithRepositoryAccess(repo)`, sync progress goes only to the user who started the sync. Webhook events for repositories nobody synced are dropped. `ALLOWED_GITHUB_USERS` and `ALLOWED_GITHUB_ORGS` restrict who may sign in; with both empty the previous behaviour (any GitHub account) is kept so existing deployments do not lock their users out.

## Alternatives considered

- PostgreSQL row level security with `SET app.user_id`. Rejected because the memory storage mode has no equivalent, the application would still need the user ID on every call, and RLS hides the authorization logic from the code that reviewers read.
- Re-check GitHub permissions on every read. Rejected because it multiplies GitHub API calls per page view and still needs a local answer for WebSocket fan-out.
- Mandatory allowlist. Rejected because it breaks every existing single-tenant deployment on upgrade; the allowlist is documented as the recommended production setting instead.

## Consequences

A repository synced by two users is visible to both and both receive its events. Users must sync once after upgrading before they see data again, because existing rows have no access grants. The `Storage` interface carries a user ID on scoped methods, which makes an unscoped query a compile-time choice rather than an omission. Organizations are visible only through repositories the user may see.
