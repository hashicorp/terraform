---
layout: "consul"
page_title: "Consul: consul_key_prefix"
sidebar_current: "docs-consul-data-source-key-prefix"
description: |-
  Reads values from the Consul key/value store.
---

# consul\_key\_prefix

The `consul_key_prefix` resource reads values from the Consul key/value store.
This is a powerful way dynamically set values in templates.

## Example Usage

```
data "consul_key_prefix" "app" {
    datacenter = "nyc1"
    token = "abcd"

    # Read the AMI from Consul
    path_prefix = "service/app/"
}

# Start our instance with the dynamic ami value
resource "aws_instance" "app" {
    ami = "${data.consul_key_prefix.app.var.ami}"
    ...
}
```

## Argument Reference

The following arguments are supported:

* `datacenter` - (Optional) The datacenter to use. This overrides the
  datacenter in the provider setup and the agent's default datacenter.

* `token` - (Optional) The ACL token to use. This overrides the
  token that the agent provides by default.

* `path_prefix` - (Required) Specifies the common prefix shared by all keys
  to be retrieved in Consul.

## Attributes Reference

The following attributes are exported:

* `datacenter` - The datacenter the keys are being read from to.
* `var.<name>` - For each name given, the corresponding attribute
  has the value of the key.
