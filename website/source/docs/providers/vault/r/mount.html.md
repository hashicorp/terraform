---
layout: "vault"
page_title: "Vault: vault_mount resource"
sidebar_current: "docs-vault-resource-mount"
description: |-
  Managing the mounting of secret backends in Vault
---

# vault\_mount


## Example Usage

```hcl
resource "vault_mount" "example" {
  path = "dummy"
  type = "generic"
  description = "This is an example mount"
}
```

## Argument Reference

The following arguments are supported:

* `path` - (Required) Where the secret backend will be mounted

* `type` - (Required) Type of the backend, such as "aws"

* `description` - (Optional) Human-friendly description of the mount

* `default_lease_ttl_seconds` - (Optional) Default lease duration for tokens and secrets in seconds

* `max_lease_ttl_seconds` - (Optional) Maximum possible lease duration for tokens and secrets in seconds

## Attributes Reference

No additional attributes are exported by this resource.
