---
layout: "consul"
page_title: "Provider: Consul"
sidebar_current: "docs-consul-index"
description: |-
  Consul is a tool for service discovery, configuration and orchestration. The Consul provider exposes resources used to interact with a Consul cluster. Configuration of the provider is optional, as it provides defaults for all arguments.
---

# Consul Provider

[Consul](https://www.consul.io) is a tool for service discovery, configuration
and orchestration. The Consul provider exposes resources used to interact with a
Consul cluster. Configuration of the provider is optional, as it provides
defaults for all arguments.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the Consul provider
provider "consul" {
  address    = "demo.consul.io:80"
  datacenter = "nyc1"
}

# Access a key in Consul
resource "consul_keys" "app" {
  key {
    name    = "ami"
    path    = "service/app/launch_ami"
    default = "ami-1234"
  }
}

# Use our variable from Consul
resource "aws_instance" "app" {
  ami = "${consul_keys.app.var.ami}"
}
```

## Argument Reference

The following arguments are supported:

* `address` - (Optional) The HTTP(S) API address of the agent to use. Defaults to "127.0.0.1:8500".
* `scheme` - (Optional) The URL scheme of the agent to use ("http" or "https"). Defaults to "http".
* `http_auth` - (Optional) HTTP Basic Authentication credentials to be used when communicating with Consul, in the format of either `user` or `user:pass`. This may also be specified using the `CONSUL_HTTP_AUTH` environment variable.
* `datacenter` - (Optional) The datacenter to use. Defaults to that of the agent.
* `token` - (Optional) The ACL token to use by default when making requests to the agent.
* `ca_file` - (Optional) A path to a PEM-encoded certificate authority used to verify the remote agent's certificate.
* `cert_file` - (Optional) A path to a PEM-encoded certificate provided to the remote agent; requires use of `key_file`.
* `key_file`- (Optional) A path to a PEM-encoded private key, required if `cert_file` is specified.
