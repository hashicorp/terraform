---
layout: "consul"
page_title: "Consul: consul_keys"
sidebar_current: "docs-consul-data-source-keys"
description: |-
  Reads values from the Consul key/value store.
---

# consul_keys

The `consul_keys` resource reads values from the Consul key/value store.
This is a powerful way dynamically set values in templates.

## Example Usage

```hcl
data "consul_keys" "app" {
  datacenter = "nyc1"
  token      = "abcd"

  # Read the launch AMI from Consul
  key {
    name    = "ami"
    path    = "service/app/launch_ami"
    default = "ami-1234"
  }
}

# Start our instance with the dynamic ami value
resource "aws_instance" "app" {
  ami = "${data.consul_keys.app.var.ami}"

  # ...
}
```

## Argument Reference

The following arguments are supported:

* `datacenter` - (Optional) The datacenter to use. This overrides the
  datacenter in the provider setup and the agent's default datacenter.

* `token` - (Optional) The ACL token to use. This overrides the
  token that the agent provides by default.

* `key` - (Required) Specifies a key in Consul to be read or written.
  Supported values documented below.

The `key` block supports the following:

* `name` - (Required) This is the name of the key. This value of the
  key is exposed as `var.<name>`. This is not the path of the key
  in Consul.

* `path` - (Required) This is the path in Consul that should be read
  or written to.

* `default` - (Optional) This is the default value to set for `var.<name>`
  if the key does not exist in Consul. Defaults to an empty string.

## Attributes Reference

The following attributes are exported:

* `datacenter` - The datacenter the keys are being read from to.
* `var.<name>` - For each name given, the corresponding attribute
  has the value of the key.
