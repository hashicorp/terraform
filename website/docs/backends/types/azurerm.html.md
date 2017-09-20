---
layout: "backend-types"
page_title: "Backend Type: azurerm"
sidebar_current: "docs-backends-types-standard-azurerm"
description: |-
  Terraform can store state remotely in Azure Blob Storage.

---

# azurerm (formerly azure)

**Kind: Standard (with state locking)**

Stores the state as a given key in a given bucket on [Microsoft Azure Storage](https://azure.microsoft.com/en-us/documentation/articles/storage-introduction/).

## Example Configuration

```hcl
terraform {
  backend "azurerm" {
    storage_account_name = "abcd1234"
    container_name       = "tfstate"
    key                  = "prod.terraform.tfstate"
  }
}
```

Note that for the access credentials we recommend using a
[partial configuration](/docs/backends/config.html).

## Example Referencing

```hcl
data "terraform_remote_state" "foo" {
  backend = "azurerm"
  config {
    storage_account_name = "terraform123abc"
    container_name       = "terraform-state"
    key                  = "prod.terraform.tfstate"
  }
}
```

## Configuration variables

The following configuration options are supported:

 * `storage_account_name` - (Required) The name of the storage account
 * `container_name` - (Required) The name of the container to use within the storage account
 * `key` - (Required) The key where to place/look for state file inside the container
 * `access_key` / `ARM_ACCESS_KEY` - (Optional) Storage account access key
 * `environment` / `ARM_ENVIRONMENT` - (Optional) The cloud environment to use. Supported values are:
   * `public` (default)
   * `usgovernment`
   * `german`
   * `china`

The following configuration options must be supplied if `access_key` is not.

 * `resource_group_name` - The resource group which contains the storage account.
 * `subscription_id` / `ARM_SUBSCRIPTION_ID` - The Azure Subscription ID.
 * `client_id` / `ARM_CLIENT_ID` - The Azure Client ID.
 * `client_secret` / `ARM_CLIENT_SECRET` - The Azure Client Secret.
 * `tenant_id` / `ARM_TENANT_ID` - The Azure Tenant ID.
