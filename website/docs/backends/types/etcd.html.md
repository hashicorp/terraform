---
layout: "backend-types"
page_title: "Backend Type: etcd"
sidebar_current: "docs-backends-types-standard-etcd"
description: |-
  Terraform can store state remotely in etcd.
---

# etcd

**Kind: Standard (with no locking)**

Stores the state in [etcd](https://coreos.com/etcd/) at a given path.

## Example Configuration

```hcl
terraform {
  backend "etcd" {
    path      = "path/to/terraform.tfstate"
    endpoints = "http://one:4001 http://two:4001"
  }
}
```

## Example Referencing

```hcl
data "terraform_remote_state" "foo" {
  backend = "etcd"
  config {
    path      = "path/to/terraform.tfstate"
    endpoints = "http://one:4001 http://two:4001"
  }
}
```

## Configuration variables

The following configuration options are supported:

 * `path` - (Required) The path where to store the state
 * `endpoints` - (Required) A space-separated list of the etcd endpoints
 * `username` - (Optional) The username
 * `password` - (Optional) The password
