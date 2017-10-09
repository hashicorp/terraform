---
layout: "backend-types"
page_title: "Backend Type: manta"
sidebar_current: "docs-backends-types-standard-manta"
description: |-
  Terraform can store state in manta.
---

# manta

**Kind: Standard (with locking within Manta)**

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

 * `account` - (Required) This is the name of the Manta account. It can also be provided via the `SDC_ACCOUNT` or `TRITON_ACCOUNT` environment variables.
 * `url` - (Optional) The Manta API Endpoint. It can also be provided via the `MANTA_URL` environment variable. Defaults to `https://us-east.manta.joyent.com`.
 * `key_material` - (Optional) This is the private key of an SSH key associated with the Triton account to be used. If this is not set, the private key corresponding to the fingerprint in key_id must be available via an SSH Agent. Can be set via the `SDC_KEY_MATERIAL` or `TRITON_KEY_MATERIAL` environment variables.
 * `key_id` - (Required) This is the fingerprint of the public key matching the key specified in key_path. It can be obtained via the command ssh-keygen -l -E md5 -f /path/to/key. Can be set via the `SDC_KEY_ID` or `TRITON_KEY_ID` environment variables.
 * `insecure_skip_tls_verify` - (Optional) This allows skipping TLS verification of the Triton endpoint. It is useful when connecting to a temporary Triton installation such as Cloud-On-A-Laptop which does not generally use a certificate signed by a trusted root CA. Defaults to `false`.
 * `path` - (Required) The path relative to your private storage directory (`/$MANTA_USER/stor`) where the state file will be stored. **Please Note:** If this path does not exist, then the backend will create this folder location as part of backend creation.
 * `objectName` - (Optional) The name of the state file (defaults to `terraform.tfstate`)

The following [Manta environment variables](https://apidocs.joyent.com/manta/#setting-up-your-environment) are supported:

 * `MANTA_URL` - (Required) The API endpoint
 * `MANTA_USER` - (Required) The Manta user
 * `MANTA_KEY_ID` - (Required) The MD5 fingerprint of your SSH key
 * `MANTA_KEY_MATERIAL` - (Required) The path to the private key for accessing Manta (must align with the `MANTA_KEY_ID`). This key must *not* be protected by passphrase.
