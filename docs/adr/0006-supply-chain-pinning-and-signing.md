# 0006: Pin the supply chain and sign release artifacts

- Status: Accepted
- Date: 2026-10-05

## Context

CI resolved GitHub Actions by mutable major tags, installed `govulncheck` and `gosec` at `@latest`, and three workflows deleted `pnpm-lock.yaml` and re-resolved dependencies whenever `--frozen-lockfile` failed. Base images were floating tags, the frontend image carried OS packages with published fixes because only OpenSSL was upgraded, Trivy never failed a job, and released images and the chart were neither signed nor accompanied by provenance or an SBOM.

## Decision

Pin every action to a full commit SHA with the version in a trailing comment, and let Dependabot (new `github-actions` ecosystem) move the pins. Install `govulncheck` and `gosec` at fixed versions. Remove the lockfile fallback: a lockfile mismatch fails CI, and the existing `lockfile-update` workflow keeps bot pull requests in sync (restricted to same-repository branches). Pin base images by digest and run a full `apk upgrade` in the final stages. Trivy fails on fixable CRITICAL and HIGH findings while still uploading SARIF. Release images are built with `provenance: mode=max` and `sbom: true` and signed with keyless cosign; the chart is signed by digest after `helm push`. Patch-level dependency updates stay disabled per the repository policy.

## Alternatives considered

- Keep tag pins and rely on GitHub's "require SHA pinning" setting. Rejected because the setting is a repository toggle outside the code and tag pins remain mutable until it is enabled; the workflows pin explicitly and the setting can be enabled on top.
- Keep the lockfile fallback for bot pull requests. Rejected because it also ran for human pull requests and replaced the supply-chain contract with whatever the registry returned that day.
- Sign with a long-lived cosign key stored as a secret. Rejected in favour of keyless signing with the workflow OIDC identity: nothing to rotate or leak, and consumers verify against the workflow identity.
- Keep Trivy advisory-only. Rejected because the open alerts stayed unfixed for weeks without anyone noticing.

## Consequences

Dependabot opens pull requests for action pins and image digests; the `lockfile-update` workflow handles the npm ones. A new base image CVE with a fix blocks the security workflow until the digest is bumped, which is the intended signal. Consumers can verify images and the chart with `cosign verify --certificate-identity-regexp 'https://github.com/banshee86vr/snorlx/' --certificate-oidc-issuer https://token.actions.githubusercontent.com`. Required status checks on the `main` ruleset are a repository setting and are not changed by this decision.
