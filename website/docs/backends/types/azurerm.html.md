---
layout: "backend-types"
page_title: "Backend Type: azurerm"
sidebar_current: "docs-backends-types-standard-azurerm"
description: |-
  Terraform can store state remotely in Azure Blob Storage.

---

# azurerm

**Kind: Standard (with state locking)**

Stores the state as a Blob with the given Key within the Blob Container within [the Blob Storage Account](https://docs.microsoft.com/azure/storage/common/storage-introduction). This backend also supports state locking and consistency checking via native capabilities of Azure Blob Storage.

## Example Configuration

When authenticating using the Azure CLI or a Service Principal:

```hcl
terraform {
  backend "azurerm" {
    storage_account_name = "abcd1234"
    container_name       = "tfstate"
    key                  = "prod.terraform.tfstate"
  }
}
```

When authenticating using Managed Service Identity (MSI):

```hcl
terraform {
  backend "azurerm" {
    storage_account_name = "abcd1234"
    container_name       = "tfstate"
    key                  = "prod.terraform.tfstate"
    use_msi              = true
    subscription_id  = "00000000-0000-0000-0000-000000000000"
    tenant_id        = "00000000-0000-0000-0000-000000000000"
  }
}
```

When authenticating using the Access Key associated with the Storage Account:

```hcl
terraform {
  backend "azurerm" {
    storage_account_name = "abcd1234"
    container_name       = "tfstate"
    key                  = "prod.terraform.tfstate"

    # rather than defining this inline, the Access Key can also be sourced
    # from an Environment Variable - more information is available below.
    access_key = "abcdefghijklmnopqrstuvwxyz0123456789..."
  }
}
```

When authenticating using a SAS Token associated with the Storage Account:

```hcl
terraform {
  backend "azurerm" {
    storage_account_name = "abcd1234"
    container_name       = "tfstate"
    key                  = "prod.terraform.tfstate"

    # rather than defining this inline, the SAS Token can also be sourced
    # from an Environment Variable - more information is available below.
    sas_token = "abcdefghijklmnopqrstuvwxyz0123456789..."
  }
}
```

-> **NOTE:** When using a Service Principal or an Access Key - we recommend using a [Partial Configuration](/docs/backends/config.html) for the credentials.

## Example Referencing

When authenticating using a Service Principal:

```hcl
data "terraform_remote_state" "foo" {
  backend = "azurerm"
  config = {
    storage_account_name = "terraform123abc"
    container_name       = "terraform-state"
    key                  = "prod.terraform.tfstate"
  }
}
```

When authenticating using Managed Service Identity (MSI):

```hcl
data "terraform_remote_state" "foo" {
  backend = "azurerm"
  config = {
    storage_account_name = "terraform123abc"
    container_name       = "terraform-state"
    key                  = "prod.terraform.tfstate"
    use_msi              = true
    subscription_id  = "00000000-0000-0000-0000-000000000000"
    tenant_id        = "00000000-0000-0000-0000-000000000000"
  }
}
```

When authenticating using the Access Key associated with the Storage Account:

```hcl
data "terraform_remote_state" "foo" {
  backend = "azurerm"
  config = {
    storage_account_name = "terraform123abc"
    container_name       = "terraform-state"
    key                  = "prod.terraform.tfstate"

    # rather than defining this inline, the Access Key can also be sourced
    # from an Environment Variable - more information is available below.
    access_key = "abcdefghijklmnopqrstuvwxyz0123456789..."
  }
}

When authenticating using a SAS Token associated with the Storage Account:

```hcl
data "terraform_remote_state" "foo" {
  backend = "azurerm"
  config = {
    storage_account_name = "terraform123abc"
    container_name       = "terraform-state"
    key                  = "prod.terraform.tfstate"

    # rather than defining this inline, the SAS Token can also be sourced
    # from an Environment Variable - more information is available below.
    sas_token = "abcdefghijklmnopqrstuvwxyz0123456789..."
  }
}
```

## Configuration variables

The following configuration options are supported:

* `storage_account_name` - (Required) The Name of [the Storage Account](https://www.terraform.io/docs/providers/azurerm/r/storage_account.html).

* `container_name` - (Required) The Name of [the Storage Container](https://www.terraform.io/docs/providers/azurerm/r/storage_container.html) within the Storage Account.

* `key` - (Required) The name of the Blob used to retrieve/store Terraform's State file inside the Storage Container.

* `environment` - (Optional) The Azure Environment which should be used. This can also be sourced from the `ARM_ENVIRONMENT` environment variable. Possible values are `public`, `china`, `german`, `stack` and `usgovernment`. Defaults to `public`.

* `endpoint` - (Optional) The Custom Endpoint for Azure Resource Manager. This can also be sourced from the `ARM_ENDPOINT` environment variable.

~> **NOTE:** An `endpoint` should only be configured when using Azure Stack.

---

When authenticating using the Managed Service Identity (MSI) - the following fields are also supported:

* `subscription_id` - (Optional) The Subscription ID in which the Storage Account exists. This can also be sourced from the `ARM_SUBSCRIPTION_ID` environment variable.

* `tenant_id` - (Optional) The Tenant ID in which the Subscription exists. This can also be sourced from the `ARM_TENANT_ID` environment variable.

* `msi_endpoint` - (Optional) The path to a custom Managed Service Identity endpoint which is automatically determined if not specified. This can also be sourced from the `ARM_MSI_ENDPOINT` environment variable.

* `use_msi` - (Optional) Should Managed Service Identity authentication be used? This can also be sourced from the `ARM_USE_MSI` environment variable.

---

When authenticating using a SAS Token associated with the Storage Account - the following fields are also supported:

* `sas_token` - (Optional) The SAS Token used to access the Blob Storage Account. This can also be sourced from the `ARM_SAS_TOKEN` environment variable.

---

When authenticating using the Storage Account's Access Key - the following fields are also supported:

* `access_key` - (Optional) The Access Key used to access the Blob Storage Account. This can also be sourced from the `ARM_ACCESS_KEY` environment variable.

---

When authenticating using a Service Principal - the following fields are also supported:

* `resource_group_name` - (Required) The Name of the Resource Group in which the Storage Account exists.

* `client_id` - (Optional) The Client ID of the Service Principal. This can also be sourced from the `ARM_CLIENT_ID` environment variable.

* `client_secret` - (Optional) The Client Secret of the Service Principal. This can also be sourced from the `ARM_CLIENT_SECRET` environment variable.

* `subscription_id` - (Optional) The Subscription ID in which the Storage Account exists. This can also be sourced from the `ARM_SUBSCRIPTION_ID` environment variable.

* `tenant_id` - (Optional) The Tenant ID in which the Subscription exists. This can also be sourced from the `ARM_TENANT_ID` environment variable.
