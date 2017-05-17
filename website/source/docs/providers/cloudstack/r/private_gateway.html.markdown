---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_private_gateway"
sidebar_current: "docs-cloudstack-resource-private-gateway"
description: |-
  Creates a private gateway.
---

# cloudstack_private_gateway

Creates a private gateway for the given VPC.

*NOTE: private gateway can only be created using a ROOT account!*

## Example Usage

```hcl
resource "cloudstack_private_gateway" "default" {
  gateway    = "10.0.0.1"
  ip_address = "10.0.0.2"
  netmask    = "255.255.255.252"
  vlan       = "200"
  vpc_id     = "76f6e8dc-07e3-4971-b2a2-8831b0cc4cb4"
}
```

## Argument Reference

The following arguments are supported:

* `gateway` - (Required) the gateway of the Private gateway. Changing this
    forces a new resource to be created.

* `ip_address` - (Required) the IP address of the Private gateway. Changing this forces
    a new resource to be created.

* `netmask` - (Required) The netmask of the Private gateway. Changing
    this forces a new resource to be created.

* `vlan` - (Required) The VLAN number (1-4095) the network will use.

* `physical_network_id` - (Optional) The ID of the physical network this private
    gateway belongs to.

* `network_offering` - (Optional) The name or ID of the network offering to use for
    the private gateways network connection.

* `acl_id` - (Required) The ACL ID that should be attached to the network.

* `vpc_id` - (Required) The VPC ID in which to create this Private gateway. Changing
    this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the private gateway.
