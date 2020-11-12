---
layout: "language"
page_title: "Backend Type: cos"
sidebar_current: "docs-backends-types-standard-cos"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# COS

**Kind: Standard (with locking)**

Stores the state as an object in a configurable prefix in a given bucket on [Tencent Cloud Object Storage](https://intl.cloud.tencent.com/product/cos) (COS).
This backend also supports [state locking](/docs/state/locking.html).

~> **Warning!** It is highly recommended that you enable [Object Versioning](https://intl.cloud.tencent.com/document/product/436/19883)
on the COS bucket to allow for state recovery in the case of accidental deletions and human error.

## Example Configuration

```hcl
terraform {
  backend "cos" {
    region = "ap-guangzhou"
    bucket = "bucket-for-terraform-state-1258798060"
    prefix = "terraform/state"
  }
}
```

This assumes we have a [COS Bucket](https://www.terraform.io/docs/providers/tencentcloud/r/cos_bucket.html) created named `bucket-for-terraform-state-1258798060`,
Terraform state will be written into the file `terraform/state/terraform.tfstate`.

## Data Source Configuration

To make use of the COS remote state in another configuration, use the [`terraform_remote_state` data source](/docs/providers/terraform/d/remote_state.html).

```hcl
data "terraform_remote_state" "foo" {
  backend = "cos"

  config = {
    region = "ap-guangzhou"
    bucket = "bucket-for-terraform-state-1258798060"
    prefix = "terraform/state"
  }
}
```

## Configuration variables

The following configuration options or environment variables are supported:

 * `secret_id` - (Optional) Secret id of Tencent Cloud. It supports environment variables `TENCENTCLOUD_SECRET_ID`.
 * `secret_key` - (Optional) Secret key of Tencent Cloud. It supports environment variables `TENCENTCLOUD_SECRET_KEY`.
 * `region` - (Optional) The region of the COS bucket. It supports environment variables `TENCENTCLOUD_REGION`.
 * `bucket` - (Required) The name of the COS bucket. You shall manually create it first.
 * `prefix` - (Optional) The directory for saving the state file in bucket. Default to "env:".
 * `key` - (Optional) The path for saving the state file in bucket. Defaults to `terraform.tfstate`.
 * `encrypt` - (Optional) Whether to enable server side encryption of the state file. If it is true, COS will use 'AES256' encryption algorithm to encrypt state file.
 * `acl` - (Optional) Object ACL to be applied to the state file, allows `private` and `public-read`. Defaults to `private`.
