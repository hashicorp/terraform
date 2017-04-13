---
layout: "backend-types"
page_title: "Backend Type: gcs"
sidebar_current: "docs-backends-types-standard-gcs"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# gcs

**Kind: Standard (with no locking)**

Stores the state as a given key in a given bucket on [Google Cloud Storage](https://cloud.google.com/storage/).

## Example Configuration

```hcl
terraform {
  backend "gcs" {
    bucket  = "tf-state-prod"
    path    = "path/terraform.tfstate"
    project = "myproject"
  }
}
```

## Example Referencing

```hcl
data "terraform_remote_state" "foo" {
  backend = "gcs"
  config {
    bucket  = "terraform-state-prod"
    path    = "network/terraform.tfstate"
    project = "goopro"
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

 * `bucket` - (Required) The name of the GCS bucket
 * `path` - (Required) The path where to place/look for state file inside the bucket
 * `credentials` / `GOOGLE_CREDENTIALS` - (Required) Google Cloud Platform account credentials in json format
