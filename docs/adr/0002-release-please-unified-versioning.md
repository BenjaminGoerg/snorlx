# 0002: Release the product with release-please

- Status: Accepted
- Date: 2026-10-05

## Context

Frontend, backend, and the Helm chart shipped as version 1.0.0 in source with no tags, GitHub Releases, or published images. A release has to move those versions together, publish `ghcr.io/banshee86vr/snorlx-backend`, `ghcr.io/banshee86vr/snorlx-frontend`, and the chart at `oci://ghcr.io/banshee86vr/charts/snorlx`, and stay compatible with GitHub immutable releases.

## Decision

Use release-please in manifest mode with one root `node` package. `extra-files` bumps the frontend and MCP manifests, both Helm chart versions, and the annotated lines in `backend/internal/version/version.go` and `mcp/src/server.ts`. One tag `vX.Y.Z` and one GitHub Release. The workflow uses a GitHub App token so pull request CI runs. Releases are published immediately with no assets, which fits immutable releases without a draft-first upload. `gh release verify` checks the attestation. Images are multi-arch and tagged only `X.Y.Z`. The chart image tag defaults to `.Chart.AppVersion`. The first release is `1.0.0`, with changelog history starting at `be22b7eba2ed1e0417fba776b8bdea8ed4d76cfe`.

## Alternatives considered

- Linked or independent component versions. Rejected because the frontend, backend, and chart deploy as one product.
- `GITHUB_TOKEN` or a personal access token. Rejected because token-created pull requests do not run CI, and a personal token expires and is tied to one account.
- Floating image tags (`X.Y`, `X`, `latest`). Rejected because GHCR tags are mutable and the release model is immutable.
- Attach the chart `.tgz` to the GitHub Release. Rejected because immutable releases lock assets at publish time, which needs a draft-first flow, and the OCI chart is the install path.
- chart-releaser on `gh-pages`. Rejected because the chart is installed from GHCR, not a git index.
- Inject the backend version with Docker `-ldflags`. Rejected because release-please would not bump a source file and local `go run` would not show the release version.

## Consequences

A broken release cannot reuse its tag. Ship a new patch. Image and chart jobs can be re-run because GHCR tags can be overwritten. A later run publishes a `vX.Y.Z` tag that already points at the commit, so a failed immutable-release check does not strand the artifacts. `chore` commits still open a patch release pull request; merge it only when shipping. The GitHub App and the immutable-releases setting are manual and must exist before the first release.
