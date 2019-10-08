---
layout: "backend-types"
page_title: "Backend Type: gcs"
sidebar_current: "docs-backends-types-standard-gcs"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# gcs

**Kind: Standard (with locking)**

Stores the state as an object in a configurable prefix in a given bucket on [Google Cloud Storage](https://cloud.google.com/storage/) (GCS).
This backend also supports [state locking](/docs/state/locking.html).

~> **Warning!** It is highly recommended that you enable
[Object Versioning](https://cloud.google.com/storage/docs/object-versioning)
on the GCS bucket to allow for state recovery in the case of accidental deletions and human error.

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
  config = {
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
 * `access_token` - (Optional) A temporary [OAuth 2.0 access token] obtained from
    the Google Authorization server, i.e. the `Authorization: Bearer` token used to
    authenticate HTTP requests to GCP APIs. This is an alternative to `credentials`. If both are specified, `access_token` will be used over the `credentials` field.
 *  `prefix` - (Optional) GCS prefix inside the bucket. Named states for workspaces are stored in an object called `<prefix>/<name>.tfstate`.
 *  `path` - (Deprecated) GCS path to the state file of the default state. For backwards compatibility only, use `prefix` instead.
 *  `encryption_key` / `GOOGLE_ENCRYPTION_KEY` - (Optional) A 32 byte base64 encoded 'customer supplied encryption key' used to encrypt all state. For more information see [Customer Supplied Encryption Keys](https://cloud.google.com/storage/docs/encryption#customer-supplied).
