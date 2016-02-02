---
layout: "vault"
page_title: "Vault: vault_policy"
sidebar_current: "docs-vault-resource-policy"
description: |-
  Provides a resource for managing Vault policies.
---

# vault\_policy

Provides a resource for managing Vault policies.

## Example Usage

```
resource "vault_policy" "foo" {
  name  = "operator"
  rules = "${file("${path.module}/policies/operator.hcl")}"
}
```

## Argument Reference

More detail about each of these fields can be found in the [official
Vault API documentation](https://www.vaultproject.io/docs/http/sys-policy.html).

The following arguments are supported:

* `name` - _(String, Required)_ The name of the policy. This will be used as an identifier for the policy.
* `rules` - _(String, Required)_ The policy document. See [Vault documentation on policies](https://www.vaultproject.io/docs/concepts/policies.html) for details on Vault policy syntax.
