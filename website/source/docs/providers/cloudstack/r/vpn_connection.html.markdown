---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_vpn_connection"
sidebar_current: "docs-cloudstack-resource-vpn-connection"
description: |-
  Creates a site to site VPN connection.
---

# cloudstack\_vpn\_connection

Creates a site to site VPN connection.

## Example Usage

Basic usage:

```
resource "cloudstack_vpn_connection" "default" {
    customergatewayid = "xxx"
    vpngatewayid = "xxx"
}
```

## Argument Reference

The following arguments are supported:

* `customergatewayid` - (Required) The Customer Gateway ID to connect.
    Changing this forces a new resource to be created.

* `vpngatewayid` - (Required) The VPN Gateway ID to connect.
    Changing this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the VPN Connection.
