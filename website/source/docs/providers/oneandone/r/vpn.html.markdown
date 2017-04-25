---
layout: "oneandone"
page_title: "1&1: oneandone_vpn"
sidebar_current: "docs-oneandone-resource-vpn"
description: |-
  Creates and manages 1&1 VPN.
---

# oneandone\_vpn

Manages a VPN on 1&1

## Example Usage

```hcl
resource "oneandone_public_ip" "ip" {
  "ip_type" = "IPV4"
  "reverse_dns" = "test.1and1.com"
  "datacenter" = "GB"
}
```

## Argument Reference

The following arguments are supported:

* `datacenter` - (Optional)[string] Location of desired 1and1 datacenter ["DE", "GB", "US", "ES" ]
* `ip_type` - (Required)[string] IPV4 or IPV6
* `reverese_dns` - [Optional](string)

