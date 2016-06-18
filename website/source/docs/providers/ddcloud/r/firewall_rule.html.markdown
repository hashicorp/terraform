---
layout: "ddcloud"
page_title: "Dimension Data Managed Cloud Platform: firewall_rule"
sidebar_current: "docs-ddcloud-resource-firewall_rule"
description: |-
  Allows Terraform to manage a Managed Cloud Platform firewall rule.
---

# ddcloud\_firewall_rule

A firewall rule is part of the configuration for a network domain; it permits or denies traffic to and from IPv4 / IPv6 addresses within the network domain.

While all IPv6 addresses within a network domain are publicly routable, the default firewall configuration does not permit traffic from outside the network domain.

Most IPv4 addresses within a network domain are not publicly routable, and NAT rules must be set up to forward traffic to them from public IPv4 addresses. It is these public IPv4 addresses that must be referenced by firewall rules to permit external traffic.

~> **Note:** Due to current infrastructure limitations, MCP 2.0 cannot perform more than one concurrent deployment operation for network domains and VLANs (all other operations can however be performed concurrently).  
If necessary, use the `depends_on` attribute to ensure that resources that relate to the same network domain are not run in parallel.

## Example Usage

The following configuration permits TCP traffic over IPv4 on port 80 from any source address to the public address associated with a NAT rule.

```
resource "ddcloud_firewall_rule" "my_server_http_in" {
  name                = "test_vm.HTTP.Inbound"
  placement           = "first"
  action              = "accept"
  enabled             = true

  ip_version          = "ipv4"
  protocol            = "tcp"

  destination_address = "${ddcloud_nat.myserver_nat.public_ipv4}"
  destination_port    = "80"

  networkdomain       = "${ddcloud_networkdomain.mydomain.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A name for the firewall rule.  
Note that rule names can only contain letters, numbers, and periods (`.`).
* `description` - (Optional) A description for the firewall rule.
* `action` - (Required) The action performed by the firewall rule.  
Can be `accept` or `drop`.
* `enabled` - (Optional) Determines whether the firewall rule is enabled.  
Default is true.
* `placement` - (Required) Where in the firewall ACL this particular rule will be created.  
Can be one of `first`, `last`, `before`, or `after`.
* `placement_relative_to` - (Optional) When `placement` is `before` or `after`, specifies the name of the firewall rule to which the placement instruction refers.
* `ip_version` - (Required) The IP version to which the firewall rule applies.  
Can be `ipv4` or `ipv6`.
* `protocol` - (Required) The protocol to which the rule applies.  
Can be `ip`, `icmp`, `tcp`, or `udp`.
* `source_address` - (Optional) The source IP address to be matched by the rule.  
Cannot be specified with `source_network`.
* `source_network` - (Optional) The source network to be matched by the rule.  
Cannot be specified with `source_address`.
* `source_port` - (Optional) The source port (if any) to be matched by the rule.
* `destination_address` - (Optional) The destination IP address to be matched by the rule.  
Cannot be specified with `destination_network`.
* `destination_network` - (Optional) The destination network to be matched by the rule.  
Cannot be specified with `destination_address`.
* `destination_port` - (Optional) The destination port (if any) to be matched by the rule.
* `networkdomain` - (Required) The Id of the network domain to which the firewall rule applies.
* `private_ipv4` - (Required) The private IPv4 address to which traffic will be forwarded.
* `public_ipv4` - (Optional) A specific public IPv4 address from which traffic is to be forwarded.

## Attributes Reference

There are currently no additional attributes for `ddcloud_firewall_rule`.
