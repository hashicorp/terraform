---
layout: "consul"
page_title: "Provider: Consul"
sidebar_current: "docs-consul-index"
---

# Consul Provider

The Consul provider exposes resources used to interact with
the Consul catalog. The provider optionally must can be configured with
to change default behavior.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the Consul provider
provider "consul" {
    address = "demo.consul.io:80"
    datacenter = "nyc1"
}

# Access a key in Consul
resource "consul_keys" "app" {
    key {
        name = "ami"
        path = "service/app/launch_ami"
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

* `address` - (Optional) The HTP API address of the agent to use. Defaults to "127.0.0.1:8500".
* `datacenter` - (Optional) The datacenter to use. Defaults to that of the agent.

