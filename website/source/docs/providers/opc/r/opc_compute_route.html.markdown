---
layout: "opc"
page_title: "Oracle: opc_compute_route"
sidebar_current: "docs-opc-resource-route"
description: |-
  Creates and manages a Route resource for an IP Network
---

# opc\_compute\_route

The ``opc_compute_route`` resource creates and manages a route for an IP Network.

## Example Usage

```hcl
resource "opc_compute_route" "foo" {
  name              = "my-route"
  description       = "my IP Network route"
  admin_distance    = 1
  ip_address_prefix = "10.0.1.0/24"
  next_hop_vnic_set = "${opc_compute_vnic_set.bar.name}"
  tags              = ["tag1", "tag2"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the route.

* `description` - (Optional) The description of the route.

* `admin_distance` - (Optional) The route's administrative distance. Defaults to `0`.

* `ip_address_prefix` - (Required) The IPv4 address prefix, in CIDR format, of the external network from which to route traffic.

* `next_hop_vnic_set` - (Required) Name of the virtual NIC set to route matching packets to. Routed flows are load-balanced among all the virtual NICs in the virtual NIC set.

## Attributes Reference

The following attributes are exported:

* `name` The name of the route

* `description` - The description of the route.

* `admin_distance` - The route's administrative distance. Defaults to `0`.

* `ip_address_prefix` - The IPv4 address prefix, in CIDR format, of the external network from which to route traffic.

* `next_hop_vnic_set` - Name of the virtual NIC set to route matching packets to. Routed flows are load-balanced among all the virtual NICs in the virtual NIC set.

## Import

Route's can be imported using the `resource name`, e.g.

```shell
$ terraform import opc_compute_route.route1 example
```
