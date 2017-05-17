---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_vpn_gateway"
sidebar_current: "docs-cloudstack-resource-vpn-gateway"
description: |-
  Creates a site to site VPN local gateway.
---

# cloudstack_vpn_gateway

Creates a site to site VPN local gateway.

## Example Usage

Basic usage:

```hcl
resource "cloudstack_vpn_gateway" "default" {
  vpc_id = "f8141e2f-4e7e-4c63-9362-986c908b7ea7"
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC for which to create the VPN Gateway.
    Changing this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the VPN Gateway.
* `public_ip` - The public IP address associated with the VPN Gateway.
