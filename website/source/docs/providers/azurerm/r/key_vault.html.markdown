---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_key_vault"
sidebar_current: "docs-azurerm-resource-key-vault"
description: |-
  Create a Key Vault.
---

# azurerm\_key\_vault

Create a Key Vault.

## Example Usage

```hcl
resource "azurerm_resource_group" "test" {
  name     = "resourceGroup1"
  location = "West US"
}

resource "azurerm_key_vault" "test" {
  name                = "testvault"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"

  sku {
    name = "standard"
  }

  tenant_id = "d6e396d0-5584-41dc-9fc0-268df99bc610"

  access_policy {
    tenant_id = "d6e396d0-5584-41dc-9fc0-268df99bc610"
    object_id = "d746815a-0433-4a21-b95d-fc437d2d475b"

    key_permissions = [
      "all",
    ]

    secret_permissions = [
      "get",
    ]
  }

  enabled_for_disk_encryption = true

  tags {
    environment = "Production"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the Key Vault resource. Changing this
    forces a new resource to be created.

* `location` - (Required) Specifies the supported Azure location where the resource exists.
    Changing this forces a new resource to be created.

* `resource_group_name` - (Required) The name of the resource group in which to
    create the namespace. Changing this forces a new resource to be created.

* `sku` - (Required) An SKU block as described below.

* `tenant_id` - (Required) The Azure Active Directory tenant ID that should be
    used for authenticating requests to the key vault.

* `access_policy` - (Required) An access policy block as described below. At least
    one policy is required up to a maximum of 16.

* `enabled_for_deployment` - (Optional) Boolean flag to specify whether Azure Virtual
    Machines are permitted to retrieve certificates stored as secrets from the key
    vault. Defaults to false.

* `enabled_for_disk_encryption` - (Optional) Boolean flag to specify whether Azure
    Disk Encryption is permitted to retrieve secrets from the vault and unwrap keys.
    Defaults to false.

* `enabled_for_template_deployment` - (Optional) Boolean flag to specify whether
    Azure Resource Manager is permitted to retrieve secrets from the key vault.
    Defaults to false.

* `tags` - (Optional) A mapping of tags to assign to the resource.

`sku` supports the following:

* `name` - (Required) SKU name to specify whether the key vault is a `standard`
    or `premium` vault.

`access_policy` supports the following:

* `tenant_id` - (Required) The Azure Active Directory tenant ID that should be used
    for authenticating requests to the key vault. Must match the `tenant_id` used
    above.

* `object_id` - (Required) The object ID of a user, service principal or security
    group in the Azure Active Directory tenant for the vault. The object ID must
    be unique for the list of access policies.

* `key_permissions` - (Required) List of key permissions, must be one or more from
    the following: `all`, `backup`, `create`, `decrypt`, `delete`, `encrypt`, `get`,
    `import`, `list`, `restore`, `sign`, `unwrapKey`, `update`, `verify`, `wrapKey`.

* `secret_permissions` - (Required) List of secret permissions, must be one or more
    from the following: `all`, `delete`, `get`, `list`, `set`.

## Attributes Reference

The following attributes are exported:

* `id` - The Vault ID.
* `vault_uri` - The URI of the vault for performing operations on keys and secrets.
