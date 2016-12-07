---
layout: "ddcloud"
page_title: "Dimension Data Managed Cloud Platform: nat"
sidebar_current: "docs-ddcloud-resource-nat"
description: |-
  Allows Terraform to manage a Managed Cloud Platform IPv4 NAT rule.
---

# ddcloud\_nat

A Network Address Translation (NAT) rule is part of the configuration for a network domain. It forwards traffic from a public IPv4 address to a private IPv4 address.

~> **Note:** Due to current infrastructure limitations, MCP 2.0 cannot perform more than one concurrent deployment operation for network domains and VLANs (all other operations can however be performed concurrently).  
If necessary, use the `depends_on` attribute to ensure that resources that relate to the same network domain are not run in parallel.

## Example Usage

```
resource "ddcloud_nat" "test-vm-nat" {
  networkdomain = "${ddcloud_networkdomain.my-domain.id}"
  private_ipv4	= "${ddcloud_server.my-server.primary_adapter_ipv4}"

  depends_on    = ["ddcloud_vlan.my-vlan"]
}
```

## Argument Reference

The following arguments are supported:

* `networkdomain` - (Required) The Id of the network domain to which the NAT rule applies.
* `private_ipv4` - (Required) The private IPv4 address to which traffic will be forwarded.
* `public_ipv4` - (Optional) A specific public IPv4 address from which traffic is to be forwarded.

## Attributes Reference

* `public_ipv4` - The public IPv4 address from which traffic is forwarded.  
If not specified as an argument, the first available public IP address will be used. If there are no public IPv4 addresses available, a new block will be allocated.
