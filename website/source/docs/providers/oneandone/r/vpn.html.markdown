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

* `datacenter` - (Optional) Location of desired 1and1 datacenter. Can be `DE`, `GB`, `US` or `ES`.
* `ip_type` - (Required) IPV4 or IPV6
* `reverese_dns` - (Optional)

