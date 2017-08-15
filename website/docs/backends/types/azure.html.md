---
layout: "backend-types"
page_title: "Backend Type: azure"
sidebar_current: "docs-backends-types-standard-azure"
description: |-
  Terraform can store state remotely in Azure Storage.
---

# azure

**Kind: Standard (with no locking)**

Stores the state as a given key in a given bucket on [Microsoft Azure Storage](https://azure.microsoft.com/en-us/documentation/articles/storage-introduction/).

## Example Configuration

```hcl
terraform {
  backend "azure" {
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
  backend = "azure"
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
 * `access_key` / `ARM_ACCESS_KEY` - (Required) Storage account access key
 * `lease_id` / `ARM_LEASE_ID` - (Optional) If set, will be used when writing to storage blob.
 * `resource_group_name` - (Optional) The name of the resource group for the storage account. Required if `access_key` isn't specified.
 * `environment` / `ARM_ENVIRONMENT` - (Optional) The cloud environment to use. Supported values are:
   * `public` (default)
   * `usgovernment`
   * `german`
   * `china`
