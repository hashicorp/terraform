---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_vpn_customer_gateway"
sidebar_current: "docs-cloudstack-resource-vpn-customer-gateway"
description: |-
  Creates a site to site VPN local customer gateway.
---

# cloudstack\_vpn\_customer\_gateway

Creates a site to site VPN local customer gateway.

## Example Usage

Basic usage:

```
resource "cloudstack_vpn_customer_gateway" "default" {
    name = "test-vpc"
    cidr = "10.0.0.0/8"
    esp_policy = "aes256-sha1"
    gateway = "192.168.0.1"
    ike_policy = "aes256-sha1"
    ipsec_psk = "terraform"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the VPN Customer Gateway.

* `cidr` - (Required) The CIDR block that needs to be routed through this gateway.

* `esp_policy` - (Required) The ESP policy to use for this VPN Customer Gateway.

* `gateway` - (Required) The public IP address of the related VPN Gateway.

* `ike_policy` - (Required) The IKE policy to use for this VPN Customer Gateway.

* `ipsec_psk` - (Required) The IPSEC pre-shared key used for this gateway.

* `dpd` - (Optional) If DPD is enabled for the related VPN connection (defaults false)

* `esp_lifetime` - (Optional) The ESP lifetime of phase 2 VPN connection to this
    VPN Customer Gateway in seconds (defaults 86400)

* `ike_lifetime` - (Optional) The IKE lifetime of phase 2 VPN connection to this
    VPN Customer Gateway in seconds (defaults 86400)

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the VPN Customer Gateway.
* `dpd` - Enable or disable DPD is enabled for the related VPN connection.
* `esp_lifetime` - The ESP lifetime of phase 2 VPN connection to this VPN Customer Gateway.
* `ike_lifetime` - The IKE lifetime of phase 2 VPN connection to this VPN Customer Gateway.
