---
layout: "backend-types"
page_title: "Backend Type: manta"
sidebar_current: "docs-backends-types-standard-manta"
description: |-
  Terraform can store state in manta.
---

# manta

**Kind: Standard (with no locking)**

Stores the state as an artifact in [Manta](https://www.joyent.com/manta).

## Example Configuration

```hcl
terraform {
  backend "manta" {
    path       = "random/path"
    objectName = "terraform.tfstate"
  }
}
```

Note that for the access credentials we recommend using a
[partial configuration](/docs/backends/config.html).

## Example Referencing

```hcl
data "terraform_remote_state" "foo" {
  backend = "manta"
  config {
    path       = "random/path"
    objectName = "terraform.tfstate"
  }
}
```

## Configuration variables

The following configuration options are supported:

 * `path` - (Required) The path relative to your private storage directory (`/$MANTA_USER/stor`) where the state file will be stored
 * `objectName` - (Optional) The name of the state file (defaults to `terraform.tfstate`)

The following [Manta environment variables](https://apidocs.joyent.com/manta/#setting-up-your-environment) are supported:

 * `MANTA_URL` - (Required) The API endpoint
 * `MANTA_USER` - (Required) The Manta user
 * `MANTA_KEY_ID` - (Required) The MD5 fingerprint of your SSH key
 * `MANTA_KEY_MATERIAL` - (Required) The path to the private key for accessing Manta (must align with the `MANTA_KEY_ID`). This key must *not* be protected by passphrase.
