---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_vpn_gateway"
sidebar_current: "docs-cloudstack-resource-vpn-gateway"
description: |-
  Creates a site to site VPN local gateway.
---

# cloudstack\_vpn\_gateway

Creates a site to site VPN local gateway.

## Example Usage

Basic usage:

```
resource "cloudstack_vpn_gateway" "default" {
    vpc = "test-vpc"
}
```

## Argument Reference

The following arguments are supported:

* `vpc` - (Required) The name or ID of the VPC for which to create the VPN Gateway.
    Changing this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the VPN Gateway.
* `public_ip` - The public IP address associated with the VPN Gateway.
