---
layout: "vault"
page_title: "Vault: vault_audit_backend"
sidebar_current: "docs-vault-resource-audit-backend"
description: |-
  Provides a resoruce for mounting Vault audit backends.
---

# vault\_audit\_backend

Provides a resource for mounting Vault audit backends.

## Example Usage

```
resource "vault_audit_backend" "foo" {
  type        = "file"
  path        = "foo/bar"
  description = "Store logs in a file"
  options {
    path = "/var/log/vault-audit.log"
  }
}
```

## Argument Reference

More detail about each of these fields can be found in the [official
Vault API documentation](https://www.vaultproject.io/docs/http/sys-audit.html).

The following arguments are supported:

* `type` - _(String, Required)_ The name of the backend type, such as "app-id". The Vault documentation has the [full list of available audit backend types](https://www.vaultproject.io/docs/audit/index.html).
* `path` - _(String, Optional)_ The path where you'd like to mount the audit backend. Defaults to the audit type name.
* `description` - _(String, Optional)_ A human-friendly description for the mount. Defaults to "Managed by Terraform".
* `options` - _(Map of Strings, Optional)_ A set of options to configure the backend. Options depend on the backend type. For example, the [file backend](https://www.vaultproject.io/docs/audit/file.html) requires `path`. See the [Vault documentation on audit backends](https://www.vaultproject.io/docs/audit/index.html) for details.
