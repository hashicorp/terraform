---
layout: "oneandone"
page_title: "1&1: oneandone_private_network"
sidebar_current: "docs-oneandone-resource-private-network"
description: |-
  Creates and manages 1&1 Private Network.
---

# oneandone\_server

Manages a Private Network on 1&1

## Example Usage

```hcl
resource "oneandone_private_network" "pn" {
  name = "pn_test",
  description = "new stuff001"
  datacenter = "GB"
  network_address = "192.168.7.0"
  subnet_mask = "255.255.255.0"
  server_ids = [
    "${oneandone_server.server.id}",
    "${oneandone_server.server02.id}",
  ]
}
```

## Argument Reference

The following arguments are supported:

* `datacenter` - (Optional) Location of desired 1and1 datacenter. Can be `DE`, `GB`, `US` or `ES`.
* `description` - (Optional) Description for the shared storage
* `name` - (Required) The name of the private network
* `network_address` - (Optional) Network address for the private network
* `subnet_mask` - (Optional) Subnet mask for the private network
* `server_ids`  (Optional) List of servers that are to be associated with the private network
