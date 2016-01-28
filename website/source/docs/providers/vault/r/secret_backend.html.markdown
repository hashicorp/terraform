---
layout: "vault"
page_title: "Vault: vault_secret_backend"
sidebar_current: "docs-vault-resource-secret-backend"
description: |-
  Provides a resoruce for mounting Vault secret backends.
---

# vault\_secret\_backend

Provides a resource for mounting Vault secret backends.

## Example Usage

```
resource "vault_secret_backend" "foo" {
  type              = "generic"
  path              = "foo/bar"
  description       = "My secrets"
  default_lease_ttl = "100m"
  max_lease_ttl     = "300m"
}
```

## Argument Reference

More detail about each of these fields can be found in the [official
Vault documentation](https://www.vaultproject.io/docs/http/sys-mounts.html).

The following arguments are supported:

* `type` - _(String, Required)_ The name of the backend type, such as "aws".
* `path` - _(String, Optional)_ The path where you'd like to mount the secret backend. Defaults to the type name.
* `description` - _(String, Optional)_ A human-friendly description for the mount. Defaults to "Managed by Terraform".
* `default_lease_ttl` - _(String, Optional)_ Overrides the global setting for the default lease TTL. Must be a string parseable by [`time.ParseDuration`](https://golang.org/pkg/time/#ParseDuration).
* `max_lease_ttl` - _(String, Optional)_ Overrides the global setting for the max lease TTL. Must be a string parseable by [`time.ParseDuration`](https://golang.org/pkg/time/#ParseDuration).
