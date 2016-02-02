---
layout: "vault"
page_title: "Vault: vault_auth_backend"
sidebar_current: "docs-vault-resource-auth-backend"
description: |-
  Provides a resoruce for mounting Vault auth backends.
---

# vault\_auth\_backend

Provides a resource for mounting Vault auth backends.

## Example Usage

```
resource "vault_auth_backend" "foo" {
  type        = "app-id"
  path        = "foo/bar"
  description = "Authentication for Apps"
}
```

## Argument Reference

More detail about each of these fields can be found in the [official
Vault API documentation](https://www.vaultproject.io/docs/http/sys-auth.html).

The following arguments are supported:

* `type` - _(String, Required)_ The name of the backend type, such as "app-id". The Vault documentation has the [full list of available auth backend types](https://www.vaultproject.io/docs/auth/index.html).
* `path` - _(String, Optional)_ The path where you'd like to mount the auth backend. Defaults to the auth type name.
* `description` - _(String, Optional)_ A human-friendly description for the mount. Defaults to "Managed by Terraform".
