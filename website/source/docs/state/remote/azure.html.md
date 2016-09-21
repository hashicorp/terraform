---
layout: "remotestate"
page_title: "Remote State Backend: azure"
sidebar_current: "docs-state-remote-azure"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# azure

Stores the state as a given key in a given bucket on [Microsoft Azure Storage](https://azure.microsoft.com/en-us/documentation/articles/storage-introduction/).

-> **Note:** Passing credentials directly via config options will
make them included in cleartext inside the persisted state.
Use of environment variables or config file is recommended.

## Example Usage

```
terraform remote config \
  -backend=azure \
  -backend-config="storage_account_name=terraform123abc" \
  -backend-config="container_name=terraform-state" \
  -backend-config="key=prod.terraform.tfstate"
```

## Example Referencing

```hcl
# setup remote state data source
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
