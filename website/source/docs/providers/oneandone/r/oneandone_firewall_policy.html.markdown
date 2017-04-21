---
layout: "oneandone"
page_title: "1&1: oneandone_firewall_policy"
sidebar_current: "docs-oneandone-resource-firwall-policy"
description: |-
  Creates and manages 1&1 Firewall Policy.
---

# oneandone\_server

Manages a Firewall Policy on 1&1

## Example Usage

```
resource "oneandone_firewall_policy" "fw" {
  name = "test_fw_011"
  rules = [
    {
      "protocol" = "TCP"
      "port_from" = 80
      "port_to" = 80
      "source_ip" = "0.0.0.0"
    },
    {
      "protocol" = "ICMP"
      "source_ip" = "0.0.0.0"
    },
    {
      "protocol" = "TCP"
      "port_from" = 43
      "port_to" = 43
      "source_ip" = "0.0.0.0"
    },
    {
      "protocol" = "TCP"
      "port_from" = 22
      "port_to" = 22
      "source_ip" = "0.0.0.0"
    }
  ]
}
```

## Argument Reference

The following arguments are supported:

* `description` - (Optional) [string] Description for the VPN
* `name` - (Required) [string] The name of the VPN.

Firewall Policy Rules (`rules`) support the follwing:

* `protocol` - (Required) [String]  The protocol for the rule ["TCP", "UDP", "TCP/UDP", "ICMP", "IPSEC"]
* `port_from` - (Optional) [String]  Defines the start range of the allowed port
* `port_to` - (Optional) [String]  Defines the end range of the allowed port
* `source_ip` - (Optional) [String]  Only traffic directed to the respective IP address

