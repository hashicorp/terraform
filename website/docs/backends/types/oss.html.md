---
layout: "backend-types"
page_title: "Backend Type: oss"
sidebar_current: "docs-backends-types-standard-oss"
description: |-
  Terraform can store state remotely in OSS and lock that state with OSS.
---

# OSS

**Kind: Standard (with locking via TableStore)**

Stores the state as a given key in a given bucket on Stores
[Alibaba Cloud OSS](https://www.alibabacloud.com/help/product/31815.htm).
This backend also supports state locking and consistency checking via
[Alibaba Cloud Table Store](https://www.alibabacloud.com/help/doc-detail/27280.htm), which can be enabled by setting
the `tablestore_table` field to an existing TableStore table name.

## Example Configuration

```hcl
terraform {
  backend "oss" {
    bucket = "bucket-for-terraform-state"
    prefix   = "path/mystate"
    key   = "version-1.tfstate"
    region = "cn-beijing"
    tablestore_endpoint = "https://terraform-remote.cn-hangzhou.ots.aliyuncs.com"
    tablestore_table = "statelock"
  }
}
```

This assumes we have a [OSS Bucket](https://www.terraform.io/docs/providers/alicloud/r/oss_bucket.html) created called `bucket-for-terraform-state`,
a [OTS Instance](https://www.terraform.io/docs/providers/alicloud/r/ots_instance.html) called `terraform-remote` and
a [OTS TableStore](https://www.terraform.io/docs/providers/alicloud/r/ots_table.html) called `statelock`. The
Terraform state will be written into the file `path/mystate/version-1.tfstate`.


## Using the OSS remote state

To make use of the OSS remote state we can use the
[`terraform_remote_state` data
source](/docs/providers/terraform/d/remote_state.html).

```hcl
terraform {
  backend "oss" {
    bucket = "remote-state-dns"
    prefix = "mystate/state"
    key    = "terraform.tfstate"
    region = "cn-beijing"
  }
}
```

The `terraform_remote_state` data source will return all of the root outputs
defined in the referenced remote state, an example output might look like:

```
data "terraform_remote_state" "network" {
    backend   = "oss"
    config    = {
        bucket = "remote-state-dns"
        key    = "terraform.tfstate"
        prefix = "mystate/state"
        region = "cn-beijing"
    }
    outputs   = {}
    workspace = "default"
}
```

## Configuration variables

The following configuration options or environment variables are supported:

 * `access_key` - (Optional) Alicloud access key. It supports environment variables `ALICLOUD_ACCESS_KEY` and  `ALICLOUD_ACCESS_KEY_ID`.
 * `secret_key` - (Optional) Alicloud secret access key. It supports environment variables `ALICLOUD_SECRET_KEY` and  `ALICLOUD_ACCESS_KEY_SECRET`.
 * `security_token` - (Optional) STS access token. It supports environment variable `ALICLOUD_SECURITY_TOKEN`.
 * `region` - (Optional) The region of the OSS bucket. It supports environment variables `ALICLOUD_REGION` and `ALICLOUD_DEFAULT_REGION`.
 * `endpoint` - (Optional) A custom endpoint for the OSS API. It supports environment variables `ALICLOUD_OSS_ENDPOINT` and `OSS_ENDPOINT`.
 * `bucket` - (Required) The name of the OSS bucket.
 * `prefix` - (Opeional) The path directory of the state file will be stored. Default to "env:".
 * `key` - (Optional) The name of the state file. Defaults to `terraform.tfstate`.
 * `tablestore_endpoint` / `ALICLOUD_TABLESTORE_ENDPOINT` - (Optional) A custom endpoint for the TableStore API.
 * `tablestore_table` - (Optional) A TableStore table for state locking and consistency.
 * `encrypt` - (Optional) Whether to enable server side
   encryption of the state file. If it is true, OSS will use 'AES256' encryption algorithm to encrypt state file.
 * `acl` - (Optional) [Object
   ACL](https://www.alibabacloud.com/help/doc-detail/52284.htm)
   to be applied to the state file.

-> **Note:** If you want to store state in the custom OSS endpoint, you can specify a enviornment variable `OSS_ENDPOINT`, like "oss-cn-beijing-internal.aliyuncs.com"

