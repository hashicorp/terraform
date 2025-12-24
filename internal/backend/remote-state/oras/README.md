# Terraform ORAS Remote State Backend

## The Big Idea

Use an OCI registry as a remote backend for Terraform state.

This backend stores each workspace's state as an OCI manifest + layer in a repository you control (e.g. a registry you already use for containers), and stores the lock as a separate manifest/tag. That means you can reuse existing registry authentication, authorization, and operational tooling.

> **Note**: Currently only tested with GitHub Container Registry (ghcr.io). Other registries should work but haven't been validated.

## Parameters

All settings are configured in the Terraform backend block. Some can also be set via environment variables.

- `repository` (required): OCI repository in the form `<registry>/<repository>`, without tag or digest.
  - Env: `TF_BACKEND_ORAS_REPOSITORY`
- `compression` (optional, default `none`): `none` or `gzip`.
- `insecure` (optional, default `false`): skip TLS certificate verification.
- `ca_file` (optional): path to a PEM-encoded CA bundle to trust.

Retry/backoff:
- `retry_max` (optional, default `2`): number of retries for transient registry requests. Note: attempts = `retry_max + 1`.
  - Env: `TF_BACKEND_ORAS_RETRY_MAX`
- `retry_wait_min` (optional, default `1`): minimum backoff in seconds.
  - Env: `TF_BACKEND_ORAS_RETRY_WAIT_MIN`
- `retry_wait_max` (optional, default `30`): maximum backoff in seconds.
  - Env: `TF_BACKEND_ORAS_RETRY_WAIT_MAX`

Locking:
- `lock_ttl` (optional, default `0`): lock TTL in seconds. If non-zero, stale locks older than this may be cleared when attempting to acquire a lock. `0` disables.
  - Env: `TF_BACKEND_ORAS_LOCK_TTL`

Rate limiting:
- `rate_limit` (optional, default `0`): maximum registry requests per second. `0` disables.
  - Env: `TF_BACKEND_ORAS_RATE_LIMIT`
- `rate_limit_burst` (optional, default `0`): maximum burst size. If `rate_limit > 0` and burst is `0`, Terraform uses burst `1`.
  - Env: `TF_BACKEND_ORAS_RATE_LIMIT_BURST`

Versioning:
- `versioning { ... }` (optional block): enables version tags for state.
  - `enabled` (optional, default `false`)
  - `max_versions` (optional): maximum historical versions to keep. `0` means unlimited.

## How State Is Stored (Tags)

The backend uses tags as stable pointers:

- State: `state-<workspaceTag>`
- Lock: `locked-<workspaceTag>`

`workspaceTag` is the workspace name if it is a valid OCI tag. Otherwise Terraform uses a stable `ws-<hash>` form and stores the real workspace name in OCI annotations.

If versioning is enabled, each successful state write also tags the same manifest as:

- `state-<workspaceTag>-v<integer>`

Version numbers are chosen by scanning existing tags and taking `(max + 1)`.

## Under the Hood (Wire Format)

Terraform writes OCI manifests using ORAS "manifest v1.1" packing.

State objects:
- Manifest `artifactType`: `application/vnd.terraform.state.v1`
- Layer media type:
  - `application/vnd.terraform.statefile.v1` (no compression)
  - `application/vnd.terraform.statefile.v1+gzip` (gzip)
- Annotations:
  - `org.terraform.workspace`: workspace name
  - `org.terraform.state.updated_at`: RFC3339 timestamp (changes on every Put)

Lock objects:
- Manifest `artifactType`: `application/vnd.terraform.lock.v1`
- Annotations:
  - `org.terraform.workspace`: workspace name
  - `org.terraform.lock.id`: lock ID
  - `org.terraform.lock.info`: JSON-encoded lock metadata

Strict reads:
- Terraform rejects unexpected `artifactType` for both state and lock.
- Terraform rejects unknown state layer media types (it will not "guess" how to decode).
- Invalid lock metadata is treated as an error rather than "no lock".

## Locking

Terraform represents the lock as a dedicated OCI reference (`locked-*`) whose manifest carries lock information in annotations.

Unlocking normally deletes the referenced lock manifest. Some registries do not support manifest deletion via OCI Distribution `DELETE`; in that case Terraform falls back to retagging the lock reference to an `unlocked-*` placeholder.

If `lock_ttl` is set, Terraform may clear a lock older than the TTL during a later lock attempt.

**Note**: There's a theoretical race condition where two concurrent `terraform apply` runs could both believe they acquired the lock. Use CI job concurrency controls for safety.

## Versioning & Retention

If `versioning.enabled` is true, Terraform creates an additional `-vN` tag for each successful state write.

If `versioning.max_versions > 0`, Terraform prunes older versions as part of writing new state. This is not a background task.

Some registries (notably `ghcr.io`) commonly return HTTP 405 for OCI Distribution `DELETE`. In that case, Terraform falls back to deleting the corresponding GitHub Container Package version via the GitHub Packages API. This fallback requires credentials that GitHub accepts for package version enumeration and deletion (the code expects a token with `delete:packages`; exact requirements can vary by token type and package visibility). If deletion can't be performed, old versions may remain and/or the write may fail.

## Authentication

Registry credentials are discovered in this order:

1. Docker credential store (Docker config / credential helpers).
2. Terraform CLI host credentials (from the CLI config / `terraform login`).

If Docker credentials fail or are missing, Terraform falls back to CLI credentials.

## Usage

Typical usage is a backend block in configuration and then `terraform init`.

Example (minimal):

```hcl
terraform {
  backend "oras" {
    repository = "ghcr.io/acme/terraform-state"
  }
}
```

Example (gzip + versioning):

```hcl
terraform {
  backend "oras" {
    repository  = "ghcr.io/myorg/infra-state"
    compression = "gzip"

    versioning {
      enabled      = true
      max_versions = 10
    }
  }
}
```

Example (GitHub Actions CI):

```yaml
- name: Terraform Init
  env:
    TF_BACKEND_ORAS_REPOSITORY: ghcr.io/${{ github.repository_owner }}/tf-state
  run: |
    echo "${{ secrets.GITHUB_TOKEN }}" | docker login ghcr.io -u $ --password-stdin
    terraform init
```

The `GITHUB_TOKEN` needs `packages:write` scope. For version retention/deletion, you'll also need `delete:packages`.

## Troubleshooting

**"unauthorized" or "denied" errors**
- Check `docker login <registry>` works
- For GHCR: token needs `read:packages` and `write:packages` scopes

**Lock stuck after crashed run**
- Set `lock_ttl = 300` to auto-expire locks after 5 minutes
- Or manually delete the `locked-<workspace>` tag in the registry UI

**Version deletion fails on GHCR (405 error)**
- GHCR doesn't support OCI DELETE, we fall back to GitHub API
- Your token needs `delete:packages` scope
- If it still fails, old versions will accumulate (not a blocker, just messy)

**Debug mode**
```bash
TF_LOG=DEBUG terraform plan
```

## Limitations / Future Enhancements

Limitations:
- Only GHCR is tested; other registries may have quirks.
- Deletion semantics vary by registry. Lock/unlock and retention behavior can degrade when OCI Distribution `DELETE` is unsupported.
- Retention is enforced only on writes; there is no background garbage collection.
- `lock_ttl` is evaluated during lock attempts, not proactively.
- `insecure = true` disables TLS verification and should be used only for controlled environments.

Future enhancements:
- Clearer introspection commands for lock/version tags.
- Better registry-specific retention strategies beyond GHCR.
- Testing with ECR, GCR, ACR, Harbor, etc.
