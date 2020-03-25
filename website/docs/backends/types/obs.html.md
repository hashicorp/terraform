---
layout: "backend-types"
page_title: "Backend Type: obs"
sidebar_current: "docs-backends-types-standard-obs"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# OBS

**Kind: Standard (with locking)**

Stores the state as a given key in a given bucket on
[HuaweiCloud OBS](https://www.huaweicloud.com/intl/en-us/product/obs.html).
This backend also supports [state locking](/docs/state/locking.html).

~> **Warning!** It is highly recommended that you enable
[Bucket Versioning](https://support.huaweicloud.com/intl/en-us/usermanual-obs/en-us_topic_0045853504.html)
on the OBS bucket to allow for state recovery in the case of accidental deletions and human error.

## Example Configuration

```hcl
terraform {
  backend "obs" {
    bucket = "testbucket"
    prefix = "terraform/state"
    key    = "terraform.tfstate"
    region = "cn-north-1"
  }
}
```

This assumes we have a bucket created called `testbucket`. The
Terraform state will be written to the key `terraform/state/terraform.tfstate`.

Note that for the access credentials we recommend using
`OS_ACCESS_KEY` and `OS_SECRET_KEY`.

## Using the OBS remote state

To make use of the OBS remote state we can use the
[`terraform_remote_state` data
source](/docs/providers/terraform/d/remote_state.html).

```hcl
data "terraform_remote_state" "foo" {
  backend = "obs"

  config = {
    bucket = "tf-backend"
    prefix = "terraform/state"
    key    = "terraform.tfstate"
    region = "cn-north-1"
  }
}
```

The `terraform_remote_state` data source will return all of the root module
outputs defined in the referenced remote state (but not any outputs from
nested modules unless they are explicitly output again in the root).

## Configuration variables

The following configuration options or environment variables are supported:

 * `access_key_id` - (Optional) HuaweiCloud access key. It supports environment variable `OS_ACCESS_KEY`.
 * `secret_key_id` - (Optional) HuaweiCloud secret access key. It supports environment variable `OS_SECRET_KEY`.
 * `region` - (Required) The region of the OBS bucket. It supports environment variable `OS_REGION_NAME`.
 * `bucket` - (Required) The name of the OBS bucket. You shall manually create it first.
 * `prefix` - (Optional) The directory for saving the state file in bucket.
 * `key` - (Optional) The path to the state file inside the bucket. Defaults to `terraform.tfstate`.
 * `encrypt` - (Optional) Whether to enable [server side
   encryption](https://support.huaweicloud.com/intl/en-us/usermanual-obs/en-us_topic_0066036553.html)
   of the state file.
 * `kms_key_id` - (Optional) The KMS Key to use for encrypting the state.
 * `acl` - Object ACL to be applied to the state file. Defaults to `private`.
 * `endpoint` - (Optional) A custom endpoint for the OBS API.
