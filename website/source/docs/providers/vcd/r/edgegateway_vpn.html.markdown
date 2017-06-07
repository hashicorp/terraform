---
layout: "vcd"
page_title: "vCloudDirector: vcd_edgegateway_vpn"
sidebar_current: "docs-vcd-resource-edgegateway-vpn"
description: |-
  Provides a vCloud Director IPsec VPN. This can be used to create, modify, and delete VPN settings and rules.
---

# vcd\_edgegateway\_vpn

Provides a vCloud Director IPsec VPN. This can be used to create,
modify, and delete VPN settings and rules.

## Example Usage

```
resource "vcd_edgegateway_vpn" "vpn" {
    edge_gateway        = "Internet_01(nti0000bi2_123-456-2)"
    name                = "west-to-east"
  description         = "Description"
  encryption_protocol = "AES256"
    mtu                 = 1400
    peer_id             = "64.121.123.11"
    peer_ip_address     = "64.121.123.11"
    local_id            = "64.121.123.10"
    local_ip_address    = "64.121.123.10"
    shared_secret       = "***********************"
    
    peer_subnets {
        peer_subnet_name = "DMZ_WEST"
        peer_subnet_gateway = "10.0.10.1"
        peer_subnet_mask = "255.255.255.0"
    }

    peer_subnets {
        peer_subnet_name = "WEB_WEST"
        peer_subnet_gateway = "10.0.20.1"
        peer_subnet_mask = "255.255.255.0"
    }

    local_subnets {
        local_subnet_name = "DMZ_EAST"
        local_subnet_gateway = "10.0.1.1"
        local_subnet_mask = "255.255.255.0"
    }

    local_subnets {
        local_subnet_name = "WEB_EAST"
        local_subnet_gateway = "10.0.22.1"
        local_subnet_mask = "255.255.255.0"
    }
}
```

## Argument Reference

The following arguments are supported:

* `edge_gateway` - (Required) The name of the edge gateway on which to apply the Firewall Rules
* `name` - (Required) The name of the VPN 
* `description` - (Required) A description for the VPN
* `encryption_protocol` - (Required) - E.g. `AES256`
* `local_ip_address` - (Required) - Local IP Address
* `local_id` - (Required) - Local ID
* `mtu` - (Required) - The MTU setting
* `peer_ip_address` - (Required) - Peer IP Address
* `peer_id` - (Required) - Peer ID
* `shared_secret` - (Required) - Shared Secret
* `local_subnets` - (Required) - List of Local Subnets see [Local Subnets](#localsubnets) below for details.
* `peer_subnets` - (Required) - List of Peer Subnets see [Peer Subnets](#peersubnets) below for details.

<a id="localsubnets"></a>
## Local Subnets

Each Local Subnet supports the following attributes:

* `local_subnet_name` - (Required) Name of the local subnet
* `local_subnet_gateway` - (Required) Gateway of the local subnet
* `local_subnet_mask` - (Required) Subnet mask of the local subnet

<a id="peersubnets"></a>
## Peer Subnets

Each Peer Subnet supports the following attributes:

* `peer_subnet_name` - (Required) Name of the peer subnet
* `peer_subnet_gateway` - (Required) Gateway of the peer subnet
* `peer_subnet_mask` - (Required) Subnet mask of the peer subnet