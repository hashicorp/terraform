---
layout: "language"
page_title: "Backend Type: etcdv3"
sidebar_current: "docs-backends-types-standard-etcdv3"
description: |-
  Terraform can store state remotely in etcd 3.x.
---

# etcdv3

Stores the state in the [etcd](https://etcd.io/) KV store with a given prefix.

This backend supports [state locking](/docs/language/state/locking.html).

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
[partial configuration](/docs/language/settings/backends/configuration.html#partial-configuration).

## Data Source Configuration

```hcl
data "terraform_remote_state" "foo" {
  backend = "etcdv3"
  config = {
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
 * `cacert_path` - (Optional) The path to a PEM-encoded CA bundle with which to verify certificates of TLS-enabled etcd servers.
 * `cert_path` - (Optional) The path to a PEM-encoded certificate to provide to etcd for secure client identification.
 * `key_path` - (Optional) The path to a PEM-encoded key to provide to etcd for secure client identification.
 * `max_request_bytes` - (Optional) The max request size to send to etcd. This can be increased to enable storage of larger state. You must set the corresponding server-side flag [--max-request-bytes](https://etcd.io/docs/current/dev-guide/limit/#request-size-limit) as well and the value should be less than the client setting. Defaults to `2097152` (2.0 MiB). **Please Note:** Increasing etcd's request size limit may negatively impact overall latency.
