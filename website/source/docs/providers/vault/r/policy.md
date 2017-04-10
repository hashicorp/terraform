---
layout: "vault"
page_title: "Vault: vault_policy resource"
sidebar_current: "docs-vault-resource-policy"
description: |-
  Writes arbitrary policies for Vault
---

# vault\_policy


## Example Usage

```hcl
resource "vault_policy" "example" {
  name = "dev-team"

  policy = <<EOT
path "secret/my_app" {
  policy = "write"
}
EOT
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the policy

* `policy` - (Required) String containing a Vault policy

## Attributes Reference

No additional attributes are exported by this resource.
