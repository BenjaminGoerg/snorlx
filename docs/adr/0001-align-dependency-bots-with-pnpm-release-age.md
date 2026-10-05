# 0001: Wait two days before dependency bot updates

- Status: Accepted
- Date: 2026-10-05

## Context

pnpm rejects lockfile entries published less than 1440 minutes ago. Dependabot and Renovate were opening pull requests for releases inside that window. `pnpm install` then failed the supply-chain check, including the lockfile sync workflow, because a lockfile that already matches the manifests is verified before resolution runs again.

## Decision

Set `minimumReleaseAge` to 1440 in `pnpm-workspace.yaml`. Renovate waits two days before proposing a release. Dependabot waits two days before opening an npm pull request. The frontend image installs the pnpm version declared in `package.json` `packageManager`.

## Alternatives considered

- Set pnpm `minimumReleaseAge` to 0 so bots can ship same-day releases. Rejected because that removes the supply-chain delay that was failing these installs.
- Keep adding each bot bump to `minimumReleaseAgeExclude`. Rejected because every new release needs a manual exception and the pull request stays red until that exception exists.
- Wait exactly one day in the bots. Rejected because a scheduled run can still select a release a few minutes inside pnpm's cutoff, and CI evaluates the cutoff again at install time.

## Consequences

Releases younger than two days do not get a bot pull request. A fix that must land sooner still needs a `minimumReleaseAgeExclude` entry. The frontend image follows `package.json` when the pnpm version changes.
