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
    project = "myproject"
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
    project = "goopro"
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

 * `bucket` - (Required) The name of the GCS bucket.
 * `credentials` / `GOOGLE_CREDENTIALS` - (Required) Local path to Google Cloud Platform account credentials in JSON format.
 * `prefix` - (Optional) GCS prefix inside the bucket. Named states are stored in an object called `<prefix>/<name>.tfstate`.
 * `path` - (Legacy) GCS path to the state file of the default state. For backwards compatibility only, use `prefix` instead.
