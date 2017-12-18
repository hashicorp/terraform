---
layout: "backend-types"
page_title: "Backend Type: gcs"
sidebar_current: "docs-backends-types-standard-gcs"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# gcs

**Kind: Standard (with locking)**

Stores the state as an object in a configurable prefix and bucket on [Google Cloud Storage](https://cloud.google.com/storage/) (GCS).

## Example Configuration

```hcl
terraform {
  backend "gcs" {
    bucket  = "tf-state-prod"
    prefix  = "terraform/state"
  }
}
```

## Example Referencing

```hcl
data "terraform_remote_state" "foo" {
  backend = "gcs"
  config {
    bucket  = "terraform-state"
    prefix  = "prod"
  }
}

resource "template_file" "bar" {
  template = "${greeting}"

  vars {
    greeting = "${data.terraform_remote_state.foo.greeting}"
  }
}
```

## Configuration variables

The following configuration options are supported:

 *  `bucket` - (Required) The name of the GCS bucket.
    This name must be globally unique.
    For more information, see [Bucket Naming Guidelines](https://cloud.google.com/storage/docs/bucketnaming.html#requirements).
 *  `credentials` / `GOOGLE_CREDENTIALS` - (Optional) Local path to Google Cloud Platform account credentials in JSON format.
    If unset, [Google Application Default Credentials](https://developers.google.com/identity/protocols/application-default-credentials) are used.
    The provided credentials need to have the `devstorage.read_write` scope and `WRITER` permissions on the bucket.
 *  `prefix` - (Optional) GCS prefix inside the bucket. Named states are stored in an object called `<prefix>/<name>.tfstate`.
 *  `path` - (Deprecated) GCS path to the state file of the default state. For backwards compatibility only, use `prefix` instead.
 *  `project` / `GOOGLE_PROJECT` - (Optional) The project ID to which the bucket belongs. This is only used when creating a new bucket during initialization.
    Since buckets have globally unique names, the project ID is not required to access the bucket during normal operation.
 *  `region` / `GOOGLE_REGION` - (Optional) The region in which a new bucket is created.
    For more information, see [Bucket Locations](https://cloud.google.com/storage/docs/bucket-locations).
 *  `encryption_key` / `GOOGLE_ENCRYPTION_KEY` - (Optional) A 32 byte base64 encoded 'customer supplied encryption key' used to encrypt all state. For more information see [Customer Supplied Encryption Keys](https://cloud.google.com/storage/docs/encryption#customer-supplied).
