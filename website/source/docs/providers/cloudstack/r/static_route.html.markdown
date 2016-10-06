---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_static_route"
sidebar_current: "docs-cloudstack-resource-static-route"
description: |-
  Creates a static route.
---

# cloudstack\_static\_route

Creates a static route for the given private gateway or VPC.

## Example Usage

```
resource "cloudstack_static_route" "default" {
    cidr = "10.0.0.0/16"
    gateway_id = "76f607e3-e8dc-4971-8831-b2a2b0cc4cb4"
}
```

## Argument Reference

The following arguments are supported:

* `cidr` - (Required) The CIDR block for the static route. Changing this forces 
    a new resource to be created.

* `gateway_id` - (Optional) The ip_address on the VPC. Changing this forces
    a new resource to be created.

* `vpc_id` - (Optional) The VPC ID in which to create this network. Changing
    this forces a new resource to be created.

* `nexthop` - (Optional) The nexthop for the CIDR.

NOTE: Either private_gateway_id or vpc_id should have a value! Nexthop is required 
      in combination with vpc_id!
## Attributes Reference

The following attributes are exported:

* `id` - The ID of the static route.

