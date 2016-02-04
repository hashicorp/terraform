---
layout: "vault"
page_title: "Vault: vault_secret"
sidebar_current: "docs-vault-resource-secret"
description: |-
  Provides a resource for managing Vault secrets.
---

# vault\_secret

Provides a resource for managing Vault secrets in generic or cubbyhole
backends.

## Example Usage

Writing a secret to a generic backend that we mount ourselves:

```
resource "vault_secret_backend" "foo" {
  type = "generic"
  path = "my-secrets"
}

resource "vault_secret" "foo" {
  path = "${vault_secret_backend.foo}/a-secret"
  data {
    hush = "secrets"
  }
}
```

Writing a secret to the built-in cubbyhole backend:

```
resource "vault_secret" "foo" {
  path = "cubbyhole/a-secret"
  data {
    hush = "secrets"
  }
}
```

## Argument Reference

More detail about each of these fields can be found in the official
Vault API documentation on the [generic](https://www.vaultproject.io/docs/secrets/generic/index.html) and [cubbyhole](https://www.vaultproject.io/docs/secrets/cubbyhole/index.html) backends.

The following arguments are supported:

* `path` (String, Required) The location and identifier for this secret.
* `data` (Map of Strings, Required) Data to be written for this secret.
* `ttl` (String, Optional) The Time To Live for the entry. This value, converted to seconds, is round-tripped on read operations as the `lease_duration` parameter. Vault takes no action when this value expires; it is only meant as a way for a writer of a value to indicate to readers how often they should check for new entries. (Note this argument is only valid for `generic` backends. It has no meaning for `cubbyhole` secrets.)
