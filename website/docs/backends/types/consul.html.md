---
layout: "backend-types"
page_title: "Backend Type: consul"
sidebar_current: "docs-backends-types-standard-consul"
description: |-
  Terraform can store state in Consul.
---

# consul

**Kind: Standard (with locking)**

Stores the state in the [Consul](https://www.consul.io/) KV store at a given path.

This backend supports [state locking](/docs/state/locking.html).

## Example Configuration

```hcl
terraform {
  backend "consul" {
    address = "demo.consul.io"
    scheme  = "https"
    path    = "full/path"
  }
}
```

Note that for the access credentials we recommend using a
[partial configuration](/docs/backends/config.html).

## Example Referencing

```hcl
data "terraform_remote_state" "foo" {
  backend = "consul"
  config = {
    path = "full/path"
  }
}
```

## Configuration variables

The following configuration options / environment variables are supported:

 * `path` - (Required) Path in the Consul KV store
 * `access_token` / `CONSUL_HTTP_TOKEN` - (Required) Access token
 * `address` / `CONSUL_HTTP_ADDR` - (Optional) DNS name and port of your Consul endpoint specified in the
   format `dnsname:port`. Defaults to the local agent HTTP listener.
 * `scheme` - (Optional) Specifies what protocol to use when talking to the given
   `address`, either `http` or `https`. SSL support can also be triggered
   by setting then environment variable `CONSUL_HTTP_SSL` to `true`.
 * `datacenter` - (Optional) The datacenter to use. Defaults to that of the agent.
 * `http_auth` / `CONSUL_HTTP_AUTH` - (Optional) HTTP Basic Authentication credentials to be used when
   communicating with Consul, in the format of either `user` or `user:pass`.
 * `gzip` - (Optional) `true` to compress the state data using gzip, or `false` (the default) to leave it uncompressed.
 * `lock` - (Optional) `false` to disable locking. This defaults to true, but will require session permissions with Consul and at least kv write permissions on `$path/.lock` to perform locking. 
 * `ca_file` / `CONSUL_CAFILE` - (Optional) A path to a PEM-encoded certificate authority used to verify the remote agent's certificate.
 * `cert_file` / `CONSUL_CLIENT_CERT` - (Optional) A path to a PEM-encoded certificate provided to the remote agent; requires use of `key_file`.
 * `key_file` / `CONSUL_CLIENT_KEY` - (Optional) A path to a PEM-encoded private key, required if `cert_file` is specified.
