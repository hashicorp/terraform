---
layout: "backend-types"
page_title: "Backend Type: tikv"
sidebar_current: "docs-backends-types-standard-tikv"
description: |-
  Terraform can store state in TiKV.
---

# tikv

**Kind: Standard (with locking)**

Stores the state in the [TiKV](https://tikv.org/) KV store at a given prefix.

This backend supports [state locking](/docs/state/locking.html).

## Example Configuration

```hcl
terraform {
  backend "tikv" {
    pd_address = ["127.0.0.1:2379", "127.0.0.1:2380", "127.0.0.1:2381"]
    prefix     = "demo
  }
}
```

Note that for the access credentials we recommend using a [partial configuration](/docs/backends/config.html).

## Example Referencing

```hcl
data "terraform_remote_state" "foo" {
  backend = "tikv"
  config = {
    path = "full/path"
  }
}
```

## Configuration variables

The following configuration options / environment variables are supported:

 * `prefix` - (Required) The prefix of key in the TiKV store
 * `pd_address` - (Required) The addresses of tikv pd node, used for discovery tikv node, format `dnsname:port`.
 * `lock` - (Optional) `false` to disable locking. This defaults to true.
 * `ca_file` - (Optional) A path to a PEM-encoded certificate authority used to verify the remote agent's certificate.
 * `cert_file` - (Optional) A path to a PEM-encoded certificate provided to the remote agent; requires use of `key_file`.
 * `key_file` - (Optional) A path to a PEM-encoded private key, required if `cert_file` is specified.
 