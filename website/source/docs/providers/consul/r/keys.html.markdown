---
layout: "consul"
page_title: "Consul: consul_keys"
sidebar_current: "docs-consul-resource-keys"
description: |-
  Provides access to Key/Value data in Consul. This can be used to both read keys from Consul, but also to set the value of keys in Consul. This is a powerful way dynamically set values in templates, and to expose infrastructure details to clients.
---

# consul\_keys

Provides access to Key/Value data in Consul. This can be used
to both read keys from Consul, but also to set the value of keys
in Consul. This is a powerful way dynamically set values in templates,
and to expose infrastructure details to clients.

This resource manages individual keys, and thus it can create, update and
delete the keys explicitly given. Howver, It is not able to detect and remove
additional keys that have been added by non-Terraform means. To manage
*all* keys sharing a common prefix, and thus have Terraform remove errant keys
not present in the configuration, consider using the `consul_key_prefix`
resource instead.

## Example Usage

```
resource "consul_keys" "app" {
    datacenter = "nyc1"
    token = "abcd"

    # Read the launch AMI from Consul
    key {
        name = "ami"
        path = "service/app/launch_ami"
        default = "ami-1234"
    }

    # Set the CNAME of our load balancer as a key
    key {
        name = "elb_cname"
        path = "service/app/elb_address"
        value = "${aws_elb.app.dns_name}"
    }
}

# Start our instance with the dynamic ami value
resource "aws_instance" "app" {
    ami = "${consul_keys.app.var.ami}"
    ...
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
  if the key does not exist in Consul. Defaults to the empty string.

* `value` - (Optional) If set, the key will be set to this value.
  This allows a key to be written to.

* `delete` - (Optional) If true, then the key will be deleted when
  either its configuration block is removed from the configuration or
  the entire resource is destroyed. Otherwise, it will be left in Consul.
  Defaults to false.

## Attributes Reference

The following attributes are exported:

* `datacenter` - The datacenter the keys are being read/written to.
* `var.<name>` - For each name given, the corresponding attribute
  has the value of the key.
