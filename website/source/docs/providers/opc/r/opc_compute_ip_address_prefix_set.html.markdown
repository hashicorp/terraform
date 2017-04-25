---
layout: "opc"
page_title: "Oracle: opc_compute_ip_address_prefix_set"
sidebar_current: "docs-opc-resource-ip-address-prefix-set"
description: |-
  Creates and manages an IP address prefix set in an OPC identity domain.
---

# opc\_compute\_ip\_address\_prefix\_set

The ``opc_compute_ip_address_prefix_set`` resource creates and manages an IP address prefix set in an OPC identity domain.

## Example Usage

```hcl
resource "opc_compute_ip_address_prefix_set" "default" {
  name     = "PrefixSet1"
  prefixes = ["192.168.0.0/16", "172.120.0.0/24"]
  tags     = ["tags1", "tags2"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the ip address prefix set.

* `prefixes` - (Optional) List of CIDR IPv4 prefixes assigned in the virtual network.

* `description` - (Optional) A description of the ip address prefix set.

* `tags` - (Optional) List of tags that may be applied to the ip address prefix set.

In addition to the above, the following variables are exported:

* `uri` - (Computed) The Uniform Resource Identifier of the ip address prefix set.

## Import

IP Address Prefix Set can be imported using the `resource name`, e.g.

```shell
$ terraform import opc_compute_ip_address_prefix_set.default example
```
