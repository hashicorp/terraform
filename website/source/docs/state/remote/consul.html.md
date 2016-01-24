---
layout: "remotestate"
page_title: "Remote State Backend: consul"
sidebar_current: "docs-state-remote-consul"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# consul

Stores the state in the [Consul](https://www.consul.io/) KV store at a given path.

-> **Note:** Specifying `access_token` directly makes it included in
cleartext inside the persisted, shard state.
Use of the environment variable `CONSUL_HTTP_TOKEN` is recommended.

## Example Usage

```
terraform remote config \
	-backend=consul \
	-backend-config="path=full/path"
```

## Example Referencing

```
resource "terraform_remote_state" "foo" {
	backend = "consul"
	config {
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
 * `http_auth` / `CONSUL_HTTP_AUTH` - (Optional) HTTP Basic Authentication credentials to be used when
   communicating with Consul, in the format of either `user` or `user:pass`.
