---
layout: "oneandone"
page_title: "1&1: oneandone_public_ip"
sidebar_current: "docs-oneandone-resource-public-ip"
description: |-
  Creates and manages 1&1 Public IP.
---

# oneandone\_vpn

Manages a Public IP on 1&1

## Example Usage

```hcl
resource "oneandone_vpn" "vpn" {
  datacenter = "GB"
  name = "test_vpn_01"
  description = "ttest descr"
}
```

## Argument Reference

The following arguments are supported:

* `datacenter` - (Optional) Location of desired 1and1 datacenter. Can be `DE`, `GB`, `US` or `ES`
* `description` - (Optional) Description of the VPN
* `name` -(Required) The name of the VPN.
