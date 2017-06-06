---
layout: "vault"
page_title: "Vault: vault_auth_backend resource"
sidebar_current: "docs-vault-auth-backend"
description: |-
  Writes arbitrary policies for Vault
---

# vault\_auth\_backend


## Example Usage

```hcl
resource "vault_auth_backend" "example" {
  type = "github"
}
```

## Argument Reference

The following arguments are supported:

* `type` - (Required) The name of the policy

* `path` - (Optional) The path to mount the auth backend. This defaults to the name.

* `description` - (Optional) A description of the auth backend

## Attributes Reference

No additional attributes are exported by this resource.
