---
layout: "backend-types"
page_title: "Backend Type: etcdv3"
sidebar_current: "docs-backends-types-standard-etcdv3"
description: |-
  Terraform can store state remotely in etcd 3.x.
---

# etcdv3

**Kind: Standard (with locking)**

Stores the state in the [etcd](https://coreos.com/etcd/) KV store wit a given prefix.

This backend supports [state locking](/docs/state/locking.html).

## Example Configuration

```hcl
terraform {
  backend "etcdv3" {
    endpoints = ["etcd-1:2379", "etcd-2:2379", "etcd-3:2379"]
    lock      = true
    prefix    = "terraform-state/"
  }
}
```

Note that for the access credentials we recommend using a
[partial configuration](/docs/backends/config.html).

## Example Referencing

```hcl
data "terraform_remote_state" "foo" {
  backend = "etcdv3"
  config {
    endpoints = ["etcd-1:2379", "etcd-2:2379", "etcd-3:2379"]
    lock      = true
    prefix    = "terraform-state/"
  }
}
```

## Configuration variables

The following configuration options / environment variables are supported:

 * `endpoints` - (Required) The list of 'etcd' endpoints which to connect to.
 * `username` / `ETCDV3_USERNAME` - (Optional) Username used to connect to the etcd cluster.
 * `password` / `ETCDV3_PASSWORD` - (Optional) Password used to connect to the etcd  cluster.
 * `prefix` - (Optional) An optional prefix to be added to keys when to storing state in etcd. Defaults to `""`.
 * `lock` - (Optional) Whether to lock state access. Defaults to `true`.
