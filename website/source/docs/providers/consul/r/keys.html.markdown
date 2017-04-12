---
layout: "consul"
page_title: "Consul: consul_keys"
sidebar_current: "docs-consul-resource-keys"
description: |-
  Writes values into the Consul key/value store.
---

# consul_keys

The `consul_keys` resource writes sets of individual values into Consul.
This is a powerful way to expose infrastructure details to clients.

This resource manages individual keys, and thus it can create, update
and delete the keys explicitly given. However, it is not able to detect
and remove additional keys that have been added by non-Terraform means.
To manage *all* keys sharing a common prefix, and thus have Terraform
remove errant keys not present in the configuration, consider using the
`consul_key_prefix` resource instead.

## Example Usage

```hcl
resource "consul_keys" "app" {
  datacenter = "nyc1"
  token      = "abcd"

  # Set the CNAME of our load balancer as a key
  key {
    path  = "service/app/elb_address"
    value = "${aws_elb.app.dns_name}"
  }
}
```

## Argument Reference

The following arguments are supported:

* `datacenter` - (Optional) The datacenter to use. This overrides the
  datacenter in the provider setup and the agent's default datacenter.

* `token` - (Optional) The ACL token to use. This overrides the
  token that the agent provides by default.

* `key` - (Required) Specifies a key in Consul to be written.
  Supported values documented below.

The `key` block supports the following:

* `path` - (Required) This is the path in Consul that should be written to.

* `value` - (Required) The value to write to the given path.

* `delete` - (Optional) If true, then the key will be deleted when
  either its configuration block is removed from the configuration or
  the entire resource is destroyed. Otherwise, it will be left in Consul.
  Defaults to false.

### Deprecated `key` arguments

Prior to Terraform 0.7, this resource was used both to read *and* write the
Consul key/value store. The read functionality has moved to the `consul_keys`
*data source*, whose documentation can be found via the navigation.

The pre-0.7 interface for reading keys is still supported for backward compatibility,
but will be removed in a future version of Terraform.

## Attributes Reference

The following attributes are exported:

* `datacenter` - The datacenter the keys are being written to.
