# 0004: Encrypt stored GitHub tokens with a key derived from SESSION_SECRET

- Status: Accepted
- Date: 2026-10-05

## Context

GitHub OAuth App tokens with the `repo` scope were stored in clear text in `users.access_token`. They do not expire, so a database dump or backup gave write access to every repository of every user. `SESSION_SECRET` was required by the configuration and documented as an encryption key but was not used anywhere.

## Decision

Encrypt `access_token` and `refresh_token` with AES-256-GCM in the database storage adapter (`internal/tokencrypt`). The key is derived from `SESSION_SECRET` with HKDF-SHA256 and a fixed info string, so operators keep configuring one secret. Stored values carry the prefix `enc:v1:`; a value without the prefix is treated as legacy plaintext, returned as is, and re-encrypted on the next login. Outside `DEV_MODE` the secret must be at least 32 characters. The memory storage keeps tokens in process memory only and does not encrypt. The Helm chart generates the secret on first install and reads it back from the release Secret on upgrade so the key stays stable.

## Alternatives considered

- Separate `TOKEN_ENCRYPTION_KEY` variable. Rejected because it adds a second secret with the same lifecycle and `SESSION_SECRET` was already mandatory and documented for this purpose.
- External KMS or Vault transit encryption. Rejected for a self-hosted dashboard: it adds an infrastructure dependency the target deployments do not have, and the key derivation can be swapped later behind the same `Cipher` type.
- Not storing the token and asking users to re-authenticate on every sync. Rejected because background syncs and webhook-driven refreshes need a token without a browser session.

## Consequences

Rotating `SESSION_SECRET` makes stored tokens undecryptable: authentication fails for those users until they sign in again, which re-encrypts with the new key. A database compromise alone no longer yields usable GitHub tokens; the attacker also needs the application secret. Existing plaintext rows are migrated lazily, so a one-off re-encryption job is not required.
