---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_private_gateway"
sidebar_current: "docs-cloudstack-resource-private-gateway"
description: |-
  Creates a private gateway.
---

# cloudstack\_private\_gateway

Creates a private gateway for the given VPC.

## Example Usage

```
resource "cloudstack_private_gateway" "default" {
    gateway = 10.0.0.1
    ip_address = "10.0.0.2"
    netmask = "255.255.255.252"
    vlan = "200"
	vpc_id = "76f6e8dc-07e3-4971-b2a2-8831b0cc4cb4"
}
```

## Argument Reference

The following arguments are supported:

* `gateway` - (Required) The nexthop for the static routes. Changing this
    forces a new resource to be created.

* `ip_address` - (Required) The ip_address on the VPC. Changing this forces
    a new resource to be created.

* `netmask` - (Required) The netmask of the private gateway on the VPC. Changing 
    this forces a new resource to be created.

* `vlan` - (Required) The VLAN number (1-4095) the network will use. This might be
    required by the Network Offering if specifyVlan=true is set. Only the ROOT 
    admin can set this value.

* `physical_network_id` - (Optional) The ID of the physical network this private
    gateway belongs to.

* `network_offering` - (Optional) The name or ID of the network offering to use
    for this private gateway.

* `vpc_id` - (Optional) The VPC ID in which to create this network. Changing
    this forces a new resource to be created.

* `acl_id` - (Optional) The ACL ID that should be attached to the network or
    `none` if you do not want to attach an ACL. You can dynamically attach and
    swap ACL's, but if you want to detach an attached ACL and revert to using
    `none`, this will force a new resource to be created. (defaults `none`)

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the private gateway.

