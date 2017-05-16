---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_vpn_connection"
sidebar_current: "docs-cloudstack-resource-vpn-connection"
description: |-
  Creates a site to site VPN connection.
---

# cloudstack_vpn_connection

Creates a site to site VPN connection.

## Example Usage

Basic usage:

```hcl
resource "cloudstack_vpn_connection" "default" {
  customer_gateway_id = "8dab9381-ae73-48b8-9a3d-c460933ef5f7"
  vpn_gateway_id      = "a7900060-f8a8-44eb-be15-ea54cf499703"
}
```

## Argument Reference

The following arguments are supported:

* `customer_gateway_id` - (Required) The Customer Gateway ID to connect.
    Changing this forces a new resource to be created.

* `vpn_gateway_id` - (Required) The VPN Gateway ID to connect. Changing
    this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the VPN Connection.
